# Middleware Pattern

Lamar SDK supports a middleware chain for request/response processing, enabling extensibility without inheritance.

---

## Overview

The middleware pattern allows you to wrap the core generation logic with cross-cutting concerns like:

- Logging
- Retry logic
- Rate limiting
- Metrics collection
- Tracing
- Caching
- Request/response transformation

---

## Handler Interface

```go
// middleware/middleware.go
package middleware

import (
    "context"
)

// Handler processes a generation request.
type Handler interface {
    Handle(ctx context.Context, req *Request) (*Response, error)
}

// HandlerFunc is an adapter that allows using functions as handlers.
type HandlerFunc func(ctx context.Context, req *Request) (*Response, error)

func (f HandlerFunc) Handle(ctx context.Context, req *Request) (*Response, error) {
    return f(ctx, req)
}

// Request represents a generation request for the middleware chain.
type Request struct {
    Model   Generator
    Prompt  string
    Config  *GenerateConfig
}

// Response represents a generation response for the middleware chain.
type Response struct {
    Text         string
    Content      []Content
    ToolCalls    []ToolCall
    ToolResults  []ToolResult
    FinishReason FinishReason
    Usage        Usage
}
```

---

## Middleware Definition

```go
// Middleware wraps a handler with additional behavior.
type Middleware func(next Handler) Handler

// Chain combines multiple middlewares into one.
// Middlewares are applied in order (first middleware is outermost).
func Chain(middlewares ...Middleware) Middleware {
    return func(final Handler) Handler {
        for i := len(middlewares) - 1; i >= 0; i-- {
            final = middlewares[i](final)
        }
        return final
    }
}

// Apply wraps a handler with a chain of middlewares.
func Apply(handler Handler, middlewares ...Middleware) Handler {
    return Chain(middlewares...)(handler)
}
```

---

## Built-in Middlewares

### Logging Middleware

```go
// middleware/logging.go
package middleware

import (
    "context"
    "time"
)

// LoggingMiddleware logs all requests and responses.
func LoggingMiddleware(logger Logger) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            start := time.Now()
            
            logger.Debug("request starting",
                "provider", req.Model.Provider(),
                "model", req.Model.ModelID(),
                "prompt_length", len(req.Prompt),
            )
            
            resp, err := next.Handle(ctx, req)
            
            duration := time.Since(start)
            
            if err != nil {
                logger.Error("request failed",
                    "provider", req.Model.Provider(),
                    "model", req.Model.ModelID(),
                    "duration", duration,
                    "error", err,
                )
            } else {
                logger.Info("request completed",
                    "provider", req.Model.Provider(),
                    "model", req.Model.ModelID(),
                    "duration", duration,
                    "tokens", resp.Usage.TotalTokens,
                    "finish_reason", resp.FinishReason,
                )
            }
            
            return resp, err
        })
    }
}
```

### Retry Middleware

