package image

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/ctxutil"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

const DefaultTimeout = 120 * time.Second

func Generate(ctx context.Context, model provider.ImageModel, prompt string, opts ...Option) (*Result, error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
	}
	if prompt == "" {
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

	req := &provider.ImageRequest{
		Prompt:  prompt,
		N:       cfg.N,
		Size:    cfg.Size,
		Quality: cfg.Quality,
		Format:  cfg.Format,
	}

	start := time.Now()
	cfg.logDebug("image: starting", "provider", model.Provider(), "model", model.ModelID())

	result, err := model.GenerateImage(ctx, req)
	if err != nil {
		cfg.logError("image: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("image: completed", "provider", model.Provider(), "model", model.ModelID(), "count", len(result.Images))
	cfg.recordMetrics(ctx, model, time.Since(start), nil)

	return &Result{
		Images:         result.Images,
		MediaType:      determineMediaType(cfg.Format),
		RevisedPrompts: result.RevisedPrompts,
	}, nil
}

func determineMediaType(format string) string {
	if format != "" {
		switch format {
		case "png":
			return "image/png"
		case "jpeg", "jpg":
			return "image/jpeg"
		case "webp":
			return "image/webp"
		}
	}
	return "image/png"
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
