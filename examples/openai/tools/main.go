// Package main demonstrates tool calling with OpenAI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
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

	// Create a type-safe tool
	weatherTool := tool.NewTool("get_weather", "Get current weather for a location",
		func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
			// Mock implementation - integrate with real weather API
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
		"What's the weather like in Tokyo and San Francisco?",
		generate.Tools(tool.ToDefinition(weatherTool)),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	// Handle tool calls
	toolCalls := result.ToolCalls()
	if len(toolCalls) > 0 {
		fmt.Printf("Model requested %d tool call(s):\n", len(toolCalls))

		for _, tc := range toolCalls {
			fmt.Printf("\nTool: %s\n", tc.Name)
			fmt.Printf("Input: %s\n", string(tc.Input))

			// Execute tool
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
	} else {
		fmt.Println("No tool calls requested.")
		fmt.Printf("Response: %s\n", result.Text)
	}

	fmt.Printf("\nFinish reason: %s\n", result.FinishReason())
	fmt.Printf("Tokens: %d\n", result.Usage().TotalTokens)
}

// Example of using SuccessResult and ErrorResult for structured tool responses
func _() {
	// Create a tool that returns structured error information
	weatherToolWithErrorHandling := tool.NewTool("get_weather", "Get current weather for a location",
		func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
			if input.Location == "" {
				// Return structured error for LLM interpretation
				return WeatherOutput{}, fmt.Errorf("location is required")
			}

			// Simulate API call
			if input.Location == "unknown" {
				// Return structured result with success=false for better LLM interpretation
				var output WeatherOutput
				return output, nil
			}

			return WeatherOutput{
				Temperature: 22.5,
				Condition:   "sunny",
				Humidity:    65,
			}, nil
		},
	)

	// Example of sending tool results back to continue conversation
	_ = weatherToolWithErrorHandling
	var _ = func() {
		var toolResults []provider.ToolResult
		// for _, tc := range result.ToolCalls {
		// 	output, _ := weatherToolWithErrorHandling.Execute(context.Background(), tc.Input)
		// 	toolResults = append(toolResults, provider.ToolResult{
		// 		ID:     tc.ID,
		// 		Result: output,
		// 	})
		// }

		// Continue conversation with tool results
		_ = toolResults
		// generate.Generate(ctx, model, "",
		// 	generate.Messages(
		// 		provider.Message{Role: provider.RoleUser, Content: []provider.Content{provider.Text("...")}},
		// 		provider.Message{Role: provider.RoleAssistant, Content: result.Content},
		// 		provider.Message{Role: provider.RoleTool, Content: ... tool results ...},
		// 	),
		// )
	}
}
