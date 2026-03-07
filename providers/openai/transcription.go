package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type TranscriptionModel struct {
	id       string
	provider *Provider
	config   TranscriptionConfig
}

type TranscriptionConfig struct {
	Language             string
	Prompt               string
	Temperature          float64
	TimestampGranularity string
}

type TranscriptionOption func(*TranscriptionConfig)

func TranscriptionLanguage(language string) TranscriptionOption {
	return func(c *TranscriptionConfig) { c.Language = language }
}

func TranscriptionPrompt(prompt string) TranscriptionOption {
	return func(c *TranscriptionConfig) { c.Prompt = prompt }
}

func TranscriptionTemperature(temp float64) TranscriptionOption {
	return func(c *TranscriptionConfig) { c.Temperature = temp }
}

func TranscriptionTimestampGranularity(granularity string) TranscriptionOption {
	return func(c *TranscriptionConfig) { c.TimestampGranularity = granularity }
}

func NewTranscriptionModel(id string, p *Provider, opts ...TranscriptionOption) *TranscriptionModel {
	cfg := &TranscriptionConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &TranscriptionModel{
		id:       id,
		provider: p,
		config:   *cfg,
	}
}

func (m *TranscriptionModel) Provider() string { return "openai" }
func (m *TranscriptionModel) ModelID() string  { return m.id }

func (m *TranscriptionModel) Transcribe(ctx context.Context, req *provider.TranscriptionRequest) (*provider.TranscriptionResult, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("model", m.id); err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write model field", Cause: err}
	}

	ext := mediaTypeToExtension(req.MediaType)
	part, err := writer.CreateFormFile("file", "audio."+ext)
	if err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to create file part", Cause: err}
	}
	if _, err := part.Write(req.Audio); err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write audio data", Cause: err}
	}

	if req.Language != "" {
		if err := writer.WriteField("language", req.Language); err != nil {
			return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write language field", Cause: err}
		}
	} else if m.config.Language != "" {
		if err := writer.WriteField("language", m.config.Language); err != nil {
			return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write language field", Cause: err}
		}
	}
	if req.Prompt != "" {
		if err := writer.WriteField("prompt", req.Prompt); err != nil {
			return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write prompt field", Cause: err}
		}
	} else if m.config.Prompt != "" {
		if err := writer.WriteField("prompt", m.config.Prompt); err != nil {
			return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write prompt field", Cause: err}
		}
	}
	if err := writer.WriteField("response_format", "verbose_json"); err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write response_format field", Cause: err}
	}
	if m.config.Temperature > 0 {
		if err := writer.WriteField("temperature", jsonNumber(m.config.Temperature)); err != nil {
			return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write temperature field", Cause: err}
		}
	}
	if m.config.TimestampGranularity != "" {
		if err := writer.WriteField("timestamp_granularities[]", m.config.TimestampGranularity); err != nil {
			return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to write timestamp_granularities field", Cause: err}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to close multipart writer", Cause: err}
	}

	url := m.provider.client.BaseURL + "/audio/transcriptions"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, &provider.Error{Code: provider.CodeInvalidRequest, Message: "failed to create request", Cause: err}
	}

	for key, value := range m.provider.client.Headers {
		httpReq.Header.Set(key, value)
	}
	httpReq.Header.Set("Content-Type", writer.FormDataContentType())

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
		return nil, parseTranscriptionError(resp, respBody)
	}

	var transcriptionResp TranscriptionResponse
	if err := json.Unmarshal(respBody, &transcriptionResp); err != nil {
		return nil, &provider.Error{Code: provider.CodeParseError, Message: "failed to unmarshal response", Cause: err}
	}

	segments := make([]provider.TranscriptSegment, 0)
	if len(transcriptionResp.Segments) > 0 {
		for _, s := range transcriptionResp.Segments {
			segments = append(segments, provider.TranscriptSegment{
				Text:        s.Text,
				StartSecond: s.Start,
				EndSecond:   s.End,
			})
		}
	} else if len(transcriptionResp.Words) > 0 {
		for _, w := range transcriptionResp.Words {
			segments = append(segments, provider.TranscriptSegment{
				Text:        w.Word,
				StartSecond: w.Start,
				EndSecond:   w.End,
			})
		}
	}

	return &provider.TranscriptionResult{
		Text:     transcriptionResp.Text,
		Segments: segments,
		Language: transcriptionResp.Language,
		Duration: transcriptionResp.Duration,
	}, nil
}

func mediaTypeToExtension(mediaType string) string {
	switch mediaType {
	case "audio/mp3", "audio/mpeg":
		return "mp3"
	case "audio/mp4", "audio/m4a":
		return "mp4"
	case "audio/wav":
		return "wav"
	case "audio/webm":
		return "webm"
	case "audio/ogg":
		return "oga"
	default:
		return "mp3"
	}
}

func jsonNumber(f float64) string {
	b, _ := json.Marshal(f)
	return string(b)
}

func parseTranscriptionError(resp *http.Response, body []byte) *provider.Error {
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
			RetryAfter: parseRetryAfterHeader(resp.Header),
		}
	}

	return &provider.Error{
		Code:       code,
		Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		Provider:   "openai",
		StatusCode: resp.StatusCode,
	}
}

func parseRetryAfterHeader(headers http.Header) time.Duration {
	retryAfter := headers.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}
	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}
	return 0
}

type TranscriptionResponse struct {
	Text     string  `json:"text"`
	Language string  `json:"language,omitempty"`
	Duration float64 `json:"duration,omitempty"`
	Segments []struct {
		Text  string  `json:"text"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"segments,omitempty"`
	Words []struct {
		Word  string  `json:"word"`
		Start float64 `json:"start"`
		End   float64 `json:"end"`
	} `json:"words,omitempty"`
}
