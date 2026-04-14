package main

import (
	"context"
	"fmt"
	"log"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/ollama"
)

func main() {
	client := ollama.NewProvider()
	model := client.Llama32()

	ctx := context.Background()

	result, err := generate.Generate(ctx, model, "Say hello in 3 languages")
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println("Response:")
	fmt.Println(result.Text())
	usage := result.Usage()
	fmt.Printf("\nTokens: %d prompt, %d completion, %d total\n",
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
	)
}
