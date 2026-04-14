package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/ollama"
	"github.com/seifalmotaz/lamar-ai-sdk/tool"
)

type WeatherInput struct {
	Location string `json:"location" jsonschema:"required,description=City name"`
	Unit     string `json:"unit" jsonschema:"enum=celsius,enum=fahrenheit"`
}

type WeatherOutput struct {
	Temperature float64 `json:"temperature"`
	Condition   string  `json:"condition"`
	Humidity    int     `json:"humidity"`
}

func main() {
	client := ollama.NewProvider()
	model := client.StreamingModel("llama3.2")

	weatherTool := tool.NewTool(
		"get_weather",
		"Get the current weather for a location",
		func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
			fmt.Printf("\n[Calling weather API for %s...]\n", input.Location)
			return WeatherOutput{
				Temperature: 22.5,
				Condition:   "Partly cloudy",
				Humidity:    65,
			}, nil
		},
	)

	toolDefs := tool.ToDefinitions(weatherTool)

	ctx := context.Background()

	result, err := generate.Generate(ctx, model, "What's the weather like in Tokyo and Paris?",
		generate.Tools(toolDefs...),
		generate.ToolChoice(provider.ToolChoiceAuto()),
	)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("\n=== First Response ===")
	fmt.Printf("Tool calls: %d\n", len(result.ToolCalls()))
	for i, tc := range result.ToolCalls() {
		fmt.Printf("  %d. %s(%s)\n", i+1, tc.Name, string(tc.Input))
	}

	if len(result.ToolCalls()) > 0 {
		fmt.Println("\n=== Executing Tools ===")
		content := make([]provider.Content, 0, len(result.ToolCalls()))
		for _, tc := range result.ToolCalls() {
			content = append(content, provider.NewToolCallContent(tc.ID, tc.Name, tc.Input))
		}
		messages := []provider.Message{
			provider.UserMessage("What's the weather like in Tokyo and Paris?"),
			{Role: provider.RoleAssistant, Content: content},
		}

		for _, tc := range result.ToolCalls() {
			output, err := weatherTool.Execute(ctx, tc.Input)
			if err != nil {
				log.Printf("Tool execution error: %v", err)
				continue
			}

			outputJSON, _ := json.Marshal(output)
			fmt.Printf("Result: %s\n", string(outputJSON))

			toolResult := provider.NewToolResultContent(tc.ID, tc.Name, outputJSON, false)
			messages = append(messages, provider.ToolResultMessage(toolResult))
		}

		fmt.Println("\n=== Getting Final Response ===")
		finalResult, err := generate.Generate(ctx, model, "",
			generate.Messages(messages...),
		)
		if err != nil {
			log.Fatalf("Error: %v", err)
		}

		fmt.Printf("\nFinal response:\n%s\n", finalResult.Text())
	} else {
		fmt.Printf("\nNo tool calls. Response:\n%s\n", result.Text())
	}
}
