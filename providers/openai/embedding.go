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
	embeddings, usage, err := m.provider.wrapEmbed(ctx, m.id, req.Texts, func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error) {
		openaiReq := &EmbeddingRequest{
			Model: m.id,
			Input: texts,
		}

		var resp EmbeddingResponse
		if err := m.provider.client.Post(ctx, "/embeddings", openaiReq, &resp); err != nil {
			return nil, provider.Usage{}, err
		}

		embeddings := make([][]float64, len(resp.Data))
		for _, d := range resp.Data {
			embeddings[d.Index] = d.Embedding
		}

		return embeddings, provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}, nil
	})
	if err != nil {
		return nil, err
	}

	return &provider.EmbedResult{
		Embeddings: embeddings,
		Usage:      usage,
	}, nil
}
