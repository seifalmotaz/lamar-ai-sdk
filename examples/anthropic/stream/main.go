// Package main demonstrates streaming text generation with Anthropic Claude.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-sdk/provider"
	"github.com/seifalmotaz/lamar-sdk/providers/anthropic"
	"github.com/seifalmotaz/lamar-sdk/stream"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	fmt.Println("Streaming text generation...")
	fmt.Println("-------------------------")

	result := stream.Stream(context.Background(), model, "Write a short poem about coding.")

	for part := range result.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			fmt.Print(p.Delta)
		case provider.StreamErrorPart:
			fmt.Printf("\nError: %v\n", p.Error)
		case provider.StreamFinishPart:
			fmt.Println("\n-------------------------")
		}
	}

	text, err := result.Text()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nFull text:\n%s\n", text)

	usage, _ := result.Usage()
	fmt.Printf("\nTokens: %d prompt + %d completion = %d total\n",
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens)
}
