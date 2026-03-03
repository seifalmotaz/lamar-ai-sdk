# Lamar SDK - Final Architecture Decisions

This document captures all architectural decisions for the Lamar Go AI SDK.

---

## Summary of Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Package name | `lamar` | Short and simple |
| Schema extraction | Struct tags + `invopop/jsonschema` | Auto-generate from Go types |
| Phase 1 providers | OpenAI only | Focus and perfect one provider |
| Core APIs location | Root package | `lamar.Generate()`, `lamar.Embed()` - cleanest |
| Streaming | Channel-based + atomic | `<-chan StreamPart` with thread-safe result |
| Zero value handling | Return error | Explicit validation |
| Structured output | Separate methods | `Generate()` vs `GenerateObject[T]()` |
| Tools API | Generics | `NewTool[In, Out]()` - type safe |
| Errors | Typed errors with codes | `Error{Code, Message, Provider, ModelID}` |
| Provider options | Typed config structs | Compile-time safety |
| Embed | Separate methods | `Embed()` for single, `EmbedBatch()` for multiple |
| Go version | 1.23+ | Latest stable with modern features |
| Internal utilities | Hidden in `internal/` | Clean public API |
| Examples | By provider then feature | `examples/openai/chat/` |
| Content types | Interface-based sum type | Type-safe polymorphism |
| Model interfaces | Segregated + composable | `Model`, `Generator`, `Streamer` |
| Middleware | Handler chain | Extensibility without inheritance |
| Observability | Interface-based | Compatible with slog, otel, prometheus |
| Testing | Contract test suite | Ensure provider compliance |

---

## Package Structure

```
github.com/yourorg/lamar/
├── lamar.go                  # Re-exports from subpackages
├── provider/
│   ├── provider.go           # Model, Generator, Streamer interfaces
│   ├── types.go              # Message, Content types, Usage, etc.
│   └── errors.go             # Error types with codes
├── generate/
│   ├── generate.go           # Generate(), GenerateObject[T]()
│   └── options.go            # GenerateOption, config
├── stream/
│   └── stream.go             # Stream(), StreamObject[T]()
├── embed/
│   └── embed.go              # Embed(), EmbedBatch()
├── tool/
│   └── tool.go               # NewTool[In, Out]()
├── middleware/
│   └── middleware.go         # Handler, Middleware interface
├── doc.go                    # Package documentation
│
├── internal/                 # Hidden implementation details
│   ├── sse/                  # Server-sent events parser
│   ├── httpx/                # HTTP utilities
│   ├── schema/               # JSON schema generation
│   └── contract/             # Contract testing helpers
│
├── providers/                # Provider implementations
│   └── openai/
│       ├── provider.go       # NewProvider(), Model()
│       ├── chat.go           # Chat model implementation
│       ├── embedding.go      # Embedding model implementation
│       ├── config.go         # OpenAIConfig struct
│       └── types.go          # OpenAI API types
│
├── examples/                 # Example applications
│   └── openai/
│       ├── chat/
│       │   └── main.go
│       ├── stream/
│       │   └── main.go
│       ├── embed/
│       │   └── main.go
│       └── tools/
│           └── main.go
│
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

---

## Core Interfaces

```go
// provider/provider.go
package provider

import "context"

// Model is the base interface for all AI models.
type Model interface {
    Provider() string
    ModelID() string
}

// Generator is a model that supports non-streaming generation.
type Generator interface {
    Model
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error)
}

// Streamer is a model that supports streaming generation.
// Not all models support streaming - use type assertion to check.
type Streamer interface {
    Model
    Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)
}

// LanguageModel is a full-featured model supporting both generation modes.
// Models may implement Generator, Streamer, or both.
type LanguageModel interface {
    Generator
    Streamer
}

// EmbeddingModel represents an embedding model.
type EmbeddingModel interface {
    Model
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error)
    MaxEmbeddingsPerCall() int
}

// Capability checking helpers
func CanGenerate(m Model) bool {
    _, ok := m.(Generator)
    return ok
}

func CanStream(m Model) bool {
    _, ok := m.(Streamer)
    return ok
}
```

---

## Model Capabilities

```go
// provider/capability.go
package provider

// Capability represents a model capability.
type Capability string

const (
    CapStreaming    Capability = "streaming"
    CapTools        Capability = "tools"
    CapVision       Capability = "vision"
    CapAudio        Capability = "audio"
    CapJSON         Capability = "json"
    CapReasoning    Capability = "reasoning"
)

// ModelInfo provides metadata about a model.
type ModelInfo struct {
    Provider     string
    ModelID      string
    Capabilities []Capability
    MaxTokens    int
    ContextSize  int
}

