// Package main demonstrates extended thinking/reasoning with Anthropic Claude.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/anthropic"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := anthropic.NewProvider(anthropic.APIKey(apiKey))

	model := client.Claude35Sonnet(
		anthropic.ThinkingEnabled(1024),
	)

	fmt.Println("Extended Thinking Example")
	fmt.Println("==========================")
	fmt.Println("Asking Claude to solve a complex problem with reasoning...")
	fmt.Println()

	ctx := context.Background()
	result, err := generate.Generate(ctx, model,
		"A farmer has 17 sheep. All but 9 run away. How many sheep does the farmer have left?",
		generate.MaxTokens(500),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Response:")
	fmt.Println(result.Text())

	fmt.Println("\n---")

	var hasReasoning bool
	for _, content := range result.Content() {
		if _, ok := content.(provider.ReasoningContent); ok {
			hasReasoning = true
			break
		}
	}

	if hasReasoning {
		fmt.Println("✓ Model provided reasoning content")
	} else {
		fmt.Println("Note: Reasoning content was used internally")
	}

	fmt.Printf("\nTokens: prompt=%d, completion=%d, total=%d\n",
		result.Usage().PromptTokens,
		result.Usage().CompletionTokens,
		result.Usage().TotalTokens)

	fmt.Println("\n---")
	fmt.Println("Extended thinking allows Claude to reason through complex")
	fmt.Println("problems step by step, improving accuracy on tasks requiring")
	fmt.Println("multi-step reasoning.")
}
