// Package main demonstrates embedding generation with OpenAI.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-sdk/embed"
	"github.com/seifalmotaz/lamar-sdk/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	// Single embedding
	fmt.Println("Single embedding:")
	result, err := embed.Embed(context.Background(), model, "Hello, world!")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Embedding dimension: %d\n", len(result.Embedding))
	fmt.Printf("  First 5 values: %v\n", result.Embedding[:5])
	fmt.Printf("  Tokens: %d\n\n", result.Usage.TotalTokens)

	// Batch embeddings
	fmt.Println("Batch embeddings:")
	texts := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Machine learning is a subset of artificial intelligence.",
		"Go is a programming language created at Google.",
	}

	batchResult, err := embed.EmbedBatch(context.Background(), model, texts)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("  Number of embeddings: %d\n", len(batchResult.Embeddings))
	for i, emb := range batchResult.Embeddings {
		fmt.Printf("  Text %d: dimension=%d, first 3 values=%v\n", i+1, len(emb), emb[:3])
	}
	fmt.Printf("  Total tokens: %d\n", batchResult.Usage.TotalTokens)
}
