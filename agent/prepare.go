package agent

import (
	"context"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/tool"
)

// PrepareStep is a function called before each step to allow dynamic configuration.
// It receives context and parameters about the current state of execution.
// Return nil to use defaults, or return updates to override settings for the next step.
//
// Example: Switch to a cheaper model after 5 steps for cost optimization.
//
//	agent := agent.New(model,
//	    agent.WithPrepareStep(func(ctx context.Context, p agent.PrepareStepParams) *agent.PrepareStepResult {
//	        if p.StepNumber > 5 {
//	            return &agent.PrepareStepResult{
//	                Model: cheapModel,
//	            }
//	        }
//	        return nil
//	    }),
//	)
type PrepareStep func(ctx context.Context, params PrepareStepParams) *PrepareStepResult

// PrepareStepParams provides context for the prepare step function.
type PrepareStepParams struct {
	// StepNumber is the current step number (1-indexed).
	StepNumber int

	// Steps contains all previously executed steps.
	Steps []StepResult

	// Model is the current model being used.
	Model provider.LanguageModel

	// Messages is the current message history.
	Messages []provider.Message

	// ToolCalls from the last step (if any).
	ToolCalls []provider.ToolCall

	// ExperimentalCtx is custom context passed through steps.
	ExperimentalCtx any
}

// PrepareStepResult allows overriding settings for the next step.
// All fields are optional - nil values keep current settings.
type PrepareStepResult struct {
	// Model overrides the model for this step (nil = keep current).
	Model provider.LanguageModel

	// Tools overrides the available tools (nil = keep current).
	Tools []tool.Tool

	// ToolChoice overrides the tool choice strategy.
	ToolChoice *provider.ToolChoice

	// System overrides the system prompt.
	System *string

	// MaxTokens overrides the max tokens.
	MaxTokens *int

	// Temperature overrides the temperature.
	Temperature *float64

	// ExperimentalCtx updates the custom context.
	ExperimentalCtx any
}
