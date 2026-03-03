package embed

import "github.com/seifalmotaz/lamar-sdk/provider"

type Result struct {
	Embedding []float64
	Usage     provider.Usage
}

type BatchResult struct {
	Embeddings [][]float64
	Usage      provider.Usage
}