```go
// middleware/retry.go
package middleware

import (
    "context"
    "time"
)

// BackoffStrategy determines wait time between retries.
type BackoffStrategy interface {
    Duration(attempt int) time.Duration
}

// ExponentialBackoff implements exponential backoff.
type ExponentialBackoff struct {
    Initial time.Duration
    Max     time.Duration
    Factor  float64
}

func (b *ExponentialBackoff) Duration(attempt int) time.Duration {
    d := b.Initial
    for i := 0; i < attempt; i++ {
        d = time.Duration(float64(d) * b.Factor)
        if d > b.Max {
            d = b.Max
        }
    }
    return d
}

// DefaultBackoff provides sensible defaults.
var DefaultBackoff = &ExponentialBackoff{
    Initial: 100 * time.Millisecond,
    Max:     30 * time.Second,
    Factor:  2.0,
}

// RetryMiddleware retries failed requests.
func RetryMiddleware(maxRetries int, backoff BackoffStrategy) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            var lastErr error
            
            for attempt := 0; attempt <= maxRetries; attempt++ {
                // Check context before each attempt
                if ctx.Err() != nil {
                    return nil, ctx.Err()
                }
                
                resp, err := next.Handle(ctx, req)
                if err == nil {
                    return resp, nil
                }
                
                lastErr = err
                
                // Only retry on retryable errors
                if !IsRetryable(err) {
                    return nil, err
                }
                
                // Don't wait after the last attempt
                if attempt < maxRetries {
                    wait := backoff.Duration(attempt)
                    select {
                    case <-ctx.Done():
                        return nil, ctx.Err()
                    case <-time.After(wait):
                    }
                }
            }
            
            return nil, lastErr
        })
    }
}

// IsRetryable determines if an error is retryable.
func IsRetryable(err error) bool {
    switch ErrorCodeOf(err) {
    case CodeRateLimited:
        return true
    case CodeAPITimeout:
        return true
    default:
        // Retry on 5xx errors
        if apiErr, ok := err.(*Error); ok && apiErr.StatusCode >= 500 {
            return true
        }
    }
    return false
}
```

### Rate Limiting Middleware

```go
// middleware/rate_limit.go
package middleware

import (
    "context"
    "golang.org/x/time/rate"
)

// RateLimitMiddleware limits request rate.
func RateLimitMiddleware(limiter *rate.Limiter) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            if err := limiter.Wait(ctx); err != nil {
                return nil, &Error{
                    Code:    CodeRateLimited,
                    Message: "rate limit exceeded",
                    Cause:   err,
                }
            }
            return next.Handle(ctx, req)
        })
    }
}

// PerProviderRateLimit creates rate limiters per provider.
func PerProviderRateLimit(limits map[string]*rate.Limiter, defaultLimiter *rate.Limiter) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            limiter := defaultLimiter
            if l, ok := limits[req.Model.Provider()]; ok {
                limiter = l
            }
            
            if limiter == nil {
                return next.Handle(ctx, req)
            }
            
            if err := limiter.Wait(ctx); err != nil {
                return nil, &Error{
                    Code:    CodeRateLimited,
                    Message: "rate limit exceeded",
                    Cause:   err,
                }
            }
            
            return next.Handle(ctx, req)
        })
    }
}
```

### Metrics Middleware

```go
// middleware/metrics.go
package middleware

import (
    "context"
    "time"
)

// MetricsMiddleware records request metrics.
func MetricsMiddleware(collector MetricsCollector) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            start := time.Now()
            
            collector.RecordStreamStart(ctx, req.Model.Provider(), req.Model.ModelID())
            
            resp, err := next.Handle(ctx, req)
            
            duration := time.Since(start)
            
            collector.RecordRequest(ctx,
                req.Model.Provider(),
                req.Model.ModelID(),
                duration,
                err,
            )
            
            if resp != nil {
                collector.RecordTokens(ctx,
                    req.Model.Provider(),
                    req.Model.ModelID(),
                    resp.Usage.PromptTokens,
                    resp.Usage.CompletionTokens,
                )
            }
            
            collector.RecordStreamEnd(ctx,
                req.Model.Provider(),
                req.Model.ModelID(),
                duration,
            )
            
            return resp, err
        })
    }
}
```

### Timeout Middleware

```go
// middleware/timeout.go
package middleware

import (
    "context"
    "time"
)

// TimeoutMiddleware enforces a request timeout.
func TimeoutMiddleware(timeout time.Duration) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            // Check if context already has a deadline
            if _, hasDeadline := ctx.Deadline(); hasDeadline {
                return next.Handle(ctx, req)
            }
            
            // Apply timeout
            ctx, cancel := context.WithTimeout(ctx, timeout)
            defer cancel()
            
            return next.Handle(ctx, req)
        })
    }
}

// PerOperationTimeout applies different timeouts per operation type.
func PerOperationTimeout(defaultTimeout time.Duration, timeouts map[string]time.Duration) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            if _, hasDeadline := ctx.Deadline(); hasDeadline {
                return next.Handle(ctx, req)
            }
            
            timeout := defaultTimeout
            if t, ok := timeouts[req.Model.Provider()]; ok {
                timeout = t
            }
            
            ctx, cancel := context.WithTimeout(ctx, timeout)
            defer cancel()
            
            return next.Handle(ctx, req)
        })
    }
}
```