// ModelWithInfo is a model that can describe itself.
type ModelWithInfo interface {
    Model
    Info() ModelInfo
    HasCapability(cap Capability) bool
}

// HasCapability checks if a model supports a given capability.
func HasCapability(m Model, cap Capability) bool {
    if mi, ok := m.(ModelWithInfo); ok {
        return mi.HasCapability(cap)
    }
    // Fallback: infer from interface implementation
    switch cap {
    case CapStreaming:
        return CanStream(m)
    default:
        return false
    }
}
```

---

## Core Types

```go
// provider/types.go
package provider

import "encoding/json"

// Content represents a polymorphic content part in a message.
// Use type assertion to determine the specific type.
type Content interface {
    content()
}

// TextContent represents text content.
type TextContent struct {
    Text string
}

func (TextContent) content() {}

// ImageContent represents image content.
type ImageContent struct {
    Data      []byte
    MediaType string // "image/png", "image/jpeg", etc.
}

func (ImageContent) content() {}

// AudioContent represents audio content.
type AudioContent struct {
    Data      []byte
    MediaType string // "audio/mp3", "audio/wav", etc.
}

func (AudioContent) content() {}

// ToolCallContent represents a tool/function call.
type ToolCallContent struct {
    ID    string
    Name  string
    Input json.RawMessage // Raw JSON for lazy parsing
}

func (ToolCallContent) content() {}

// ToolResultContent represents the result of a tool call.
type ToolResultContent struct {
    ID      string
    Name    string
    Result  json.RawMessage
    IsError bool
}

func (ToolResultContent) content() {}

// ReasoningContent represents reasoning/thinking content.
type ReasoningContent struct {
    Text string
}

func (ReasoningContent) content() {}

// Content helpers for convenience
func Text(s string) TextContent                   { return TextContent{Text: s} }
func Image(data []byte, mediaType string) ImageContent {
    return ImageContent{Data: data, MediaType: mediaType}
}
func Audio(data []byte, mediaType string) AudioContent {
    return AudioContent{Data: data, MediaType: mediaType}
}
func ToolCall(id, name string, input json.RawMessage) ToolCallContent {
    return ToolCallContent{ID: id, Name: name, Input: input}
}
func ToolResult(id, name string, result json.RawMessage, isError bool) ToolResultContent {
    return ToolResultContent{ID: id, Name: name, Result: result, IsError: isError}
}

// Message represents a chat message.
type Message struct {
    Role    string    // "system", "user", "assistant", "tool"
    Content []Content // Polymorphic content parts
}

// Usage represents token usage.
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

// FinishReason indicates why generation stopped.
type FinishReason string

const (
    FinishReasonStop          FinishReason = "stop"
    FinishReasonLength        FinishReason = "length"
    FinishReasonToolCalls     FinishReason = "tool_calls"
    FinishReasonContentFilter FinishReason = "content_filter"
    FinishReasonError         FinishReason = "error"
)

// ToolChoice preferences
type ToolChoice struct {
    Type     string // "auto", "none", "required", "tool"
    ToolName string // Only when Type == "tool"
}
```

---

## Core Functions

### Generate

```go
// generate/generate.go
package generate

import "context"

// Generate generates text from a model.
func Generate(ctx context.Context, model Generator, prompt string, opts ...Option) (*Result, error) {
    if model == nil {
        return nil, ErrInvalidModel
    }
    if prompt == "" {
        return nil, ErrInvalidPrompt
    }
    
    cfg := defaultConfig()
    for _, opt := range opts {
        opt(cfg)
    }
    
    req := &GenerateRequest{
        Prompt:   prompt,
        Messages: cfg.Messages,
        System:   cfg.System,
        Config:   cfg,
    }
    
    return model.Generate(ctx, req)
}

// GenerateObject generates structured output matching a schema.
func GenerateObject[T any](ctx context.Context, model Generator, prompt string, opts ...Option) (*ObjectResult[T], error) {
    // Extract schema from T using struct tags
    // Implementation
}

// Result contains the result of a text generation.
type Result struct {
    Text         string
    Content      []Content
    ToolCalls    []ToolCall
    ToolResults  []ToolResult
    FinishReason FinishReason
    Usage        Usage
}

// ObjectResult contains the result of structured generation.
type ObjectResult[T any] struct {
    Object       T
    FinishReason FinishReason
    Usage        Usage
}
```

### Stream

```go
// stream/stream.go
package stream

import (
    "context"
    "sync/atomic"
)

