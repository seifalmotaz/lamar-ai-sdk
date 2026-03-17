package generate

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// DefaultTimeout is the default timeout for generate operations.
const DefaultTimeout = 30 * time.Second

// Generate performs a text generation request using the specified model.
// It returns a Result containing the generated text and metadata, or an error if the request fails.
//
// The prompt parameter is the text to generate from. If empty and no messages are provided,
// an ErrInvalidPrompt error is returned.
//
// Options can be provided to customize the generation behavior (see Option functions).
//
// Example:
//
//	result, err := generate.Generate(ctx, model, "Hello, world!",
//	    generate.MaxTokens(100),
//	    generate.Temperature(0.7),
//	)
func Generate(ctx context.Context, model provider.Generator, prompt string, opts ...Option) (*Result, error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
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

	if prompt == "" && len(cfg.Messages) == 0 {
		return nil, provider.ErrInvalidPrompt
	}

	var timeout time.Duration
	if cfg.Timeout == nil {
		timeout = DefaultTimeout
	} else {
		timeout = *cfg.Timeout
	}
	ctx, cancel := applyTimeout(ctx, timeout)
	defer cancel()

	req := toGenerateRequest(prompt, cfg)

	start := time.Now()
	cfg.logDebug("generate: starting", "provider", model.Provider(), "model", model.ModelID())

	result, err := model.Generate(ctx, req)
	if err != nil {
		cfg.logError("generate: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("generate: completed", "provider", model.Provider(), "model", model.ModelID(), "tokens", result.Usage.TotalTokens)
	cfg.recordMetrics(ctx, model, time.Since(start), nil)
	cfg.recordTokens(ctx, model, result.Usage)

	return newResult(result), nil
}

func applyTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return ctx, func() {}
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			return ctx, func() {}
		}
	}

	return context.WithTimeout(ctx, timeout)
}

func toGenerateRequest(prompt string, cfg *Config) *provider.GenerateRequest {
	return &provider.GenerateRequest{
		Prompt:   prompt,
		Messages: cfg.Messages,
		System:   cfg.System,
		Config:   toProviderConfig(cfg),
	}
}

func toProviderConfig(cfg *Config) provider.Config {
	return provider.Config{
		System:         cfg.System,
		MaxTokens:      cfg.MaxTokens,
		Temperature:    cfg.Temperature,
		TopP:           cfg.TopP,
		TopK:           cfg.TopK,
		StopSequences:  cfg.StopSequences,
		Tools:          cfg.Tools,
		ToolChoice:     cfg.ToolChoice,
		Seed:           cfg.Seed,
		ResponseFormat: cfg.ResponseFormat,
	}
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

func (c *Config) recordTokens(ctx context.Context, model provider.Model, usage provider.Usage) {
	if c.Metrics != nil {
		c.Metrics.RecordTokens(ctx, model.Provider(), model.ModelID(), usage.PromptTokens, usage.CompletionTokens)
	}
}
