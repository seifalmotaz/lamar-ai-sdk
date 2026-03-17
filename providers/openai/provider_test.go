package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/middleware"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestWithMiddleware_Streaming(t *testing.T) {
	t.Run("middleware is applied to stream", func(t *testing.T) {
		var called bool
		testMiddleware := func(next middleware.Handler) middleware.Handler {
			return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
				called = true
				return next.Handle(ctx, req)
			})
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)

			w.Write([]byte("data: {\"id\":\"test\",\"model\":\"gpt-5.4-2026-03-05\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hello\"},\"finish_reason\":null}]}\n\n"))
			flusher.Flush()
			w.Write([]byte("data: {\"id\":\"test\",\"model\":\"gpt-5.4-2026-03-05\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}],\"usage\":{\"prompt_tokens\":5,\"completion_tokens\":1,\"total_tokens\":6}}\n\n"))
			flusher.Flush()
			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		}))
		defer server.Close()

		p := NewProvider(
			APIKey("test-key"),
			BaseURL(server.URL),
			WithMiddleware(testMiddleware),
		)
		model := p.StreamingModel("gpt-5.4-2026-03-05")

		result, err := model.Stream(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var textParts []string
		for part := range result.Stream {
			switch p := part.(type) {
			case provider.StreamTextPart:
				textParts = append(textParts, p.Delta)
			case provider.StreamErrorPart:
				t.Fatalf("stream error: %v", p.Error)
			}
		}

		if !called {
			t.Error("middleware was not called for stream")
		}
		if len(textParts) == 0 {
			t.Error("expected text parts in stream")
		}
	})

	t.Run("timeout middleware works with stream", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		p := NewProvider(
			APIKey("test-key"),
			BaseURL(server.URL),
			WithMiddleware(middleware.TimeoutWithDefault(10*time.Millisecond)),
		)
		model := p.StreamingModel("gpt-5.4-2026-03-05")

		start := time.Now()
		_, err := model.Stream(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})

		elapsed := time.Since(start)
		if elapsed > 100*time.Millisecond {
			t.Errorf("timeout took too long: %v", elapsed)
		}

		if err == nil {
			t.Error("expected timeout error")
		}

		var providerErr *provider.Error
		if !provider.IsError(err, &providerErr) {
			t.Errorf("expected provider.Error, got: %v", err)
		} else if providerErr.Code != provider.CodeAPITimeout {
			t.Errorf("expected CodeAPITimeout, got: %v", providerErr.Code)
		}
	})

	t.Run("no middleware stream passthrough", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher, _ := w.(http.Flusher)

			w.Write([]byte("data: {\"id\":\"test\",\"model\":\"gpt-5.4-2026-03-05\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hi\"},\"finish_reason\":null}]}\n\n"))
			flusher.Flush()
			w.Write([]byte("data: {\"id\":\"test\",\"model\":\"gpt-5.4-2026-03-05\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n"))
			flusher.Flush()
			w.Write([]byte("data: [DONE]\n\n"))
			flusher.Flush()
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.StreamingModel("gpt-5.4-2026-03-05")

		result, err := model.Stream(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		for range result.Stream {
		}

		<-result.Done
		text, err := result.Text()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if text != "Hi" {
			t.Errorf("Text = %q, want %q", text, "Hi")
		}
	})
}

func TestNewProvider(t *testing.T) {
	t.Run("default configuration", func(t *testing.T) {
		p := NewProvider()
		if p.baseURL != DefaultBaseURL {
			t.Errorf("baseURL = %q, want %q", p.baseURL, DefaultBaseURL)
		}
		if p.client == nil {
			t.Error("client is nil")
		}
	})

	t.Run("custom configuration", func(t *testing.T) {
		p := NewProvider(
			APIKey("test-key"),
			BaseURL("https://custom.api.com/v1"),
			OrgID("org-123"),
			ProjectID("proj-456"),
		)
		if p.baseURL != "https://custom.api.com/v1" {
			t.Errorf("baseURL = %q, want %q", p.baseURL, "https://custom.api.com/v1")
		}
		if p.orgID != "org-123" {
			t.Errorf("orgID = %q, want %q", p.orgID, "org-123")
		}
		if p.projectID != "proj-456" {
			t.Errorf("projectID = %q, want %q", p.projectID, "proj-456")
		}
	})

	t.Run("headers set", func(t *testing.T) {
		p := NewProvider(APIKey("test-key"), OrgID("org-123"))
		if p.client.Headers["Authorization"] != "Bearer test-key" {
			t.Errorf("Authorization header = %q, want %q", p.client.Headers["Authorization"], "Bearer test-key")
		}
		if p.client.Headers["OpenAI-Organization"] != "org-123" {
			t.Errorf("OpenAI-Organization header = %q, want %q", p.client.Headers["OpenAI-Organization"], "org-123")
		}
	})
}

