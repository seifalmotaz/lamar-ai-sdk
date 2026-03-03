package middleware

import (
	"context"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// Logging creates a middleware that logs request lifecycle events.
// Logs are written at DEBUG level for successful requests and ERROR level for failures.
//
// Example:
//
//	handler := middleware.Logging(logger)(nextHandler)
func Logging(logger provider.Logger) Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			start := time.Now()
			logger.Debug("request started",
				"provider", req.Provider(),
				"model", req.ModelID(),
			)

			resp, err := next.Handle(ctx, req)

			duration := time.Since(start)
			if err != nil {
				logger.Error("request failed",
					"provider", req.Provider(),
					"model", req.ModelID(),
					"duration", duration,
					"error", err,
				)
			} else {
				logger.Debug("request completed",
					"provider", req.Provider(),
					"model", req.ModelID(),
					"duration", duration,
					"tokens", resp.Usage().TotalTokens,
				)
			}

			return resp, err
		})
	}
}
