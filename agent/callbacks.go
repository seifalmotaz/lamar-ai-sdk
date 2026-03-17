package agent

import (
	"context"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// OnStart is called before agent execution begins.
// Return an error to abort execution.
type OnStart func(ctx context.Context, messages []provider.Message) error

// OnStepStart is called before each step.
// Return an error to abort execution.
type OnStepStart func(ctx context.Context, stepNumber int, messages []provider.Message) error

// OnStepFinish is called after each step completes.
// Return an error to abort execution.
type OnStepFinish func(ctx context.Context, result StepResult) error

// OnToolCallStart is called before executing a tool.
// Return an error to abort execution.
type OnToolCallStart func(ctx context.Context, toolCall provider.ToolCall) error

// OnToolCallFinish is called after tool execution completes.
// Return an error to abort execution.
type OnToolCallFinish func(ctx context.Context, result ToolExecutionResult) error

// OnFinish is called after agent execution completes successfully.
// Return an error to signal a failure.
type OnFinish func(ctx context.Context, result *Result) error

// OnError is called when an error occurs during execution.
// Return nil to suppress the error and continue execution to the next step.
// Return a non-nil error to propagate that error and stop execution.
//
// Example - Suppress error and continue:
//
//	OnError: func(ctx context.Context, stepNumber int, err error) error {
//	    log.Printf("Step %d failed: %v, continuing...", stepNumber, err)
//	    return nil // Suppress error, continue execution
//	}
//
// Example - Transform error:
//
//	OnError: func(ctx context.Context, stepNumber int, err error) error {
//	    return fmt.Errorf("step %d failed: %w", stepNumber, err)
//	}
type OnError func(ctx context.Context, stepNumber int, err error) error

// Callbacks groups all callback functions for the agent.
// All callbacks are optional - nil callbacks are skipped.
type Callbacks struct {
	// OnStart is called before agent execution begins.
	OnStart OnStart

	// OnStepStart is called before each step.
	OnStepStart OnStepStart

	// OnStepFinish is called after each step completes.
	OnStepFinish OnStepFinish

	// OnToolCallStart is called before executing a tool.
	OnToolCallStart OnToolCallStart

	// OnToolCallFinish is called after tool execution completes.
	OnToolCallFinish OnToolCallFinish

	// OnFinish is called after agent execution completes.
	OnFinish OnFinish

	// OnError is called when an error occurs.
	// If nil is returned, the error is suppressed and execution continues.
	// If a non-nil error is returned, that error is propagated instead.
	OnError OnError
}
