package contract

import (
	"strings"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/stream"
)

func AssertTextGenerated(t *testing.T, result *provider.GenerateResult, expectedMinLen int) {
	t.Helper()
	if result.Text == "" && expectedMinLen > 0 {
		t.Error("result.Text is empty")
	}
	if len(result.Text) < expectedMinLen {
		t.Errorf("text length = %d, want at least %d", len(result.Text), expectedMinLen)
	}
}

func AssertStreamCompleted(t *testing.T, result *stream.Result) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	text, err := result.Text()
	if err != nil {
		t.Fatalf("failed to get text: %v", err)
	}
	if text == "" {
		t.Error("stream produced empty text")
	}
}

func AssertStreamHasParts(t *testing.T, result *stream.Result, minParts int) int {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	count := 0
	for range result.Stream() {
		count++
	}
	if count < minParts {
		t.Errorf("stream produced %d parts, want at least %d", count, minParts)
	}
	return count
}

func AssertToolCalled(t *testing.T, result *provider.GenerateResult, toolName string) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	for _, content := range result.Content {
		if tc, ok := content.(provider.ToolCallContent); ok {
			if tc.Name == toolName {
				return
			}
		}
	}
	t.Errorf("tool %q not found in result", toolName)
}

func AssertObjectType[T any](t *testing.T, result *provider.GenerateResult) T {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	var obj T
	text := result.Text
	if text == "" {
		t.Fatal("result.Text is empty")
	}
	return obj
}

func AssertErrorIs(t *testing.T, err error, code provider.ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	actualCode := provider.ErrorCodeOf(err)
	if actualCode != code {
		t.Errorf("error code = %v, want %v", actualCode, code)
	}
}

func AssertErrorContains(t *testing.T, err error, substr string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), substr) {
		t.Errorf("error %q does not contain %q", err.Error(), substr)
	}
}

func AssertNonEmptyUsage(t *testing.T, usage provider.Usage) {
	t.Helper()
	if usage.PromptTokens <= 0 {
		t.Errorf("PromptTokens = %d, want > 0", usage.PromptTokens)
	}
	if usage.CompletionTokens <= 0 {
		t.Errorf("CompletionTokens = %d, want > 0", usage.CompletionTokens)
	}
	if usage.TotalTokens <= 0 {
		t.Errorf("TotalTokens = %d, want > 0", usage.TotalTokens)
	}
}

func AssertEmbeddingDimension(t *testing.T, embedding []float32, expectedDim int) {
	t.Helper()
	if len(embedding) != expectedDim {
		t.Errorf("embedding dimension = %d, want %d", len(embedding), expectedDim)
	}
}

func AssertFinishReason(t *testing.T, result *stream.Result, expected provider.FinishReason) {
	t.Helper()
	if result == nil {
		t.Fatal("result is nil")
	}
	reason, err := result.FinishReason()
	if err != nil {
		t.Fatalf("failed to get finish reason: %v", err)
	}
	if expected != "" && reason != expected {
		t.Errorf("finish reason = %v, want %v", reason, expected)
	}
}

type ContentMatcher func(provider.Content) bool

func AssertContentMatches(t *testing.T, contents []provider.Content, matcher ContentMatcher) {
	t.Helper()
	for _, c := range contents {
		if matcher(c) {
			return
		}
	}
	t.Error("no content matched")
}

func IsTextContent(textContains string) ContentMatcher {
	return func(c provider.Content) bool {
		if tc, ok := c.(provider.TextContent); ok {
			return strings.Contains(tc.Text, textContains)
		}
		return false
	}
}

func IsToolCallContent(toolName string) ContentMatcher {
	return func(c provider.Content) bool {
		if tc, ok := c.(provider.ToolCallContent); ok {
			return tc.Name == toolName
		}
		return false
	}
}

func IsImageContent() ContentMatcher {
	return func(c provider.Content) bool {
		_, ok := c.(provider.ImageContent)
		return ok
	}
}

func SkipIfMissingAPIKey(t *testing.T, apiKey string, providerName string) {
	t.Helper()
	if apiKey == "" {
		t.Skipf("skipping: %s API key not set", providerName)
	}
}
