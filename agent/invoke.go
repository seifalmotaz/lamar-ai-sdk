package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
	"github.com/seifalmotaz/lamar-sdk/tool"
)

// InvokeOption configures a single invocation of the agent.
type InvokeOption func(*invokeConfig)

type invokeConfig struct {
	messages  []provider.Message
	system    *string
	callbacks Callbacks
}

// WithMessages sets the initial messages for this invocation.
func WithMessages(messages ...provider.Message) InvokeOption {
	return func(cfg *invokeConfig) {
		cfg.messages = messages
	}
}

// WithInvokeSystem overrides the system prompt for this invocation.
func WithInvokeSystem(system string) InvokeOption {
	return func(cfg *invokeConfig) {
		cfg.system = &system
	}
}

// WithInvokeCallbacks sets callbacks for this invocation.
// This is primarily used internally by Stream to inject event-emitting callbacks.
func WithInvokeCallbacks(cb Callbacks) InvokeOption {
	return func(cfg *invokeConfig) {
		cfg.callbacks = cb
	}
}

// mergeCallbacks merges two callback structs, using non-nil values from both.
// Values in 'overrides' take precedence over values in 'base'.
func mergeCallbacks(base, overrides Callbacks) Callbacks {
	result := base
	if overrides.OnStart != nil {
		result.OnStart = overrides.OnStart
	}
	if overrides.OnStepStart != nil {
		result.OnStepStart = overrides.OnStepStart
	}
	if overrides.OnStepFinish != nil {
		result.OnStepFinish = overrides.OnStepFinish
	}
	if overrides.OnToolCallStart != nil {
		result.OnToolCallStart = overrides.OnToolCallStart
	}
	if overrides.OnToolCallFinish != nil {
		result.OnToolCallFinish = overrides.OnToolCallFinish
	}
	if overrides.OnFinish != nil {
		result.OnFinish = overrides.OnFinish
	}
	if overrides.OnError != nil {
		result.OnError = overrides.OnError
	}
	return result
}

