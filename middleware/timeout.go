package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// TimeoutConfig configures timeout behavior for the Timeout middleware.
type TimeoutConfig struct {
	// Default is the default timeout applied to all requests.
	// If zero, no default timeout is applied.
	Default time.Duration

	// PerProvider maps provider names to specific timeouts.
	// These override the Default for specific providers.
	PerProvider map[string]time.Duration

	// PerModel maps model IDs to specific timeouts.
	// These override both Default and PerProvider.
	PerModel map[string]time.Duration
}

// Timeout returns a middleware that enforces request timeouts.
//
// If the context already has a deadline, the middleware respects it and does not
// apply an additional timeout. This allows callers to set their own deadlines.
//
// If no deadline exists, the middleware applies a timeout based on:
//  1. PerModel[modelID] if the model has a specific timeout
//  2. PerProvider[providerName] if the provider has a specific timeout
//  3. Default timeout if configured
//  4. No timeout otherwise (passthrough)
//
// Example:
//
//	cfg := TimeoutConfig{
//	    Default: 30 * time.Second,
//	    PerProvider: map[string]time.Duration{
//	        "openai": 60 * time.Second,
//	    },
//	}
//	handler := middleware.Timeout(cfg)(baseHandler)
func Timeout(cfg TimeoutConfig) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			// If context already has a deadline, respect it
			if _, hasDeadline := ctx.Deadline(); hasDeadline {
				return next.Handle(ctx, req)
			}

			// Determine timeout based on config hierarchy
			timeout := cfg.Default

			// Check per-provider override
			if cfg.PerProvider != nil {
				if t, ok := cfg.PerProvider[req.Provider()]; ok {
					timeout = t
				}
			}

			// Check per-model override (highest priority)
			if cfg.PerModel != nil {
				if t, ok := cfg.PerModel[req.ModelID()]; ok {
					timeout = t
				}
			}

			// If no timeout configured, passthrough
			if timeout <= 0 {
				return next.Handle(ctx, req)
			}

			// Apply timeout
			ctx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			resp, err := next.Handle(ctx, req)
			if err != nil {
				// Wrap context cancellation errors with provider error
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, &provider.Error{
						Code:       provider.CodeAPITimeout,
						Message:    "request timed out",
						Cause:      err,
						Provider:   req.Provider(),
						ModelID:    req.ModelID(),
						StatusCode: 408,
					}
				}
				return nil, err
			}

			return resp, nil
		})
	}
}

// TimeoutWithDefault creates a timeout middleware with a single default timeout.
//
// This is a convenience function for the common case of applying the same timeout
// to all requests.
//
// Example:
//
//	handler := middleware.TimeoutWithDefault(30 * time.Second)(baseHandler)
func TimeoutWithDefault(timeout time.Duration) Middleware {
	return Timeout(TimeoutConfig{Default: timeout})
}

// TimeoutPerProvider creates a timeout middleware with per-provider timeouts.
//
// This is useful when different providers have different latency characteristics.
//
// Example:
//
//	timeouts := map[string]time.Duration{
//	    "openai":    60 * time.Second,
//	    "anthropic": 45 * time.Second,
//	}
//	handler := middleware.TimeoutPerProvider(timeouts)(baseHandler)
func TimeoutPerProvider(perProvider map[string]time.Duration) Middleware {
	return Timeout(TimeoutConfig{PerProvider: perProvider})
}
