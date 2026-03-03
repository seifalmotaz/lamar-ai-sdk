package middleware

import (
	"context"
	"errors"
	"math/rand"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// RetryConfig configures retry behavior.
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts (including the initial request).
	MaxAttempts int

	// InitialDelay is the initial backoff delay.
	InitialDelay time.Duration

	// MaxDelay is the maximum backoff delay.
	MaxDelay time.Duration

	// Multiplier is the backoff multiplier (default: 2.0).
	Multiplier float64

	// RetryOn is a function that determines if a request should be retried.
	// If nil, retries on rate limits (429) and transient errors (5xx).
	RetryOn func(err error) bool
}

// DefaultRetryConfig returns a sensible default retry configuration.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		RetryOn:      DefaultRetryPredicate,
	}
}

// DefaultRetryPredicate returns true for retriable errors.
func DefaultRetryPredicate(err error) bool {
	var providerErr *provider.Error
	if !isError(err, &providerErr) {
		return false
	}

	// Retry on rate limits
	if providerErr.Code == provider.CodeRateLimited {
		return true
	}

	// Retry on timeouts
	if providerErr.Code == provider.CodeAPITimeout {
		return true
	}

	// Retry on server errors (5xx)
	if providerErr.StatusCode >= 500 && providerErr.StatusCode < 600 {
		return true
	}

	return false
}

// isError helper to check if error is a provider.Error
func isError(err error, target **provider.Error) bool {
	return errors.As(err, target)
}

// Retry returns a middleware that retries failed requests with exponential backoff.
//
// Example:
//
//	cfg := RetryConfig{
//	    MaxAttempts:  5,
//	    InitialDelay: 1 * time.Second,
//	    MaxDelay:     30 * time.Second,
//	}
//	handler := middleware.Retry(cfg)(baseHandler)
func Retry(cfg RetryConfig) Middleware {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 1
	}
	if cfg.InitialDelay <= 0 {
		cfg.InitialDelay = 1 * time.Second
	}
	if cfg.MaxDelay <= 0 {
		cfg.MaxDelay = 30 * time.Second
	}
	if cfg.Multiplier <= 0 {
		cfg.Multiplier = 2.0
	}
	if cfg.RetryOn == nil {
		cfg.RetryOn = DefaultRetryPredicate
	}

	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			var lastErr error
			delay := cfg.InitialDelay

			for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
				resp, err := next.Handle(ctx, req)
				if err == nil {
					return resp, nil
				}

				lastErr = err

				// Check if we should retry
				if !cfg.RetryOn(err) {
					return nil, err
				}

				// Check if we've exhausted attempts
				if attempt >= cfg.MaxAttempts {
					return nil, err
				}

				// Check context cancellation
				if ctx.Err() != nil {
					return nil, ctx.Err()
				}

				// Get retry-after hint from error if available
				retryAfter := provider.RetryAfter(err)
				if retryAfter > 0 {
					delay = retryAfter
				}

				// Wait before retrying
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(delay):
				}

				// Calculate next delay with multiplier and cap
				delay = time.Duration(float64(delay) * cfg.Multiplier)
				if delay > cfg.MaxDelay {
					delay = cfg.MaxDelay
				}

				// Add jitter (10% of delay)
				jitter := time.Duration(rand.Float64() * float64(delay) * 0.1)
				delay += jitter
			}

			return nil, lastErr
		})
	}
}
