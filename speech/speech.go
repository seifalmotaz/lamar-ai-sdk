package speech

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-sdk/internal/ctxutil"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

const DefaultTimeout = 60 * time.Second

func Synthesize(ctx context.Context, model provider.SpeechModel, text string, opts ...Option) (*Result, error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
	}
	if text == "" {
		return nil, provider.ErrInvalidInput
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

	req := &provider.SpeechRequest{
		Text:         text,
		Voice:        cfg.Voice,
		Format:       cfg.Format,
		Speed:        cfg.Speed,
		Instructions: cfg.Instructions,
	}

	start := time.Now()
	cfg.logDebug("speech: starting", "provider", model.Provider(), "model", model.ModelID())

	result, err := model.Synthesize(ctx, req)
	if err != nil {
		cfg.logError("speech: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("speech: completed", "provider", model.Provider(), "model", model.ModelID(), "bytes", len(result.Audio))
	cfg.recordMetrics(ctx, model, time.Since(start), nil)

	return &Result{
		Audio:     result.Audio,
		MediaType: result.MediaType,
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
