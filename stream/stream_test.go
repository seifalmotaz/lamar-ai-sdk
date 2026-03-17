package stream

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type mockStreamer struct {
	parts []provider.StreamPart
	err   error
}

func (m *mockStreamer) Provider() string { return "mock" }
func (m *mockStreamer) ModelID() string  { return "mock-model" }

func (m *mockStreamer) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Create stream channel with capacity for all parts
	stream := make(chan provider.StreamPart, len(m.parts))
	for _, part := range m.parts {
		stream <- part
	}
	close(stream)

	// Pre-calculate values for the accessor functions
	var textBuilder strings.Builder
	var usage provider.Usage
	var reason provider.FinishReason
	for _, part := range m.parts {
		switch p := part.(type) {
		case provider.StreamTextPart:
			textBuilder.WriteString(p.Delta)
		case provider.StreamFinishPart:
			usage = p.Usage
			reason = p.FinishReason
		}
	}

	// Create done channel that closes immediately since stream is already done
	done := make(chan struct{})
	close(done)

	result := &provider.StreamResult{
		Stream: stream,
		Done:   done,
		Text: func() (string, error) {
			return textBuilder.String(), nil
		},
		Usage: func() (provider.Usage, error) {
			return usage, nil
		},
		FinishReason: func() (provider.FinishReason, error) {
			return reason, nil
		},
	}

	return result, nil
}

func TestStreamBasic(t *testing.T) {
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: "Hello "},
		provider.StreamTextPart{Delta: "world"},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop, Usage: provider.Usage{TotalTokens: 10}},
	}

	mock := &mockStreamer{parts: parts}
	result := Stream(context.Background(), mock, "test prompt")

	// Collect all parts
	var collectedParts []provider.StreamPart
	for part := range result.Stream() {
		collectedParts = append(collectedParts, part)
	}

	if len(collectedParts) != 3 {
		t.Errorf("expected 3 parts, got %d", len(collectedParts))
	}

	// Check text
	text, err := result.Text()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "Hello world" {
		t.Errorf("expected text %q, got %q", "Hello world", text)
	}

	// Check usage
	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if usage.TotalTokens != 10 {
		t.Errorf("expected 10 tokens, got %d", usage.TotalTokens)
	}

	// Check finish reason
	reason, err := result.FinishReason()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if reason != provider.FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", provider.FinishReasonStop, reason)
	}
}

func TestStreamWithTimeout(t *testing.T) {
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: "test"},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop},
	}

	mock := &mockStreamer{parts: parts}
	ctx := context.Background()
	result := Stream(ctx, mock, "test prompt", WithTimeout(30*time.Second))

	// Wait for completion
	<-result.Wait()

	text, err := result.Text()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if text != "test" {
		t.Errorf("expected text %q, got %q", "test", text)
	}
}

func TestStreamError(t *testing.T) {
	testErr := &provider.Error{Code: provider.CodeInvalidRequest, Message: "test error"}
	mock := &mockStreamer{err: testErr}

	result := Stream(context.Background(), mock, "test prompt")

	// Should receive error part
	var receivedError bool
	for part := range result.Stream() {
		if errPart, ok := part.(provider.StreamErrorPart); ok {
			receivedError = true
			if errPart.Error != testErr {
				t.Errorf("expected error %v, got %v", testErr, errPart.Error)
			}
		}
	}

	if !receivedError {
		t.Error("expected error part in stream")
	}

	// Err() should return the error
	err := result.Err()
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.Error
	if errors.As(err, &providerErr) {
		if providerErr.Code != provider.CodeInvalidRequest {
			t.Errorf("expected code %v, got %v", provider.CodeInvalidRequest, providerErr.Code)
		}
	} else {
		t.Errorf("expected provider.Error, got %T", err)
	}
}

func TestStreamContext(t *testing.T) {
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: "Hello"},
	}

	mock := &mockStreamer{parts: parts}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result := Stream(ctx, mock, "test prompt")

	// The context cancellation should propagate
	<-result.Wait()
}

func TestResultWait(t *testing.T) {
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: "test"},
	}

	mock := &mockStreamer{parts: parts}
	result := Stream(context.Background(), mock, "test prompt")

	// Wait should block until streaming is complete
	select {
	case <-result.Wait():
		// Good - streaming completed
	case <-time.After(5 * time.Second):
		t.Error("Wait did not return within timeout")
	}
}
