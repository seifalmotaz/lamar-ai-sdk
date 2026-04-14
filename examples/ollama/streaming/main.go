package main

import (
	"context"
	"fmt"
	"log"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/ollama"
	"github.com/seifalmotaz/lamar-ai-sdk/stream"
)

func main() {
	client := ollama.NewProvider()
	model := client.StreamingModel("llama3.2")

	ctx := context.Background()

	result := stream.Stream(ctx, model, "Tell me a short story about a robot")

	fmt.Println("Streaming response:")
	for part := range result.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			fmt.Print(p.Delta)
		case provider.StreamToolCallPart:
			fmt.Printf("\n[Tool: %s]\n", p.ToolCall.Name)
		case provider.StreamErrorPart:
			fmt.Printf("\nError: %v\n", p.Error)
		case provider.StreamFinishPart:
			fmt.Printf("\n\n--- Finished: %s ---\n", p.FinishReason)
		}
	}

	<-result.Wait()

	text, err := result.Text()
	if err != nil {
		log.Fatalf("Error getting final text: %v", err)
	}

	fmt.Printf("\nFinal text length: %d characters\n", len(text))

	usage, _ := result.Usage()
	fmt.Printf("Tokens: %d prompt, %d completion, %d total\n",
		usage.PromptTokens,
		usage.CompletionTokens,
		usage.TotalTokens,
	)
}
