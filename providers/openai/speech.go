package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type SpeechModel struct {
	id       string
	provider *Provider
	config   SpeechConfig
}

type SpeechConfig struct {
	Voice        string
	Format       string
	Speed        float64
	Instructions string
}

type SpeechOption func(*SpeechConfig)

func SpeechVoice(voice string) SpeechOption {
	return func(c *SpeechConfig) { c.Voice = voice }
}

func SpeechFormat(format string) SpeechOption {
	return func(c *SpeechConfig) { c.Format = format }
}

func SpeechSpeed(speed float64) SpeechOption {
	return func(c *SpeechConfig) { c.Speed = speed }
}

func SpeechInstructions(instructions string) SpeechOption {
	return func(c *SpeechConfig) { c.Instructions = instructions }
}

func NewSpeechModel(id string, p *Provider, opts ...SpeechOption) *SpeechModel {
	cfg := &SpeechConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &SpeechModel{
		id:       id,
		provider: p,
		config:   *cfg,
	}
}

func (m *SpeechModel) Provider() string { return "openai" }
func (m *SpeechModel) ModelID() string  { return m.id }

func (m *SpeechModel) Synthesize(ctx context.Context, req *provider.SpeechRequest) (*provider.SpeechResult, error) {
	body := map[string]any{
		"model": m.id,
		"input": req.Text,
	}

	if req.Voice != "" {
		body["voice"] = req.Voice
	} else if m.config.Voice != "" {
		body["voice"] = m.config.Voice
	} else {
		body["voice"] = "alloy"
	}

	if req.Format != "" {
		body["response_format"] = req.Format
	} else if m.config.Format != "" {
		body["response_format"] = m.config.Format
	} else {
		body["response_format"] = "mp3"
	}

	if req.Speed > 0 {
		body["speed"] = req.Speed
	} else if m.config.Speed > 0 {
		body["speed"] = m.config.Speed
	}

	if req.Instructions != "" {
		body["instructions"] = req.Instructions
	} else if m.config.Instructions != "" {
		body["instructions"] = m.config.Instructions
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to marshal request body", Cause: err}
	}

	url := m.provider.client.BaseURL + "/audio/speech"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, &provider.Error{Code: provider.CodeInvalidRequest, Message: "failed to create request", Cause: err}
	}

	for key, value := range m.provider.client.Headers {
		httpReq.Header.Set(key, value)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := m.provider.client.HTTPClient.Do(httpReq)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, provider.ErrContextCanceled
		}
		return nil, &provider.Error{Code: provider.CodeAPITimeout, Message: "request failed", Cause: err}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to read response body", Cause: err}
	}

	if resp.StatusCode >= 400 {
		return nil, parseSpeechError(resp, respBody)
	}

	mediaType := "audio/mpeg"
	if req.Format != "" {
		mediaType = formatToMediaType(req.Format)
	} else if m.config.Format != "" {
		mediaType = formatToMediaType(m.config.Format)
	}

	return &provider.SpeechResult{
		Audio:     respBody,
		MediaType: mediaType,
	}, nil
}

func formatToMediaType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "opus":
		return "audio/opus"
	case "aac":
		return "audio/aac"
	case "flac":
		return "audio/flac"
	case "wav":
		return "audio/wav"
	case "pcm":
		return "audio/pcm"
	default:
		return "audio/mpeg"
	}
}

func parseSpeechError(resp *http.Response, body []byte) *provider.Error {
	var errBody struct {
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		} `json:"error"`
	}

	code := provider.CodeUnknown
	switch resp.StatusCode {
	case 400:
		code = provider.CodeInvalidRequest
	case 401, 403:
		code = provider.CodeAuthenticationFailed
	case 404:
		code = provider.CodeModelNotFound
	case 429:
		code = provider.CodeRateLimited
	case 500, 502, 503, 504:
		code = provider.CodeAPITimeout
	}

	if err := json.Unmarshal(body, &errBody); err == nil && errBody.Error.Message != "" {
		return &provider.Error{
			Code:       code,
			Message:    errBody.Error.Message,
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			RetryAfter: parseRetryAfterSpeech(resp.Header),
		}
	}

	return &provider.Error{
		Code:       code,
		Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		Provider:   "openai",
		StatusCode: resp.StatusCode,
	}
}

func parseRetryAfterSpeech(headers http.Header) time.Duration {
	retryAfter := headers.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}
	return 0
}