### Caching Middleware

```go
// middleware/cache.go
package middleware

import (
    "context"
    "crypto/sha256"
    "encoding/json"
    "fmt"
)

// CacheStore defines the cache interface.
type CacheStore interface {
    Get(ctx context.Context, key string) (*CachedResponse, error)
    Set(ctx context.Context, key string, value *CachedResponse, ttl time.Duration) error
}

// CachedResponse represents a cached response.
type CachedResponse struct {
    Text         string
    Content      []Content
    ToolCalls    []ToolCall
    ToolResults  []ToolResult
    FinishReason FinishReason
    Usage        Usage
}

// CachingMiddleware caches responses.
func CachingMiddleware(store CacheStore, ttl time.Duration) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            // Generate cache key
            key := cacheKey(req)
            
            // Try to get from cache
            if cached, err := store.Get(ctx, key); err == nil && cached != nil {
                return &Response{
                    Text:         cached.Text,
                    Content:      cached.Content,
                    ToolCalls:    cached.ToolCalls,
                    ToolResults:  cached.ToolResults,
                    FinishReason: cached.FinishReason,
                    Usage:        cached.Usage,
                }, nil
            }
            
            // Execute request
            resp, err := next.Handle(ctx, req)
            if err != nil {
                return nil, err
            }
            
            // Cache the response
            cached := &CachedResponse{
                Text:         resp.Text,
                Content:      resp.Content,
                ToolCalls:    resp.ToolCalls,
                ToolResults:  resp.ToolResults,
                FinishReason: resp.FinishReason,
                Usage:        resp.Usage,
            }
            
            // Ignore cache write errors
            _ = store.Set(ctx, key, cached, ttl)
            
            return resp, nil
        })
    }
}

func cacheKey(req *Request) string {
    h := sha256.New()
    h.Write([]byte(req.Model.Provider()))
    h.Write([]byte(req.Model.ModelID()))
    h.Write([]byte(req.Prompt))
    
    if req.Config != nil {
        data, _ := json.Marshal(req.Config)
        h.Write(data)
    }
    
    return fmt.Sprintf("%x", h.Sum(nil))
}
```

### Tracing Middleware (OpenTelemetry)

```go
// middleware/tracing.go
package middleware

import (
    "context"
    
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
    "go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates OpenTelemetry spans.
func TracingMiddleware(tp trace.TracerProvider) Middleware {
    tracer := tp.Tracer("github.com/yourorg/lamar")
    
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            ctx, span := tracer.Start(ctx, "lamar.generate",
                trace.WithAttributes(
                    attribute.String("provider", req.Model.Provider()),
                    attribute.String("model", req.Model.ModelID()),
                    attribute.Int("prompt_length", len(req.Prompt)),
                ),
            )
            defer span.End()
            
            // Add request ID to span if available
            if reqID := RequestID(ctx); reqID != "" {
                span.SetAttributes(attribute.String("request_id", reqID))
            }
            
            resp, err := next.Handle(ctx, req)
            
            if err != nil {
                span.RecordError(err)
                span.SetStatus(codes.Error, err.Error())
                return nil, err
            }
            
            span.SetAttributes(
                attribute.Int("prompt_tokens", resp.Usage.PromptTokens),
                attribute.Int("completion_tokens", resp.Usage.CompletionTokens),
                attribute.Int("total_tokens", resp.Usage.TotalTokens),
                attribute.String("finish_reason", string(resp.FinishReason)),
            )
            
            span.SetStatus(codes.Ok, "")
            
            return resp, nil
        })
    }
}
```

