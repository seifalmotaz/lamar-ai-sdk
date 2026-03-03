# Go Ecosystem Recommendations

This document outlines recommended libraries and patterns from the Go ecosystem for building the Lamar AI SDK.

---

## 1. HTTP Client & Transport

### Recommended: `net/http` + Custom Client

```go
import (
    "net/http"
    "time"
)

// Create configurable HTTP client
client := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}
```

### For Retries: `github.com/sethvargo/go-retry`

```go
import "github.com/sethvargo/go-retry"

func doWithRetry(ctx context.Context, fn func() error) error {
    return retry.Do(ctx,
        retry.WithMaxRetries(3),
        retry.WithBackoff(retry.Fibonacci(100*time.Millisecond)),
        func(ctx context.Context) error {
            return fn()
        },
    )
}
```

### Alternative: Built-in Backoff

```go
func doWithRetry(ctx context.Context, maxRetries int, fn func() error) error {
    var lastErr error
    for i := 0; i <= maxRetries; i++ {
        err := fn()
        if err == nil {
            return nil
        }
        lastErr = err
        
        if !isRetryable(err) {
            return err
        }
        
        // Exponential backoff
        time.Sleep(time.Duration(1<<uint(i)) * 100 * time.Millisecond)
    }
    return lastErr
}
```

---

## 2. JSON Schema & Validation

### Recommended: Struct Tags + `github.com/go-playground/validator`

```go
import "github.com/go-playground/validator/v10"

type Person struct {
    Name string `json:"name" validate:"required,min=1,max=100"`
    Age  int    `json:"age" validate:"required,min=0,max=150"`
    Email string `json:"email" validate:"required,email"`
}

var validate = validator.New()

func ValidateStruct(s interface{}) error {
    return validate.Struct(s)
}
```

### Schema Generation: `github.com/invopop/jsonschema`

```go
import "github.com/invopop/jsonschema"

type ToolInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
    Units    string `json:"units,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
}

// Generate JSON Schema automatically
schema := jsonschema.Reflect(&ToolInput{})
```

### Alternative: Manual Schema Builder

```go
// schema/builder.go
package schema

type Schema struct {
    Type        string                 `json:"type"`
    Properties  map[string]*Schema     `json:"properties,omitempty"`
    Required    []string               `json:"required,omitempty"`
    Description string                 `json:"description,omitempty"`
    Enum        []interface{}          `json:"enum,omitempty"`
    Items       *Schema                `json:"items,omitempty"`
}

func Object(properties map[string]*Schema, required []string) *Schema {
    return &Schema{
        Type:       "object",
        Properties: properties,
        Required:   required,
    }
}

func String(desc string) *Schema {
    return &Schema{Type: "string", Description: desc}
}

func Int(desc string) *Schema {
    return &Schema{Type: "integer", Description: desc}
}

func Number(desc string) *Schema {
    return &Schema{Type: "number", Description: desc}
}

func Enum(values ...interface{}) *Schema {
    return &Schema{Type: "string", Enum: values}
}
```

---

## 3. Streaming & Server-Sent Events

### SSE Parser (Built-in)

```go
// internal/sse/decoder.go
package sse

import (
    "bufio"
    "bytes"
    "io"
    "strings"
)

type Event struct {
    ID    string
    Type  string
    Data  []byte
    Retry int
}

type Decoder struct {
    scanner *bufio.Scanner
}

func NewDecoder(r io.Reader) *Decoder {
    return &Decoder{
        scanner: bufio.NewScanner(r),
    }
}

func (d *Decoder) Decode() (Event, error) {
    var event Event
    var data bytes.Buffer
    
    for d.scanner.Scan() {
        line := d.scanner.Text()
        
        // Empty line signals end of event
        if line == "" {
            if data.Len() > 0 {
                event.Data = data.Bytes()
                return event, nil
            }
            continue
        }
        
        // Parse field: value
        field, value, _ := strings.Cut(line, ":")
        value = strings.TrimSpace(value)
        
        switch field {
        case "id":
            event.ID = value
        case "event":
            event.Type = value
        case "data":
            if data.Len() > 0 {
                data.WriteByte('\n')
            }
            data.WriteString(value)
        case "retry":
            // Parse retry duration
        }
    }
    
    if err := d.scanner.Err(); err != nil {
        return Event{}, err
    }
    
    return Event{}, io.EOF
}

