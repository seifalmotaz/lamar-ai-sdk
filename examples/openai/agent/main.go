// Package main demonstrates tool calling with the agent package.
// The agent package handles multi-turn tool-calling loops automatically.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/agent"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-ai-sdk/tool"
)

type WeatherInput struct {
	Location string `json:"location" jsonschema:"required,description=City name"`
	Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit,description=Temperature unit"`
}

type WeatherOutput struct {
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT5Mini()

	// Create tools
	weatherTool := tool.NewTool("get_weather", "Get current weather for a location",
		func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
			fmt.Printf("[Tool] Getting weather for: %s\n", input.Location)
			return WeatherOutput{
				Temperature: 22.5,
				Condition:   "sunny",
				Humidity:    65,
			}, nil
		},
	)

	// Create an agent that will run until it gets a final answer
	// The agent uses IsStopMessage internally to detect terminal responses
	ag := agent.New(model,
		agent.WithTools(weatherTool),
		agent.WithStopWhen(agent.StepCountIs(10)), // Max iterations
	)

	fmt.Println("Agent tool-calling example:")
	fmt.Println("---------------------------")

	// Run the agent
	result, err := ag.Invoke(context.Background(),
		agent.WithMessages(
			provider.UserMessage("What's the weather like in Tokyo? Should I bring an umbrella?"),
		),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Print final response
	fmt.Printf("\nFinal response:\n%s\n", result.FinalText)
	fmt.Printf("\nSteps: %d\n", len(result.Steps))
	fmt.Printf("Total tokens: %d\n", result.TotalUsage.TotalTokens)
}
