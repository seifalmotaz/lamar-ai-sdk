// Package agent provides multi-step LLM tool-calling loops with stop conditions.
//
// The agent package orchestrates the cycle of: call model → execute tools → call model again
// until a stop condition is met or the model doesn't request any tool calls.
//
// # Basic Usage
//
// Create an agent with a model and tools:
//
//	agent := agent.New(model,
//	    agent.WithTools(weatherTool, calculatorTool),
//	    agent.WithStopWhen(agent.StepCountIs(10)),
//	)
//
// Run the agent synchronously:
//
//	result, err := agent.Invoke(ctx,
//	    agent.WithMessages(
//	        provider.UserMessage("What's the weather in Tokyo?"),
//	    ),
//	)
//
// Or stream events:
//
//	stream := agent.Stream(ctx,
//	    agent.WithMessages(
//	        provider.UserMessage("Calculate 42 * 7"),
//	    ),
//	)
//	for event := range stream {
//	    switch e := event.(type) {
//	    case agent.StreamEventStepStart:
//	        fmt.Printf("Step %d starting\n", e.StepNumber)
//	    case agent.StreamEventToolCall:
//	        fmt.Printf("Tool call: %s\n", e.ToolCall.Name)
//	    case agent.StreamEventFinish:
//	        fmt.Printf("Done! Final text: %s\n", e.Result.FinalText)
//	    }
//	}
//
// # Stop Conditions
//
// Stop conditions control when the agent loop terminates:
//
//	// Stop after 5 steps
//	agent.New(model, agent.WithStopWhen(agent.StepCountIs(5)))
//
//	// Stop when a specific tool is called
//	agent.New(model, agent.WithStopWhen(agent.HasToolCall("submit_answer")))
//
//	// Stop when any condition is met
//	agent.New(model, agent.WithStopWhen(
//	    agent.StopWhenAny(
//	        agent.StepCountIs(10),
//	        agent.HasToolCall("finish"),
//	    ),
//	))
//
// # Dynamic Configuration
//
// Use PrepareStep to change model, tools, or settings between steps:
//
//	agent := agent.New(model,
//	    agent.WithPrepareStep(func(ctx context.Context, p agent.PrepareStepParams) *agent.PrepareStepResult {
//	        if p.StepNumber > 3 {
//	            return &agent.PrepareStepResult{
//	                Model: cheapModel,
//	            }
//	        }
//	        return nil
//	    }),
//	)
//
// # Callbacks
//
// Add callbacks for observability:
//
//	agent := agent.New(model,
//	    agent.WithCallbacks(agent.Callbacks{
//	        OnStepStart: func(ctx context.Context, stepNumber int, messages []provider.Message) error {
//	            log.Printf("Step %d starting with %d messages", stepNumber, len(messages))
//	            return nil
//	        },
//	        OnToolCallStart: func(ctx context.Context, tc provider.ToolCall) error {
//	            log.Printf("Calling tool: %s", tc.Name)
//	            return nil
//	        },
//	        OnToolCallFinish: func(ctx context.Context, result agent.ToolExecutionResult) error {
//	            log.Printf("Tool %s completed in %v", result.ToolName, result.Duration)
//	            return nil
//	        },
//	    }),
//	)
//
// # Result
//
// The Result contains all executed steps and accumulated usage:
//
//	result, err := agent.Invoke(ctx, ...)
//	for _, step := range result.Steps {
//	    fmt.Printf("Step %d: %d tool calls, %d tokens\n",
//	        step.StepNumber,
//	        len(step.ToolCalls),
//	        step.Usage.TotalTokens,
//	    )
//	}
//	fmt.Printf("Total tokens: %d\n", result.TotalUsage.TotalTokens)
//	fmt.Printf("Final text: %s\n", result.FinalText)
package agent
