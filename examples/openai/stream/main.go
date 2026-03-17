// Package main demonstrates streaming text generation with OpenAI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-ai-sdk/stream"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT5Mini()

	fmt.Println("Streaming text generation...")
	fmt.Println("-------------------------")

	result := stream.Stream(context.Background(), model, "Write a short poem about coding.")

	// Consume stream in real-time
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

	// Wait and get full text
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
