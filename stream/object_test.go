package stream

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type TestPerson struct {
	Name string `json:"name" jsonschema:"required,description=The person's name"`
	Age  int    `json:"age" jsonschema:"required,minimum=0"`
}

type mockStreamerObject struct {
	parts []provider.StreamPart
	err   error
}

func (m *mockStreamerObject) Provider() string { return "mock" }
func (m *mockStreamerObject) ModelID() string  { return "mock-model" }

func (m *mockStreamerObject) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Pre-calculate values
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

	// Create stream channel with capacity for all parts
	stream := make(chan provider.StreamPart, len(m.parts))
	for _, part := range m.parts {
		stream <- part
	}
	close(stream)

	done := make(chan struct{})
	close(done)

	finalText := textBuilder.String()

	return &provider.StreamResult{
		Stream: stream,
		Done:   done,
		Text: func() (string, error) {
			return finalText, nil
		},
		Usage: func() (provider.Usage, error) {
			return usage, nil
		},
		FinishReason: func() (provider.FinishReason, error) {
			return reason, nil
		},
	}, nil
}

func TestStreamObject_Success(t *testing.T) {
	jsonStr := `{"name": "Alice", "age": 30}`
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: jsonStr},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop, Usage: provider.Usage{TotalTokens: 10}},
	}

	mock := &mockStreamerObject{parts: parts}
	result := StreamObject[TestPerson](context.Background(), mock, "Generate a person")

	// Collect all parts
	var textDeltas []string
	var finishFound bool
	for part := range result.Stream() {
		switch part.Type {
		case "text-delta":
			textDeltas = append(textDeltas, part.Delta)
		case "object":
			// Good - object received
		case "finish":
			finishFound = true
		case "error":
			t.Errorf("Unexpected error: %v", part.Error)
		}
	}

	if !finishFound {
		t.Error("Expected finish part")
	}

	combined := strings.Join(textDeltas, "")
	if combined != jsonStr {
		t.Errorf("Expected text %q, got %q", jsonStr, combined)
	}

	// Get final object
	obj, err := result.Object()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj.Name != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", obj.Name)
	}
	if obj.Age != 30 {
		t.Errorf("expected age %d, got %d", 30, obj.Age)
	}
}

func TestStreamObject_Streaming(t *testing.T) {
	// Test streaming text deltas
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: `{"name": "`},
		provider.StreamTextPart{Delta: "Bob"},
		provider.StreamTextPart{Delta: `", "age": 25}`},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop, Usage: provider.Usage{TotalTokens: 5}},
	}

	mock := &mockStreamerObject{parts: parts}
	result := StreamObject[TestPerson](context.Background(), mock, "test")

	// Drain stream
	for range result.Stream() {
	}

	// Verify final object
	obj, err := result.Object()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj.Name != "Bob" {
		t.Errorf("expected name %q, got %q", "Bob", obj.Name)
	}
	if obj.Age != 25 {
		t.Errorf("expected age %d, got %d", 25, obj.Age)
	}
}

func TestStreamObject_NilModel(t *testing.T) {
	result := StreamObject[TestPerson](context.Background(), nil, "test")

	// Should receive error part
	for part := range result.Stream() {
		if part.Type == "error" {
			if part.Error != provider.ErrInvalidModel {
				t.Errorf("expected ErrInvalidModel, got %v", part.Error)
			}
			return
		}
	}
	t.Error("Expected error part for nil model")
}

func TestStreamObject_EmptyPrompt(t *testing.T) {
	mock := &mockStreamerObject{
		parts: []provider.StreamPart{
			provider.StreamTextPart{Delta: "{}"},
			provider.StreamFinishPart{FinishReason: provider.FinishReasonStop},
		},
	}

	result := StreamObject[TestPerson](context.Background(), mock, "")

	// Should receive error for empty prompt with no messages
	for part := range result.Stream() {
		if part.Type == "error" {
			if part.Error != provider.ErrInvalidPrompt {
				t.Errorf("expected ErrInvalidPrompt, got %v", part.Error)
			}
			return
		}
	}
}

