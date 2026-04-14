package main

import (
	"context"
	"fmt"
	"log"

	"github.com/seifalmotaz/lamar-ai-sdk/embed"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/ollama"
)

func main() {
	client := ollama.NewProvider()
	model := client.NomicEmbedText()

	ctx := context.Background()

	texts := []string{
		"The quick brown fox jumps over the lazy dog",
		"Machine learning is a subset of artificial intelligence",
		"Go is a programming language developed at Google",
	}

	result, err := embed.EmbedBatch(ctx, model, texts)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Generated %d embeddings\n", len(result.Embeddings))
	for i, emb := range result.Embeddings {
		fmt.Printf("Text %d: %d dimensions\n", i+1, len(emb))
	}

	fmt.Printf("\nTotal tokens: %d\n", result.Usage.TotalTokens)

	// Compute similarity between first two embeddings
	if len(result.Embeddings) >= 2 {
		similarity := cosineSimilarity(result.Embeddings[0], result.Embeddings[1])
		fmt.Printf("\nCosine similarity between text 1 and 2: %.4f\n", similarity)
	}
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (sqrt(normA) * sqrt(normB))
}

func sqrt(x float64) float64 {
	if x <= 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = z - (z*z-x)/(2*z)
	}
	return z
}
