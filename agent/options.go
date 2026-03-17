package agent

import (
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/tool"
)

// Option configures an Agent.
type Option func(*Agent)

// WithTools sets the tools available to the agent.
func WithTools(tools ...tool.Tool) Option {
	return func(a *Agent) {
		a.tools = tools
		a.toolMap = make(map[string]tool.Tool, len(tools))
		for _, t := range tools {
			a.toolMap[t.Name()] = t
		}
	}
}

// WithStopWhen sets stop conditions.
// The agent will stop when ANY of the conditions is true.
// Default is StepCountIs(1) (single step, no tool loop).
//
// Example:
//
//	agent := agent.New(model,
//	    agent.WithTools(weatherTool),
//	    agent.WithStopWhen(agent.StepCountIs(10)),
//	)
func WithStopWhen(conditions ...StopCondition) Option {
	return func(a *Agent) {
		// If no conditions provided, use default single step
		if len(conditions) == 0 {
			a.stopWhen = []StopCondition{StepCountIs(1)}
			return
		}
		a.stopWhen = conditions
	}
}

// WithPrepareStep sets the prepare step function for dynamic configuration.
// The function is called before each step and can override model, tools, etc.
func WithPrepareStep(fn PrepareStep) Option {
	return func(a *Agent) {
		a.prepare = fn
	}
}

// WithCallbacks sets callback functions for observability and side effects.
func WithCallbacks(cb Callbacks) Option {
	return func(a *Agent) {
		a.callbacks = cb
	}
}

// WithMaxRetries sets the maximum retries for model calls.
// Default is 2. Set to 0 for no retries.
func WithMaxRetries(n int) Option {
	return func(a *Agent) {
		a.maxRetries = n
	}
}

// WithSystem sets the system prompt for the agent.
func WithSystem(system string) Option {
	return func(a *Agent) {
		a.system = system
	}
}

// WithToolChoice sets the tool choice strategy.
func WithToolChoice(choice provider.ToolChoice) Option {
	return func(a *Agent) {
		a.toolChoice = choice
	}
}

// WithExperimentalContext sets custom context that is passed through all prepare step calls.
func WithExperimentalContext(ctx any) Option {
	return func(a *Agent) {
		a.experimentalCtx = ctx
	}
}

// WithTemperature sets the temperature for model calls.
func WithTemperature(temp float64) Option {
	return func(a *Agent) {
		a.temperature = &temp
	}
}

// WithMaxTokens sets the maximum tokens for model calls.
func WithMaxTokens(maxTokens int) Option {
	return func(a *Agent) {
		a.maxTokens = &maxTokens
	}
}
