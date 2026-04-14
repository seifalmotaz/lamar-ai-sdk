package ollama

import (
	"context"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type EmbeddingModel struct {
	id       string
	provider *Provider
}

func (m *EmbeddingModel) Provider() string { return "ollama" }
func (m *EmbeddingModel) ModelID() string  { return m.id }

func (m *EmbeddingModel) MaxEmbeddingsPerCall() int {
	return 1
}

func (m *EmbeddingModel) Embed(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
	embeddings, usage, err := m.provider.wrapEmbed(ctx, m.id, req.Texts, func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error) {
		if len(texts) == 0 {
			return nil, provider.Usage{}, nil
		}

		if len(texts) == 1 {
			ollamaReq := &EmbedRequest{
				Model: m.id,
				Input: texts[0],
			}

			var resp EmbedResponse
			if err := m.provider.client.Post(ctx, "/api/embed", ollamaReq, &resp); err != nil {
				return nil, provider.Usage{}, m.mapError(err)
			}

			if len(resp.Embeddings) == 0 {
				return nil, provider.Usage{}, nil
			}

			return [][]float64{resp.Embeddings[0]}, provider.Usage{
				PromptTokens:     resp.PromptEvalCount,
				CompletionTokens: 0,
				TotalTokens:      resp.PromptEvalCount,
			}, nil
		}

		ollamaReq := &EmbedBatchRequest{
			Model: m.id,
			Input: texts,
		}

		var resp EmbedResponse
		if err := m.provider.client.Post(ctx, "/api/embed", ollamaReq, &resp); err != nil {
			return nil, provider.Usage{}, m.mapError(err)
		}

		return resp.Embeddings, provider.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: 0,
			TotalTokens:      resp.PromptEvalCount,
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

func (m *EmbeddingModel) mapError(err error) error {
	return &provider.Error{
		Code:     provider.CodeUnknown,
		Message:  err.Error(),
		Provider: "ollama",
		ModelID:  m.id,
		Cause:    err,
	}
}
