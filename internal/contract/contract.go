package contract

import (
	"context"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type GeneratorTests struct {
	Name  string
	Model provider.Generator
	Tests []GeneratorTest
}

type GeneratorTest struct {
	Name     string
	Prompt   string
	Validate func(t *testing.T, result *provider.GenerateResult, err error)
}

func TestGenerator(t *testing.T, tt GeneratorTests) {
	for _, test := range tt.Tests {
		t.Run(test.Name, func(t *testing.T) {
			req := &provider.GenerateRequest{
				Prompt: test.Prompt,
			}
			result, err := tt.Model.Generate(context.Background(), req)
			test.Validate(t, result, err)
		})
	}
}

type EmbeddingModelTests struct {
	Name  string
	Model provider.EmbeddingModel
	Tests []EmbeddingModelTest
}

type EmbeddingModelTest struct {
	Name     string
	Texts    []string
	Validate func(t *testing.T, result *provider.EmbedResult, err error)
}

func TestEmbeddingModel(t *testing.T, tt EmbeddingModelTests) {
	for _, test := range tt.Tests {
		t.Run(test.Name, func(t *testing.T) {
			req := &provider.EmbedRequest{
				Texts: test.Texts,
			}
			result, err := tt.Model.Embed(context.Background(), req)
			test.Validate(t, result, err)
		})
	}
}

func RequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func RequireError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func RequireErrorCode(t *testing.T, err error, code provider.ErrorCode) {
	t.Helper()
	actualCode := provider.ErrorCodeOf(err)
	if actualCode != code {
		t.Errorf("error code = %v, want %v", actualCode, code)
	}
}

func RequireNonEmptyText(t *testing.T, result *provider.GenerateResult) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.Text == "" {
		t.Error("result.Text is empty")
	}
}

func RequireNonEmptyEmbedding(t *testing.T, result *provider.EmbedResult, index int) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	if index >= len(result.Embeddings) {
		t.Fatalf("index %d out of range (len=%d)", index, len(result.Embeddings))
	}
	if len(result.Embeddings[index]) == 0 {
		t.Errorf("embedding[%d] is empty", index)
	}
}

func RequirePositiveUsage(t *testing.T, usage provider.Usage) {
	t.Helper()
	if usage.TotalTokens <= 0 {
		t.Errorf("TotalTokens = %d, want > 0", usage.TotalTokens)
	}
}

// StreamerTests defines contract tests for Streamer implementations.
type StreamerTests struct {
	Name  string
	Model provider.Streamer
	Tests []StreamerTest
}

// StreamerTest defines a single streaming test case.
type StreamerTest struct {
	Name     string
	Prompt   string
	Validate func(t *testing.T, result *provider.StreamResult, err error)
}

// TestStreamer runs contract tests for Streamer implementations.
func TestStreamer(t *testing.T, tt StreamerTests) {
	for _, test := range tt.Tests {
		t.Run(test.Name, func(t *testing.T) {
			req := &provider.GenerateRequest{
				Prompt: test.Prompt,
			}
			result, err := tt.Model.Stream(context.Background(), req)
			test.Validate(t, result, err)
		})
	}
}

// RequireStreamParts validates that the stream produces at least minParts parts.
func RequireStreamParts(t *testing.T, result *provider.StreamResult, minParts int) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	count := 0
	for range result.Stream {
		count++
	}
	if count < minParts {
		t.Errorf("stream produced %d parts, want at least %d", count, minParts)
	}
}

// RequireStreamText validates the stream produces text containing substring.
func RequireStreamText(t *testing.T, result *provider.StreamResult, contains string) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	text, err := result.Text()
	if err != nil {
		t.Fatalf("failed to get text: %v", err)
	}
	if contains != "" && text == "" {
		t.Error("expected non-empty text")
	}
}

// RequireStreamUsage validates the stream returns usage statistics.
func RequireStreamUsage(t *testing.T, result *provider.StreamResult) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("failed to get usage: %v", err)
	}
	if usage.TotalTokens <= 0 {
		t.Errorf("TotalTokens = %d, want > 0", usage.TotalTokens)
	}
}

// RequireStreamFinish validates the stream has a valid finish reason.
func RequireStreamFinish(t *testing.T, result *provider.StreamResult) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	reason, err := result.FinishReason()
	if err != nil {
		t.Fatalf("failed to get finish reason: %v", err)
	}
	if reason == "" {
		t.Error("finish reason is empty")
	}
}
