package openai

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/seifalmotaz/lamar-sdk/internal/sse"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// Stream streams a chat completion from OpenAI.
// It implements the provider.Streamer interface.
func (m *ChatModel) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	return m.provider.wrapStream(ctx, m.id, req, func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
		openaiReq, err := m.buildRequest(req)
		if err != nil {
			return nil, err
		}

		// Enable streaming
		openaiReq.Stream = true
		// Request usage in final chunk
		openaiReq.StreamOptions = &StreamOptions{IncludeUsage: true}

		rc, err := m.provider.client.DoStream(ctx, "POST", "/chat/completions", openaiReq)
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

			// Track tool call accumulation across chunks
			toolCalls := make(map[int]*toolCallBuilder)

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

				var chunk ChatCompletionChunk
				if err := json.Unmarshal([]byte(event.Data), &chunk); err != nil {
					continue
				}

				// Process choices
				for _, choice := range chunk.Choices {
					// Handle role (first chunk)
					if choice.Delta.Role != "" {
						// Role is set, nothing to send for now
					}

					// Handle content delta
					if choice.Delta.Content != "" {
						textBuilder.WriteString(choice.Delta.Content)
						stream <- provider.StreamTextPart{Delta: choice.Delta.Content}
					}

					// Handle tool calls
					for _, tc := range choice.Delta.ToolCalls {
						builder := toolCalls[tc.Index]
						if builder == nil {
							builder = &toolCallBuilder{index: tc.Index}
							toolCalls[tc.Index] = builder
						}
						if tc.ID != "" {
							builder.id = tc.ID
						}
						if tc.Function.Name != "" {
							builder.name = tc.Function.Name
						}
						if tc.Function.Arguments != "" {
							builder.arguments = append(builder.arguments, tc.Function.Arguments)
						}
					}

					// Handle finish reason
					if choice.FinishReason != nil {
						finishReason = mapFinishReason(*choice.FinishReason)
					}
				}

				// Handle usage (only in final chunk)
				if chunk.Usage != nil {
					usage = provider.Usage{
						PromptTokens:     chunk.Usage.PromptTokens,
						CompletionTokens: chunk.Usage.CompletionTokens,
						TotalTokens:      chunk.Usage.TotalTokens,
					}
				}
			}

			// Send accumulated tool calls
			for _, tc := range toolCalls {
				args := strings.Join(tc.arguments, "")
				stream <- provider.StreamToolCallPart{
					ToolCall: provider.ToolCall{
						ID:    tc.id,
						Name:  tc.name,
						Input: json.RawMessage(args),
					},
				}
			}

			// Send finish
			stream <- provider.StreamFinishPart{
				FinishReason: finishReason,
				Usage:        usage,
			}

			// Set accessor functions
			finalText := textBuilder.String()
			result.Text = func() (string, error) { return finalText, streamErr }
			result.Usage = func() (provider.Usage, error) { return usage, streamErr }
			result.FinishReason = func() (provider.FinishReason, error) { return finishReason, streamErr }
		}()

		return result, nil
	})
}

type toolCallBuilder struct {
	index     int
	id        string
	name      string
	arguments []string
}
