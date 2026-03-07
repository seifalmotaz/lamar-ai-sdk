// Package main demonstrates basic text generation with OpenAI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-sdk/generate"
	"github.com/seifalmotaz/lamar-sdk/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT5Mini()

	fmt.Println("Generating text...")
	result, err := generate.Generate(context.Background(), model, "Say hello in 3 different languages, one per line.")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\nResult:")
	fmt.Println(result.Text())
	fmt.Printf("\nTokens: %d prompt + %d completion = %d total\n",
		result.Usage().PromptTokens,
		result.Usage().CompletionTokens,
		result.Usage().TotalTokens)
}
