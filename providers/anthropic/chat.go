package anthropic

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/seifalmotaz/lamar-sdk/internal/sse"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

const DefaultMaxTokens = 4096

type ChatModel struct {
	id       string
	provider *Provider
	config   *ChatConfig
}

var _ provider.Generator = (*ChatModel)(nil)
var _ provider.Streamer = (*ChatModel)(nil)
var _ provider.LanguageModel = (*ChatModel)(nil)

func NewChatModel(id string, p *Provider, opts ...ChatOption) *ChatModel {
	config := mergeChatConfig(opts...)
	return &ChatModel{
		id:       id,
		provider: p,
		config:   config,
	}
}

func (m *ChatModel) Provider() string {
	return m.provider.name()
}

func (m *ChatModel) ModelID() string {
	return m.id
}

func (m *ChatModel) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	return m.provider.wrapGenerate(ctx, m.id, req, m.generateCore)
}

func (m *ChatModel) generateCore(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	anthropicReq, err := m.buildRequest(req)
	if err != nil {
		return nil, err
	}

	var resp MessageResponse
	headers := m.buildExtraHeaders()
	if err := m.provider.client.DoWithHeaders(ctx, "POST", "/messages", anthropicReq, &resp, headers); err != nil {
		return nil, err
	}

	return m.buildResult(&resp)
}

func (m *ChatModel) buildExtraHeaders() map[string]string {
	headers := make(map[string]string)
	if betaHeaders := m.buildBetaHeaders(); len(betaHeaders) > 0 {
		headers["anthropic-beta"] = mergeBetaHeaders(betaHeaders)
	}
	return headers
}

func (m *ChatModel) buildRequest(req *provider.GenerateRequest) (*MessageRequest, error) {
	anthropicReq := &MessageRequest{
		Model:     m.id,
		MaxTokens: DefaultMaxTokens,
	}

	if req.Config.MaxTokens > 0 {
		anthropicReq.MaxTokens = req.Config.MaxTokens
	}

	if req.Config.Temperature > 0 {
		temp := req.Config.Temperature
		anthropicReq.Temperature = &temp
	}

	if req.Config.TopP > 0 {
		topP := req.Config.TopP
		anthropicReq.TopP = &topP
	}

	if req.Config.TopK > 0 {
		topK := req.Config.TopK
		anthropicReq.TopK = &topK
	}

	if len(req.Config.StopSequences) > 0 {
		anthropicReq.StopSequences = req.Config.StopSequences
	}

	messages, systemBlocks, err := convertMessages(req.Messages, req.Config.System)
	if err != nil {
		return nil, err
	}
	anthropicReq.Messages = messages
	anthropicReq.System = systemBlocks

	if len(req.Config.Tools) > 0 {
		tools, err := convertTools(req.Config.Tools)
		if err != nil {
			return nil, err
		}
		anthropicReq.Tools = tools

		if req.Config.ToolChoice.Type != "" {
			anthropicReq.ToolChoice = convertToolChoice(req.Config.ToolChoice, m.config.DisableParallelToolUse)
		}
	}

	if m.config != nil && m.config.Thinking != nil {
		anthropicReq.Thinking = &ThinkingRequest{
			Type: m.config.Thinking.Type,
		}
		if m.config.Thinking.BudgetTokens > 0 {
			anthropicReq.Thinking.BudgetTokens = m.config.Thinking.BudgetTokens
		}
	}

	if m.config != nil && len(m.config.MCPServers) > 0 {
		anthropicReq.MCPServers = convertMCPServers(m.config.MCPServers)
	}

	if m.config != nil && m.config.Container != nil {
		anthropicReq.Container = convertContainer(m.config.Container)
	}

	return anthropicReq, nil
}

func (m *ChatModel) buildResult(resp *MessageResponse) (*provider.GenerateResult, error) {
	contents, text, toolCalls, err := convertResponseContent(resp.Content)
	if err != nil {
		return nil, err
	}

	result := &provider.GenerateResult{
		Text:         text,
		Content:      contents,
		ToolCalls:    toolCalls,
		FinishReason: mapStopReason(resp.StopReason),
		Usage: provider.Usage{
			PromptTokens:     resp.Usage.InputTokens,
			CompletionTokens: resp.Usage.OutputTokens,
			TotalTokens:      resp.Usage.InputTokens + resp.Usage.OutputTokens,
		},
	}

	return result, nil
}

func convertTools(tools []provider.ToolDefinition) ([]Tool, error) {
	result := make([]Tool, len(tools))
	for i, tool := range tools {
		result[i] = Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		}
	}
	return result, nil
}

func convertToolChoice(choice provider.ToolChoice, disableParallel bool) *ToolChoice {
	tc := &ToolChoice{DisableParallelToolUse: disableParallel}

	switch choice.Type {
	case "auto":
		tc.Type = "auto"
	case "required":
		tc.Type = "any"
	case "tool":
		tc.Type = "tool"
		tc.Name = choice.ToolName
	default:
		tc.Type = "auto"
	}

	return tc
}

