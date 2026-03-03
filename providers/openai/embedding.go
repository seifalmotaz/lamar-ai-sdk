package openai

import (
	"context"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type EmbeddingModel struct {
	id       string
	provider *Provider
}

func (m *EmbeddingModel) Provider() string { return "openai" }
func (m *EmbeddingModel) ModelID() string  { return m.id }

func (m *EmbeddingModel) MaxEmbeddingsPerCall() int {
	return 2048
}

func (m *EmbeddingModel) Embed(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
	openaiReq := &EmbeddingRequest{
		Model: m.id,
		Input: req.Texts,
	}

	var resp EmbeddingResponse
	if err := m.provider.client.Post(ctx, "/embeddings", openaiReq, &resp); err != nil {
		return nil, err
	}

	embeddings := make([][]float64, len(resp.Data))
	for _, d := range resp.Data {
		embeddings[d.Index] = d.Embedding
	}

	return &provider.EmbedResult{
		Embeddings: embeddings,
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}
