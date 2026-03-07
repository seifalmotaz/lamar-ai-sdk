package ctxutil

import (
	"context"
	"time"
)

// ApplyTimeout applies a timeout to the context if one isn't already set.
// If timeout is 0, returns the context unchanged with a no-op cancel function.
// If the context already has a deadline sooner than timeout, returns unchanged.
func ApplyTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
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
