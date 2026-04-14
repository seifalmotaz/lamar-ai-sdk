package ollama

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func convertMessages(messages []provider.Message) []ChatMessage {
	result := make([]ChatMessage, 0, len(messages))

	for _, msg := range messages {
		converted := convertMessage(msg)
		if m, ok := converted.(ChatMessage); ok {
			result = append(result, m)
		} else if msgs, ok := converted.([]ChatMessage); ok {
			result = append(result, msgs...)
		}
	}

	return result
}

func convertMessage(msg provider.Message) any {
	var textParts []string
	var images []string
	var toolCalls []ToolCall
	var toolResult *provider.ToolResultContent
	var reasoningParts []string

	for _, content := range msg.Content {
		switch c := content.(type) {
		case provider.TextContent:
			if c.Text != "" {
				textParts = append(textParts, c.Text)
			}

		case provider.ImageContent:
			if c.MediaType == "url" {
			} else {
				encoded := base64.StdEncoding.EncodeToString(c.Data)
				images = append(images, encoded)
			}

		case provider.AudioContent:
		case provider.ToolCallContent:
			toolCalls = append(toolCalls, ToolCall{
				ID: c.ID,
				Function: FunctionCall{
					Name:      c.Name,
					Arguments: string(c.Input),
				},
			})

		case provider.ToolResultContent:
			toolResult = &c

		case provider.ReasoningContent:
			if c.Text != "" {
				reasoningParts = append(reasoningParts, c.Text)
			}
		}
	}

	if toolResult != nil {
		return ChatMessage{
			Role:    "tool",
			Content: string(toolResult.Result),
		}
	}

	if len(reasoningParts) > 0 {
		textParts = append(reasoningParts, textParts...)
	}

	content := strings.Join(textParts, "\n")

	role := string(msg.Role)

	ollamaMsg := ChatMessage{
		Role:    role,
		Content: content,
	}

	if len(images) > 0 {
		ollamaMsg.Images = images
	}

	if len(toolCalls) > 0 {
		ollamaMsg.ToolCalls = toolCalls
	}

	return ollamaMsg
}

func convertTools(tools []provider.ToolDefinition) []Tool {
	if len(tools) == 0 {
		return nil
	}

	result := make([]Tool, len(tools))
	for i, t := range tools {
		result[i] = Tool{
			Type: "function",
			Function: ToolFunction{
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
			"name": tc.ToolName,
		}
	default:
		return "auto"
	}
}

func mapFinishReason(reason string) provider.FinishReason {
	switch reason {
	case "stop":
		return provider.FinishReasonStop
	case "length":
		return provider.FinishReasonLength
	case "tool_calls":
		return provider.FinishReasonToolCalls
	default:
		return provider.FinishReasonStop
	}
}

func buildResult(resp *ChatResponse) *provider.GenerateResult {
	result := &provider.GenerateResult{
		FinishReason: mapFinishReason(resp.DoneReason),
		Usage: provider.Usage{
			PromptTokens:     resp.PromptEvalCount,
			CompletionTokens: resp.EvalCount,
			TotalTokens:      resp.PromptEvalCount + resp.EvalCount,
		},
	}

	content := make([]provider.Content, 0)

	if resp.Message.Thinking != "" {
		content = append(content, provider.ReasoningContent{
			Text: resp.Message.Thinking,
		})
	}

	if resp.Message.Content != "" {
		result.Text = resp.Message.Content
		content = append(content, provider.Text(resp.Message.Content))
	}

	if len(resp.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]provider.ToolCall, len(resp.Message.ToolCalls))
		for i, tc := range resp.Message.ToolCalls {
			result.ToolCalls[i] = provider.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			}
			content = append(content, provider.NewToolCallContent(
				tc.ID,
				tc.Function.Name,
				json.RawMessage(tc.Function.Arguments),
			))
		}
	}

	result.Content = content
	return result
}

func buildStreamResult(chunks []ChatChunk) *provider.GenerateResult {
	var textBuilder strings.Builder
	var toolCalls []provider.ToolCall
	var finishReason provider.FinishReason
	var usage provider.Usage

	for _, chunk := range chunks {
		if chunk.Message.Content != "" {
			textBuilder.WriteString(chunk.Message.Content)
		}

		if chunk.Message.Thinking != "" {
		}

		for _, tc := range chunk.Message.ToolCalls {
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: json.RawMessage(tc.Function.Arguments),
			})
		}

		if chunk.Done {
			if chunk.DoneReason != "" {
				finishReason = mapFinishReason(chunk.DoneReason)
			}
			usage.PromptTokens = chunk.PromptEvalCount
			usage.CompletionTokens = chunk.EvalCount
			usage.TotalTokens = chunk.PromptEvalCount + chunk.EvalCount
		}
	}

	if finishReason == "" {
		finishReason = provider.FinishReasonStop
	}

	content := make([]provider.Content, 0)
	text := textBuilder.String()

	if text != "" {
		content = append(content, provider.Text(text))
	}

	for _, tc := range toolCalls {
		content = append(content, provider.NewToolCallContent(tc.ID, tc.Name, tc.Input))
	}

	return &provider.GenerateResult{
		Text:         text,
		Content:      content,
		ToolCalls:    toolCalls,
		FinishReason: finishReason,
		Usage:        usage,
	}
}