// Invoke runs the agent synchronously and returns the final result.
// The agent will continue calling the model and executing tools until
// a stop condition is met or the model doesn't request any tool calls.
//
// Example:
//
//	result, err := agent.Invoke(ctx,
//	    agent.WithMessages(
//	        provider.UserMessage("What's the weather in Tokyo?"),
//	    ),
//	)
func (a *Agent) Invoke(ctx context.Context, opts ...InvokeOption) (*Result, error) {
	cfg := &invokeConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	messages := cfg.messages
	if len(messages) == 0 {
		return nil, NewError("invalid_input", "no messages provided", 0, nil)
	}

	// Merge callbacks: agent's callbacks as base, config callbacks as overrides
	callbacks := mergeCallbacks(a.callbacks, cfg.callbacks)

	steps := make([]StepResult, 0)
	totalUsage := provider.Usage{}

	if callbacks.OnStart != nil {
		if err := callbacks.OnStart(ctx, messages); err != nil {
			return nil, NewError("callback_error", "OnStart callback failed", 0, err)
		}
	}

	for {
		if ctx.Err() != nil {
			return nil, NewError("context_canceled", "context canceled", len(steps), ctx.Err())
		}

		for _, cond := range a.stopWhen {
			if cond(ctx, steps) {
				result := a.buildResult(steps, messages)
				if callbacks.OnFinish != nil {
					if err := callbacks.OnFinish(ctx, result); err != nil {
						return nil, NewError("callback_error", "OnFinish callback failed", len(steps), err)
					}
				}
				return result, nil
			}
		}

		stepNumber := len(steps) + 1

		stepModel := a.model
		stepTools := a.tools
		stepToolChoice := a.toolChoice
		stepSystem := a.system
		if cfg.system != nil {
			stepSystem = *cfg.system
		}
		var stepMaxTokens *int
		var stepTemperature *float64

		if a.prepare != nil {
			prepareResult := a.prepare(ctx, PrepareStepParams{
				StepNumber:      stepNumber,
				Steps:           steps,
				Model:           a.model,
				Messages:        messages,
				ToolCalls:       a.lastToolCalls(steps),
				ExperimentalCtx: a.experimentalCtx,
			})
			if prepareResult != nil {
				if prepareResult.Model != nil {
					stepModel = prepareResult.Model
				}
				if prepareResult.Tools != nil {
					stepTools = prepareResult.Tools
				}
				if prepareResult.ToolChoice != nil {
					stepToolChoice = *prepareResult.ToolChoice
				}
				if prepareResult.System != nil {
					stepSystem = *prepareResult.System
				}
				if prepareResult.MaxTokens != nil {
					stepMaxTokens = prepareResult.MaxTokens
				}
				if prepareResult.Temperature != nil {
					stepTemperature = prepareResult.Temperature
				}
				if prepareResult.ExperimentalCtx != nil {
					a.experimentalCtx = prepareResult.ExperimentalCtx
				}
			}
		}

		if callbacks.OnStepStart != nil {
			if err := callbacks.OnStepStart(ctx, stepNumber, messages); err != nil {
				return nil, NewError("callback_error", "OnStepStart callback failed", len(steps), err)
			}
		}

		stepStart := time.Now()

		var reqMessages []provider.Message
		if stepSystem != "" {
			reqMessages = append(reqMessages, provider.SystemMessage(stepSystem))
		}
		reqMessages = append(reqMessages, messages...)

		req := &provider.GenerateRequest{
			Messages: reqMessages,
			Config: provider.Config{
				Tools:      tool.ToDefinitions(stepTools...),
				ToolChoice: stepToolChoice,
			},
		}

		if stepMaxTokens != nil {
			req.Config.MaxTokens = *stepMaxTokens
		} else if a.maxTokens != nil {
			req.Config.MaxTokens = *a.maxTokens
		}

		if stepTemperature != nil {
			req.Config.Temperature = *stepTemperature
		} else if a.temperature != nil {
			req.Config.Temperature = *a.temperature
		}

		response, err := a.callModelWithRetry(ctx, stepModel, req)
		if err != nil {
			if callbacks.OnError != nil {
				if cbErr := callbacks.OnError(ctx, stepNumber, err); cbErr != nil {
					return nil, NewError("model_error", "model call failed", stepNumber, cbErr)
				}
				continue
			}
			return nil, NewError("model_error", "model call failed", stepNumber, err)
		}

		toolResults := make([]ToolExecutionResult, 0, len(response.ToolCalls))
		for _, tc := range response.ToolCalls {
			if callbacks.OnToolCallStart != nil {
				if err := callbacks.OnToolCallStart(ctx, tc); err != nil {
					return nil, NewError("callback_error", "OnToolCallStart callback failed", len(steps), err)
				}
			}

			result := a.executeTool(ctx, tc, stepTools)
			if callbacks.OnToolCallFinish != nil {
				if err := callbacks.OnToolCallFinish(ctx, result); err != nil {
					return nil, NewError("callback_error", "OnToolCallFinish callback failed", len(steps), err)
				}
			}
			toolResults = append(toolResults, result)
		}

		stepResult := StepResult{
			StepNumber:   stepNumber,
			Content:      response.Content,
			ToolCalls:    response.ToolCalls,
			ToolResults:  toolResults,
			Usage:        response.Usage,
			FinishReason: response.FinishReason,
			Model: ModelInfo{
				Provider: stepModel.Provider(),
				ModelID:  stepModel.ModelID(),
			},
			Duration: time.Since(stepStart),
		}
		steps = append(steps, stepResult)
		totalUsage = totalUsage.Add(response.Usage)

		if callbacks.OnStepFinish != nil {
			if err := callbacks.OnStepFinish(ctx, stepResult); err != nil {
				return nil, NewError("callback_error", "OnStepFinish callback failed", len(steps), err)
			}
		}

		if len(response.ToolCalls) == 0 {
			result := a.buildResult(steps, messages)
			if callbacks.OnFinish != nil {
				if err := callbacks.OnFinish(ctx, result); err != nil {
					return nil, NewError("callback_error", "OnFinish callback failed", len(steps), err)
				}
			}
			return result, nil
		}

		// Append assistant message with content and tool calls
		if len(response.Content) > 0 || len(response.ToolCalls) > 0 {
			toolCallContents := make([]provider.ToolCallContent, len(response.ToolCalls))
			for i, tc := range response.ToolCalls {
				toolCallContents[i] = provider.NewToolCallContent(tc.ID, tc.Name, tc.Input)
			}

			assistantContent := make([]provider.Content, 0, len(response.Content)+len(toolCallContents))
			assistantContent = append(assistantContent, response.Content...)
			for _, tc := range toolCallContents {
				assistantContent = append(assistantContent, tc)
			}

			messages = append(messages, provider.Message{
				Role:    provider.RoleAssistant,
				Content: assistantContent,
			})
		}

		// Append tool results
		for _, tr := range toolResults {
			var resultJSON json.RawMessage
			if tr.Error != nil {
				// Marshal error message as JSON string for safety
				resultJSON, _ = json.Marshal(tr.Error.Error())
			} else {
				resultJSON, _ = json.Marshal(tr.Output)
			}
			messages = append(messages, provider.ToolResultMessage(
				provider.NewToolResultContent(tr.ToolCallID, tr.ToolName, resultJSON, tr.Error != nil),
			))
		}
	}
}

