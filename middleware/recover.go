package middleware

import (
	"context"
	"fmt"
	"runtime/debug"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Recover creates a middleware that recovers from panics in the handler chain.
// If a panic occurs, it returns a provider.Error with the panic message and stack trace.
//
// Example:
//
//	handler := middleware.Recover()(nextHandler)
func Recover() Middleware {
	return func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (resp Response, err error) {
			defer func() {
				if r := recover(); r != nil {
					stack := string(debug.Stack())
					err = &provider.Error{
						Code:    provider.CodeUnknown,
						Message: fmt.Sprintf("panic recovered: %v", r),
						Cause:   fmt.Errorf("%s", stack),
					}
				}
			}()
			return next.Handle(ctx, req)
		})
	}
}
