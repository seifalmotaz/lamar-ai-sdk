package agent

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// StreamEvent represents an event during streaming execution.
// Use type assertion to determine the specific event type.
type StreamEvent interface {
	eventType() string
}

// StreamEventStepStart is emitted when a step begins.
type StreamEventStepStart struct {
	// StepNumber is the 1-indexed step number.
	StepNumber int

	// Messages is the current message history.
	Messages []provider.Message
}

func (e StreamEventStepStart) eventType() string { return "step_start" }

// StreamEventContentDelta is emitted for each content delta during streaming.
// This is emitted during the model response within a step.
type StreamEventContentDelta struct {
	// Delta is the text content chunk from the model.
	Delta string
}

func (e StreamEventContentDelta) eventType() string { return "content_delta" }

// StreamEventToolCall is emitted when the model requests a tool.
type StreamEventToolCall struct {
	// ToolCall contains the tool call details.
	ToolCall provider.ToolCall
}

func (e StreamEventToolCall) eventType() string { return "tool_call" }

// StreamEventToolResult is emitted when a tool completes.
type StreamEventToolResult struct {
	// Result contains the tool execution result.
	Result ToolExecutionResult
}

func (e StreamEventToolResult) eventType() string { return "tool_result" }

// StreamEventStepFinish is emitted when a step completes.
type StreamEventStepFinish struct {
	// Result contains the step result.
	Result StepResult
}

func (e StreamEventStepFinish) eventType() string { return "step_finish" }

// StreamEventFinish is emitted when the agent finishes.
type StreamEventFinish struct {
	// Result contains the final result.
	Result *Result
}

func (e StreamEventFinish) eventType() string { return "finish" }

// StreamEventError is emitted when an error occurs.
type StreamEventError struct {
	// Error contains the error.
	Error error
}

func (e StreamEventError) eventType() string { return "error" }

// StreamOption configures a streaming invocation.
type StreamOption func(*streamConfig)

type streamConfig struct {
	messages          []provider.Message
	system            *string
	overrideCallbacks Callbacks
}

// WithStreamMessages sets the initial messages for this streaming invocation.
func WithStreamMessages(messages ...provider.Message) StreamOption {
	return func(cfg *streamConfig) {
		cfg.messages = messages
	}
}

// WithStreamSystem overrides the system prompt for this streaming invocation.
func WithStreamSystem(system string) StreamOption {
	return func(cfg *streamConfig) {
		cfg.system = &system
	}
}

// WithStreamCallbacks sets callbacks for streaming that emit events.
func WithStreamCallbacks(cb Callbacks) StreamOption {
	return func(cfg *streamConfig) {
		cfg.overrideCallbacks = cb
	}
}

// Stream runs the agent and returns a channel of events.
// The channel is closed when execution completes or when an error occurs.
// Context cancellation will stop the stream.
//
// Example:
//
//	stream := agent.Stream(ctx,
//	    agent.WithMessages(
//	        provider.UserMessage("Calculate 42 * 7"),
//	    ),
//	)
//
//	for event := range stream {
//	    switch e := event.(type) {
//	    case agent.StreamEventStepStart:
//	        fmt.Printf("Step %d starting\n", e.StepNumber)
//	    case agent.StreamEventToolCall:
//	        fmt.Printf("Tool call: %s\n", e.ToolCall.Name)
//	    case agent.StreamEventFinish:
//	        fmt.Printf("Done! Final text: %s\n", e.Result.FinalText)
//	    case agent.StreamEventError:
//	        log.Printf("Error: %v", e.Error)
//	    }
//	}
func (a *Agent) Stream(ctx context.Context, opts ...StreamOption) <-chan StreamEvent {
	events := make(chan StreamEvent, 64)

	cfg := &streamConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	go func() {
		defer close(events)

		// Validate messages
		if len(cfg.messages) == 0 {
			events <- StreamEventError{Error: NewError("invalid_input", "no messages provided", 0, nil)}
			return
		}

		// Create stream-emitting callbacks and merge with override callbacks
		streamCallbacks := toStreamCallbacks(events)
		mergedCallbacks := mergeCallbacks(streamCallbacks, cfg.overrideCallbacks)

		invokeOpts := []InvokeOption{
			WithMessages(cfg.messages...),
			WithInvokeCallbacks(mergedCallbacks),
		}
		if cfg.system != nil {
			invokeOpts = append(invokeOpts, WithInvokeSystem(*cfg.system))
		}

		result, err := a.Invoke(ctx, invokeOpts...)
		if err != nil {
			events <- StreamEventError{Error: err}
			return
		}

		events <- StreamEventFinish{Result: result}
	}()

	return events
}

// toStreamCallbacks creates callbacks that emit stream events.
// This allows streaming events while still getting the complete result at the end.
func toStreamCallbacks(events chan<- StreamEvent) Callbacks {
	return Callbacks{
		OnStepStart: func(ctx context.Context, stepNumber int, messages []provider.Message) error {
			events <- StreamEventStepStart{
				StepNumber: stepNumber,
				Messages:   messages,
			}
			return nil
		},
		OnToolCallStart: func(ctx context.Context, tc provider.ToolCall) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case events <- StreamEventToolCall{ToolCall: tc}:
				return nil
			}
		},
		OnToolCallFinish: func(ctx context.Context, result ToolExecutionResult) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case events <- StreamEventToolResult{Result: result}:
				return nil
			}
		},
		OnStepFinish: func(ctx context.Context, result StepResult) error {
			events <- StreamEventStepFinish{Result: result}
			return nil
		},
	}
}

// StreamWithTimeout runs the agent with a timeout.
// IMPORTANT: The returned channel MUST be fully drained to prevent goroutine leaks.
// Events are buffered (64 capacity), so small responses won't block, but for
// large responses or if you don't need all events, use context cancellation instead.
func (a *Agent) StreamWithTimeout(ctx context.Context, timeout time.Duration, opts ...StreamOption) <-chan StreamEvent {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	stream := a.Stream(ctx, opts...)

	// Wrap stream to ensure cancellation when done
	wrapped := make(chan StreamEvent, 64)
	go func() {
		defer cancel()
		defer close(wrapped)
		for event := range stream {
			wrapped <- event
		}
	}()

	return wrapped
}
