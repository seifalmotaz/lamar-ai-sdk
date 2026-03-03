package embed

import "github.com/seifalmotaz/lamar-sdk/provider"

// Result contains the result of a single embedding request.
type Result struct {
	// Embedding is the embedding vector for the input text.
	Embedding []float64
	// Usage contains token usage statistics.
	Usage provider.Usage
}

// BatchResult contains the results of a batch embedding request.
type BatchResult struct {
	// Embeddings are the embedding vectors for all input texts.
	Embeddings [][]float64
	// Usage contains aggregated token usage statistics.
	Usage provider.Usage
}