// Stream streams text generation from a model.
func Stream(ctx context.Context, model Streamer, prompt string, opts ...generate.Option) *Result {
    result := &Result{
        stream: make(chan Part, 100),
        done:   make(chan struct{}),
    }
    
    go func() {
        defer close(result.done)
        defer close(result.stream)
        
        streamResult, err := model.Stream(ctx, &GenerateRequest{
            Prompt: prompt,
            // ...
        })
        if err != nil {
            result.result.Store(&streamData{err: err})
            result.stream <- Part{Type: PartError, Error: err}
            return
        }
        
        var data streamData
        var textBuilder strings.Builder
        
        for part := range streamResult.Stream {
            result.stream <- part
            
            if textPart, ok := part.(TextPart); ok {
                textBuilder.WriteString(textPart.Delta)
            }
            if finishPart, ok := part.(FinishPart); ok {
                data.text = textBuilder.String()
                data.usage = finishPart.Usage
                data.finishReason = finishPart.FinishReason
            }
        }
        
        result.result.Store(&data)
    }()
    
    return result
}

// Result contains a streaming text generation result.
// Thread-safe for concurrent access.
type Result struct {
    stream chan Part
    result atomic.Pointer[streamData]
    done   chan struct{}
}

type streamData struct {
    text         string
    usage        Usage
    finishReason FinishReason
    err          error
}

// Stream returns the stream channel.
func (r *Result) Stream() <-chan Part {
    return r.stream
}

// Text blocks until streaming completes and returns the full text.
func (r *Result) Text() (string, error) {
    <-r.done
    data := r.result.Load()
    if data == nil {
        return "", errors.New("stream not completed")
    }
    return data.text, data.err
}

// Usage blocks until streaming completes and returns usage stats.
func (r *Result) Usage() (Usage, error) {
    <-r.done
    data := r.result.Load()
    if data == nil {
        return Usage{}, errors.New("stream not completed")
    }
    return data.usage, data.err
}

// Part represents a stream part. Use type assertion.
type Part interface {
    part()
}

// TextPart is a text delta in the stream.
type TextPart struct {
    Delta string
}

func (TextPart) part() {}

// ToolCallPart is a tool call in the stream.
type ToolCallPart struct {
    ToolCall ToolCall
}

func (ToolCallPart) part() {}

// FinishPart signals stream completion.
type FinishPart struct {
    FinishReason FinishReason
    Usage        Usage
}

func (FinishPart) part() {}

// ErrorPart signals an error in the stream.
type ErrorPart struct {
    Error error
}

func (ErrorPart) part() {}

// Part type constants for type assertion
const (
    PartText     = "text"
    PartToolCall = "tool_call"
    PartFinish   = "finish"
    PartError    = "error"
)
```

### Embed

```go
// embed/embed.go
package embed

import "context"

// Embed generates an embedding for a single text.
func Embed(ctx context.Context, model EmbeddingModel, text string, opts ...Option) (*Result, error) {
    if model == nil {
        return nil, ErrInvalidModel
    }
    if text == "" {
        return nil, ErrInvalidInput
    }
    
    return model.Embed(ctx, &EmbedRequest{
        Texts: []string{text},
    })
}

// EmbedBatch generates embeddings for multiple texts.
// Automatically handles batching based on model.MaxEmbeddingsPerCall().
func EmbedBatch(ctx context.Context, model EmbeddingModel, texts []string, opts ...Option) (*BatchResult, error) {
    if model == nil {
        return nil, ErrInvalidModel
    }
    if len(texts) == 0 {
        return nil, ErrInvalidInput
    }
    
    maxPerCall := model.MaxEmbeddingsPerCall()
    if maxPerCall <= 0 || maxPerCall >= len(texts) {
        // Single call
        return model.Embed(ctx, &EmbedRequest{Texts: texts})
    }
    
    // Batch processing with parallel execution
    return processBatches(ctx, model, texts, maxPerCall)
}

// Result contains a single embedding result.
type Result struct {
    Embedding []float64
    Usage     Usage
}

// BatchResult contains batch embedding results.
type BatchResult struct {
    Embeddings [][]float64
    Usage      Usage
}
```

---

## Schema Auto-Extraction

```go
// internal/schema/schema.go
package schema

import (
    "github.com/invopop/jsonschema"
)

// FromStruct generates JSON Schema from a struct type.
func FromStruct[T any]() *jsonschema.Schema {
    var zero T
    return jsonschema.Reflect(&zero)
}

// FromStructWithExamples generates schema with examples.
func FromStructWithExamples[T any](examples ...T) *jsonschema.Schema {
    var zero T
    schema := jsonschema.Reflect(&zero)
    if len(examples) > 0 {
        schema.Examples = make([]any, len(examples))
        for i, ex := range examples {
            schema.Examples[i] = ex
        }
    }
    return schema
}
```

### Usage

```go
package main