// DecodeStream returns a channel of events
func DecodeStream(r io.Reader) <-chan Event {
    ch := make(chan Event, 100)
    
    go func() {
        defer close(ch)
        decoder := NewDecoder(r)
        
        for {
            event, err := decoder.Decode()
            if err != nil {
                return
            }
            ch <- event
        }
    }()
    
    return ch
}
```

---

## 4. Structured Logging

### Recommended: `log/slog` (Go 1.21+)

```go
import "log/slog"

type GenerateOptions struct {
    // ...
    Logger *slog.Logger // Optional logger
}

func Generate(ctx context.Context, model LanguageModel, prompt string, opts ...GenerateOption) (*GenerateResult, error) {
    logger := slog.Default()
    
    logger.Debug("starting generation",
        "provider", model.Provider(),
        "model", model.ModelID(),
        "prompt_length", len(prompt),
    )
    
    // ...
    
    logger.Debug("generation complete",
        "tokens", result.Usage.TotalTokens,
        "duration", time.Since(start),
    )
}
```

### Logger Interface (for flexibility)

```go
// Logger interface that works with slog, zap, logrus, etc.
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}
```

---

## 5. Environment Variables

### Recommended: `os.Getenv` + Fallbacks

```go
import "os"

func getEnv(key, fallback string) string {
    if value := os.Getenv(key); value != "" {
        return value
    }
    return fallback
}

func getEnvRequired(key string) (string, error) {
    value := os.Getenv(key)
    if value == "" {
        return "", fmt.Errorf("%s environment variable is required", key)
    }
    return value, nil
}
```

### Alternative: `github.com/kelseyhightower/envconfig`

```go
import "github.com/kelseyhightower/envconfig"

type OpenAIConfig struct {
    APIKey string `required:"true" envconfig:"OPENAI_API_KEY"`
    OrgID  string `envconfig:"OPENAI_ORG_ID"`
    BaseURL string `default:"https://api.openai.com/v1" envconfig:"OPENAI_BASE_URL"`
}

var config OpenAIConfig
if err := envconfig.Process("", &config); err != nil {
    log.Fatal(err)
}
```

---

## 6. Context Patterns

### Timeout Pattern

```go
func Generate(ctx context.Context, model LanguageModel, prompt string, opts ...GenerateOption) (*GenerateResult, error) {
    // Apply default timeout if none set
    _, hasDeadline := ctx.Deadline()
    if !hasDeadline {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
        defer cancel()
    }
    
    // ... implementation
}
```

### Request Tracing

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
)

func Generate(ctx context.Context, model LanguageModel, prompt string, opts ...GenerateOption) (*GenerateResult, error) {
    tracer := otel.Tracer("github.com/yourorg/lamar-sdk")
    ctx, span := tracer.Start(ctx, "Generate")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("provider", model.Provider()),
        attribute.String("model", model.ModelID()),
    )
    
    // ... implementation
}
```

---

## 7. JSON Utilities

### Type-Safe JSON Parsing

```go
// jsonx/json.go
package jsonx

import (
    "encoding/json"
    "io"
)

// Decode decodes JSON from reader into T
func Decode[T any](r io.Reader) (T, error) {
    var result T
    decoder := json.NewDecoder(r)
    err := decoder.Decode(&result)
    return result, err
}

// Encode encodes T into JSON bytes
func Encode[T any](v T) ([]byte, error) {
    return json.Marshal(v)
}

// Map transforms T1 to T2 using a function
func Map[T1, T2 any](v T1, fn func(T1) T2) T2 {
    return fn(v)
}
```

### RawMessage for Flexible Types

```go
type ToolCall struct {
    ID    string          `json:"id"`
    Name  string          `json:"name"`
    Input json.RawMessage `json:"input"` // Keep raw JSON for later parsing
}

func (tc *ToolCall) ParseInput(v interface{}) error {
    return json.Unmarshal(tc.Input, v)
}
```

---

## 8. Rate Limiting

### Recommended: `golang.org/x/time/rate`