func convertMCPServers(servers []MCPServerConfig) []MCPServerAPI {
	result := make([]MCPServerAPI, len(servers))
	for i, s := range servers {
		result[i] = MCPServerAPI{
			Type:               s.Type,
			Name:               s.Name,
			URL:                s.URL,
			AuthorizationToken: s.AuthorizationToken,
		}
		if s.ToolConfiguration != nil {
			result[i].ToolConfiguration = &MCPToolConfigurationAPI{
				Enabled:      s.ToolConfiguration.Enabled,
				AllowedTools: s.ToolConfiguration.AllowedTools,
			}
		}
	}
	return result
}

func convertContainer(c *ContainerConfig) *ContainerAPI {
	if c == nil {
		return nil
	}
	return &ContainerAPI{
		ID:     c.ID,
		Skills: c.Skills,
	}
}

func (m *ChatModel) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	return m.provider.wrapStream(ctx, m.id, req, m.streamCore)
}

func (m *ChatModel) streamCore(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	anthropicReq, err := m.buildRequest(req)
	if err != nil {
		return nil, err
	}
	anthropicReq.Stream = true

	headers := m.buildExtraHeaders()
	rc, err := m.provider.client.DoStreamWithHeaders(ctx, "POST", "/messages", anthropicReq, headers)
	if err != nil {
		return nil, err
	}

	stream := make(chan provider.StreamPart, 100)
	done := make(chan struct{})

	result := &provider.StreamResult{
		Stream: stream,
		Done:   done,
	}

	go func() {
		defer close(stream)
		defer close(done)
		defer rc.Close()

		reader := sse.NewReader(rc)
		var textBuilder strings.Builder
		var usage provider.Usage
		var finishReason provider.FinishReason
		var streamErr error

		toolCallBuilders := make(map[int]*toolCallBuilder)

		for {
			event, err := reader.ReadEvent()
			if err != nil {
				if err == sse.Done || err == io.EOF {
					break
				}
				stream <- provider.StreamErrorPart{Error: err}
				streamErr = err
				break
			}

			var streamEvent StreamEvent
			if err := json.Unmarshal([]byte(event.Data), &streamEvent); err != nil {
				continue
			}

			switch streamEvent.Type {
			case "message_start":
				if streamEvent.Message != nil {
					usage.PromptTokens = streamEvent.Message.Usage.InputTokens
				}

			case "content_block_start":
				if streamEvent.ContentBlock != nil {
					switch streamEvent.ContentBlock.Type {
					case "tool_use":
						toolCallBuilders[streamEvent.Index] = &toolCallBuilder{
							index: streamEvent.Index,
							id:    streamEvent.ContentBlock.ID,
							name:  streamEvent.ContentBlock.Name,
						}
					}
				}

			case "content_block_delta":
				var delta StreamDelta
				if err := json.Unmarshal(streamEvent.Delta, &delta); err != nil {
					continue
				}

				switch delta.Type {
				case "text_delta":
					if delta.Text != "" {
						textBuilder.WriteString(delta.Text)
						stream <- provider.StreamTextPart{Delta: delta.Text}
					}
				case "thinking_delta":
					if delta.Thinking != "" {
						stream <- provider.StreamTextPart{Delta: delta.Thinking}
					}
				case "input_json_delta":
					if delta.PartialJSON != "" {
						if builder, ok := toolCallBuilders[streamEvent.Index]; ok {
							builder.arguments = append(builder.arguments, delta.PartialJSON)
						}
					}
				}

			case "content_block_stop":
				for _, builder := range toolCallBuilders {
					if builder.index == streamEvent.Index && len(builder.arguments) > 0 {
						args := strings.Join(builder.arguments, "")
						stream <- provider.StreamToolCallPart{
							ToolCall: provider.ToolCall{
								ID:    builder.id,
								Name:  builder.name,
								Input: json.RawMessage(args),
							},
						}
						break
					}
				}

			case "message_delta":
				var msgDelta StreamMessageDelta
				if err := json.Unmarshal(streamEvent.Delta, &msgDelta); err != nil {
					continue
				}
				if msgDelta.StopReason != "" {
					finishReason = mapStopReason(msgDelta.StopReason)
				}
				if msgDelta.Usage.OutputTokens > 0 {
					usage.CompletionTokens = msgDelta.Usage.OutputTokens
					usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
				}

			case "message_stop":
				break

			case "error":
				if streamEvent.Error != nil {
					streamErr = fmt.Errorf("%s: %s", streamEvent.Error.Error.Type, streamEvent.Error.Error.Message)
					stream <- provider.StreamErrorPart{Error: streamErr}
				}

			case "ping":
			}
		}

		if finishReason == "" {
			finishReason = provider.FinishReasonStop
		}

		stream <- provider.StreamFinishPart{
			FinishReason: finishReason,
			Usage:        usage,
		}

		finalText := textBuilder.String()
		result.Text = func() (string, error) { return finalText, streamErr }
		result.Usage = func() (provider.Usage, error) { return usage, streamErr }
		result.FinishReason = func() (provider.FinishReason, error) { return finishReason, streamErr }
	}()

	return result, nil
}

type toolCallBuilder struct {
	index     int
	id        string
	name      string
	arguments []string
}
