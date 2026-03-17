package anthropic

import (
	"encoding/json"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestConvertMessages(t *testing.T) {
	t.Run("simple user message", func(t *testing.T) {
		messages := []provider.Message{
			provider.UserMessage("Hello, world!"),
		}

		apiMessages, systemBlocks, err := convertMessages(messages, "")
		if err != nil {
			t.Fatalf("convertMessages error: %v", err)
		}

		if len(systemBlocks) != 0 {
			t.Errorf("expected 0 system blocks, got %d", len(systemBlocks))
		}

		if len(apiMessages) != 1 {
			t.Fatalf("expected 1 message, got %d", len(apiMessages))
		}

		if apiMessages[0].Role != "user" {
			t.Errorf("role = %q, want %q", apiMessages[0].Role, "user")
		}

		if len(apiMessages[0].Content) != 1 {
			t.Fatalf("expected 1 content block, got %d", len(apiMessages[0].Content))
		}

		if apiMessages[0].Content[0].Type != "text" {
			t.Errorf("content type = %q, want %q", apiMessages[0].Content[0].Type, "text")
		}

		if apiMessages[0].Content[0].Text != "Hello, world!" {
			t.Errorf("text = %q, want %q", apiMessages[0].Content[0].Text, "Hello, world!")
		}
	})

	t.Run("system prompt", func(t *testing.T) {
		messages := []provider.Message{
			provider.UserMessage("Hello"),
		}

		_, systemBlocks, err := convertMessages(messages, "You are a helpful assistant.")
		if err != nil {
			t.Fatalf("convertMessages error: %v", err)
		}

		if len(systemBlocks) != 1 {
			t.Fatalf("expected 1 system block, got %d", len(systemBlocks))
		}

		if systemBlocks[0].Type != "text" {
			t.Errorf("system block type = %q, want %q", systemBlocks[0].Type, "text")
		}

		if systemBlocks[0].Text != "You are a helpful assistant." {
			t.Errorf("system block text = %q, want %q", systemBlocks[0].Text, "You are a helpful assistant.")
		}
	})

	t.Run("system message in messages", func(t *testing.T) {
		messages := []provider.Message{
			{
				Role:    provider.RoleSystem,
				Content: []provider.Content{provider.Text("System instruction")},
			},
			provider.UserMessage("Hello"),
		}

		apiMsgs, systemBlocks, err := convertMessages(messages, "")
		if err != nil {
			t.Fatalf("convertMessages error: %v", err)
		}

		if len(systemBlocks) != 1 {
			t.Fatalf("expected 1 system block, got %d", len(systemBlocks))
		}

		if len(apiMsgs) != 1 {
			t.Errorf("expected 1 API message, got %d", len(apiMsgs))
		}
	})

	t.Run("assistant message", func(t *testing.T) {
		messages := []provider.Message{
			provider.AssistantMessage("Hi there!"),
		}

		apiMessages, _, err := convertMessages(messages, "")
		if err != nil {
			t.Fatalf("convertMessages error: %v", err)
		}

		if apiMessages[0].Role != "assistant" {
			t.Errorf("role = %q, want %q", apiMessages[0].Role, "assistant")
		}
	})
}

func TestConvertContent(t *testing.T) {
	t.Run("text content", func(t *testing.T) {
		contents := []provider.Content{
			provider.Text("Hello"),
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if len(blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(blocks))
		}

		if blocks[0].Type != "text" {
			t.Errorf("type = %q, want %q", blocks[0].Type, "text")
		}

		if blocks[0].Text != "Hello" {
			t.Errorf("text = %q, want %q", blocks[0].Text, "Hello")
		}
	})

	t.Run("image content base64", func(t *testing.T) {
		imageData := []byte("test-image-data")
		contents := []provider.Content{
			provider.Image(imageData, "image/png"),
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if len(blocks) != 1 {
			t.Fatalf("expected 1 block, got %d", len(blocks))
		}

		if blocks[0].Type != "image" {
			t.Errorf("type = %q, want %q", blocks[0].Type, "image")
		}

		if blocks[0].Source == nil {
			t.Fatal("expected source to be set")
		}

		if blocks[0].Source.Type != "base64" {
			t.Errorf("source type = %q, want %q", blocks[0].Source.Type, "base64")
		}

		if blocks[0].Source.MediaType != "image/png" {
			t.Errorf("media type = %q, want %q", blocks[0].Source.MediaType, "image/png")
		}
	})

	t.Run("image content from URL", func(t *testing.T) {
		contents := []provider.Content{
			provider.ImageFromURL("https://example.com/image.png"),
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if blocks[0].Source.Type != "url" {
			t.Errorf("source type = %q, want %q", blocks[0].Source.Type, "url")
		}

		if blocks[0].Source.URL != "https://example.com/image.png" {
			t.Errorf("url = %q, want %q", blocks[0].Source.URL, "https://example.com/image.png")
		}
	})

	t.Run("tool call content", func(t *testing.T) {
		input := json.RawMessage(`{"location":"Tokyo"}`)
		contents := []provider.Content{
			provider.NewToolCallContent("toolu_123", "get_weather", input),
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if blocks[0].Type != "tool_use" {
			t.Errorf("type = %q, want %q", blocks[0].Type, "tool_use")
		}

		if blocks[0].ID != "toolu_123" {
			t.Errorf("id = %q, want %q", blocks[0].ID, "toolu_123")
		}

		if blocks[0].Name != "get_weather" {
			t.Errorf("name = %q, want %q", blocks[0].Name, "get_weather")
		}
	})

	t.Run("tool result content", func(t *testing.T) {
		result := json.RawMessage(`{"temperature":22}`)
		contents := []provider.Content{
			provider.ToolResultContent{
				ID:     "toolu_123",
				Name:   "get_weather",
				Result: result,
			},
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if blocks[0].Type != "tool_result" {
			t.Errorf("type = %q, want %q", blocks[0].Type, "tool_result")
		}

		if blocks[0].ToolUseID != "toolu_123" {
			t.Errorf("tool_use_id = %q, want %q", blocks[0].ToolUseID, "toolu_123")
		}
	})

	t.Run("reasoning content", func(t *testing.T) {
		contents := []provider.Content{
			provider.ReasoningContent{Text: "Let me think about this..."},
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if blocks[0].Type != "thinking" {
			t.Errorf("type = %q, want %q", blocks[0].Type, "thinking")
		}

		if blocks[0].Thinking != "Let me think about this..." {
			t.Errorf("thinking = %q, want %q", blocks[0].Thinking, "Let me think about this...")
		}
	})

	t.Run("multiple content blocks", func(t *testing.T) {
		contents := []provider.Content{
			provider.Text("Hello"),
			provider.Text("World"),
		}

		blocks, err := convertContent(contents)
		if err != nil {
			t.Fatalf("convertContent error: %v", err)
		}

		if len(blocks) != 2 {
			t.Errorf("expected 2 blocks, got %d", len(blocks))
		}
	})
}

func TestConvertResponseContent(t *testing.T) {
	t.Run("text block", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: "text", Text: "Hello"},
		}

		contents, text, toolCalls, err := convertResponseContent(blocks)
		if err != nil {
			t.Fatalf("convertResponseContent error: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("expected 1 content, got %d", len(contents))
		}

		if text != "Hello" {
			t.Errorf("text = %q, want %q", text, "Hello")
		}

		if len(toolCalls) != 0 {
			t.Errorf("expected 0 tool calls, got %d", len(toolCalls))
		}
	})

	t.Run("tool use block", func(t *testing.T) {
		input := json.RawMessage(`{"location":"Tokyo"}`)
		blocks := []ContentBlock{
			{Type: "tool_use", ID: "toolu_123", Name: "get_weather", Input: input},
		}

		contents, _, toolCalls, err := convertResponseContent(blocks)
		if err != nil {
			t.Fatalf("convertResponseContent error: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("expected 1 content, got %d", len(contents))
		}

		if tc, ok := contents[0].(provider.ToolCallContent); !ok {
			t.Error("expected ToolCallContent")
		} else {
			if tc.ID != "toolu_123" {
				t.Errorf("id = %q, want %q", tc.ID, "toolu_123")
			}
		}

		if len(toolCalls) != 1 {
			t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
		}

		if toolCalls[0].Name != "get_weather" {
			t.Errorf("tool call name = %q, want %q", toolCalls[0].Name, "get_weather")
		}
	})

	t.Run("thinking block", func(t *testing.T) {
		blocks := []ContentBlock{
			{Type: "thinking", Thinking: "Let me think..."},
		}

		contents, _, _, err := convertResponseContent(blocks)
		if err != nil {
			t.Fatalf("convertResponseContent error: %v", err)
		}

		if len(contents) != 1 {
			t.Fatalf("expected 1 content, got %d", len(contents))
		}

		if rc, ok := contents[0].(provider.ReasoningContent); !ok {
			t.Error("expected ReasoningContent")
		} else {
			if rc.Text != "Let me think..." {
				t.Errorf("thinking = %q, want %q", rc.Text, "Let me think...")
			}
		}
	})

	t.Run("multiple blocks", func(t *testing.T) {
		input := json.RawMessage(`{}`)
		blocks := []ContentBlock{
			{Type: "thinking", Thinking: "Thinking..."},
			{Type: "text", Text: "Response"},
			{Type: "tool_use", ID: "toolu_1", Name: "tool", Input: input},
		}

		contents, text, toolCalls, err := convertResponseContent(blocks)
		if err != nil {
			t.Fatalf("convertResponseContent error: %v", err)
		}

		if len(contents) != 3 {
			t.Errorf("expected 3 contents, got %d", len(contents))
		}

		if text != "Response" {
			t.Errorf("text = %q, want %q", text, "Response")
		}

		if len(toolCalls) != 1 {
			t.Errorf("expected 1 tool call, got %d", len(toolCalls))
		}
	})
}

func TestConvertTools(t *testing.T) {
	tools := []provider.ToolDefinition{
		{
			Name:        "get_weather",
			Description: "Get weather for a location",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
		},
	}

	result, err := convertTools(tools)
	if err != nil {
		t.Fatalf("convertTools error: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	if result[0].Name != "get_weather" {
		t.Errorf("name = %q, want %q", result[0].Name, "get_weather")
	}

	if result[0].Description != "Get weather for a location" {
		t.Errorf("description = %q, want %q", result[0].Description, "Get weather for a location")
	}
}
