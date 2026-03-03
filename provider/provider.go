package provider

import "context"

type Model interface {
	Provider() string
	ModelID() string
}

type Generator interface {
	Model
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error)
}

type Streamer interface {
	Model
	Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)
}

type LanguageModel interface {
	Generator
	Streamer
}

type EmbeddingModel interface {
	Model
	Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error)
	MaxEmbeddingsPerCall() int
}

func CanGenerate(m Model) bool {
	_, ok := m.(Generator)
	return ok
}

func CanStream(m Model) bool {
	_, ok := m.(Streamer)
	return ok
}

func CanEmbed(m Model) bool {
	_, ok := m.(EmbeddingModel)
	return ok
}

func IsLanguageModel(m Model) bool {
	_, ok := m.(LanguageModel)
	return ok
}

type ModelBuilder struct {
	provider string
	modelID  string
}

func NewModelBuilder(provider, modelID string) *ModelBuilder {
	return &ModelBuilder{
		provider: provider,
		modelID:  modelID,
	}
}

func (b *ModelBuilder) Build() Model {
	return &simpleModel{
		provider: b.provider,
		modelID:  b.modelID,
	}
}

type simpleModel struct {
	provider string
	modelID  string
}

func (m *simpleModel) Provider() string { return m.provider }
func (m *simpleModel) ModelID() string  { return m.modelID }
