package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type ChatModel struct {
	id       string
	provider *Provider
	config   ChatConfig
}

func (m *ChatModel) Provider() string { return "openai" }
func (m *ChatModel) ModelID() string  { return m.id }

func (m *ChatModel) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	return m.provider.wrapper.Generate(ctx, m.id, req, m.generateCore)
}

func (m *ChatModel) generateCore(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	openaiReq, err := m.buildRequest(req)
	if err != nil {
		return nil, err
	}

	var resp ChatCompletionResponse
	if err := m.provider.client.Post(ctx, "/chat/completions", openaiReq, &resp); err != nil {
		return nil, err
	}

	return m.buildResult(&resp)
}

func (m *ChatModel) buildRequest(req *provider.GenerateRequest) (*ChatCompletionRequest, error) {
	messages := make([]ChatMessage, 0, len(req.Messages)+1)

	if req.System != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	if len(req.Messages) > 0 {
		for _, msg := range req.Messages {
			messages = append(messages, convertMessage(msg))
		}
	} else if req.Prompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	openaiReq := &ChatCompletionRequest{
		Model:       m.id,
		Messages:    messages,
		Temperature: req.Config.Temperature,
		TopP:        req.Config.TopP,
		Stop:        req.Config.StopSequences,
		Seed:        req.Config.Seed,
	}

	if req.Config.MaxTokens > 0 {
		if isGPT5Model(m.id) {
			openaiReq.MaxCompletionTokens = req.Config.MaxTokens
		} else {
			openaiReq.MaxTokens = req.Config.MaxTokens
		}
	}

	if req.Config.ToolChoice.Type != "" {
		openaiReq.ToolChoice = convertToolChoice(req.Config.ToolChoice)
	}

	if len(req.Config.Tools) > 0 {
		openaiReq.Tools = convertTools(req.Config.Tools)
	}

	if req.Config.ResponseFormat != nil {
		openaiReq.ResponseFormat = convertResponseFormat(*req.Config.ResponseFormat)
	}

	if m.config.LogitBias != nil {
		openaiReq.LogitBias = m.config.LogitBias
	}

	if m.config.ReasoningEffort != "" {
		openaiReq.ReasoningEffort = m.config.ReasoningEffort
	}

	if m.config.User != "" {
		openaiReq.User = m.config.User
	}

	return openaiReq, nil
}

func isGPT5Model(modelID string) bool {
	return strings.HasPrefix(modelID, "gpt-5") || strings.HasPrefix(modelID, "o1")
}

func convertMessage(msg provider.Message) ChatMessage {
	cm := ChatMessage{
		Role: string(msg.Role),
	}

	if len(msg.Content) == 0 {
		return cm
	}

	if len(msg.Content) == 1 {
		switch c := msg.Content[0].(type) {
		case provider.TextContent:
			cm.Content = c.Text
			return cm
		case provider.ToolCallContent:
			cm.ToolCalls = []ToolCall{{
				ID:   c.ID,
				Type: "function",
				Function: FunctionCallData{
					Name:      c.Name,
					Arguments: string(c.Input),
				},
			}}
			return cm
		case provider.ToolResultContent:
			cm.Role = "tool"
			cm.ToolCallID = c.ID
			cm.Content = string(c.Result)
			return cm
		}
	}

	var textParts []string
	var toolCalls []ToolCall
	var otherParts []ContentPart

	for _, c := range msg.Content {
		switch content := c.(type) {
		case provider.TextContent:
			if content.Text != "" {
				textParts = append(textParts, content.Text)
			}
		case provider.ToolCallContent:
			toolCalls = append(toolCalls, ToolCall{
				ID:   content.ID,
				Type: "function",
				Function: FunctionCallData{
					Name:      content.Name,
					Arguments: string(content.Input),
				},
			})
		case provider.ToolResultContent:
			cm.Role = "tool"
			cm.ToolCallID = content.ID
			cm.Content = string(content.Result)
			return cm
		case provider.ImageContent:
			if content.MediaType == "url" {
				otherParts = append(otherParts, ContentPart{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: string(content.Data),
					},
				})
			} else {
				otherParts = append(otherParts, ContentPart{
					Type: "image_url",
					ImageURL: &ImageURL{
						URL: "data:" + content.MediaType + ";base64," + encodeBase64(content.Data),
					},
				})
			}
		case provider.AudioContent:
			format := extractAudioFormat(content.MediaType)
			otherParts = append(otherParts, ContentPart{
				Type: "input_audio",
				InputAudio: &InputAudio{
					Data:   encodeBase64(content.Data),
					Format: format,
				},
			})
		}
	}

	if len(toolCalls) > 0 {
		cm.ToolCalls = toolCalls
		if len(textParts) == 1 {
			cm.Content = textParts[0]
		} else if len(textParts) > 1 {
			parts := make([]ContentPart, len(textParts)+len(otherParts))
			for i, t := range textParts {
				parts[i] = ContentPart{Type: "text", Text: t}
			}
			copy(parts[len(textParts):], otherParts)
			cm.Content = parts
		} else if len(otherParts) > 0 {
			cm.Content = otherParts
		}
	} else if len(textParts) == 1 && len(otherParts) == 0 {
		cm.Content = textParts[0]
	} else if len(textParts) > 0 || len(otherParts) > 0 {
		parts := make([]ContentPart, 0, len(textParts)+len(otherParts))
		for _, t := range textParts {
			parts = append(parts, ContentPart{Type: "text", Text: t})
		}
		parts = append(parts, otherParts...)
		cm.Content = parts
	}

	return cm
}

