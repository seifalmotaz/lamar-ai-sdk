package agent

import (
	"github.com/seifalmotaz/lamar-sdk/provider"
	"github.com/seifalmotaz/lamar-sdk/tool"
)

// Agent orchestrates multi-step LLM tool-calling loops.
// It handles the cycle of: call model → execute tools → call model again.
//
// Use New to create an Agent with the default configuration,
// then use Invoke for synchronous execution or Stream for event-based streaming.
type Agent struct {
	model           provider.LanguageModel
	tools           []tool.Tool
	toolMap         map[string]tool.Tool
	stopWhen        []StopCondition
	prepare         PrepareStep
	callbacks       Callbacks
	maxRetries      int
	system          string
	toolChoice      provider.ToolChoice
	temperature     *float64
	maxTokens       *int
	experimentalCtx any
}

// New creates a new Agent with the given model and optional configuration.
// The default configuration is:
//   - No tools (model can't call tools)
//   - Stop after 1 step (single call, no loop)
//   - No callbacks
//   - Max 2 retries
//
// Use Option functions to customize the agent:
//
//	agent := agent.New(model,
//	    agent.WithTools(weatherTool, calculatorTool),
//	    agent.WithStopWhen(agent.StepCountIs(10)),
//	    agent.WithCallbacks(agent.Callbacks{
//	        OnStepStart: func(ctx context.Context, stepNumber int, messages []provider.Message) error {
//	            log.Printf("Step %d starting", stepNumber)
//	            return nil
//	        },
//	    }),
//	)
func New(model provider.LanguageModel, opts ...Option) *Agent {
	a := &Agent{
		model:      model,
		stopWhen:   []StopCondition{StepCountIs(1)},
		maxRetries: 2,
		toolChoice: provider.ToolChoiceAuto(),
	}
	for _, opt := range opts {
		opt(a)
	}
	// Build tool map for O(1) lookup
	a.toolMap = make(map[string]tool.Tool, len(a.tools))
	for _, t := range a.tools {
		a.toolMap[t.Name()] = t
	}
	return a
}