### Request/Response Transformation

```go
// middleware/transform.go
package middleware

// RequestTransformer transforms requests before processing.
type RequestTransformer func(ctx context.Context, req *Request) (*Request, error)

// ResponseTransformer transforms responses after processing.
type ResponseTransformer func(ctx context.Context, resp *Response) (*Response, error)

// TransformMiddleware applies request and response transformations.
func TransformMiddleware(
    reqTransform RequestTransformer,
    respTransform ResponseTransformer,
) Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            // Transform request
            if reqTransform != nil {
                transformed, err := reqTransform(ctx, req)
                if err != nil {
                    return nil, err
                }
                req = transformed
            }
            
            // Execute
            resp, err := next.Handle(ctx, req)
            if err != nil {
                return nil, err
            }
            
            // Transform response
            if respTransform != nil {
                transformed, err := respTransform(ctx, resp)
                if err != nil {
                    return nil, err
                }
                resp = transformed
            }
            
            return resp, nil
        })
    }
}

// Example: Add default system prompt
func DefaultSystemPrompt(system string) Middleware {
    return TransformMiddleware(
        func(ctx context.Context, req *Request) (*Request, error) {
            if req.Config.System == "" {
                req.Config.System = system
            }
            return req, nil
        },
        nil,
    )
}
```

---

## Provider Integration

### Using Middleware with Providers

```go
// providers/openai/provider.go enhancement
package openai

type Provider struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    orgID      string
    
    // Middleware chain
    middlewares []middleware.Middleware
    handler     middleware.Handler
}

// WithMiddleware adds middleware to the provider.
func WithMiddleware(middlewares ...middleware.Middleware) Option {
    return func(p *Provider) {
        p.middlewares = append(p.middlewares, middlewares...)
    }
}

func (p *Provider) buildHandler() middleware.Handler {
    // Core handler
    core := middleware.HandlerFunc(p.handleGenerate)
    
    // Apply middleware chain
    return middleware.Apply(core, p.middlewares...)
}
```

### Complete Example

```go
package main

import (
    "context"
    "log/slog"
    "time"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/middleware"
    "github.com/yourorg/lamar/providers/openai"
    "golang.org/x/time/rate"
)

func main() {
    logger := slog.Default()
    
    // Create provider with middleware
    client := openai.NewProvider(
        openai.WithMiddleware(
            middleware.LoggingMiddleware(lamar.NewSlogAdapter(logger)),
            middleware.TimeoutMiddleware(30*time.Second),
            middleware.RetryMiddleware(3, middleware.DefaultBackoff),
            middleware.RateLimitMiddleware(rate.NewLimiter(rate.Every(time.Second), 10)),
            middleware.MetricsMiddleware(lamar.NewSlogAdapter(logger)),
        ),
    )
    
    // Generate with all middleware applied
    result, err := lamar.Generate(
        context.Background(),
        client.GPT4oMini(),
        "Say hello",
        lamar.MaxTokens(100),
    )
    if err != nil {
        panic(err)
    }
    
    log.Println(result.Text)
}
```

---

## Creating Custom Middleware

### Pattern

```go
func MyCustomMiddleware(config MyConfig) middleware.Middleware {
    return func(next middleware.Handler) middleware.Handler {
        return middleware.HandlerFunc(func(ctx context.Context, req *middleware.Request) (*middleware.Response, error) {
            // Pre-processing
            // ...
            
            // Call next handler
            resp, err := next.Handle(ctx, req)
            if err != nil {
                // Error handling
                return nil, err
            }
            
            // Post-processing
            // ...
            
            return resp, nil
        })
    }
}
```

### Example: Content Filter

