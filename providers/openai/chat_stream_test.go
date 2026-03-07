package openai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestChatModel_Stream(t *testing.T) {
	// Mock SSE server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Expected Accept: text/event-stream, got %s", r.Header.Get("Accept"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send streaming chunks
		chunks := []string{
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"content":" world"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":5,"completion_tokens":2,"total_tokens":7}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	// Create provider
	cfg := Config{APIKey: "test-key", BaseURL: server.URL}
	p := NewProviderWithConfig(cfg)

	model := p.Model("gpt-5.4-2026-03-05")

	req := &provider.GenerateRequest{
		Prompt: "Say hello",
	}

	result, err := model.(provider.Streamer).Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Collect all parts
	var textParts []string
	var finishPart provider.StreamFinishPart
	var gotFinish bool

	for part := range result.Stream {
		switch p := part.(type) {
		case provider.StreamTextPart:
			textParts = append(textParts, p.Delta)
		case provider.StreamFinishPart:
			finishPart = p
			gotFinish = true
		case provider.StreamErrorPart:
			t.Errorf("Unexpected error part: %v", p.Error)
		}
	}

	if !gotFinish {
		t.Error("Expected finish part in stream")
	}

	// Verify text
	combined := strings.Join(textParts, "")
	if combined != "Hello world" {
		t.Errorf("Expected text %q, got %q", "Hello world", combined)
	}

	// Verify usage
	if finishPart.Usage.TotalTokens != 7 {
		t.Errorf("Expected 7 total tokens, got %d", finishPart.Usage.TotalTokens)
	}

	// Verify finish reason
	if finishPart.FinishReason != provider.FinishReasonStop {
		t.Errorf("Expected finish reason %q, got %q", provider.FinishReasonStop, finishPart.FinishReason)
	}
}

func TestChatModel_StreamWithToolCalls(t *testing.T) {
	// Mock SSE server with tool calls
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Tool call chunks are split across multiple events
		chunks := []string{
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"get_weather","arguments":""}}]},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"location\":"}}]},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"Tokyo\"}"}}]},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}],"usage":{"prompt_tokens":10,"completion_tokens":20,"total_tokens":30}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	cfg := Config{APIKey: "test-key", BaseURL: server.URL}
	p := NewProviderWithConfig(cfg)

	model := p.Model("gpt-5.4-2026-03-05")

	req := &provider.GenerateRequest{
		Prompt: "What's the weather in Tokyo?",
	}

	result, err := model.(provider.Streamer).Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Collect parts
	var toolCallParts []provider.StreamToolCallPart
	var finishPart provider.StreamFinishPart

	for part := range result.Stream {
		switch p := part.(type) {
		case provider.StreamToolCallPart:
			toolCallParts = append(toolCallParts, p)
		case provider.StreamFinishPart:
			finishPart = p
		case provider.StreamErrorPart:
			t.Errorf("Unexpected error part: %v", p.Error)
		}
	}

	// Should have one tool call
	if len(toolCallParts) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(toolCallParts))
	}

	tc := toolCallParts[0].ToolCall
	if tc.ID != "call_123" {
		t.Errorf("Expected tool call ID %q, got %q", "call_123", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("Expected tool name %q, got %q", "get_weather", tc.Name)
	}

	// Verify finish reason
	if finishPart.FinishReason != provider.FinishReasonToolCalls {
		t.Errorf("Expected finish reason %q, got %q", provider.FinishReasonToolCalls, finishPart.FinishReason)
	}
}

func TestChatModel_StreamError(t *testing.T) {
	// Mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error","code":"invalid_api_key"}}`))
	}))
	defer server.Close()

	cfg := Config{APIKey: "invalid-key", BaseURL: server.URL}
	p := NewProviderWithConfig(cfg)

	model := p.Model("gpt-5.4-2026-03-05")

	req := &provider.GenerateRequest{
		Prompt: "test",
	}

	_, err := model.(provider.Streamer).Stream(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("Expected provider.Error, got %T", err)
	}
}