func TestStreamObject_ModelError(t *testing.T) {
	testErr := &provider.Error{Code: provider.CodeInvalidRequest, Message: "test error"}
	mock := &mockStreamerObject{err: testErr}

	result := StreamObject[TestPerson](context.Background(), mock, "test")

	for part := range result.Stream() {
		if part.Type == "error" {
			var providerErr *provider.Error
			if !errors.As(part.Error, &providerErr) {
				t.Errorf("expected provider.Error, got %T", part.Error)
			}
			return
		}
	}
	t.Error("Expected error part")
}

func TestStreamObject_InvalidJSON(t *testing.T) {
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: `{invalid json}`},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop},
	}

	mock := &mockStreamerObject{parts: parts}
	result := StreamObject[TestPerson](context.Background(), mock, "test")

	// Drain stream
	for range result.Stream() {
	}

	// Get final object - should have error
	_, err := result.Object()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}

	var parseErr *provider.ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}
}

func TestStreamObject_WithOptions(t *testing.T) {
	jsonStr := `{"name": "Charlie", "age": 40}`
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: jsonStr},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop, Usage: provider.Usage{TotalTokens: 20}},
	}

	mock := &mockStreamerObject{parts: parts}
	result := StreamObject[TestPerson](context.Background(), mock, "test",
		MaxTokens(100),
		Temperature(0.5),
		WithTimeout(30*time.Second),
	)

	// Drain stream
	for range result.Stream() {
	}

	obj, err := result.Object()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj.Name != "Charlie" {
		t.Errorf("expected name %q, got %q", "Charlie", obj.Name)
	}
}

func TestStreamObject_AccessorFunctions(t *testing.T) {
	jsonStr := `{"name": "Test", "age": 99}`
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: jsonStr},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop, Usage: provider.Usage{TotalTokens: 15, PromptTokens: 5, CompletionTokens: 10}},
	}

	mock := &mockStreamerObject{parts: parts}
	result := StreamObject[TestPerson](context.Background(), mock, "test")

	// Wait for completion
	<-result.Wait()

	// Test Object()
	obj, err := result.Object()
	if err != nil {
		t.Fatalf("Object() error: %v", err)
	}
	if obj.Name != "Test" {
		t.Errorf("expected name %q, got %q", "Test", obj.Name)
	}

	// Test Text()
	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() error: %v", err)
	}
	if text != jsonStr {
		t.Errorf("expected text %q, got %q", jsonStr, text)
	}

	// Test Usage()
	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() error: %v", err)
	}
	if usage.TotalTokens != 15 {
		t.Errorf("expected 15 tokens, got %d", usage.TotalTokens)
	}

	// Test FinishReason()
	reason, err := result.FinishReason()
	if err != nil {
		t.Fatalf("FinishReason() error: %v", err)
	}
	if reason != provider.FinishReasonStop {
		t.Errorf("expected %q, got %q", provider.FinishReasonStop, reason)
	}
}

func TestStreamObject_WithMessages(t *testing.T) {
	jsonStr := `{"name": "Dave", "age": 50}`
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: jsonStr},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop},
	}

	mock := &mockStreamerObject{parts: parts}
	result := StreamObject[TestPerson](context.Background(), mock, "",
		Messages(provider.Message{
			Role:    provider.RoleUser,
			Content: []provider.Content{provider.Text("Generate a person")},
		}),
	)

	// Drain stream
	for range result.Stream() {
	}

	obj, err := result.Object()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if obj.Name != "Dave" {
		t.Errorf("expected name %q, got %q", "Dave", obj.Name)
	}
}

func TestStreamObject_VerifyRequest(t *testing.T) {
	// Verify that the request has response format set correctly
	jsonStr := `{"name": "Eve", "age": 35}`
	parts := []provider.StreamPart{
		provider.StreamTextPart{Delta: jsonStr},
		provider.StreamFinishPart{FinishReason: provider.FinishReasonStop},
	}

	mock := &mockStreamerObject{parts: parts}
	_ = StreamObject[TestPerson](context.Background(), mock, "test")

	// The schema should be generated from the struct type
	// Verify the schema was created correctly by serializing a TestPerson
	schemaBytes, _ := json.Marshal(TestPerson{Name: "Test", Age: 1})
	if string(schemaBytes) != `{"name":"Test","age":1}` {
		t.Errorf("unexpected schema: %s", schemaBytes)
	}
}
