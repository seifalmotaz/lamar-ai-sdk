package ollama

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type ChatModel struct {
	id       string
	provider *Provider
	config   ChatConfig
}

func (m *ChatModel) Provider() string { return "ollama" }
func (m *ChatModel) ModelID() string  { return m.id }

var _ provider.LanguageModel = (*ChatModel)(nil)

func (m *ChatModel) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	return m.provider.wrapper.Generate(ctx, m.id, req, m.generateCore)
}

func (m *ChatModel) generateCore(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	ollamaReq, err := m.buildRequest(req)
	if err != nil {
		return nil, err
	}

	var resp ChatResponse
	if err := m.provider.client.Post(ctx, "/api/chat", ollamaReq, &resp); err != nil {
		return nil, m.mapError(err)
	}

	return buildResult(&resp), nil
}

func (m *ChatModel) buildRequest(req *provider.GenerateRequest) (*ChatRequest, error) {
	messages := make([]ChatMessage, 0)

	if req.System != "" {
		messages = append(messages, ChatMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	if len(req.Messages) > 0 {
		messages = append(messages, convertMessages(req.Messages)...)
	} else if req.Prompt != "" {
		messages = append(messages, ChatMessage{
			Role:    "user",
			Content: req.Prompt,
		})
	}

	ollamaReq := &ChatRequest{
		Model:    m.id,
		Messages: messages,
	}

	if req.Config.MaxTokens > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.NumPredict = &req.Config.MaxTokens
	}

	if req.Config.Temperature > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Temperature = &req.Config.Temperature
	}

	if req.Config.TopP > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.TopP = &req.Config.TopP
	}

	if req.Config.TopK > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.TopK = &req.Config.TopK
	}

	if len(req.Config.StopSequences) > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Stop = req.Config.StopSequences
	}

	if req.Config.Seed != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Seed = req.Config.Seed
	}

	if len(req.Config.Tools) > 0 {
		ollamaReq.Tools = convertTools(req.Config.Tools)

		if req.Config.ToolChoice.Type != "" {
			ollamaReq.Format = convertToolChoice(req.Config.ToolChoice)
		}
	}

	if req.Config.ResponseFormat != nil {
		if req.Config.ResponseFormat.Type == "json_object" || req.Config.ResponseFormat.Type == "json_schema" {
			ollamaReq.Format = "json"
		}
	}

	if m.config.Temperature != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Temperature = m.config.Temperature
	}
	if m.config.TopP != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.TopP = m.config.TopP
	}
	if m.config.TopK != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.TopK = m.config.TopK
	}
	if m.config.Seed != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Seed = m.config.Seed
	}
	if m.config.MaxTokens > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.NumPredict = &m.config.MaxTokens
	}
	if len(m.config.StopSequences) > 0 {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Stop = m.config.StopSequences
	}
	if m.config.Mirostat != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.Mirostat = m.config.Mirostat
	}
	if m.config.MirostatEta != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.MirostatEta = m.config.MirostatEta
	}
	if m.config.MirostatTau != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.MirostatTau = m.config.MirostatTau
	}
	if m.config.NumCtx != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.NumCtx = m.config.NumCtx
	}
	if m.config.NumPredict != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.NumPredict = m.config.NumPredict
	}
	if m.config.NumKeep != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.NumKeep = m.config.NumKeep
	}
	if m.config.RepeatLastN != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.RepeatLastN = m.config.RepeatLastN
	}
	if m.config.RepeatPenalty != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.RepeatPenalty = m.config.RepeatPenalty
	}
	if m.config.PresencePenalty != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.PresencePenalty = m.config.PresencePenalty
	}
	if m.config.FrequencyPenalty != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.FrequencyPenalty = m.config.FrequencyPenalty
	}
	if m.config.TFSZ != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.TFSZ = m.config.TFSZ
	}
	if m.config.TypicalP != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.TypicalP = m.config.TypicalP
	}
	if m.config.MinP != nil {
		if ollamaReq.Options == nil {
			ollamaReq.Options = &ChatOptions{}
		}
		ollamaReq.Options.MinP = m.config.MinP
	}

	if m.config.KeepAlive != "" {
		ollamaReq.KeepAlive = m.config.KeepAlive
	}

	if m.config.Think != nil && *m.config.Think {
		ollamaReq.Format = map[string]any{"type": "thinking"}
	}

	switch v := ollamaReq.Format.(type) {
	case string:
	case map[string]any:
	case json.RawMessage:
		ollamaReq.Format = v
	case *json.RawMessage:
		ollamaReq.Format = *v
	}

	return ollamaReq, nil
}

func (m *ChatModel) mapError(err error) error {
	if pErr, ok := err.(*provider.Error); ok {
		pErr.Provider = "ollama"
		pErr.ModelID = m.id
		return pErr
	}

	if strings.Contains(err.Error(), "model") && strings.Contains(err.Error(), "not found") {
		return &provider.Error{
			Code:     provider.CodeModelNotFound,
			Message:  err.Error(),
			Provider: "ollama",
			ModelID:  m.id,
			Cause:    err,
		}
	}

	return &provider.Error{
		Code:     provider.CodeUnknown,
		Message:  err.Error(),
		Provider: "ollama",
		ModelID:  m.id,
		Cause:    err,
	}
}

func ptr[T any](v T) *T {
	return &v
}
