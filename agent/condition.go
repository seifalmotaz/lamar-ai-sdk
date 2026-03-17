package agent

import (
	"context"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// StopCondition is a function that determines when the agent should stop.
// It receives the context for cancellation and all executed steps so far.
// Return true to stop the agent, false to continue execution.
type StopCondition func(ctx context.Context, steps []StepResult) bool

// StepCountIs returns a StopCondition that stops after exactly n steps.
// This is equivalent to "run exactly n iterations".
//
// Example:
//
//	agent := agent.New(model, agent.WithStopWhen(agent.StepCountIs(5)))
func StepCountIs(n int) StopCondition {
	return func(_ context.Context, steps []StepResult) bool {
		return len(steps) >= n
	}
}

// HasToolCall returns a StopCondition that stops when a specific tool is called.
// This is useful for agents that should stop when they call a particular tool.
//
// Example:
//
//	agent := agent.New(model, agent.WithStopWhen(agent.HasToolCall("submit_answer")))
func HasToolCall(name string) StopCondition {
	return func(_ context.Context, steps []StepResult) bool {
		if len(steps) == 0 {
			return false
		}
		last := steps[len(steps)-1]
		for _, tc := range last.ToolCalls {
			if tc.Name == name {
				return true
			}
		}
		return false
	}
}

// HasFinishReason returns a StopCondition that stops when the model finishes
// with one of the specified finish reasons.
//
// Example:
//
//	agent := agent.New(model,
//	    agent.WithStopWhen(agent.HasFinishReason(
//	        provider.FinishReasonStop,
//	        provider.FinishReasonLength,
//	    )),
//	)
func HasFinishReason(reasons ...provider.FinishReason) StopCondition {
	reasonSet := make(map[provider.FinishReason]bool, len(reasons))
	for _, r := range reasons {
		reasonSet[r] = true
	}
	return func(_ context.Context, steps []StepResult) bool {
		if len(steps) == 0 {
			return false
		}
		last := steps[len(steps)-1]
		return reasonSet[last.FinishReason]
	}
}

// StopWhenAny returns a StopCondition that stops when any of the given conditions is true.
//
// Example:
//
//	agent := agent.New(model,
//	    agent.WithStopWhen(agent.StopWhenAny(
//	        agent.StepCountIs(10),
//	        agent.HasToolCall("finish"),
//	    )),
//	)
func StopWhenAny(conditions ...StopCondition) StopCondition {
	return func(ctx context.Context, steps []StepResult) bool {
		for _, cond := range conditions {
			if cond(ctx, steps) {
				return true
			}
		}
		return false
	}
}

// StopWhenAll returns a StopCondition that stops when all of the given conditions are true.
//
// Example:
//
//	agent := agent.New(model,
//	    agent.WithStopWhen(agent.StopWhenAll(
//	        agent.StepCountAtLeast(3),
//	        agent.HasToolCall("submit"),
//	    )),
//	)
func StopWhenAll(conditions ...StopCondition) StopCondition {
	return func(ctx context.Context, steps []StepResult) bool {
		for _, cond := range conditions {
			if !cond(ctx, steps) {
				return false
			}
		}
		return true
	}
}
