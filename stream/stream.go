package stream

import (
	"context"
	"strings"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// DefaultTimeout is the default timeout for streaming operations.
const DefaultTimeout = 2 * time.Minute

// Stream streams text generation from a model.
// It returns a Result immediately, which can be used to consume
// the stream in real-time or wait for completion.
//
// The model must implement the Streamer interface. Use provider.CanStream(m)
// to check if a model supports streaming.
//
// Example:
//
//	result := stream.Stream(ctx, model, "Tell me a story")
//
//	// Real-time consumption
//	for part := range result.Stream() {
//	    if text, ok := part.(provider.StreamTextPart); ok {
//	        fmt.Print(text.Delta)
//	    }
//	}
//
//	// Wait and get result
//	text, err := result.Text()
func Stream(ctx context.Context, model provider.Streamer, prompt string, opts ...Option) *Result {
	if model == nil {
		result := newResult()
		result.setError(provider.ErrInvalidModel)
		result.close()
		return result
	}

	// Check if context is already canceled
	select {
	case <-ctx.Done():
		result := newResult()
		result.setError(provider.NewError(provider.CodeContextCanceled, "context already canceled", ctx.Err()))
		result.close()
		return result
	default:
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	result := newResult()

	go func() {
		defer result.close()

		start := time.Now()
		cfg.logDebug("stream: starting", "provider", model.Provider(), "model", model.ModelID())
		cfg.recordStreamStart(ctx, model)

		// Build request
		req := &provider.GenerateRequest{
			Prompt:   prompt,
			Messages: cfg.Messages,
			System:   cfg.System,
			Config: provider.Config{
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
			},
		}

		// Apply timeout
		var timeout time.Duration
		if cfg.Timeout == nil {
			timeout = DefaultTimeout
		} else {
			timeout = *cfg.Timeout
		}
		streamCtx, cancel := applyTimeout(ctx, timeout)
		defer cancel()

		// Call model
		streamResult, err := model.Stream(streamCtx, req)
		if err != nil {
			// Send error part
			result.stream <- provider.StreamErrorPart{Error: err}
			result.setError(err)
			cfg.logError("stream: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
			cfg.recordMetricsWithErr(ctx, model, time.Since(start), err)
			cfg.recordStreamEnd(ctx, model, time.Since(start))
			return
		}

		// Accumulate text and parts
		var textBuilder strings.Builder
		var usage provider.Usage
		var finishReason provider.FinishReason
		var streamErr error

		// Stream parts from model
		for part := range streamResult.Stream {
			// Forward to consumer
			result.stream <- part

			switch p := part.(type) {
			case provider.StreamTextPart:
				textBuilder.WriteString(p.Delta)
			case provider.StreamFinishPart:
				usage = p.Usage
				finishReason = p.FinishReason
			case provider.StreamErrorPart:
				streamErr = p.Error
			}
		}

		// Store final result
		result.setText(textBuilder.String())
		result.setUsage(usage)
		result.setFinishReason(finishReason)
		if streamErr != nil {
			result.setError(streamErr)
		}

		cfg.logDebug("stream: completed", "provider", model.Provider(), "model", model.ModelID(), "tokens", usage.TotalTokens)
		cfg.recordMetricsWithErr(ctx, model, time.Since(start), streamErr)
		cfg.recordStreamEnd(ctx, model, time.Since(start))
	}()

	return result
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