func convertTools(tools []provider.ToolDefinition) []Tool {
	result := make([]Tool, len(tools))
	for i, t := range tools {
		result[i] = Tool{
			Type: "function",
			Function: Function{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		}
	}
	return result
}

func convertToolChoice(tc provider.ToolChoice) any {
	switch tc.Type {
	case "auto":
		return "auto"
	case "none":
		return "none"
	case "required":
		return "required"
	case "tool":
		return map[string]any{
			"type": "function",
			"function": map[string]string{
				"name": tc.ToolName,
			},
		}
	default:
		return "auto"
	}
}

func convertResponseFormat(rf provider.ResponseFormat) *ResponseFormat {
	if rf.Type != "json_schema" {
		return &ResponseFormat{
			Type:       rf.Type,
			JSONSchema: rf.JSONSchema,
		}
	}

	var schemaName string
	var schemaData json.RawMessage

	var raw map[string]any
	if err := json.Unmarshal(rf.JSONSchema, &raw); err == nil {
		if title, ok := raw["title"].(string); ok {
			schemaName = title
		} else if t, ok := raw["type"].(string); ok {
			schemaName = t + "_schema"
		}
		schemaData = rf.JSONSchema
	}

	if schemaName == "" {
		schemaName = "response"
		schemaData = rf.JSONSchema
	}

	wrapper := JSONSchemaWrapper{
		Name:   schemaName,
		Strict: false,
		Schema: schemaData,
	}
	wrapperBytes, _ := json.Marshal(wrapper)

	return &ResponseFormat{
		Type:       rf.Type,
		JSONSchema: wrapperBytes,
	}
}

func (m *ChatModel) buildResult(resp *ChatCompletionResponse) (*provider.GenerateResult, error) {
	if len(resp.Choices) == 0 {
		return &provider.GenerateResult{
			FinishReason: provider.FinishReasonError,
			Usage: provider.Usage{
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				TotalTokens:      resp.Usage.TotalTokens,
			},
		}, nil
	}

	choice := resp.Choices[0]
	result := &provider.GenerateResult{
		FinishReason: mapFinishReason(choice.FinishReason),
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}

	content := make([]provider.Content, 0)
	if str, ok := choice.Message.Content.(string); ok && str != "" {
		result.Text = str
		content = append(content, provider.Text(str))
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]provider.ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			result.ToolCalls[i] = provider.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			}
			content = append(content, provider.NewToolCallContent(tc.ID, tc.Function.Name, json.RawMessage(tc.Function.Arguments)))
		}
	}

	result.Content = content
	return result, nil
}

func mapFinishReason(reason string) provider.FinishReason {
	switch reason {
	case "stop":
		return provider.FinishReasonStop
	case "length":
		return provider.FinishReasonLength
	case "tool_calls":
		return provider.FinishReasonToolCalls
	case "content_filter":
		return provider.FinishReasonContentFilter
	default:
		return provider.FinishReasonError
	}
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func extractAudioFormat(mediaType string) string {
	if mediaType == "" {
		return "wav"
	}
	parts := strings.Split(mediaType, "/")
	if len(parts) == 2 {
		return parts[1]
	}
	return mediaType
}