func TestProviderModelMethods(t *testing.T) {
	p := NewProvider(APIKey("test-key"))

	t.Run("Model", func(t *testing.T) {
		m := p.Model("gpt-5.4-2026-03-05")
		if m.Provider() != "openai" {
			t.Errorf("Provider() = %q, want %q", m.Provider(), "openai")
		}
		if m.ModelID() != "gpt-5.4-2026-03-05" {
			t.Errorf("ModelID() = %q, want %q", m.ModelID(), "gpt-5.4-2026-03-05")
		}
	})

	t.Run("Embedding", func(t *testing.T) {
		m := p.Embedding("text-embedding-3-small")
		if m.Provider() != "openai" {
			t.Errorf("Provider() = %q, want %q", m.Provider(), "openai")
		}
		if m.ModelID() != "text-embedding-3-small" {
			t.Errorf("ModelID() = %q, want %q", m.ModelID(), "text-embedding-3-small")
		}
		if m.MaxEmbeddingsPerCall() != 2048 {
			t.Errorf("MaxEmbeddingsPerCall() = %d, want 2048", m.MaxEmbeddingsPerCall())
		}
	})
}

func TestProviderConvenienceMethods(t *testing.T) {
	p := NewProvider(APIKey("test-key"))

	tests := []struct {
		name     string
		getModel func() provider.Model
		wantID   string
	}{
		{"GPT5Mini", func() provider.Model { return p.GPT5Mini() }, "gpt-5-mini-2025-08-07"},
		{"GPT51", func() provider.Model { return p.GPT51() }, "gpt-5.1-2025-11-13"},
		{"GPT52", func() provider.Model { return p.GPT52() }, "gpt-5.2-2025-12-11"},
		{"GPT54", func() provider.Model { return p.GPT54() }, "gpt-5.4-2026-03-05"},
		{"GPT4oAudioPreview", func() provider.Model { return p.GPT4oAudioPreview() }, "gpt-4o-audio-preview"},
		{"O1", func() provider.Model { return p.O1() }, "o1"},
		{"O1Mini", func() provider.Model { return p.O1Mini() }, "o1-mini"},
		{"O1Preview", func() provider.Model { return p.O1Preview() }, "o1-preview"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.getModel()
			if m.ModelID() != tt.wantID {
				t.Errorf("ModelID() = %q, want %q", m.ModelID(), tt.wantID)
			}
		})
	}
}

func TestPackageConvenienceFunctions(t *testing.T) {
	tests := []struct {
		name     string
		getModel func() provider.Model
		wantID   string
	}{
		{"GPT5Mini", func() provider.Model { return GPT5Mini() }, "gpt-5-mini-2025-08-07"},
		{"GPT51", func() provider.Model { return GPT51() }, "gpt-5.1-2025-11-13"},
		{"GPT52", func() provider.Model { return GPT52() }, "gpt-5.2-2025-12-11"},
		{"GPT54", func() provider.Model { return GPT54() }, "gpt-5.4-2026-03-05"},
		{"GPT4oAudioPreview", func() provider.Model { return GPT4oAudioPreview() }, "gpt-4o-audio-preview"},
		{"O1", func() provider.Model { return O1() }, "o1"},
		{"O1Mini", func() provider.Model { return O1Mini() }, "o1-mini"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.getModel()
			if m.ModelID() != tt.wantID {
				t.Errorf("ModelID() = %q, want %q", m.ModelID(), tt.wantID)
			}
		})
	}
}