```go
//middleware/content_filter.go
package middleware

import (
    "context"
    "strings"
)

type ContentFilter struct {
    blockedWords []string
}

func ContentFilterMiddleware(blockedWords ...string) Middleware {
    filter := &ContentFilter{blockedWords: blockedWords}
    
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            // Check prompt
            for _, word := range filter.blockedWords {
                if strings.Contains(strings.ToLower(req.Prompt), word) {
                    return nil, &Error{
                        Code:    CodeContentFiltered,
                        Message: "prompt contains blocked content",
                    }
                }
            }
            
            // Execute
            resp, err := next.Handle(ctx, req)
            if err != nil {
                return nil, err
            }
            
            // Check response
            for _, word := range filter.blockedWords {
                if strings.Contains(strings.ToLower(resp.Text), word) {
                    return nil, &Error{
                        Code:    CodeContentFiltered,
                        Message: "response contains blocked content",
                    }
                }
            }
            
            return resp, nil
        })
    }
}
```

### Example: Request ID Injection

```go
// middleware/request_id.go
package middleware

import (
    "context"
    "github.com/google/uuid"
)

// RequestIDMiddleware injects a unique request ID.
func RequestIDMiddleware() Middleware {
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            // Generate request ID if not present
            if RequestID(ctx) == "" {
                ctx = WithRequestID(ctx, uuid.New().String())
            }
            
            return next.Handle(ctx, req)
        })
    }
}
```

---

## Testing Middleware

```go
//middleware/middleware_test.go
package middleware_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/lamar/middleware"
)

func TestLoggingMiddleware(t *testing.T) {
    // Mock logger
    logger := &mockLogger{}
    
    // Create middleware chain
    handler := middleware.LoggingMiddleware(logger)(
        middleware.HandlerFunc(func(ctx context.Context, req *middleware.Request) (*middleware.Response, error) {
            return &middleware.Response{Text: "Hello"}, nil
        }),
    )
    
    // Execute
    resp, err := handler.Handle(context.Background(), &middleware.Request{
        Prompt: "Test",
    })
    
    require.NoError(t, err)
    assert.Equal(t, "Hello", resp.Text)
    assert.True(t, logger.infoCalled)
}

// mockLogger implements Logger for testing
type mockLogger struct {
    infoCalled bool
}

func (m *mockLogger) Debug(msg string, args ...any) {}
func (m *mockLogger) Info(msg string, args ...any)  { m.infoCalled = true }
func (m *mockLogger) Warn(msg string, args ...any)  {}
func (m *mockLogger) Error(msg string, args ...any) {}
```

---

## Best Practices

1. **Order Matters**: Middleware is applied in order. Put logging first, rate limiting early.
2. **Short-circuit on Error**: Return immediately on non-retryable errors.
3. **Context Awareness**: Always check `ctx.Err()` before operations.
4. **Thread Safety**: Middleware may be called concurrently.
5. **Simplicity**: Keep middleware focused on a single concern.

### Recommended Order

```go
client := openai.NewProvider(
    openai.WithMiddleware(
        // 1. Request ID (outermost)
        middleware.RequestIDMiddleware(),
        
        // 2. Logging
        middleware.LoggingMiddleware(logger),
        
        // 3. Tracing
        middleware.TracingMiddleware(tracerProvider),
        
        // 4. Metrics
        middleware.MetricsMiddleware(metricsCollector),
        
        // 5. Rate Limiting
        middleware.RateLimitMiddleware(limiter),
        
        // 6. Timeout
        middleware.TimeoutMiddleware(30*time.Second),
        
        // 7. Retry
        middleware.RetryMiddleware(3, middleware.DefaultBackoff),
        
        // 8. Caching (innermost, after retries)
        middleware.CachingMiddleware(cache, 5*time.Minute),
    ),
)
```

---

## Summary

The middleware pattern provides:

- **Extensibility**: Add functionality without modifying core code
- **Composability**: Mix and match middleware as needed
- **Separation of Concerns**: Each middleware handles one thing
- **Testability**: Test middleware in isolation
- **Flexibility**: Different middleware stacks per provider