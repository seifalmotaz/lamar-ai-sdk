package ollama

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func getenv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

func skipIfNoOllama(t *testing.T) {
	if os.Getenv("OLLAMA_HOST") == "" && os.Getenv("TEST_OLLAMA") == "" {
		t.Skip("Skipping: set OLLAMA_HOST or TEST_OLLAMA to run integration tests")
	}
}

func TestIntegrationGenerate(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model := p.StreamingModel(getenv("OLLAMA_MODEL", "llama3.2"))

	result, err := model.Generate(ctx, &provider.GenerateRequest{
		Prompt: "Say 'hello' in exactly one word",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.Text == "" {
		t.Error("Generate() returned empty text")
	}

	if result.FinishReason == "" {
		t.Error("Generate() returned empty finish reason")
	}

	t.Logf("Response: %s", result.Text)
}

func TestIntegrationGenerateWithMessages(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model := p.StreamingModel(getenv("OLLAMA_MODEL", "llama3.2"))

	result, err := model.Generate(ctx, &provider.GenerateRequest{
		Messages: []provider.Message{
			provider.SystemMessage("You are a helpful assistant. Be very brief."),
			provider.UserMessage("What is 2+2?"),
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.Text == "" {
		t.Error("Generate() returned empty text")
	}

	t.Logf("Response: %s", result.Text)
}

func TestIntegrationStream(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	model := p.StreamingModel(getenv("OLLAMA_MODEL", "llama3.2"))

	result, err := model.Stream(ctx, &provider.GenerateRequest{
		Prompt: "Count from 1 to 5",
	})
	if err != nil {
		t.Fatalf("Stream() error = %v", err)
	}

	var textParts []string
	for part := range result.Stream {
		switch p := part.(type) {
		case provider.StreamTextPart:
			textParts = append(textParts, p.Delta)
			t.Logf("Text: %s", p.Delta)
		case provider.StreamErrorPart:
			t.Errorf("Stream error: %v", p.Error)
		case provider.StreamFinishPart:
			t.Logf("Finish: %s", p.FinishReason)
		}
	}

	<-result.Done

	finalText, err := result.Text()
	if err != nil {
		t.Errorf("Text() error = %v", err)
	}

	t.Logf("Final text: %s", finalText)

	if finalText == "" && len(textParts) == 0 {
		t.Error("Stream() returned no text")
	}
}

func TestIntegrationEmbed(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model := p.Embedding(getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text"))

	result, err := model.Embed(ctx, &provider.EmbedRequest{
		Texts: []string{"hello world", "goodbye world"},
	})
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}

	if len(result.Embeddings) != 2 {
		t.Errorf("Embed() returned %d embeddings, want 2", len(result.Embeddings))
	}

	if len(result.Embeddings[0]) == 0 {
		t.Error("Embed() returned empty embedding vector")
	}

	t.Logf("Embedding dimension: %d", len(result.Embeddings[0]))
}

func TestIntegrationGenerateWithConfig(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	temperature := 0.5
	config := &ChatConfig{
		Temperature: &temperature,
	}

	model := p.ModelWithConfig(getenv("OLLAMA_MODEL", "llama3.2"), config)

	result, err := model.Generate(ctx, &provider.GenerateRequest{
		Prompt: "Say 'test'",
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if result.Text == "" {
		t.Error("Generate() returned empty text")
	}

	t.Logf("Response: %s", result.Text)
}

func TestIntegrationGenerateWithMaxTokens(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model := p.StreamingModel(getenv("OLLAMA_MODEL", "llama3.2"))

	result, err := model.Generate(ctx, &provider.GenerateRequest{
		Prompt: "Write a long story about a cat",
		Config: provider.Config{
			MaxTokens: 50,
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Logf("Response: %s", result.Text)

	if result.Usage.CompletionTokens > 100 {
		t.Logf("Warning: completion tokens %d exceeded limit by a lot", result.Usage.CompletionTokens)
	}
}

func TestIntegrationGenerateWithStopSequence(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	model := p.StreamingModel(getenv("OLLAMA_MODEL", "llama3.2"))

	result, err := model.Generate(ctx, &provider.GenerateRequest{
		Prompt: "Count from 1 to 10: ",
		Config: provider.Config{
			StopSequences: []string{"5"},
		},
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	t.Logf("Response: %s", result.Text)
}

func TestIntegrationModelNotFound(t *testing.T) {
	skipIfNoOllama(t)

	baseURL := getenv("OLLAMA_HOST", DefaultBaseURL)
	p := NewProvider(BaseURL(baseURL))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	model := p.StreamingModel("nonexistent-model-xyz")

	_, err := model.Generate(ctx, &provider.GenerateRequest{
		Prompt: "test",
	})
	if err == nil {
		t.Error("Generate() should fail for non-existent model")
	}

	t.Logf("Error (expected): %v", err)
}
