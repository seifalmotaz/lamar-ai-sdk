package ollama

import (
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestConvertMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []provider.Message
		want     []ChatMessage
	}{
		{
			name: "simple user message",
			messages: []provider.Message{
				provider.UserMessage("Hello"),
			},
			want: []ChatMessage{
				{Role: "user", Content: "Hello"},
			},
		},
		{
			name: "system and user message",
			messages: []provider.Message{
				provider.SystemMessage("You are helpful"),
				provider.UserMessage("Hello"),
			},
			want: []ChatMessage{
				{Role: "system", Content: "You are helpful"},
				{Role: "user", Content: "Hello"},
			},
		},
		{
			name: "assistant message",
			messages: []provider.Message{
				provider.UserMessage("Hello"),
				provider.AssistantMessage("Hi there!"),
			},
			want: []ChatMessage{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi there!"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertMessages(tt.messages)
			if len(got) != len(tt.want) {
				t.Errorf("convertMessages() got %d messages, want %d", len(got), len(tt.want))
				return
			}
			for i, msg := range got {
				if msg.Role != tt.want[i].Role {
					t.Errorf("convertMessages()[%d].Role = %s, want %s", i, msg.Role, tt.want[i].Role)
				}
				if msg.Content != tt.want[i].Content {
					t.Errorf("convertMessages()[%d].Content = %s, want %s", i, msg.Content, tt.want[i].Content)
				}
			}
		})
	}
}

func TestConvertTools(t *testing.T) {
	tools := []provider.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get weather info",
			InputSchema: []byte(`{"type":"object","properties":{"location":{"type":"string"}}}`),
		},
	}

	got := convertTools(tools)

	if len(got) != 1 {
		t.Fatalf("convertTools() got %d tools, want 1", len(got))
	}

	if got[0].Type != "function" {
		t.Errorf("convertTools()[0].Type = %s, want function", got[0].Type)
	}

	if got[0].Function.Name != "get_weather" {
		t.Errorf("convertTools()[0].Function.Name = %s, want get_weather", got[0].Function.Name)
	}

	if got[0].Function.Description != "Get weather info" {
		t.Errorf("convertTools()[0].Function.Description = %s, want Get weather info", got[0].Function.Description)
	}
}

