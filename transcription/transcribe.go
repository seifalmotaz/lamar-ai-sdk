package transcription

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/ctxutil"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

const DefaultTimeout = 120 * time.Second

func Transcribe(ctx context.Context, model provider.TranscriptionModel, audio []byte, mediaType string, opts ...Option) (*Result, error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
	}
	if len(audio) == 0 {
		return nil, provider.ErrInvalidInput
	}
	if mediaType == "" {
		return nil, provider.ErrInvalidMediaType
	}

	select {
	case <-ctx.Done():
		return nil, provider.NewError(provider.CodeContextCanceled, "context already canceled", ctx.Err())
	default:
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	} else if timeout < 0 {
		timeout = 0
	}
	ctx, cancel := ctxutil.ApplyTimeout(ctx, timeout)
	defer cancel()

	req := &provider.TranscriptionRequest{
		Audio:     audio,
		MediaType: mediaType,
		Language:  cfg.Language,
		Prompt:    cfg.Prompt,
	}

	start := time.Now()
	cfg.logDebug("transcription: starting", "provider", model.Provider(), "model", model.ModelID())

	result, err := model.Transcribe(ctx, req)
	if err != nil {
		cfg.logError("transcription: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("transcription: completed", "provider", model.Provider(), "model", model.ModelID(), "duration", result.Duration)
	cfg.recordMetrics(ctx, model, time.Since(start), nil)

	return &Result{
		Text:     result.Text,
		Segments: result.Segments,
		Language: result.Language,
		Duration: result.Duration,
	}, nil
}

func (c *Config) logDebug(msg string, args ...any) {
	if c.Logger != nil {
		c.Logger.Debug(msg, args...)
	}
}

func (c *Config) logError(msg string, args ...any) {
	if c.Logger != nil {
		c.Logger.Error(msg, args...)
	}
}

func (c *Config) recordMetrics(ctx context.Context, model provider.Model, duration time.Duration, err error) {
	if c.Metrics != nil {
		c.Metrics.RecordRequest(ctx, model.Provider(), model.ModelID(), duration, err)
	}
}