func (a *Agent) callModelWithRetry(ctx context.Context, model provider.LanguageModel, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	var lastErr error
	for attempt := 0; attempt <= a.maxRetries; attempt++ {
		response, err := model.Generate(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err
		if !isErrorRetryable(err) {
			return nil, err
		}

		// Check for RetryAfter from provider (e.g., rate limit headers)
		var backoff time.Duration
		if retryAfter := provider.RetryAfter(err); retryAfter > 0 {
			backoff = retryAfter
		} else {
			// Exponential backoff: 1s, 2s, 4s, ...
			backoff = time.Duration(1<<uint(attempt)) * time.Second
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(backoff):
			continue
		}
	}
	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isErrorRetryable(err error) bool {
	var genErr *provider.Error
	if errors.As(err, &genErr) {
		return genErr.Code == provider.CodeRateLimited ||
			genErr.Code == provider.CodeAPITimeout ||
			genErr.Code == provider.CodeUnknown
	}
	return false
}

func (a *Agent) executeTool(ctx context.Context, tc provider.ToolCall, tools []tool.Tool) ToolExecutionResult {
	start := time.Now()

	var targetTool tool.Tool
	if a.toolMap != nil {
		targetTool = a.toolMap[tc.Name]
	} else {
		// Fallback for edge cases (shouldn't happen in normal use)
		for _, t := range tools {
			if t.Name() == tc.Name {
				targetTool = t
				break
			}
		}
	}

	if targetTool == nil {
		return ToolExecutionResult{
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Input:      tc.Input,
			Error:      &ToolNotFoundError{ToolName: tc.Name},
			Duration:   time.Since(start),
		}
	}

	output, err := targetTool.Execute(ctx, tc.Input)
	duration := time.Since(start)

	if err != nil {
		return ToolExecutionResult{
			ToolCallID: tc.ID,
			ToolName:   tc.Name,
			Input:      tc.Input,
			Error:      err,
			Duration:   duration,
		}
	}

	var parsedOutput any
	if err := json.Unmarshal(output, &parsedOutput); err != nil {
		parsedOutput = string(output)
	}

	return ToolExecutionResult{
		ToolCallID: tc.ID,
		ToolName:   tc.Name,
		Input:      tc.Input,
		Output:     parsedOutput,
		Duration:   duration,
	}
}

func (a *Agent) lastToolCalls(steps []StepResult) []provider.ToolCall {
	if len(steps) == 0 {
		return nil
	}
	return steps[len(steps)-1].ToolCalls
}

func (a *Agent) buildResult(steps []StepResult, messages []provider.Message) *Result {
	if len(steps) == 0 {
		return &Result{
			Steps:         steps,
			FinalMessages: messages,
		}
	}

	last := steps[len(steps)-1]
	totalUsage := provider.Usage{}
	totalDuration := time.Duration(0)

	for _, s := range steps {
		totalUsage = totalUsage.Add(s.Usage)
		totalDuration += s.Duration
	}

	return &Result{
		Steps:         steps,
		FinalMessages: messages,
		FinalContent:  last.Content,
		FinalText:     ExtractText(last.Content),
		TotalUsage:    totalUsage,
		TotalDuration: totalDuration,
		FinishReason:  last.FinishReason,
	}
}
