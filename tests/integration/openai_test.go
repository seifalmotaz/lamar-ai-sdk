package integration

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/embed"
	"github.com/seifalmotaz/lamar-sdk/generate"
	"github.com/seifalmotaz/lamar-sdk/provider"
	"github.com/seifalmotaz/lamar-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-sdk/stream"
)

const testModel = "gpt-5-mini-2025-08-07"
const embeddingModel = "text-embedding-3-small"

func getAPIKey(t *testing.T) string {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	return key
}

func getTimeout() time.Duration {
	if timeout := os.Getenv("TEST_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			return d
		}
	}
	return 60 * time.Second
}

func TestOpenAI_Generate(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'hello' and nothing else.",
		generate.MaxTokens(10),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}
	if result.Usage().TotalTokens == 0 {
		t.Error("Expected non-zero token usage")
	}

	t.Logf("Response: %s", result.Text())
	t.Logf("Tokens: %d", result.Usage().TotalTokens)
}

func TestOpenAI_GenerateWithSystem(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "What is 2+2?",
		generate.System("You are a helpful math teacher. Always answer briefly."),
		generate.MaxTokens(20),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", result.Text())
}

func TestOpenAI_Stream(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result := stream.Stream(ctx, model, "Count from 1 to 5",
		stream.MaxTokens(20),
	)

	var textParts []string
	var finishFound bool

	for part := range result.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			textParts = append(textParts, p.Delta)
		case provider.StreamFinishPart:
			finishFound = true
		case provider.StreamErrorPart:
			t.Fatalf("Stream error: %v", p.Error)
		}
	}

	if !finishFound {
		t.Error("Expected finish part in stream")
	}

	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}
	if text == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", text)

	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() failed: %v", err)
	}
	t.Logf("Tokens: %d", usage.TotalTokens)
}

func TestOpenAI_Embed(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := embed.Embed(ctx, model, "Hello, world!")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(result.Embedding) == 0 {
		t.Error("Expected non-empty embedding")
	}
	if result.Usage.TotalTokens == 0 {
		t.Error("Expected non-zero token usage")
	}

	t.Logf("Embedding dimension: %d", len(result.Embedding))
	t.Logf("Tokens: %d", result.Usage.TotalTokens)
}

func TestOpenAI_EmbedBatch(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	texts := []string{
		"The quick brown fox",
		"jumps over the lazy dog",
		"Hello world",
	}

	result, err := embed.EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(result.Embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}

	for i, emb := range result.Embeddings {
		if len(emb) == 0 {
			t.Errorf("Embedding %d is empty", i)
		}
	}

	t.Logf("Generated %d embeddings", len(result.Embeddings))
	t.Logf("Tokens: %d", result.Usage.TotalTokens)
}

func TestOpenAI_GenerateObject(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type Person struct {
		Name string `json:"name" jsonschema:"required,description=The person's name"`
		Age  int    `json:"age" jsonschema:"required,minimum=0,description=Age in years"`
	}

	result, err := generate.GenerateObject[Person](ctx, model,
		"Generate a random fictional person. Respond with just a name and age.",
		generate.MaxTokens(50),
	)
	if err != nil {
		t.Fatalf("GenerateObject failed: %v", err)
	}

	if result.Object.Name == "" {
		t.Error("Expected non-empty name")
	}
	if result.Object.Age < 0 {
		t.Error("Age should not be negative")
	}

	t.Logf("Generated person: %+v", result.Object)
	t.Logf("Tokens: %d", result.Usage.TotalTokens)
}

func TestOpenAI_StreamObject(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type Color struct {
		Name    string `json:"name" jsonschema:"required,description=Color name"`
		Hex     string `json:"hex" jsonschema:"required,description=Hex color code"`
		Popular bool   `json:"popular" jsonschema:"description=Is this color popular"`
	}

	result := stream.StreamObject[Color](ctx, model,
		"Generate a random color. Respond with name, hex code, and whether it's popular.",
		stream.MaxTokens(50),
		stream.WithTimeout(30*time.Second),
	)

	// Drain stream
	for part := range result.Stream() {
		if part.Type == "error" {
			t.Fatalf("Stream error: %v", part.Error)
		}
	}

	obj, err := result.Object()
	if err != nil {
		t.Fatalf("Object() failed: %v", err)
	}

	if obj.Name == "" {
		t.Error("Expected non-empty name")
	}

	t.Logf("Generated color: %+v", obj)

	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() failed: %v", err)
	}
	t.Logf("Tokens: %d", usage.TotalTokens)
}

func TestOpenAI_GenerateWithContextCancellation(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := generate.Generate(ctx, model, "Hello")
	if err == nil {
		t.Error("Expected error for canceled context")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("Expected provider.Error, got %T", err)
		return
	}
	if providerErr.Code != provider.CodeContextCanceled {
		t.Errorf("Expected CodeContextCanceled, got %v", providerErr.Code)
	}
}

func TestOpenAI_StreamWithTimeout(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx := context.Background()

	result := stream.Stream(ctx, model, "Hello",
		stream.WithTimeout(30*time.Second),
	)

	// Drain stream
	for range result.Stream() {
	}

	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if text == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", text)
}

func TestOpenAI_GenerateTemperature(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'test'",
		generate.MaxTokens(5),
		generate.Temperature(0.1), // Low temperature for deterministic output
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	t.Logf("Response: %s", result.Text())
}

func TestOpenAI_EmbedBatchUneven(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	// Test with 10 items - will create uneven batches
	texts := []string{
		"one", "two", "three", "four", "five",
		"six", "seven", "eight", "nine", "ten",
	}

	result, err := embed.EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(result.Embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}

	// Verify all embeddings are present
	for i, emb := range result.Embeddings {
		if len(emb) == 0 {
			t.Errorf("Embedding %d is empty", i)
		}
	}

	t.Logf("Generated %d embeddings", len(result.Embeddings))
}