import "github.com/yourorg/lamar"

// Define your struct with tags
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature" jsonschema:"description=Temperature in the requested unit"`
    Condition   string  `json:"condition" jsonschema:"description=Weather condition"`
}

// Tool automatically extracts schema from struct tags
weatherTool := lamar.NewTool[WeatherInput, WeatherOutput](
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
        return WeatherOutput{Temperature: 22.5, Condition: "sunny"}, nil
    },
)
```

---

## Tool Definition

```go
// tool/tool.go
package tool

import (
    "context"
    "encoding/json"
    
    "github.com/yourorg/lamar/internal/schema"
)

// Tool represents a callable tool.
type Tool interface {
    Name() string
    Description() string
    InputSchema() json.RawMessage
    Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

// typedTool is a type-safe tool implementation.
type typedTool[In, Out any] struct {
    name        string
    description string
    inputSchema json.RawMessage
    fn          func(ctx context.Context, input In) (Out, error)
}

// NewTool creates a type-safe tool from function and struct types.
func NewTool[In, Out any](name, description string, fn func(ctx context.Context, input In) (Out, error)) Tool {
    sch := schema.FromStruct[In]()
    schemaBytes, _ := json.Marshal(sch)
    
    return &typedTool[In, Out]{
        name:        name,
        description: description,
        inputSchema: json.RawMessage(schemaBytes),
        fn:          fn,
    }
}

func (t *typedTool[In, Out]) Name() string              { return t.name }
func (t *typedTool[In, Out]) Description() string       { return t.description }
func (t *typedTool[In, Out]) InputSchema() json.RawMessage { return t.inputSchema }

func (t *typedTool[In, Out]) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
    var in In
    if err := json.Unmarshal(input, &in); err != nil {
        return nil, &ParseError{Field: "input", Err: err}
    }
    
    out, err := t.fn(ctx, in)
    if err != nil {
        return nil, err
    }
    
    return json.Marshal(out)
}
```

---

## Errors

```go
// provider/errors.go
package provider

import (
    "errors"
    "fmt"
    "strings"
    "time"
)

// ErrorCode represents a structured error type.
type ErrorCode int

const (
    CodeUnknown ErrorCode = iota
    CodeInvalidRequest
    CodeInvalidModel
    CodeInvalidPrompt
    CodeInvalidInput
    CodeAuthenticationFailed
    CodeRateLimited
    CodeModelNotFound
    CodeContentFiltered
    CodeContextCanceled
    CodeAPITimeout
    CodeParseError
)

// String returns the error code as a string.
func (e ErrorCode) String() string {
    switch e {
    case CodeInvalidRequest:
        return "INVALID_REQUEST"
    case CodeInvalidModel:
        return "INVALID_MODEL"
    case CodeInvalidPrompt:
        return "INVALID_PROMPT"
    case CodeInvalidInput:
        return "INVALID_INPUT"
    case CodeAuthenticationFailed:
        return "AUTHENTICATION_FAILED"
    case CodeRateLimited:
        return "RATE_LIMITED"
    case CodeModelNotFound:
        return "MODEL_NOT_FOUND"
    case CodeContentFiltered:
        return "CONTENT_FILTERED"
    case CodeContextCanceled:
        return "CONTEXT_CANCELED"
    case CodeAPITimeout:
        return "API_TIMEOUT"
    case CodeParseError:
        return "PARSE_ERROR"
    default:
        return "UNKNOWN"
    }
}

// Error is the base structured error type.
type Error struct {
    Code       ErrorCode
    Message    string
    Cause      error
    Provider   string        // Which provider
    ModelID    string        // Which model
    RetryAfter time.Duration // For rate limiting
    StatusCode int           // HTTP status if applicable
}

func (e *Error) Error() string {
    var b strings.Builder
    b.WriteString(e.Code.String())
    b.WriteString(": ")
    b.WriteString(e.Message)
    
    if e.Provider != "" {
        b.WriteString(" (provider=")
        b.WriteString(e.Provider)
        if e.ModelID != "" {
            b.WriteString(", model=")
            b.WriteString(e.ModelID)
        }
        b.WriteString(")")
    }
    
    if e.Cause != nil {
        b.WriteString(": ")
        b.WriteString(e.Cause.Error())
    }
    
    return b.String()
}

func (e *Error) Unwrap() error { return e.Cause }

// NewError creates a new Error.
func NewError(code ErrorCode, message string, cause error) *Error {
    return &Error{
        Code:    code,
        Message: message,
        Cause:   cause,
    }
}