func TestConvertToolChoice(t *testing.T) {
	tests := []struct {
		name string
		tc   provider.ToolChoice
		want any
	}{
		{
			name: "auto",
			tc:   provider.ToolChoiceAuto(),
			want: "auto",
		},
		{
			name: "none",
			tc:   provider.ToolChoiceNone(),
			want: "none",
		},
		{
			name: "required",
			tc:   provider.ToolChoiceRequired(),
			want: "required",
		},
		{
			name: "named tool",
			tc:   provider.ToolChoiceNamed("get_weather"),
			want: map[string]any{"type": "function", "name": "get_weather"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := convertToolChoice(tt.tc)
			switch v := got.(type) {
			case string:
				if v != tt.want.(string) {
					t.Errorf("convertToolChoice() = %v, want %v", got, tt.want)
				}
			case map[string]any:
				wantMap := tt.want.(map[string]any)
				if v["type"] != wantMap["type"] || v["name"] != wantMap["name"] {
					t.Errorf("convertToolChoice() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMapFinishReason(t *testing.T) {
	tests := []struct {
		reason string
		want   provider.FinishReason
	}{
		{"stop", provider.FinishReasonStop},
		{"length", provider.FinishReasonLength},
		{"tool_calls", provider.FinishReasonToolCalls},
		{"unknown", provider.FinishReasonStop},
		{"", provider.FinishReasonStop},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			if got := mapFinishReason(tt.reason); got != tt.want {
				t.Errorf("mapFinishReason(%q) = %v, want %v", tt.reason, got, tt.want)
			}
		})
	}
}

func TestNewProvider(t *testing.T) {
	p := NewProvider()
	if p == nil {
		t.Fatal("NewProvider() returned nil")
	}

	if p.baseURL != DefaultBaseURL {
		t.Errorf("NewProvider() baseURL = %s, want %s", p.baseURL, DefaultBaseURL)
	}
}

func TestNewProviderWithOptions(t *testing.T) {
	customURL := "http://localhost:8080"
	p := NewProvider(BaseURL(customURL))

	if p.baseURL != customURL {
		t.Errorf("NewProvider(BaseURL) baseURL = %s, want %s", p.baseURL, customURL)
	}
}

func TestProviderModelMethods(t *testing.T) {
	p := NewProvider()

	gen := p.Model("llama3.2")
	if gen == nil {
		t.Fatal("Provider.Model() returned nil")
	}
	if gen.Provider() != "ollama" {
		t.Errorf("Model.Provider() = %s, want ollama", gen.Provider())
	}
	if gen.ModelID() != "llama3.2" {
		t.Errorf("Model.ModelID() = %s, want llama3.2", gen.ModelID())
	}

	lm := p.StreamingModel("llama3.2")
	if lm == nil {
		t.Fatal("Provider.StreamingModel() returned nil")
	}
	if lm.Provider() != "ollama" {
		t.Errorf("StreamingModel.Provider() = %s, want ollama", lm.Provider())
	}

	emb := p.Embedding("nomic-embed-text")
	if emb == nil {
		t.Fatal("Provider.Embedding() returned nil")
	}
	if emb.Provider() != "ollama" {
		t.Errorf("Embedding.Provider() = %s, want ollama", emb.Provider())
	}
}

func TestPredefinedModels(t *testing.T) {
	p := NewProvider()

	llama := p.Llama32()
	if llama.ModelID() != "llama3.2" {
		t.Errorf("Llama32().ModelID() = %s, want llama3.2", llama.ModelID())
	}

	qwen := p.Qwen25()
	if qwen.ModelID() != "qwen2.5" {
		t.Errorf("Qwen25().ModelID() = %s, want qwen2.5", qwen.ModelID())
	}

	embed := p.NomicEmbedText()
	if embed.ModelID() != "nomic-embed-text" {
		t.Errorf("NomicEmbedText().ModelID() = %s, want nomic-embed-text", embed.ModelID())
	}
}

func TestMessageWithToolCalls(t *testing.T) {
	toolCall := provider.NewToolCallContent("call_123", "get_weather", []byte(`{"location":"Tokyo"}`))
	msg := provider.AssistantMessageWithToolCalls(toolCall)

	converted := convertMessage(msg)
	ollamaMsg, ok := converted.(ChatMessage)
	if !ok {
		t.Fatal("convertMessage() did not return ChatMessage")
	}

	if ollamaMsg.Role != "assistant" {
		t.Errorf("Role = %s, want assistant", ollamaMsg.Role)
	}

	if len(ollamaMsg.ToolCalls) != 1 {
		t.Fatalf("len(ToolCalls) = %d, want 1", len(ollamaMsg.ToolCalls))
	}

	if ollamaMsg.ToolCalls[0].Function.Name != "get_weather" {
		t.Errorf("ToolCalls[0].Function.Name = %s, want get_weather", ollamaMsg.ToolCalls[0].Function.Name)
	}
}

func TestMessageWithImage(t *testing.T) {
	imageData := []byte("fake-image-data")
	msg := provider.UserMessageWithContent(
		provider.Text("What's in this image?"),
		provider.Image(imageData, "image/png"),
	)

	converted := convertMessage(msg)
	ollamaMsg, ok := converted.(ChatMessage)
	if !ok {
		t.Fatal("convertMessage() did not return ChatMessage")
	}

	if ollamaMsg.Role != "user" {
		t.Errorf("Role = %s, want user", ollamaMsg.Role)
	}

	if len(ollamaMsg.Images) != 1 {
		t.Errorf("len(Images) = %d, want 1", len(ollamaMsg.Images))
	}
}

func TestToolResultMessage(t *testing.T) {
	result := provider.NewToolResultContent("call_123", "get_weather", []byte(`{"temp":22}`), false)
	msg := provider.ToolResultMessage(result)

	converted := convertMessage(msg)
	ollamaMsg, ok := converted.(ChatMessage)
	if !ok {
		t.Fatal("convertMessage() did not return ChatMessage")
	}

	if ollamaMsg.Role != "tool" {
		t.Errorf("Role = %s, want tool", ollamaMsg.Role)
	}
}
