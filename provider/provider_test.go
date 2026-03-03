package provider

import (
	"context"
	"testing"
)

type mockModel struct {
	provider string
	modelID  string
}

func (m *mockModel) Provider() string { return m.provider }
func (m *mockModel) ModelID() string  { return m.modelID }

type mockGenerator struct {
	*mockModel
}

func (m *mockGenerator) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error) {
	return &GenerateResult{Text: "generated"}, nil
}

type mockStreamer struct {
	*mockModel
}

func (m *mockStreamer) Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error) {
	return &StreamResult{}, nil
}

type mockLanguageModel struct {
	*mockModel
}

func (m *mockLanguageModel) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error) {
	return &GenerateResult{Text: "generated"}, nil
}

func (m *mockLanguageModel) Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error) {
	return &StreamResult{}, nil
}

type mockEmbeddingModel struct {
	*mockModel
}

func (m *mockEmbeddingModel) Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error) {
	return &EmbedResult{Embeddings: [][]float64{{0.1, 0.2}}}, nil
}

func (m *mockEmbeddingModel) MaxEmbeddingsPerCall() int {
	return 100
}

func TestModelInterface(t *testing.T) {
	model := &mockModel{provider: "test", modelID: "test-model"}

	if model.Provider() != "test" {
		t.Errorf("Provider() = %q, want %q", model.Provider(), "test")
	}
	if model.ModelID() != "test-model" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "test-model")
	}
}

func TestGeneratorInterface(t *testing.T) {
	gen := &mockGenerator{mockModel: &mockModel{provider: "test", modelID: "test-model"}}

	var _ Model = gen
	var _ Generator = gen

	result, err := gen.Generate(context.Background(), &GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Text != "generated" {
		t.Errorf("Generate() Text = %q, want %q", result.Text, "generated")
	}
}

func TestStreamerInterface(t *testing.T) {
	streamer := &mockStreamer{mockModel: &mockModel{provider: "test", modelID: "test-model"}}

	var _ Model = streamer
	var _ Streamer = streamer

	result, err := streamer.Stream(context.Background(), &GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}
	if result == nil {
		t.Error("Stream() result is nil")
	}
}

func TestLanguageModelInterface(t *testing.T) {
	lm := &mockLanguageModel{mockModel: &mockModel{provider: "test", modelID: "test-model"}}

	var _ Model = lm
	var _ Generator = lm
	var _ Streamer = lm
	var _ LanguageModel = lm
}

func TestEmbeddingModelInterface(t *testing.T) {
	em := &mockEmbeddingModel{mockModel: &mockModel{provider: "test", modelID: "test-embedding"}}

	var _ Model = em
	var _ EmbeddingModel = em

	if em.MaxEmbeddingsPerCall() != 100 {
		t.Errorf("MaxEmbeddingsPerCall() = %d, want 100", em.MaxEmbeddingsPerCall())
	}

	result, err := em.Embed(context.Background(), &EmbedRequest{Texts: []string{"test"}})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(result.Embeddings) != 1 {
		t.Errorf("Embed() Embeddings length = %d, want 1", len(result.Embeddings))
	}
}

func TestCanGenerate(t *testing.T) {
	gen := &mockGenerator{mockModel: &mockModel{}}
	streamer := &mockStreamer{mockModel: &mockModel{}}
	model := &mockModel{}

	if !CanGenerate(gen) {
		t.Error("CanGenerate(generator) should be true")
	}
	if CanGenerate(streamer) {
		t.Error("CanGenerate(streamer) should be false")
	}
	if CanGenerate(model) {
		t.Error("CanGenerate(model) should be false")
	}
}

func TestCanStream(t *testing.T) {
	gen := &mockGenerator{mockModel: &mockModel{}}
	streamer := &mockStreamer{mockModel: &mockModel{}}
	model := &mockModel{}

	if CanStream(gen) {
		t.Error("CanStream(generator) should be false")
	}
	if !CanStream(streamer) {
		t.Error("CanStream(streamer) should be true")
	}
	if CanStream(model) {
		t.Error("CanStream(model) should be false")
	}
}

func TestCanEmbed(t *testing.T) {
	gen := &mockGenerator{mockModel: &mockModel{}}
	em := &mockEmbeddingModel{mockModel: &mockModel{}}
	model := &mockModel{}

	if CanEmbed(gen) {
		t.Error("CanEmbed(generator) should be false")
	}
	if !CanEmbed(em) {
		t.Error("CanEmbed(embeddingModel) should be true")
	}
	if CanEmbed(model) {
		t.Error("CanEmbed(model) should be false")
	}
}

func TestIsLanguageModel(t *testing.T) {
	lm := &mockLanguageModel{mockModel: &mockModel{}}
	gen := &mockGenerator{mockModel: &mockModel{}}
	streamer := &mockStreamer{mockModel: &mockModel{}}

	if !IsLanguageModel(lm) {
		t.Error("IsLanguageModel(languageModel) should be true")
	}
	if IsLanguageModel(gen) {
		t.Error("IsLanguageModel(generator) should be false")
	}
	if IsLanguageModel(streamer) {
		t.Error("IsLanguageModel(streamer) should be false")
	}
}

func TestModelBuilder(t *testing.T) {
	model := NewModelBuilder("openai", "gpt-4").Build()

	if model.Provider() != "openai" {
		t.Errorf("Provider() = %q, want %q", model.Provider(), "openai")
	}
	if model.ModelID() != "gpt-4" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "gpt-4")
	}
}

func TestSimpleModel(t *testing.T) {
	model := &simpleModel{provider: "test", modelID: "test-model"}

	if model.Provider() != "test" {
		t.Errorf("Provider() = %q, want %q", model.Provider(), "test")
	}
	if model.ModelID() != "test-model" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "test-model")
	}

	var _ Model = model
}

func TestInterfaceSatisfaction(t *testing.T) {
	gen := &mockGenerator{mockModel: &mockModel{}}
	streamer := &mockStreamer{mockModel: &mockModel{}}
	lm := &mockLanguageModel{mockModel: &mockModel{}}
	em := &mockEmbeddingModel{mockModel: &mockModel{}}

	if _, ok := interface{}(gen).(Generator); !ok {
		t.Error("mockGenerator should satisfy Generator")
	}
	if _, ok := interface{}(streamer).(Streamer); !ok {
		t.Error("mockStreamer should satisfy Streamer")
	}
	if _, ok := interface{}(lm).(LanguageModel); !ok {
		t.Error("mockLanguageModel should satisfy LanguageModel")
	}
	if _, ok := interface{}(em).(EmbeddingModel); !ok {
		t.Error("mockEmbeddingModel should satisfy EmbeddingModel")
	}
}