// Sentinel errors for quick comparison
var (
    ErrInvalidModel    = &Error{Code: CodeInvalidModel, Message: "model is nil"}
    ErrInvalidPrompt   = &Error{Code: CodeInvalidPrompt, Message: "prompt cannot be empty"}
    ErrInvalidInput    = &Error{Code: CodeInvalidInput, Message: "input cannot be empty"}
    ErrRateLimited     = &Error{Code: CodeRateLimited, Message: "rate limit exceeded"}
    ErrContextCanceled = &Error{Code: CodeContextCanceled, Message: "context canceled"}
)

// Helper functions
func IsRateLimited(err error) bool {
    var e *Error
    return errors.As(err, &e) && e.Code == CodeRateLimited
}

func RetryAfter(err error) time.Duration {
    var e *Error
    if errors.As(err, &e) {
        return e.RetryAfter
    }
    return 0
}

func ErrorCodeOf(err error) ErrorCode {
    var e *Error
    if errors.As(err, &e) {
        return e.Code
    }
    return CodeUnknown
}

func IsContextCanceled(err error) bool {
    if errors.Is(err, context.Canceled) {
        return true
    }
    return ErrorCodeOf(err) == CodeContextCanceled
}

// ParseError for JSON parsing failures.
type ParseError struct {
    Field string
    Err   error
}

func (e *ParseError) Error() string {
    if e.Field != "" {
        return fmt.Sprintf("parse error in %s: %v", e.Field, e.Err)
    }
    return fmt.Sprintf("parse error: %v", e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }
```

---

## Options (Functional Pattern)

```go
// generate/options.go
package generate

import "github.com/yourorg/lamar/tool"

// Option configures generation.
type Option func(*Config)

// Config holds generation configuration.
type Config struct {
    System        string
    MaxTokens     int
    Temperature   float64
    TopP          float64
    TopK          int
    StopSequences []string
    Tools         []tool.Tool
    ToolChoice    ToolChoice
    
    // Advanced
    Seed          *int
    ResponseFormat *ResponseFormat
    
    // Provider configs (type-safe)
    ProviderConfigs map[string]any
    
    // Observability
    Logger  Logger
    Metrics MetricsCollector
}

// defaultConfig returns the default configuration.
func defaultConfig() *Config {
    return &Config{
        ProviderConfigs: make(map[string]any),
    }
}

// System sets the system prompt.
func System(prompt string) Option {
    return func(c *Config) {
        c.System = prompt
    }
}

// MaxTokens sets the maximum tokens.
func MaxTokens(n int) Option {
    return func(c *Config) {
        c.MaxTokens = n
    }
}

// Temperature sets the temperature.
func Temperature(t float64) Option {
    return func(c *Config) {
        c.Temperature = t
    }
}

// TopP sets the top-p sampling.
func TopP(p float64) Option {
    return func(c *Config) {
        c.TopP = p
    }
}

// StopSequences sets stop sequences.
func StopSequences(seqs ...string) Option {
    return func(c *Config) {
        c.StopSequences = seqs
    }
}

// Tools sets the available tools.
func Tools(tools ...tool.Tool) Option {
    return func(c *Config) {
        c.Tools = append(c.Tools, tools...)
    }
}

// Seed sets a deterministic seed.
func Seed(seed int) Option {
    return func(c *Config) {
        c.Seed = &seed
    }
}

// WithLogger sets the logger.
func WithLogger(logger Logger) Option {
    return func(c *Config) {
        c.Logger = logger
    }
}

// WithMetrics sets the metrics collector.
func WithMetrics(metrics MetricsCollector) Option {
    return func(c *Config) {
        c.Metrics = metrics
    }
}
```

### Provider-Specific Configuration

```go
// providers/openai/config.go
package openai

// Config holds OpenAI-specific configuration.
type Config struct {
    LogitBias       map[int]float64
    ReasoningEffort string // "low", "medium", "high"
    User            string
    Seed            *int
}

// ApplyToRequest applies the config to a chat completion request.
func (c *Config) ApplyToRequest(req *ChatCompletionRequest) {
    if c.LogitBias != nil {
        req.LogitBias = c.LogitBias
    }
    if c.ReasoningEffort != "" {
        req.ReasoningEffort = c.ReasoningEffort
    }
    if c.User != "" {
        req.User = c.User
    }
    if c.Seed != nil {
        req.Seed = c.Seed
    }
}

// WithOpenAIConfig adds OpenAI-specific configuration.
func WithOpenAIConfig(cfg *Config) generate.Option {
    return func(c *generate.Config) {
        c.ProviderConfigs["openai"] = cfg
    }
}

// Convenience options
func LogitBias(bias map[int]float64) generate.Option {
    return WithOpenAIConfig(&Config{LogitBias: bias})
}

func ReasoningEffort(effort string) generate.Option {
    return WithOpenAIConfig(&Config{ReasoningEffort: effort})
}
```

---

## Context Propagation

### Default Timeouts

| Operation | Default Timeout | Rationale |
|-----------|-----------------|-----------|
| `Generate()` | 30 seconds | Standard API response time |
| `Stream()` | 2 minutes | Long-running streaming |
| `Embed()` | 10 seconds | Quick operation |
| `EmbedBatch()` | 5 minutes | Batch processing |

### Context Behavior

1. **Cancellation**: Context cancellation immediately closes all streams and returns `ErrContextCanceled`
2. **Inheritance**: Child requests inherit context values (request IDs, trace info)
3. **Deadline**: If no deadline set, default timeout is applied
4. **Propagation**: HTTP client respects context deadline

### Context Helpers

```go
// context.go
package lamar

import "context"

// Context keys for propagation.
type ctxKey string

const (
    RequestIDKey ctxKey = "request-id"
    TraceIDKey   ctxKey = "trace-id"
    UserIDKey    ctxKey = "user-id"
)

// WithRequestID adds a request ID to context.
func WithRequestID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, RequestIDKey, id)
}

