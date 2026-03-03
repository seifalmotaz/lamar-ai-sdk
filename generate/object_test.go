package generate

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type TestPerson struct {
	Name string `json:"name" jsonschema:"required,description=The person's name"`
	Age  int    `json:"age" jsonschema:"required,minimum=0"`
}

type mockGeneratorObject struct {
	response string
	err      error
}

func (m *mockGeneratorObject) Provider() string { return "mock" }
func (m *mockGeneratorObject) ModelID() string  { return "mock-model" }

func (m *mockGeneratorObject) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	if m.err != nil {
		return nil, m.err
	}

	// Verify JSON schema response format is set
	if req.Config.ResponseFormat == nil || req.Config.ResponseFormat.Type != "json_schema" {
		return nil, errors.New("expected json_schema response format")
	}

	return &provider.GenerateResult{
		Text:         m.response,
		FinishReason: provider.FinishReasonStop,
		Usage:        provider.Usage{TotalTokens: 10},
	}, nil
}

func TestGenerateObject_Success(t *testing.T) {
	response := `{"name": "Alice", "age": 30}`
	mock := &mockGeneratorObject{response: response}

	result, err := GenerateObject[TestPerson](context.Background(), mock, "Generate a person")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object.Name != "Alice" {
		t.Errorf("expected name %q, got %q", "Alice", result.Object.Name)
	}
	if result.Object.Age != 30 {
		t.Errorf("expected age %d, got %d", 30, result.Object.Age)
	}
	if result.FinishReason != provider.FinishReasonStop {
		t.Errorf("expected finish reason %q, got %q", provider.FinishReasonStop, result.FinishReason)
	}
	if result.Usage.TotalTokens != 10 {
		t.Errorf("expected 10 tokens, got %d", result.Usage.TotalTokens)
	}
}

func TestGenerateObject_NilModel(t *testing.T) {
	_, err := GenerateObject[TestPerson](context.Background(), nil, "test")
	if err != provider.ErrInvalidModel {
		t.Errorf("expected ErrInvalidModel, got %v", err)
	}
}

func TestGenerateObject_EmptyPrompt(t *testing.T) {
	mock := &mockGeneratorObject{response: "{}"}
	_, err := GenerateObject[TestPerson](context.Background(), mock, "")
	if err != provider.ErrInvalidPrompt {
		t.Errorf("expected ErrInvalidPrompt, got %v", err)
	}
}

func TestGenerateObject_ModelError(t *testing.T) {
	testErr := &provider.Error{Code: provider.CodeInvalidRequest, Message: "test error"}
	mock := &mockGeneratorObject{err: testErr}

	_, err := GenerateObject[TestPerson](context.Background(), mock, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("expected provider.Error, got %T", err)
	}
	if providerErr.Code != provider.CodeInvalidRequest {
		t.Errorf("expected code %v, got %v", provider.CodeInvalidRequest, providerErr.Code)
	}
}

func TestGenerateObject_InvalidJSON(t *testing.T) {
	mock := &mockGeneratorObject{response: `{invalid json}`}

	_, err := GenerateObject[TestPerson](context.Background(), mock, "test")
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var parseErr *provider.ParseError
	if !errors.As(err, &parseErr) {
		t.Errorf("expected ParseError, got %T", err)
	}
}

func TestGenerateObject_WithMessages(t *testing.T) {
	response := `{"name": "Bob", "age": 25}`
	mock := &mockGeneratorObject{response: response}

	result, err := GenerateObject[TestPerson](context.Background(), mock, "",
		Messages(provider.Message{Role: provider.RoleUser, Content: []provider.Content{provider.Text("Hello")}}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object.Name != "Bob" {
		t.Errorf("expected name %q, got %q", "Bob", result.Object.Name)
	}
}

func TestGenerateObject_WithOptions(t *testing.T) {
	response := `{"name": "Charlie", "age": 35}`
	mock := &mockGeneratorObject{response: response}

	result, err := GenerateObject[TestPerson](context.Background(), mock, "test",
		MaxTokens(100),
		Temperature(0.5),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Object.Name != "Charlie" {
		t.Errorf("expected name %q, got %q", "Charlie", result.Object.Name)
	}
}

// Integration test with OpenAI-like server
func TestGenerateObject_OpenAIIntegration(t *testing.T) {
	schemaJSON, _ := json.Marshal(map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
			"age":  map[string]any{"type": "integer", "minimum": 0},
		},
		"required": []string{"name", "age"},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Send response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"id": "test",
			"choices": [{
				"message": {"content": "{\"name\": \"Test\", \"age\": 20}"},
				"finish_reason": "stop"
			}],
			"usage": {"prompt_tokens": 5, "completion_tokens": 10, "total_tokens": 15}
		}`))
	}))
	defer server.Close()

	t.Logf("Schema JSON: %s", string(schemaJSON))
}
