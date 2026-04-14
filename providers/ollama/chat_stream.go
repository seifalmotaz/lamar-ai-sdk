package ollama

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/ndjson"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func (m *ChatModel) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	return m.provider.wrapper.Stream(ctx, m.id, req, m.streamCore)
}

func (m *ChatModel) streamCore(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	ollamaReq, err := m.buildRequest(req)
	if err != nil {
		return nil, err
	}

	ollamaReq.Stream = true

	rc, err := m.provider.client.DoStream(ctx, "POST", "/api/chat", ollamaReq)
	if err != nil {
		return nil, m.mapError(err)
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

		reader := ndjson.NewReader(rc)
		var textBuilder strings.Builder
		var toolCallBuilders map[string]*toolCallBuilder
		var usage provider.Usage
		var finishReason provider.FinishReason
		var streamErr error

		toolCallBuilders = make(map[string]*toolCallBuilder)

		for {
			var chunk ChatChunk
			err := reader.Read(&chunk)
			if err != nil {
				if err == io.EOF {
					break
				}
				stream <- provider.StreamErrorPart{Error: err}
				streamErr = err
				break
			}

			if chunk.Message.Content != "" {
				textBuilder.WriteString(chunk.Message.Content)
				stream <- provider.StreamTextPart{Delta: chunk.Message.Content}
			}

			for _, tc := range chunk.Message.ToolCalls {
				builder := toolCallBuilders[tc.ID]
				if builder == nil {
					builder = &toolCallBuilder{id: tc.ID}
					toolCallBuilders[tc.ID] = builder
				}
				if tc.Function.Name != "" {
					builder.name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					builder.arguments = append(builder.arguments, tc.Function.Arguments)
				}
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

		for _, builder := range toolCallBuilders {
			args := strings.Join(builder.arguments, "")
			tc := provider.ToolCall{
				ID:    builder.id,
				Name:  builder.name,
				Input: json.RawMessage(args),
			}
			stream <- provider.StreamToolCallPart{ToolCall: tc}
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
	id        string
	name      string
	arguments []string
}