// RequestID extracts request ID from context.
func RequestID(ctx context.Context) string {
    if v := ctx.Value(RequestIDKey); v != nil {
        return v.(string)
    }
    return ""
}

// WithTraceID adds a trace ID for observability.
func WithTraceID(ctx context.Context, id string) context.Context {
    return context.WithValue(ctx, TraceIDKey, id)
}

// TraceID extracts trace ID from context.
func TraceID(ctx context.Context) string {
    if v := ctx.Value(TraceIDKey); v != nil {
        return v.(string)
    }
    return ""
}
```

---

## Observability

### Metrics Interface

```go
// metrics.go
package lamar

import (
    "context"
    "time"
)

// MetricsCollector collects operational metrics.
type MetricsCollector interface {
    // Request metrics
    RecordRequest(ctx context.Context, provider, model string, duration time.Duration, err error)
    RecordTokens(ctx context.Context, provider, model string, prompt, completion int)
    
    // Stream metrics
    RecordStreamStart(ctx context.Context, provider, model string)
    RecordStreamEvent(ctx context.Context, provider, model, eventType string)
    RecordStreamEnd(ctx context.Context, provider, model string, duration time.Duration)
}

// NoopMetrics is the default no-op implementation.
type NoopMetrics struct{}

func (NoopMetrics) RecordRequest(context.Context, string, string, time.Duration, error) {}
func (NoopMetrics) RecordTokens(context.Context, string, string, int, int)              {}
func (NoopMetrics) RecordStreamStart(context.Context, string, string)                   {}
func (NoopMetrics) RecordStreamEvent(context.Context, string, string, string)           {}
func (NoopMetrics) RecordStreamEnd(context.Context, string, string, time.Duration)      {}
```

### Logger Interface

```go
// logger.go
package lamar

// Logger is a minimal logging interface compatible with slog, zap, logrus, etc.
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}

// SlogAdapter adapts slog.Logger to Logger interface.
type SlogAdapter struct {
    *slog.Logger
}

func NewSlogAdapter(l *slog.Logger) SlogAdapter {
    return SlogAdapter{Logger: l}
}

func (a SlogAdapter) Debug(msg string, args ...any) { a.Logger.Debug(msg, args...) }
func (a SlogAdapter) Info(msg string, args ...any)  { a.Logger.Info(msg, args...) }
func (a SlogAdapter) Warn(msg string, args ...any)  { a.Logger.Warn(msg, args...) }
func (a SlogAdapter) Error(msg string, args ...any) { a.Logger.Error(msg, args...) }
```

### Tracing (OpenTelemetry)

```go
// middleware/tracing.go
package middleware

import (
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
)

// TracingMiddleware creates OpenTelemetry spans for each request.
func TracingMiddleware(tp trace.TracerProvider) Middleware {
    tracer := tp.Tracer("github.com/yourorg/lamar")
    
    return func(next Handler) Handler {
        return HandlerFunc(func(ctx context.Context, req *Request) (*Response, error) {
            ctx, span := tracer.Start(ctx, "lamar.generate")
            defer span.End()
            
            span.SetAttributes(
                attribute.String("provider", req.Model.Provider()),
                attribute.String("model", req.Model.ModelID()),
            )
            
            resp, err := next.Handle(ctx, req)
            
            if err != nil {
                span.RecordError(err)
                span.SetAttributes(attribute.String("error", err.Error()))
            }
            
            if resp != nil {
                span.SetAttributes(
                    attribute.Int("prompt_tokens", resp.Usage.PromptTokens),
                    attribute.Int("completion_tokens", resp.Usage.CompletionTokens),
                    attribute.String("finish_reason", string(resp.FinishReason)),
                )
            }
            
            return resp, err
        })
    }
}
```

---

## OpenAI Provider Implementation

```go
// providers/openai/provider.go
package openai

