package provider

import "context"

// Model is the base interface for all AI models.
// It provides identifying information about the model's provider and ID.
type Model interface {
	// Provider returns the name of the provider (e.g., "openai", "anthropic").
	Provider() string
	// ModelID returns the model identifier (e.g., "gpt-4o", "claude-3-opus").
	ModelID() string
}

// Generator is a model that supports non-streaming text generation.
// Models that only support non-streaming generation should implement this interface.
type Generator interface {
	Model
	// Generate performs a non-streaming generation request.
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error)
}

// Streamer is a model that supports streaming text generation.
// Models that only support streaming generation should implement this interface.
// Not all models support streaming - use CanStream to check capability.
type Streamer interface {
	Model
	// Stream performs a streaming generation request.
	Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)
}

// LanguageModel is a full-featured model supporting both streaming and non-streaming generation.
// Models that support both generation modes should implement this interface.
type LanguageModel interface {
	Generator
	Streamer
}

// EmbeddingModel represents a model that can generate text embeddings.
type EmbeddingModel interface {
	Model
	// Embed generates embeddings for the given texts.
	Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error)
	// MaxEmbeddingsPerCall returns the maximum number of texts that can be embedded in a single API call.
	MaxEmbeddingsPerCall() int
}

// CanGenerate returns true if the model supports non-streaming generation.
func CanGenerate(m Model) bool {
	_, ok := m.(Generator)
	return ok
}

// CanStream returns true if the model supports streaming generation.
func CanStream(m Model) bool {
	_, ok := m.(Streamer)
	return ok
}

// CanEmbed returns true if the model supports embedding generation.
func CanEmbed(m Model) bool {
	_, ok := m.(EmbeddingModel)
	return ok
}

// IsLanguageModel returns true if the model supports both streaming and non-streaming generation.
func IsLanguageModel(m Model) bool {
	_, ok := m.(LanguageModel)
	return ok
}

// ModelBuilder provides a convenient way to create simple Model instances.
// Use this for testing or when you need a minimal Model implementation.
type ModelBuilder struct {
	provider string
	modelID  string
}

// NewModelBuilder creates a new ModelBuilder with the specified provider and model ID.
func NewModelBuilder(provider, modelID string) *ModelBuilder {
	return &ModelBuilder{
		provider: provider,
		modelID:  modelID,
	}
}

// Build creates a simple Model instance from the builder.
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