func TestChatModel_StreamAccessorFunctions(t *testing.T) {
	// Test that accessor functions work correctly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{"content":"Test"},"finish_reason":null}]}`,
			`data: {"id":"chatcmpl-test","object":"chat.completion.chunk","created":1234567890,"model":"gpt-5.4-2026-03-05","choices":[{"index":0,"delta":{},"finish_reason":"stop"}],"usage":{"prompt_tokens":1,"completion_tokens":1,"total_tokens":2}}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	cfg := Config{APIKey: "test-key", BaseURL: server.URL}
	p := NewProviderWithConfig(cfg)

	model := p.Model("gpt-5.4-2026-03-05")

	result, err := model.(provider.Streamer).Stream(context.Background(), &provider.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Wait for completion
	<-result.Done

	// Test Text()
	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() error: %v", err)
	}
	if text != "Test" {
		t.Errorf("Expected text %q, got %q", "Test", text)
	}

	// Test Usage()
	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() error: %v", err)
	}
	if usage.TotalTokens != 2 {
		t.Errorf("Expected 2 total tokens, got %d", usage.TotalTokens)
	}

	// Test FinishReason()
	reason, err := result.FinishReason()
	if err != nil {
		t.Fatalf("FinishReason() error: %v", err)
	}
	if reason != provider.FinishReasonStop {
		t.Errorf("Expected %q, got %q", provider.FinishReasonStop, reason)
	}
}

func TestChatModel_StreamMultipleToolCalls(t *testing.T) {
	// Test multiple tool calls in parallel (index 0 and 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"get_weather","arguments":""}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"loc\":"}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"id":"call_2","type":"function","function":{"name":"get_time","arguments":""}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":0,"function":{"arguments":"\"NY\"}"}}]}}]}`,
			`data: {"choices":[{"delta":{"tool_calls":[{"index":1,"function":{"arguments":"NY"}}]}}]}`,
			`data: {"choices":[{"delta":{},"finish_reason":"tool_calls"}]}`,
			`data: [DONE]`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	cfg := Config{APIKey: "test-key", BaseURL: server.URL}
	p := NewProviderWithConfig(cfg)

	model := p.Model("gpt-5.4-2026-03-05")

	result, err := model.(provider.Streamer).Stream(context.Background(), &provider.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Collect tool calls
	var toolCalls []provider.StreamToolCallPart
	for part := range result.Stream {
		if tc, ok := part.(provider.StreamToolCallPart); ok {
			toolCalls = append(toolCalls, tc)
		}
	}

	if len(toolCalls) != 2 {
		t.Fatalf("Expected 2 tool calls, got %d", len(toolCalls))
	}

	// Verify order (should match index order)
	if toolCalls[0].ToolCall.Name != "get_weather" {
		t.Errorf("Expected first tool call name %q, got %q", "get_weather", toolCalls[0].ToolCall.Name)
	}
	if toolCalls[1].ToolCall.Name != "get_time" {
		t.Errorf("Expected second tool call name %q, got %q", "get_time", toolCalls[1].ToolCall.Name)
	}
}

func TestChatModel_StreamWithEOF(t *testing.T) {
	// Test stream ending without explicit [DONE] marker
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		// Send chunks then close connection
		chunks := []string{
			`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
			`data: {"choices":[{"delta":{"content":" world"}}]}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	cfg := Config{APIKey: "test-key", BaseURL: server.URL}
	p := NewProviderWithConfig(cfg)

	model := p.Model("gpt-5.4-2026-03-05")

	result, err := model.(provider.Streamer).Stream(context.Background(), &provider.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Just drain the stream - should handle EOF gracefully
	var textParts []string
	for part := range result.Stream {
		if text, ok := part.(provider.StreamTextPart); ok {
			textParts = append(textParts, text.Delta)
		}
	}

	// Should have received some text before EOF
	combined := strings.Join(textParts, "")
	if combined != "Hello world" {
		t.Errorf("Expected text %q, got %q", "Hello world", combined)
	}
}

func TestChatModel_StreamerInterface(t *testing.T) {
	// Verify that ChatModel implements provider.Streamer interface
	cfg := Config{APIKey: "test-key"}
	p := NewProviderWithConfig(cfg)
	model := p.Model("gpt-5.4-2026-03-05")

	var _ provider.Streamer = model.(*ChatModel)
}

// Ensure io is used
var _ = io.EOF