```go
import "golang.org/x/time/rate"

type Provider struct {
    // ...
    limiter *rate.Limiter
}

func NewProvider(opts ...ProviderOption) *Provider {
    p := &Provider{
        // ...
        limiter: rate.NewLimiter(rate.Every(time.Second), 10), // 10 req/s
    }
    // ...
}

func (p *Provider) doRequest(ctx context.Context, req *http.Request) (*http.Response, error) {
    // Wait for rate limiter
    if err := p.limiter.Wait(ctx); err != nil {
        return nil, err
    }
    
    return p.httpClient.Do(req.WithContext(ctx))
}
```

---

## 9. Connection Pooling

### HTTP Connection Pooling

```go
import "net/http"

func defaultHTTPClient() *http.Client {
    return &http.Client{
        Transport: &http.Transport{
            MaxIdleConns:        100,
            MaxIdleConnsPerHost: 10,
            IdleConnTimeout:     90 * time.Second,
            TLSHandshakeTimeout: 10 * time.Second,
            DialContext: (&net.Dialer{
                Timeout:   30 * time.Second,
                KeepAlive: 30 * time.Second,
            }).DialContext,
        },
        Timeout: 30 * time.Second,
    }
}
```

---

## 10. Graceful Shutdown

### Server Integration

```go
import (
    "context"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"
)

func main() {
    // ... setup
    
    server := &http.Server{Addr: ":8080"}
    
    go func() {
        if err := server.ListenAndServe(); err != http.ErrServerClosed {
            log.Fatal(err)
        }
    }()
    
    // Wait for interrupt signal
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit
    
    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := server.Shutdown(ctx); err != nil {
        log.Fatal(err)
    }
}
```

---

## 11. CLI Integration

### For Examples: `github.com/spf13/cobra`

```go
import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
    Use:   "lamar",
    Short: "Lamar AI SDK CLI",
}

var generateCmd = &cobra.Command{
    Use:   "generate [prompt]",
    Short: "Generate text",
    Args:  cobra.MinimumNArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        prompt := args[0]
        model, _ := cmd.Flags().GetString("model")
        provider, _ := cmd.Flags().GetString("provider")
        
        // ...
    },
}

func init() {
    generateCmd.Flags().StringP("model", "m", "gpt-4o-mini", "Model to use")
    generateCmd.Flags().StringP("provider", "p", "openai", "Provider to use")
    rootCmd.AddCommand(generateCmd)
}

func main() {
    cobra.CheckErr(rootCmd.Execute())
}
```

### Alternative: `github.com/urfave/cli/v2`

```go
import "github.com/urfave/cli/v2"

func main() {
    app := &cli.App{
        Name:  "lamar",
        Usage: "Lamar AI SDK CLI",
        Commands: []*cli.Command{
            {
                Name:  "generate",
                Usage: "Generate text",
                Flags: []cli.Flag{
                    &cli.StringFlag{
                        Name:    "model",
                        Aliases: []string{"m"},
                        Value:   "gpt-4o-mini",
                    },
                },
                Action: func(c *cli.Context) error {
                    // ...
                },
            },
        },
    }
    
    app.Run(os.Args)
}
```

---

## 12. Web Framework Integrations

### net/http Handler

```go
// httpx/chat_handler.go
package httpx

import (
    "encoding/json"
    "net/http"
)

type ChatHandler struct {
    model LanguageModel
}

func NewChatHandler(model LanguageModel) *ChatHandler {
    return &ChatHandler{model: model}
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    
    flusher, _ := w.(http.Flusher)
    
    result := Stream(r.Context(), h.model, req.Message)
    
    for part := range result.Stream() {
        data, _ := json.Marshal(part)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }
}

// Usage
http.Handle("/chat", NewChatHandler(model))
http.ListenAndServe(":8080", nil)
```

### Gin Integration

```go
// gin/chat.go
package gin

import "github.com/gin-gonic/gin"

func ChatHandler(model LanguageModel) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req ChatRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        
        result := Stream(c.Request.Context(), model, req.Message)
        
        c.Stream(func(w io.Writer) bool {
            part, ok := <-result.Stream()
            if !ok {
                return false
            }
            data, _ := json.Marshal(part)
            fmt.Fprintf(w, "data: %s\n\n", data)
            return true
        })
    }
}
```

### Echo Integration

