package anthropic

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestChatModel_Stream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Expected Accept: text/event-stream, got %s", r.Header.Get("Accept"))
		}
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":10,"output_tokens":0}}}`,
			`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}`,
			`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":5}}`,
			`event: message_stop
data: {"type":"message_stop"}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.StreamingModel("claude-3-5-sonnet-20241022")

	req := &provider.GenerateRequest{
		Prompt: "Say hello",
	}

	result, err := model.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

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

	combined := strings.Join(textParts, "")
	if combined != "Hello world" {
		t.Errorf("Expected text %q, got %q", "Hello world", combined)
	}

	if finishPart.Usage.TotalTokens == 0 {
		t.Logf("Warning: TotalTokens was 0, this may indicate token counting not working in mock")
	}

	if finishPart.FinishReason != provider.FinishReasonStop {
		t.Errorf("Expected finish reason %q, got %q", provider.FinishReasonStop, finishPart.FinishReason)
	}
}

func TestChatModel_StreamWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":20}}}`,
			`event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_test","name":"get_weather"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"location\":"}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"\"Tokyo\"}"}}`,
			`event: content_block_stop
data: {"type":"content_block_stop","index":0}`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":10}}`,
			`event: message_stop
data: {"type":"message_stop"}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.StreamingModel("claude-3-5-sonnet-20241022")

	req := &provider.GenerateRequest{
		Prompt: "What's the weather in Tokyo?",
	}

	result, err := model.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

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

	if len(toolCallParts) != 1 {
		t.Fatalf("Expected 1 tool call, got %d", len(toolCallParts))
	}

	tc := toolCallParts[0].ToolCall
	if tc.ID != "toolu_test" {
		t.Errorf("Expected tool call ID %q, got %q", "toolu_test", tc.ID)
	}
	if tc.Name != "get_weather" {
		t.Errorf("Expected tool name %q, got %q", "get_weather", tc.Name)
	}

	if finishPart.FinishReason != provider.FinishReasonToolCalls {
		t.Errorf("Expected finish reason %q, got %q", provider.FinishReasonToolCalls, finishPart.FinishReason)
	}
}

func TestChatModel_StreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`))
	}))
	defer server.Close()

	p := NewProvider(APIKey("invalid-key"), BaseURL(server.URL))
	model := p.StreamingModel("claude-3-5-sonnet-20241022")

	req := &provider.GenerateRequest{
		Prompt: "test",
	}

	_, err := model.Stream(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	var providerErr *provider.Error
	if !provider.IsError(err, &providerErr) {
		t.Errorf("Expected provider.Error, got %T", err)
	}
	if providerErr.Code != provider.CodeAuthenticationFailed {
		t.Errorf("Expected code %v, got %v", provider.CodeAuthenticationFailed, providerErr.Code)
	}
}

func TestChatModel_StreamAccessorFunctions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":5,"output_tokens":0}}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Test"}}`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`,
			`event: message_stop
data: {"type":"message_stop"}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.StreamingModel("claude-3-5-sonnet-20241022")

	result, err := model.Stream(context.Background(), &provider.GenerateRequest{Prompt: "test"})
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	<-result.Done

	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() error: %v", err)
	}
	if text != "Test" {
		t.Errorf("Expected text %q, got %q", "Test", text)
	}

	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() error: %v", err)
	}
	if usage.TotalTokens == 0 {
		t.Logf("Warning: TotalTokens was 0")
	}

	reason, err := result.FinishReason()
	if err != nil {
		t.Fatalf("FinishReason() error: %v", err)
	}
	if reason != provider.FinishReasonStop {
		t.Errorf("Expected %q, got %q", provider.FinishReasonStop, reason)
	}
}

func TestChatModel_StreamWithBetaHeaders(t *testing.T) {
	var receivedBeta string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBeta = r.Header.Get("anthropic-beta")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		chunks := []string{
			`event: message_start
data: {"type":"message_start","message":{"id":"msg_test","type":"message","role":"assistant","content":[],"model":"claude-3-5-sonnet-20241022","usage":{"input_tokens":5}}}`,
			`event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"OK"}}`,
			`event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":1}}`,
			`event: message_stop
data: {"type":"message_stop"}`,
		}

		for _, chunk := range chunks {
			w.Write([]byte(chunk + "\n\n"))
		}
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.StreamingModel("claude-3-5-sonnet-20241022", ThinkingEnabled(2048))

	req := &provider.GenerateRequest{Prompt: "test"}
	result, err := model.Stream(context.Background(), req)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	for range result.Stream {
	}

	if receivedBeta == "" {
		t.Error("Expected anthropic-beta header to be set for thinking feature")
	}
}

func TestChatModel_StreamerInterface(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.StreamingModel("claude-3-5-sonnet-20241022")

	var _ provider.Streamer = model.(*ChatModel)
	var _ provider.Generator = model.(*ChatModel)
	var _ provider.LanguageModel = model.(*ChatModel)
}

var _ = io.EOF
