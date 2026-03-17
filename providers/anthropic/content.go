package anthropic

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func convertMessages(messages []provider.Message, systemPrompt string) ([]Message, []SystemBlock, error) {
	var apiMessages []Message
	var systemBlocks []SystemBlock

	if systemPrompt != "" {
		systemBlocks = append(systemBlocks, SystemBlock{
			Type: "text",
			Text: systemPrompt,
		})
	}

	for _, msg := range messages {
		if msg.Role == provider.RoleSystem {
			for _, content := range msg.Content {
				if textContent, ok := content.(provider.TextContent); ok {
					systemBlocks = append(systemBlocks, SystemBlock{
						Type: "text",
						Text: textContent.Text,
					})
				}
			}
			continue
		}

		apiMsg := Message{Role: string(msg.Role)}
		if msg.Role == provider.RoleTool {
			apiMsg.Role = "user"
		}

		blocks, err := convertContent(msg.Content)
		if err != nil {
			return nil, nil, err
		}
		apiMsg.Content = blocks
		apiMessages = append(apiMessages, apiMsg)
	}

	return apiMessages, systemBlocks, nil
}

func convertContent(contents []provider.Content) ([]ContentBlock, error) {
	var blocks []ContentBlock

	for _, content := range contents {
		switch c := content.(type) {
		case provider.TextContent:
			blocks = append(blocks, ContentBlock{
				Type: "text",
				Text: c.Text,
			})

		case provider.ImageContent:
			if c.MediaType == "url" {
				blocks = append(blocks, ContentBlock{
					Type: "image",
					Source: &ContentSource{
						Type: "url",
						URL:  string(c.Data),
					},
				})
			} else {
				blocks = append(blocks, ContentBlock{
					Type: "image",
					Source: &ContentSource{
						Type:      "base64",
						MediaType: c.MediaType,
						Data:      base64.StdEncoding.EncodeToString(c.Data),
					},
				})
			}

		case provider.ToolCallContent:
			blocks = append(blocks, ContentBlock{
				Type:  "tool_use",
				ID:    c.ID,
				Name:  c.Name,
				Input: c.Input,
			})

		case provider.ToolResultContent:
			var content json.RawMessage
			if c.Result != nil {
				content = c.Result
			}
			blocks = append(blocks, ContentBlock{
				Type:      "tool_result",
				ToolUseID: c.ID,
				Content:   content,
				IsError:   c.IsError,
			})

		case provider.ReasoningContent:
			blocks = append(blocks, ContentBlock{
				Type:     "thinking",
				Thinking: c.Text,
			})

		default:
			return nil, fmt.Errorf("unsupported content type: %T", content)
		}
	}

	return blocks, nil
}

func convertResponseContent(blocks []ContentBlock) ([]provider.Content, string, []provider.ToolCall, error) {
	var contents []provider.Content
	var text string
	var toolCalls []provider.ToolCall

	for _, block := range blocks {
		switch block.Type {
		case "text":
			contents = append(contents, provider.TextContent{Text: block.Text})
			text += block.Text

		case "thinking":
			contents = append(contents, provider.ReasoningContent{Text: block.Thinking})

		case "tool_use":
			contents = append(contents, provider.ToolCallContent{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
			toolCalls = append(toolCalls, provider.ToolCall{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})

		default:
			// Skip unknown block types
		}
	}

	return contents, text, toolCalls, nil
}

func mapStopReason(reason string) provider.FinishReason {
	switch reason {
	case "end_turn":
		return provider.FinishReasonStop
	case "max_tokens":
		return provider.FinishReasonLength
	case "stop_sequence":
		return provider.FinishReasonStop
	case "tool_use":
		return provider.FinishReasonToolCalls
	default:
		return provider.FinishReasonStop
	}
}
