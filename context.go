package lamar

import "context"

// ctxKey is the type for context keys used by this package.
type ctxKey string

// Context key constants.
const (
	requestIDKey ctxKey = "request-id"
	traceIDKey   ctxKey = "trace-id"
	userIDKey    ctxKey = "user-id"
)

// WithRequestID adds a request ID to the context for tracing.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID extracts the request ID from the context.
// Returns an empty string if no request ID is set.
func RequestID(ctx context.Context) string {
	if v := ctx.Value(requestIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// WithTraceID adds an OpenTelemetry trace ID to the context.
func WithTraceID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceIDKey, id)
}

// TraceID extracts the trace ID from the context.
// Returns an empty string if no trace ID is set.
func TraceID(ctx context.Context) string {
	if v := ctx.Value(traceIDKey); v != nil {
		return v.(string)
	}
	return ""
}

// WithUserID adds a user ID to the context for user attribution.
func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

// UserID extracts the user ID from the context.
// Returns an empty string if no user ID is set.
func UserID(ctx context.Context) string {
	if v := ctx.Value(userIDKey); v != nil {
		return v.(string)
	}
	return ""
}
