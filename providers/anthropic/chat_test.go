package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestChatModel_Generate(t *testing.T) {
	var receivedReq MessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("missing anthropic-version header")
		}
		if r.URL.Path != "/messages" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/messages")
		}

		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := MessageResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []ContentBlock{
				{Type: "text", Text: "Hello! How can I help you?"},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage: UsageResponse{
				InputTokens:  10,
				OutputTokens: 8,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022")

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

	if receivedReq.Model != "claude-3-5-sonnet-20241022" {
		t.Errorf("Model = %q, want %q", receivedReq.Model, "claude-3-5-sonnet-20241022")
	}
}

func TestChatModel_GenerateWithSystem(t *testing.T) {
	var receivedReq MessageRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&receivedReq); err != nil {
			t.Errorf("failed to decode request: %v", err)
		}

		resp := MessageResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []ContentBlock{
				{Type: "text", Text: "Response"},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage:      UsageResponse{InputTokens: 10, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022")

	_, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "Hello",
		Config: provider.Config{
			System: "You are a helpful assistant.",
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedReq.System) != 1 {
		t.Fatalf("expected 1 system block, got %d", len(receivedReq.System))
	}
	if receivedReq.System[0].Type != "text" {
		t.Errorf("system block type = %q, want %q", receivedReq.System[0].Type, "text")
	}
	if receivedReq.System[0].Text != "You are a helpful assistant." {
		t.Errorf("system block text = %q, want %q", receivedReq.System[0].Text, "You are a helpful assistant.")
	}
}

func TestChatModel_GenerateWithToolCalls(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := MessageResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []ContentBlock{
				{
					Type:  "tool_use",
					ID:    "toolu_test",
					Name:  "get_weather",
					Input: json.RawMessage(`{"location": "Tokyo"}`),
				},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "tool_use",
			Usage:      UsageResponse{InputTokens: 20, OutputTokens: 10},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022")

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

	tc := result.ToolCalls[0]
	if tc.ID != "toolu_test" {
		t.Errorf("ToolCall ID = %q, want %q", tc.ID, "toolu_test")
	}
	if tc.Name != "get_weather" {
		t.Errorf("ToolCall Name = %q, want %q", tc.Name, "get_weather")
	}
}

func TestChatModel_GenerateWithThinking(t *testing.T) {
	var receivedBeta string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBeta = r.Header.Get("anthropic-beta")

		var req MessageRequest
		json.NewDecoder(r.Body).Decode(&req)

		resp := MessageResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []ContentBlock{
				{Type: "thinking", Thinking: "Let me think about this..."},
				{Type: "text", Text: "The answer is 42."},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage:      UsageResponse{InputTokens: 10, OutputTokens: 20},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022", ThinkingEnabled(2048))

	result, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "What is the answer?",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedBeta == "" {
		t.Error("Expected anthropic-beta header for thinking feature")
	}

	if result.Text != "The answer is 42." {
		t.Errorf("Text = %q, want %q", result.Text, "The answer is 42.")
	}

	if len(result.Content) != 2 {
		t.Errorf("Expected 2 content blocks, got %d", len(result.Content))
	}
}

func TestChatModel_GenerateError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ErrorResponse{
			Type: "error",
			Error: ErrorDetail{
				Type:    "authentication_error",
				Message: "Invalid API key",
			},
		})
	}))
	defer server.Close()

	p := NewProvider(APIKey("invalid-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022")

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

func TestChatModel_GenerateWithMessages(t *testing.T) {
	var receivedReq MessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := MessageResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []ContentBlock{
				{Type: "text", Text: "Response"},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage:      UsageResponse{InputTokens: 15, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022")

	result, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Messages: []provider.Message{
			provider.UserMessage("Hello"),
			provider.AssistantMessage("Hi there!"),
			provider.UserMessage("How are you?"),
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Text != "Response" {
		t.Errorf("Text = %q, want %q", result.Text, "Response")
	}

	if len(receivedReq.Messages) != 3 {
		t.Errorf("Expected 3 messages, got %d", len(receivedReq.Messages))
	}
}

func TestChatModel_GenerateWithTools(t *testing.T) {
	var receivedReq MessageRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&receivedReq)

		resp := MessageResponse{
			ID:   "msg_test",
			Type: "message",
			Role: "assistant",
			Content: []ContentBlock{
				{Type: "text", Text: "Response"},
			},
			Model:      "claude-3-5-sonnet-20241022",
			StopReason: "end_turn",
			Usage:      UsageResponse{InputTokens: 30, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Model("claude-3-5-sonnet-20241022")

	toolSchema := json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`)
	tools := []provider.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get weather for a location",
			InputSchema: toolSchema,
		},
	}

	_, err := model.Generate(context.Background(), &provider.GenerateRequest{
		Prompt: "What's the weather?",
		Config: provider.Config{
			Tools:      tools,
			ToolChoice: provider.ToolChoiceAuto(),
		},
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedReq.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(receivedReq.Tools))
	}

	if receivedReq.Tools[0].Name != "get_weather" {
		t.Errorf("Tool name = %q, want %q", receivedReq.Tools[0].Name, "get_weather")
	}
}

func TestBuildRequest_ContentTypes(t *testing.T) {
	tests := []struct {
		name         string
		messages     []provider.Message
		wantRole     string
		wantType     string
		wantSubType  string
		checkContent func(t *testing.T, content []ContentBlock)
	}{
		{
			name:     "text content",
			messages: []provider.Message{provider.UserMessage("Hello")},
			wantRole: "user",
			checkContent: func(t *testing.T, content []ContentBlock) {
				if len(content) != 1 {
					t.Errorf("expected 1 content block, got %d", len(content))
				}
				if content[0].Type != "text" {
					t.Errorf("content type = %q, want %q", content[0].Type, "text")
				}
			},
		},
		{
			name: "image content",
			messages: []provider.Message{
				provider.UserMessageWithContent(
					provider.Image([]byte("test"), "image/png"),
				),
			},
			wantRole: "user",
			wantType: "image",
			checkContent: func(t *testing.T, content []ContentBlock) {
				if len(content) != 1 {
					t.Errorf("expected 1 content block, got %d", len(content))
					return
				}
				if content[0].Type != "image" {
					t.Errorf("content type = %q, want %q", content[0].Type, "image")
				}
			},
		},
		{
			name: "tool call content",
			messages: []provider.Message{
				{
					Role: provider.RoleAssistant,
					Content: []provider.Content{
						provider.NewToolCallContent("toolu_123", "get_weather", json.RawMessage(`{"location":"Tokyo"}`)),
					},
				},
			},
			wantRole: "assistant",
			checkContent: func(t *testing.T, content []ContentBlock) {
				if len(content) != 1 {
					t.Errorf("expected 1 content block, got %d", len(content))
					return
				}
				if content[0].Type != "tool_use" {
					t.Errorf("content type = %q, want %q", content[0].Type, "tool_use")
				}
			},
		},
		{
			name: "tool result content",
			messages: []provider.Message{
				provider.ToolResultMessage(provider.ToolResultContent{
					ID:     "toolu_123",
					Name:   "get_weather",
					Result: json.RawMessage(`{"temperature":22}`),
				}),
			},
			wantRole: "user",
			checkContent: func(t *testing.T, content []ContentBlock) {
				if len(content) != 1 {
					t.Errorf("expected 1 content block, got %d", len(content))
					return
				}
				if content[0].Type != "tool_result" {
					t.Errorf("content type = %q, want %q", content[0].Type, "tool_result")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messages, _, err := convertMessages(tt.messages, "")
			if err != nil {
				t.Fatalf("convertMessages error: %v", err)
			}

			if len(messages) == 0 {
				t.Fatal("expected at least one message")
			}

			if messages[0].Role != tt.wantRole {
				t.Errorf("role = %q, want %q", messages[0].Role, tt.wantRole)
			}

			if tt.checkContent != nil {
				tt.checkContent(t, messages[0].Content)
			}
		})
	}
}

func TestConvertMCPServers(t *testing.T) {
	servers := []MCPServerConfig{
		{
			Type:               "url",
			Name:               "test-server",
			URL:                "https://api.example.com/mcp",
			AuthorizationToken: "test-token",
			ToolConfiguration: &MCPToolConfiguration{
				Enabled:      true,
				AllowedTools: []string{"tool1", "tool2"},
			},
		},
	}

	result := convertMCPServers(servers)

	if len(result) != 1 {
		t.Fatalf("expected 1 server, got %d", len(result))
	}

	if result[0].Name != "test-server" {
		t.Errorf("name = %q, want %q", result[0].Name, "test-server")
	}

	if result[0].ToolConfiguration == nil {
		t.Error("expected tool configuration")
	}

	if result[0].ToolConfiguration.Enabled != true {
		t.Error("expected enabled")
	}
}

func TestConvertContainer(t *testing.T) {
	config := &ContainerConfig{
		ID: "container-123",
		Skills: []ContainerSkill{
			{Type: "anthropic", SkillID: "code_execution"},
		},
	}

	result := convertContainer(config)

	if result == nil {
		t.Fatal("expected container, got nil")
	}

	if result.ID != "container-123" {
		t.Errorf("ID = %q, want %q", result.ID, "container-123")
	}

	if len(result.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(result.Skills))
	}
}

func TestBuildExtraHeaders_WithThinking(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.Model("claude-3-5-sonnet-20241022", ThinkingEnabled(2048)).(*ChatModel)

	headers := model.buildExtraHeaders()

	if _, ok := headers["anthropic-beta"]; !ok {
		t.Error("Expected anthropic-beta header with thinking enabled")
	}
}

func TestBuildExtraHeaders_WithoutThinking(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.Model("claude-3-5-sonnet-20241022").(*ChatModel)

	headers := model.buildExtraHeaders()

	if len(headers) > 0 {
		t.Errorf("Expected no extra headers without thinking, got %v", headers)
	}
}