```go
// echo/chat.go
package echo

import "github.com/labstack/echo/v4"

func ChatHandler(model LanguageModel) echo.HandlerFunc {
    return func(c echo.Context) error {
        var req ChatRequest
        if err := c.Bind(&req); err != nil {
            return err
        }
        
        result := Stream(c.Request.Context(), model, req.Message)
        
        return c.Stream(http.StatusOK, "text/event-stream", func(w io.Writer) {
            for part := range result.Stream() {
                data, _ := json.Marshal(part)
                fmt.Fprintf(w, "data: %s\n\n", data)
            }
        })
    }
}
```

---

## 13. Testing Utilities

### httptest for Mocking

```go
import "net/http/httptest"

func TestOpenAIGenerate(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        w.Write([]byte(`{
            "choices": [{
                "message": {"role": "assistant", "content": "Hello!"},
                "finish_reason": "stop"
            }]
        }`))
    }))
    defer server.Close()
    
    provider := openai.NewProvider(
        openai.WithBaseURL(server.URL),
        openai.WithAPIKey("test"),
    )
    
    result, err := aisdk.Generate(context.Background(), provider.Model("test"), "Say hello")
    require.NoError(t, err)
    assert.Equal(t, "Hello!", result.Text)
}
```

### Mock Interface Pattern

```go
// mock/model.go
package mock

// Ensure MockModel implements LanguageModel
var _ provider.LanguageModel = (*MockModel)(nil)

type MockModel struct {
    ProviderName  string
    ModelIDValue  string
    GenerateFunc  func(ctx context.Context, opts provider.GenerateOptions) (*provider.GenerateResult, error)
    StreamFunc    func(ctx context.Context, opts provider.GenerateOptions) (*provider.StreamResult, error)
}

func (m *MockModel) Provider() string { return m.ProviderName }
func (m *MockModel) ModelID() string  { return m.ModelIDValue }

func (m *MockModel) Generate(ctx context.Context, opts provider.GenerateOptions) (*provider.GenerateResult, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, opts)
    }
    return &provider.GenerateResult{
        Content:       []provider.ContentPart{{Type: "text", Text: "mock response"}},
        FinishReason:  "stop",
    }, nil
}

func (m *MockModel) Stream(ctx context.Context, opts provider.GenerateOptions) (*provider.StreamResult, error) {
    if m.StreamFunc != nil {
        return m.StreamFunc(ctx, opts)
    }
    // Default mock stream implementation
    ch := make(chan provider.StreamPart, 2)
    go func() {
        defer close(ch)
        ch <- provider.StreamPart{Type: "text", Content: "mock"}
        ch <- provider.StreamPart{Type: "finish"}
    }()
    return &provider.StreamResult{Stream: ch}, nil
}
```

---

## 14. Recommended Dependencies

### Core Dependencies

```go
// go.mod
module github.com/yourorg/lamar-sdk

go 1.22

require (
    // Validation
    github.com/go-playground/validator/v10 v10.19.0
    
    // JSON Schema generation
    github.com/invopop/jsonschema v0.12.0
    
    // Rate limiting
    golang.org/x/time v0.5.0
    
    // Retry logic
    github.com/sethvargo/go-retry v0.2.4
    
    // OpenTelemetry (optional)
    go.opentelemetry.io/otel v1.26.0
    
    // Environment variables (optional)
    github.com/kelseyhightower/envconfig v1.4.0
)
```

### Testing Dependencies

```go
// Testing
require (
    github.com/stretchr/testify v1.9.0
)
```

### Framework Integrations (Optional)

```go
// Frameworks (for integration packages)
require (
    github.com/gin-gonic/gin v1.9.1        // Gin integration
    github.com/labstack/echo/v4 v4.11.4    // Echo integration
    github.com/gofiber/fiber/v2 v2.52.1    // Fiber integration
)
```

---

## Summary: Recommended Libraries

| Purpose | Library | Notes |
|---------|---------|-------|
| HTTP Client | `net/http` | Built-in, no dependency |
| Validation | `go-playground/validator` | Struct tag based |
| JSON Schema | `invopop/jsonschema` | Auto-generation from structs |
| Rate Limiting | `golang.org/x/time/rate` | Official x package |
| Retry Logic | `sethvargo/go-retry` | Clean functional API |
| Logging | `log/slog` | Built-in (Go 1.21+) |
| CLI | `spf13/cobra` | Industry standard |
| Testing | `stretchr/testify` | Assertions and mocking |
| Tracing | `opentelemetry` | Optional, for observability |
| Env Config | `kelseyhightower/envconfig` | Optional alternative to os.Getenv |