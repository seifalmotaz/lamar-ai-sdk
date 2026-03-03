package middleware

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// Metrics creates a middleware that collects metrics for each request.
// It records request duration, errors, and token usage.
//
// Example:
//
//	handler := middleware.Metrics(collector)(nextHandler)
func Metrics(collector provider.MetricsCollector) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			start := time.Now()

			resp, err := next.Handle(ctx, req)

			duration := time.Since(start)
			collector.RecordRequest(ctx, req.Provider(), req.ModelID(), duration, err)

			if err == nil {
				usage := resp.Usage()
				collector.RecordTokens(ctx, req.Provider(), req.ModelID(), usage.PromptTokens, usage.CompletionTokens)
			}

			return resp, err
		})
	}
}