import (
    "net/http"
    "os"
    
    "github.com/yourorg/lamar/provider"
)

// Provider is the OpenAI provider.
type Provider struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    orgID      string
}

// Option configures the provider.
type Option func(*Provider)

// APIKey sets the API key.
func APIKey(key string) Option {
    return func(p *Provider) {
        p.apiKey = key
    }
}

// BaseURL sets the base URL.
func BaseURL(url string) Option {
    return func(p *Provider) {
        p.baseURL = url
    }
}

// HTTPClient sets the HTTP client.
func HTTPClient(client *http.Client) Option {
    return func(p *Provider) {
        p.httpClient = client
    }
}

// OrgID sets the organization ID.
func OrgID(orgID string) Option {
    return func(p *Provider) {
        p.orgID = orgID
    }
}

// NewProvider creates a new OpenAI provider.
func NewProvider(opts ...Option) *Provider {
    p := &Provider{
        baseURL:    "https://api.openai.com/v1",
        httpClient: http.DefaultClient,
    }
    
    for _, opt := range opts {
        opt(p)
    }
    
    if p.apiKey == "" {
        p.apiKey = os.Getenv("OPENAI_API_KEY")
    }
    
    return p
}

// Model returns a chat model.
func (p *Provider) Model(id string) provider.Generator {
    return &chatModel{
        id:       id,
        provider: p,
    }
}

// StreamingModel returns a streaming-enabled model.
func (p *Provider) StreamingModel(id string) provider.LanguageModel {
    return &chatModel{
        id:       id,
        provider: p,
    }
}

// Embedding returns an embedding model.
func (p *Provider) Embedding(id string) provider.EmbeddingModel {
    return &embeddingModel{
        id:       id,
        provider: p,
    }
}

// Convenience functions
func (p *Provider) GPT4() provider.Generator       { return p.Model("gpt-4") }
func (p *Provider) GPT4o() provider.LanguageModel  { return p.StreamingModel("gpt-4o") }
func (p *Provider) GPT4oMini() provider.LanguageModel { return p.StreamingModel("gpt-4o-mini") }
func O1() provider.Generator                       { return NewProvider().Model("o1") }
func O1Mini() provider.Generator                   { return NewProvider().Model("o1-mini") }
```

---

## Usage Examples

### Basic Generation

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/providers/openai"
)

func main() {
    client := openai.NewProvider() // Uses OPENAI_API_KEY env var
    
    result, err := lamar.Generate(
        context.Background(),
        client.GPT4oMini(),
        "Say hello in 5 different languages",
        lamar.MaxTokens(100),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println(result.Text)
}
```

### Structured Output

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/providers/openai"
)

type Person struct {
    Name string `json:"name" jsonschema:"required,description=The person's name"`
    Age  int    `json:"age" jsonschema:"required,minimum=0,description=The person's age"`
}

func main() {
    client := openai.NewProvider()
    
    result, err := lamar.GenerateObject[Person](
        context.Background(),
        client.GPT4o(),
        "Generate a random fictional person",
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("%s is %d years old\n", result.Object.Name, result.Object.Age)
}
```

### Tools

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/providers/openai"
)

type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
}

func main() {
    client := openai.NewProvider()
    
    weatherTool := lamar.NewTool[WeatherInput, WeatherOutput](
        "get_weather",
        "Get current weather",
        func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
            return WeatherOutput{
                Temperature: 22.5,
                Condition:   "sunny",
            }, nil
        },
    )
    
    result, err := lamar.Generate(
        context.Background(),
        client.GPT4o(),
        "What's the weather in Tokyo?",
        lamar.Tools(weatherTool),
    )
    if err != nil {
        panic(err)
    }
    
    fmt.Println(result.Text)
    fmt.Printf("Tool calls: %d\n", len(result.ToolCalls))
}
```

### Streaming

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/provider"
    "github.com/yourorg/lamar/providers/openai"
)

