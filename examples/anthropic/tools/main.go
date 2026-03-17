// Package main demonstrates tool calling with Anthropic Claude.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/anthropic"
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
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	weatherTool := tool.NewTool("get_weather", "Get current weather for a location",
		func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
			fmt.Printf("\n[Tool called] Getting weather for: %s\n", input.Location)
			return WeatherOutput{
				Temperature: 22.5,
				Condition:   "sunny",
				Humidity:    65,
			}, nil
		},
	)

	fmt.Println("Tool calling example:")
	fmt.Println("--------------------")

	result, err := generate.Generate(context.Background(), model,
		"What's the weather like in Tokyo?",
		generate.Tools(tool.ToDefinition(weatherTool)),
		generate.MaxTokens(200),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	toolCalls := result.ToolCalls()
	if len(toolCalls) > 0 {
		fmt.Printf("Model requested %d tool call(s):\n", len(toolCalls))

		for _, tc := range toolCalls {
			fmt.Printf("\nTool: %s\n", tc.Name)
			fmt.Printf("Input: %s\n", string(tc.Input))

			output, err := weatherTool.Execute(context.Background(), tc.Input)
			if err != nil {
				fmt.Printf("Error executing tool: %v\n", err)
				continue
			}

			var weatherOutput WeatherOutput
			json.Unmarshal(output, &weatherOutput)
			fmt.Printf("Result: Temperature=%.1f°C, Condition=%s, Humidity=%d%%\n",
				weatherOutput.Temperature,
				weatherOutput.Condition,
				weatherOutput.Humidity,
			)
		}

		messages := []provider.Message{
			provider.UserMessage("What's the weather like in Tokyo?"),
			{Role: provider.RoleAssistant, Content: result.Content()},
		}
		for _, tc := range toolCalls {
			output, _ := weatherTool.Execute(context.Background(), tc.Input)
			messages = append(messages, provider.ToolResultMessage(
				provider.NewToolResultContent(tc.ID, tc.Name, output, false),
			))
		}

		result2, err := generate.Generate(context.Background(), model, "",
			generate.Messages(messages...),
			generate.Tools(tool.ToDefinition(weatherTool)),
			generate.MaxTokens(200),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("\nFinal response: %s\n", result2.Text())
	} else {
		fmt.Println("No tool calls requested.")
		fmt.Printf("Response: %s\n", result.Text())
	}

	fmt.Printf("\nFinish reason: %s\n", result.FinishReason())
	fmt.Printf("Tokens: %d\n", result.Usage().TotalTokens)
}