func TestEmbeddingConvenienceMethods(t *testing.T) {
	p := NewProvider(APIKey("test-key"))

	tests := []struct {
		name     string
		getModel func() provider.EmbeddingModel
		wantID   string
	}{
		{"TextEmbedding3Small", p.TextEmbedding3Small, "text-embedding-3-small"},
		{"TextEmbedding3Large", p.TextEmbedding3Large, "text-embedding-3-large"},
		{"TextEmbeddingAda002", p.TextEmbeddingAda002, "text-embedding-ada-002"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.getModel()
			if m.ModelID() != tt.wantID {
				t.Errorf("ModelID() = %q, want %q", m.ModelID(), tt.wantID)
			}
		})
	}
}

func TestChatModelGenerate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing Authorization header")
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/chat/completions")
		}

		resp := ChatCompletionResponse{
			ID:    "test-id",
			Model: "gpt-5.4-2026-03-05",
			Choices: []Choice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Hello! How can I help you?",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{
				PromptTokens:     10,
				CompletionTokens: 8,
				TotalTokens:      18,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("gpt-5.4-2026-03-05")

	result, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "Hello",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello! How can I help you?" {
		t.Errorf("Text = %q, want %q", result.Text, "Hello! How can I help you?")
	}
	if result.FinishReason != provider.FinishReasonStop {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, provider.FinishReasonStop)
	}
	if result.Usage.TotalTokens != 18 {
		t.Errorf("TotalTokens = %d, want 18", result.Usage.TotalTokens)
	}
}

func TestChatModelGenerateWithSystem(t *testing.T) {
	var receivedReq ChatCompletionRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := ChatCompletionResponse{
			ID:    "test-id",
			Model: "gpt-5.4-2026-03-05",
			Choices: []Choice{
				{
					Index: 0,
					Message: ChatMessage{
						Role:    "assistant",
						Content: "Response",
					},
					FinishReason: "stop",
				},
			},
			Usage: Usage{TotalTokens: 10},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("gpt-5.4-2026-03-05")

	_, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "Hello",
		System: "You are a helpful assistant.",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedReq.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(receivedReq.Messages))
	}
	if receivedReq.Messages[0].Role != "system" {
		t.Errorf("first message role = %q, want %q", receivedReq.Messages[0].Role, "system")
	}
	if receivedReq.Messages[0].Content != "You are a helpful assistant." {
		t.Errorf("system content = %v, want %q", receivedReq.Messages[0].Content, "You are a helpful assistant.")
	}
}

func TestChatModelGenerateWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := ChatCompletionResponse{
			ID:    "test-id",
			Model: "gpt-5.4-2026-03-05",
			Choices: []Choice{
				{
					Index: 0,
					Message: ChatMessage{
						Role: "assistant",
						ToolCalls: []ToolCall{
							{
								ID:   "call-123",
								Type: "function",
								Function: FunctionCallData{
									Name:      "get_weather",
									Arguments: `{"location": "Tokyo"}`,
								},
							},
						},
					},
					FinishReason: "tool_calls",
				},
			},
			Usage: Usage{TotalTokens: 20},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("gpt-5.4-2026-03-05")

	result, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "What is the weather in Tokyo?",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.FinishReason != provider.FinishReasonToolCalls {
		t.Errorf("FinishReason = %q, want %q", result.FinishReason, provider.FinishReasonToolCalls)
	}
	if len(result.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(result.ToolCalls))
	}
	if result.ToolCalls[0].ID != "call-123" {
		t.Errorf("ToolCall ID = %q, want %q", result.ToolCalls[0].ID, "call-123")
	}
	if result.ToolCalls[0].Name != "get_weather" {
		t.Errorf("ToolCall Name = %q, want %q", result.ToolCalls[0].Name, "get_weather")
	}
}

func TestChatModelGenerateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorDetail{
				Message: "Incorrect API key provided",
				Type:    "invalid_request_error",
				Code:    "invalid_api_key",
			},
		})
	}))
	defer server.Close()

	p := NewProvider(APIKey("invalid-key"), BaseURL(server.URL))
	model := p.Model("gpt-5.4-2026-03-05")

	_, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "Hello",
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	code := provider.ErrorCodeOf(err)
	if code != provider.CodeAuthenticationFailed {
		t.Errorf("error code = %v, want %v", code, provider.CodeAuthenticationFailed)
	}
}

func TestEmbeddingModelEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing Authorization header")
		}
		if r.URL.Path != "/embeddings" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/embeddings")
		}

		resp := EmbeddingResponse{
			Object: "list",
			Model:  "text-embedding-3-small",
			Data: []EmbeddingData{
				{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2, 0.3}},
				{Object: "embedding", Index: 1, Embedding: []float64{0.4, 0.5, 0.6}},
			},
			Usage: Usage{
				PromptTokens: 10,
				TotalTokens:  10,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Embedding("text-embedding-3-small")

	result, err := model.Embed(context.Background(), &provider.EmbedRequest{
		Texts: []string{"Hello", "World"},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Embeddings) != 2 {
		t.Fatalf("expected 2 embeddings, got %d", len(result.Embeddings))
	}
	if len(result.Embeddings[0]) != 3 {
		t.Errorf("embedding[0] length = %d, want 3", len(result.Embeddings[0]))
	}
	if result.Usage.TotalTokens != 10 {
		t.Errorf("TotalTokens = %d, want 10", result.Usage.TotalTokens)
	}
}

func TestEmbeddingModelEmbedError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: ErrorDetail{
				Message: "The model 'invalid-model' does not exist",
				Type:    "invalid_request_error",
			},
		})
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Embedding("invalid-model")

	_, err := model.Embed(context.Background(), &provider.EmbedRequest{
		Texts: []string{"Hello"},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	code := provider.ErrorCodeOf(err)
	if code != provider.CodeModelNotFound {
		t.Errorf("error code = %v, want %v", code, provider.CodeModelNotFound)
	}
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		input string
		want  provider.FinishReason
	}{
		{"stop", provider.FinishReasonStop},
		{"length", provider.FinishReasonLength},
		{"tool_calls", provider.FinishReasonToolCalls},
		{"content_filter", provider.FinishReasonContentFilter},
		{"unknown", provider.FinishReasonError},
		{"", provider.FinishReasonError},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := mapFinishReason(tt.input)
			if got != tt.want {
				t.Errorf("mapFinishReason(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertMessage_AudioContent(t *testing.T) {
	tests := []struct {
		name      string
		msg       provider.Message
		wantRole  string
		wantParts int
		checkFunc func(t *testing.T, cm ChatMessage)
	}{
		{
			name: "simple text message",
			msg:  provider.UserMessage("Hello"),
			checkFunc: func(t *testing.T, cm ChatMessage) {
				if cm.Role != "user" {
					t.Errorf("role = %q, want %q", cm.Role, "user")
				}
				if cm.Content != "Hello" {
					t.Errorf("content = %q, want %q", cm.Content, "Hello")
				}
			},
		},
		{
			name: "audio content",
			msg: provider.UserMessageWithContent(
				provider.Audio([]byte("test-audio-data"), "audio/wav"),
			),
			checkFunc: func(t *testing.T, cm ChatMessage) {
				if cm.Role != "user" {
					t.Errorf("role = %q, want %q", cm.Role, "user")
				}
				parts, ok := cm.Content.([]ContentPart)
				if !ok {
					t.Fatalf("content is not []ContentPart: %T", cm.Content)
				}
				if len(parts) != 1 {
					t.Fatalf("expected 1 part, got %d", len(parts))
				}
				if parts[0].Type != "input_audio" {
					t.Errorf("part type = %q, want %q", parts[0].Type, "input_audio")
				}
				if parts[0].InputAudio == nil {
					t.Fatal("InputAudio is nil")
				}
				if parts[0].InputAudio.Format != "wav" {
					t.Errorf("format = %q, want %q", parts[0].InputAudio.Format, "wav")
				}
			},
		},
		{
			name: "audio content with mp3 format",
			msg: provider.UserMessageWithContent(
				provider.Audio([]byte("mp3-data"), "audio/mp3"),
			),
			checkFunc: func(t *testing.T, cm ChatMessage) {
				parts, ok := cm.Content.([]ContentPart)
				if !ok {
					t.Fatalf("content is not []ContentPart: %T", cm.Content)
				}
				if len(parts) != 1 {
					t.Fatalf("expected 1 part, got %d", len(parts))
				}
				if parts[0].InputAudio == nil {
					t.Fatal("InputAudio is nil")
				}
				if parts[0].InputAudio.Format != "mp3" {
					t.Errorf("format = %q, want %q", parts[0].InputAudio.Format, "mp3")
				}
			},
		},
		{
			name: "mixed text and audio content",
			msg: provider.UserMessageWithContent(
				provider.Text("What is in this audio?"),
				provider.Audio([]byte("audio-data"), "audio/webm"),
			),
			checkFunc: func(t *testing.T, cm ChatMessage) {
				parts, ok := cm.Content.([]ContentPart)
				if !ok {
					t.Fatalf("content is not []ContentPart: %T", cm.Content)
				}
				if len(parts) != 2 {
					t.Fatalf("expected 2 parts, got %d", len(parts))
				}
				if parts[0].Type != "text" {
					t.Errorf("first part type = %q, want %q", parts[0].Type, "text")
				}
				if parts[1].Type != "input_audio" {
					t.Errorf("second part type = %q, want %q", parts[1].Type, "input_audio")
				}
				if parts[1].InputAudio.Format != "webm" {
					t.Errorf("audio format = %q, want %q", parts[1].InputAudio.Format, "webm")
				}
			},
		},
		{
			name: "image content",
			msg: provider.UserMessageWithContent(
				provider.Image([]byte("image-data"), "image/png"),
			),
			checkFunc: func(t *testing.T, cm ChatMessage) {
				parts, ok := cm.Content.([]ContentPart)
				if !ok {
					t.Fatalf("content is not []ContentPart: %T", cm.Content)
				}
				if len(parts) != 1 {
					t.Fatalf("expected 1 part, got %d", len(parts))
				}
				if parts[0].Type != "image_url" {
					t.Errorf("part type = %q, want %q", parts[0].Type, "image_url")
				}
				if parts[0].ImageURL == nil {
					t.Fatal("ImageURL is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := convertMessage(tt.msg)
			if tt.checkFunc != nil {
				tt.checkFunc(t, cm)
			}
		})
	}
}

func TestExtractAudioFormat(t *testing.T) {
	tests := []struct {
		mediaType string
		want      string
	}{
		{"audio/wav", "wav"},
		{"audio/mp3", "mp3"},
		{"audio/webm", "webm"},
		{"audio/m4a", "m4a"},
		{"audio/mpeg", "mpeg"},
		{"", "wav"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			got := extractAudioFormat(tt.mediaType)
			if got != tt.want {
				t.Errorf("extractAudioFormat(%q) = %q, want %q", tt.mediaType, got, tt.want)
			}
		})
	}
}

func TestConvertToolChoice(t *testing.T) {
	tests := []struct {
		name string
		tc   provider.ToolChoice
		want any
	}{
		{"auto", provider.ToolChoiceAuto(), "auto"},
		{"none", provider.ToolChoiceNone(), "none"},
		{"required", provider.ToolChoiceRequired(), "required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToolChoice(tt.tc)
			if got != tt.want {
				t.Errorf("convertToolChoice() = %v, want %v", got, tt.want)
			}
		})
	}

	t.Run("named tool", func(t *testing.T) {
		tc := provider.ToolChoiceNamed("get_weather")
		got := convertToolChoice(tc)
		m, ok := got.(map[string]any)
		if !ok {
			t.Fatalf("expected map, got %T", got)
		}
		fn, ok := m["function"].(map[string]string)
		if !ok {
			t.Fatalf("expected function map, got %T", m["function"])
		}
		if fn["name"] != "get_weather" {
			t.Errorf("function name = %q, want %q", fn["name"], "get_weather")
		}
	})
}

func TestWithMiddleware(t *testing.T) {
	t.Run("middleware is applied to generate", func(t *testing.T) {
		var called bool
		testMiddleware := func(next middleware.Handler) middleware.Handler {
			return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
				called = true
				return next.Handle(ctx, req)
			})
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				ID:    "test-id",
				Model: "gpt-5.4-2026-03-05",
				Choices: []Choice{
					{
						Index: 0,
						Message: ChatMessage{
							Role:    "assistant",
							Content: "Response",
						},
						FinishReason: "stop",
					},
				},
				Usage: Usage{TotalTokens: 10},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(
			APIKey("test-key"),
			BaseURL(server.URL),
			WithMiddleware(testMiddleware),
		)
		model := p.Model("gpt-5.4-2026-03-05")

		_, err := model.Generate(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Error("middleware was not called")
		}
	})

	t.Run("middleware is applied to embed", func(t *testing.T) {
		var called bool
		testMiddleware := func(next middleware.Handler) middleware.Handler {
			return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
				called = true
				return next.Handle(ctx, req)
			})
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := EmbeddingResponse{
				Object: "list",
				Model:  "text-embedding-3-small",
				Data: []EmbeddingData{
					{Object: "embedding", Index: 0, Embedding: []float64{0.1, 0.2}},
				},
				Usage: Usage{TotalTokens: 5},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(
			APIKey("test-key"),
			BaseURL(server.URL),
			WithMiddleware(testMiddleware),
		)
		model := p.Embedding("text-embedding-3-small")

		_, err := model.Embed(context.Background(), &provider.EmbedRequest{
			Texts: []string{"Hello"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !called {
			t.Error("middleware was not called")
		}
	})

	t.Run("timeout middleware works with generate", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(100 * time.Millisecond)
			resp := ChatCompletionResponse{
				ID:    "test-id",
				Model: "gpt-5.4-2026-03-05",
				Choices: []Choice{
					{Index: 0, Message: ChatMessage{Role: "assistant", Content: "Response"}, FinishReason: "stop"},
				},
				Usage: Usage{TotalTokens: 10},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(
			APIKey("test-key"),
			BaseURL(server.URL),
			WithMiddleware(middleware.TimeoutWithDefault(10*time.Millisecond)),
		)
		model := p.Model("gpt-5.4-2026-03-05")

		start := time.Now()
		_, err := model.Generate(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})

		elapsed := time.Since(start)
		if elapsed > 50*time.Millisecond {
			t.Errorf("timeout took too long: %v", elapsed)
		}

		if err == nil {
			t.Error("expected timeout error")
		}

		var providerErr *provider.Error
		if !provider.IsError(err, &providerErr) {
			t.Errorf("expected provider.Error, got: %v", err)
		} else if providerErr.Code != provider.CodeAPITimeout {
			t.Errorf("expected CodeAPITimeout, got: %v", providerErr.Code)
		}
	})

	t.Run("no middleware passthrough", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				ID:    "test-id",
				Model: "gpt-5.4-2026-03-05",
				Choices: []Choice{
					{Index: 0, Message: ChatMessage{Role: "assistant", Content: "Response"}, FinishReason: "stop"},
				},
				Usage: Usage{TotalTokens: 10},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Model("gpt-5.4-2026-03-05")

		result, err := model.Generate(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Text != "Response" {
			t.Errorf("Text = %q, want %q", result.Text, "Response")
		}
	})

	t.Run("multiple middlewares chain correctly", func(t *testing.T) {
		var order []string

		middleware1 := func(next middleware.Handler) middleware.Handler {
			return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
				order = append(order, "middleware1-before")
				resp, err := next.Handle(ctx, req)
				order = append(order, "middleware1-after")
				return resp, err
			})
		}

		middleware2 := func(next middleware.Handler) middleware.Handler {
			return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
				order = append(order, "middleware2-before")
				resp, err := next.Handle(ctx, req)
				order = append(order, "middleware2-after")
				return resp, err
			})
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ChatCompletionResponse{
				ID:    "test-id",
				Model: "gpt-5.4-2026-03-05",
				Choices: []Choice{
					{Index: 0, Message: ChatMessage{Role: "assistant", Content: "Response"}, FinishReason: "stop"},
				},
				Usage: Usage{TotalTokens: 10},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(
			APIKey("test-key"),
			BaseURL(server.URL),
			WithMiddleware(middleware1, middleware2),
		)
		model := p.Model("gpt-5.4-2026-03-05")

		_, err := model.Generate(context.Background(), &provider.GenerateRequest{
			Prompt: "Hello",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{
			"middleware1-before",
			"middleware2-before",
			"middleware2-after",
			"middleware1-after",
		}
		if len(order) != len(expected) {
			t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
		}
		for i, v := range expected {
			if order[i] != v {
				t.Errorf("order[%d] = %q, want %q", i, order[i], v)
			}
		}
	})
}