func main() {
    client := openai.NewProvider()
    
    // Check if streaming is supported
    model := client.GPT4oMini()
    if !provider.CanStream(model) {
        panic("model doesn't support streaming")
    }
    
    result := lamar.Stream(
        context.Background(),
        model,
        "Write a short poem about Go programming",
    )
    
    // Stream parts as they arrive
    for part := range result.Stream() {
        switch p := part.(type) {
        case provider.TextPart:
            fmt.Print(p.Delta)
        case provider.FinishPart:
            fmt.Printf("\n\nFinished: %s\n", p.FinishReason)
        case provider.ErrorPart:
            fmt.Printf("\nError: %v\n", p.Error)
        }
    }
    
    // Or get the full text
    text, err := result.Text()
    if err != nil {
        panic(err)
    }
    fmt.Println("Full text:", text)
}
```

### Embeddings

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/providers/openai"
)

func main() {
    client := openai.NewProvider()
    
    // Single embedding
    result, err := lamar.Embed(
        context.Background(),
        client.Embedding("text-embedding-3-small"),
        "Hello, world!",
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("Embedding dimension: %d\n", len(result.Embedding))
    
    // Batch embeddings
    batch, err := lamar.EmbedBatch(
        context.Background(),
        client.Embedding("text-embedding-3-small"),
        []string{"Hello", "World", "Go"},
    )
    if err != nil {
        panic(err)
    }
    fmt.Printf("Batch size: %d\n", len(batch.Embeddings))
}
```

### Provider-Specific Options

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/providers/openai"
)

func main() {
    client := openai.NewProvider()
    
    result, err := lamar.Generate(
        context.Background(),
        client.GPT4o(),
        "Say hello",
        lamar.MaxTokens(50),
        lamar.Temperature(0.7),
        openai.LogitBias(map[int]float64{123: -100}),
        openai.ReasoningEffort("high"),
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Text)
}
```

### Capability Checking

```go
package main

import (
    "fmt"
    
    "github.com/yourorg/lamar/provider"
    "github.com/yourorg/lamar/providers/openai"
)

func main() {
    client := openai.NewProvider()
    
    model := client.GPT4oMini()
    
    // Check capabilities
    fmt.Printf("Can stream: %v\n", provider.CanStream(model))
    fmt.Printf("Can generate: %v\n", provider.CanGenerate(model))
    
    // Check model info if available
    if mi, ok := model.(provider.ModelWithInfo); ok {
        info := mi.Info()
        fmt.Printf("Provider: %s\n", info.Provider)
        fmt.Printf("Model: %s\n", info.ModelID)
        fmt.Printf("Max tokens: %d\n", info.MaxTokens)
        fmt.Printf("Has vision: %v\n", mi.HasCapability(provider.CapVision))
    }
}
```

---

## Dependencies

```go
// go.mod
module github.com/yourorg/lamar

go 1.23

require (
    github.com/invopop/jsonschema v0.12.0           // Schema generation from structs
    github.com/go-playground/validator/v10 v10.19.0 // Validation
    golang.org/x/time v0.5.0                        // Rate limiting
    go.opentelemetry.io/otel v1.26.0                // Tracing (optional)
)

// Dev dependencies
require (
    github.com/stretchr/testify v1.9.0
)
```

---

## Phase 1 Roadmap (4 Weeks)

| Week | Deliverables |
|------|--------------|
| Week 1 | Core interfaces, types, errors, options pattern, middleware interface |
| Week 2 | `Generate()`, `Embed()`, OpenAI chat model, contract tests |
| Week 3 | `Stream()`, `NewTool()`, tool execution, observability interfaces |
| Week 4 | `GenerateObject[T]()`, `StreamObject[T]()`, integration tests, docs |

---

## Key Design Principles

1. **Zero-handwritten schemas** - Extract from struct tags automatically
2. **Type-safe tools** - Generics for compile-time safety
3. **Clean imports** - `lamar.Generate()` not `lamar.aisdk.Generate()`
4. **Provider isolation** - Providers in `providers/` subfolder
5. **Hidden internals** - Implementation details in `internal/`
6. **Idiomatic Go** - Channels, functional options, typed errors
7. **Validation on entry** - Fail fast on nil model, empty prompt
8. **Context-first** - All public functions take `context.Context` first
9. **Interface segregation** - Models implement only what they support
10. **Thread-safe streaming** - Atomic operations for concurrent access
11. **Typed content** - Interface-based sum types for polymorphism
12. **Extensibility via middleware** - Handler chain pattern
13. **Observable by default** - Logger and metrics interfaces built-in
14. **Contract testing** - Verify provider implementations

---

## API Stability

See [API Stability](./11-api-stability.md) for versioning and deprecation policy.

---

## Middleware Pattern

See [Middleware Pattern](./12-middleware-pattern.md) for extensibility patterns.