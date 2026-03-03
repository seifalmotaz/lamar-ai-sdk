# Software Requirements Specification (SRS)
# Lamar AI SDK

## 1. Introduction

### 1.1 Purpose
This document specifies the technical requirements for Lamar SDK, a Go-based AI SDK for LLM integration.

### 1.2 Scope
Lamar SDK provides:
- Unified provider abstraction
- Text generation (streaming and non-streaming)
- Structured output generation
- Text embeddings
- Tool/function calling
- Middleware extensibility

### 1.3 Definitions

| Term | Definition |
|------|------------|
| Provider | An AI service implementation (OpenAI, Anthropic, etc.) |
| Model | A specific AI model instance |
| Generator | A model supporting non-streaming generation |
| Streamer | A model supporting streaming generation |
| Tool | A function callable by the AI model |
| Content | A polymorphic message part (text, image, audio, tool call) |

---

## 2. System Architecture

### 2.1 Package Structure

```
github.com/yourorg/lamar/
├── lamar.go                  # Re-exports, main API
├── provider/                 # Interfaces and types
│   ├── provider.go           # Model, Generator, Streamer interfaces
│   ├── types.go              # Message, Content, Usage types
│   ├── errors.go             # Error types with codes
│   └── capability.go         # Capability definitions
├── generate/                 # Generation API
│   ├── generate.go           # Generate(), GenerateObject[T]()
│   └── options.go            # Generation options
├── stream/                   # Streaming API
│   └── stream.go             # Stream(), StreamObject[T]()
├── embed/                    # Embedding API
│   └── embed.go              # Embed(), EmbedBatch()
├── tool/                     # Tool definitions
│   └── tool.go               # NewTool[In, Out]()
├── middleware/               # Middleware pattern
│   └── middleware.go         # Handler, Middleware interfaces
├── internal/                 # Hidden implementation
│   ├── sse/                  # Server-sent events
│   ├── httpx/                # HTTP utilities
│   └── schema/               # Schema generation
└── providers/                # Provider implementations
    └── openai/               # OpenAI provider
```

### 2.2 Component Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                      Application Layer                       │
│         lamar.Generate(), lamar.Stream(), lamar.Embed()     │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                      Core Layer                              │
│   generate/  │  stream/  │  embed/  │  tool/  │  middleware/ │
└─────────────────────────────┬───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                   Abstraction Layer                          │
│              provider/ (interfaces + types)                  │
└─────────────────────────────┬───────────────────────────────┘
                              │ implements
┌─────────────────────────────▼───────────────────────────────┐
│                   Provider Layer                             │
│          providers/openai, providers/anthropic, ...         │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. Interface Specifications

### 3.1 Core Interfaces

```go
type Model interface {
    Provider() string
    ModelID() string
}

type Generator interface {
    Model
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error)
}

type Streamer interface {
    Model
    Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)
}

type LanguageModel interface {
    Generator
    Streamer
}

type EmbeddingModel interface {
    Model
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error)
    MaxEmbeddingsPerCall() int
}
```

### 3.2 Content Types

```go
type Content interface {
    content()
}

type TextContent struct{ Text string }
type ImageContent struct{ Data []byte; MediaType string }
type AudioContent struct{ Data []byte; MediaType string }
type ToolCallContent struct{ ID, Name string; Input json.RawMessage }
type ToolResultContent struct{ ID, Name string; Result json.RawMessage; IsError bool }
type ReasoningContent struct{ Text string }
```

### 3.3 Error Codes

| Code | Constant | HTTP Status |
|------|----------|-------------|
| UNKNOWN | CodeUnknown | - |
| INVALID_REQUEST | CodeInvalidRequest | 400 |
| INVALID_MODEL | CodeInvalidModel | - |
| INVALID_PROMPT | CodeInvalidPrompt | - |
| AUTHENTICATION_FAILED | CodeAuthenticationFailed | 401 |
| RATE_LIMITED | CodeRateLimited | 429 |
| MODEL_NOT_FOUND | CodeModelNotFound | 404 |
| CONTENT_FILTERED | CodeContentFiltered | 400 |
| CONTEXT_CANCELED | CodeContextCanceled | - |
| API_TIMEOUT | CodeAPITimeout | 408 |
| PARSE_ERROR | CodeParseError | - |

---

## 4. Functional Specifications

### 4.1 Generate Function

**Signature:**
```go
func Generate(ctx context.Context, model Generator, prompt string, opts ...Option) (*Result, error)
```

**Preconditions:**
- `model` must not be nil
- `prompt` must not be empty
- Context must not be expired

**Postconditions:**
- Returns `*Result` with `Text`, `Content`, `ToolCalls`, `Usage`
- Returns `*Error` with structured error on failure
- Context cancellation returns `ErrContextCanceled`

**Options:**
| Option | Type | Default |
|--------|------|---------|
| System | string | "" |
| MaxTokens | int | provider default |
| Temperature | float64 | provider default |
| TopP | float64 | provider default |
| TopK | int | provider default |
| StopSequences | []string | nil |
| Tools | []Tool | nil |
| ToolChoice | ToolChoice | "auto" |
| Seed | *int | nil |

### 4.2 Stream Function

**Signature:**
```go
func Stream(ctx context.Context, model Streamer, prompt string, opts ...generate.Option) *stream.Result
```

**Behavior:**
- Returns immediately with `*Result`
- `Result.Stream()` returns `<-chan Part` for consuming stream
- `Result.Text()` blocks until completion and returns full text
- `Result.Usage()` blocks until completion and returns usage stats

**Stream Parts:**
| Type | Fields |
|------|--------|
| TextPart | Delta string |
| ToolCallPart | ToolCall ToolCall |
| FinishPart | FinishReason, Usage |
| ErrorPart | Error error |

**Thread Safety:**
- `Result` uses `atomic.Pointer[streamData]` for concurrent access
- Multiple goroutines may read from `Stream()` channel
- Exactly one goroutine writes to channel

### 4.3 GenerateObject Function

**Signature:**
```go
func GenerateObject[T any](ctx context.Context, model Generator, prompt string, opts ...Option) (*ObjectResult[T], error)
```

**Behavior:**
- Extracts JSON schema from `T` using struct tags
- Sets response format to JSON with schema
- Parses response into `T`
- Validates against schema

**Schema Extraction:**
- Uses `json` tags for field names
- Uses `jsonschema` tags for validation rules
- Supports nested structs, slices, maps
- Supports custom validation via go-playground/validator

### 4.4 Embed Function

**Signature:**
```go
func Embed(ctx context.Context, model EmbeddingModel, text string, opts ...Option) (*Result, error)
func EmbedBatch(ctx context.Context, model EmbeddingModel, texts []string, opts ...Option) (*BatchResult, error)
```

**Behavior:**
- Single embedding: direct API call
- Batch embedding: automatically chunks based on `MaxEmbeddingsPerCall()`
- Parallel execution for batches
- Returns aggregated usage

### 4.5 Tool Definition

**Signature:**
```go
func NewTool[In, Out any](name, description string, fn func(ctx context.Context, input In) (Out, error)) Tool
```

**Behavior:**
- Auto-generates input schema from `In` type
- Type-safe execution with automatic JSON marshaling
- Returns `Tool` interface for provider integration

---

## 5. Provider Implementation Requirements

### 5.1 OpenAI Provider

**Configuration:**
```go
type ProviderConfig struct {
    APIKey     string        // Required, falls back to OPENAI_API_KEY
    BaseURL    string        // Optional, defaults to https://api.openai.com/v1
    HTTPClient *http.Client  // Optional, defaults to http.DefaultClient
    OrgID      string        // Optional, for organization context
    Timeout    time.Duration // Optional, defaults to 30s
}
```

**Supported Models:**
| Model ID | Capabilities |
|----------|--------------|
| gpt-4 | text, tools |
| gpt-4-turbo | text, tools, vision |
| gpt-4o | text, tools, vision, audio |
| gpt-4o-mini | text, tools, vision |
| o1 | text, reasoning |
| o1-mini | text, reasoning |
| o1-pro | text, reasoning |
| text-embedding-3-small | embeddings |
| text-embedding-3-large | embeddings |
| text-embedding-ada-002 | embeddings |

**Provider-Specific Options:**
```go
type OpenAIConfig struct {
    LogitBias       map[int]float64
    ReasoningEffort string // "low", "medium", "high"
    User            string
    Seed            *int
}
```

### 5.2 Request/Response Mapping

**GenerateRequest → OpenAI ChatCompletionRequest:**
| Lamar Field | OpenAI Field |
|-------------|--------------|
| Prompt | messages[-1].content |
| System | messages[0].content with role "system" |
| MaxTokens | max_completion_tokens |
| Temperature | temperature |
| TopP | top_p |
| StopSequences | stop |
| Tools | tools array |
| ToolChoice | tool_choice |

**OpenAI ChatCompletionResponse → GenerateResult:**
| OpenAI Field | Lamar Field |
|--------------|-------------|
| choices[0].message.content | Text, Content[TextContent] |
| choices[0].message.tool_calls | ToolCalls |
| choices[0].finish_reason | FinishReason |
| usage.prompt_tokens | Usage.PromptTokens |
| usage.completion_tokens | Usage.CompletionTokens |

---

## 6. Non-Functional Specifications

### 6.1 Performance Requirements

| ID | Requirement | Measurement |
|----|-------------|-------------|
| NFR-P01 | API call overhead < 1ms | Benchmark |
| NFR-P02 | Memory per request < 1KB | pprof |
| NFR-P03 | Zero allocations in hot path | benchmark -benchmem |
| NFR-P04 | HTTP connection reuse | net/http tracing |
| NFR-P05 | Streaming first token < 50ms | Integration test |

### 6.2 Reliability Requirements

| ID | Requirement |
|----|-------------|
| NFR-R01 | All network operations respect context deadline |
| NFR-R02 | HTTP timeouts configurable per operation |
| NFR-R03 | Automatic retry on 5xx errors (max 3 attempts) |
| NFR-R04 | Rate limit handling with Retry-After header |
| NFR-R05 | Graceful stream termination on error |

### 6.3 Security Requirements

| ID | Requirement |
|----|-------------|
| NFR-S01 | API keys never logged |
| NFR-S02 | API keys redacted in error strings |
| NFR-S03 | Input validation before API calls |
| NFR-S04 | No sensitive data in debug output |

### 6.4 Compatibility Requirements

| ID | Requirement |
|----|-------------|
| NFR-C01 | Go 1.23+ support |
| NFR-C02 | Backward compatible API changes |
| NFR-C03 | Cross-platform (darwin, linux, windows) |
| NFR-C04 | No CGO dependencies |

---

## 7. Testing Requirements

### 7.1 Unit Tests

| Component | Coverage Target |
|-----------|-----------------|
| provider/ | 95% |
| generate/ | 90% |
| stream/ | 90% |
| embed/ | 90% |
| tool/ | 95% |
| internal/ | 85% |

### 7.2 Contract Tests

Every provider must pass the model contract:

```go
func ModelContract(t *testing.T, model provider.Generator) {
    t.Run("Generate_BasicPrompt", func(t *testing.T) { ... })
    t.Run("Generate_SystemPrompt", func(t *testing.T) { ... })
    t.Run("Generate_WithTools", func(t *testing.T) { ... })
    t.Run("Generate_ContextCancellation", func(t *testing.T) { ... })
    t.Run("Generate_InvalidPrompt", func(t *testing.T) { ... })
}
```

### 7.3 Integration Tests

| Test | Description |
|------|-------------|
| OpenAI Chat | Full chat completion cycle |
| OpenAI Streaming | SSE streaming with cancellation |
| OpenAI Embeddings | Single and batch embeddings |
| OpenAI Tools | Tool calling and execution |
| Error Handling | Rate limits, timeouts, auth errors |

---

## 8. Documentation Requirements

### 8.1 GoDoc

- All exported types must have documentation comments
- All exported functions must have documentation comments
- Examples for all major functions
- Package-level documentation

### 8.2 Examples

| Example | Location |
|---------|----------|
| Basic Generation | examples/openai/chat/ |
| Streaming | examples/openai/stream/ |
| Structured Output | examples/openai/structured/ |
| Tools | examples/openai/tools/ |
| Embeddings | examples/openai/embed/ |
| Middleware | examples/middleware/ |
| Error Handling | examples/errors/ |

---

## 9. API Stability Guarantees

### 9.1 Stability Tiers

| Tier | Guarantee | Examples |
|------|-----------|----------|
| Stable | No breaking changes | Generate, Stream, Embed |
| Beta | Deprecated before removal | Middleware interface |
| Alpha | May change | Additional model types |

### 9.2 Breaking Change Definition

A breaking change is:
1. Removing or renaming an exported type/function
2. Adding required parameters to a function
3. Changing a function's return type
4. Changing interface method signatures
5. Changing struct field types

### 9.3 Deprecation Policy

1. Mark as deprecated with comment
2. Add runtime warning (if feasible)
3. Wait 2 minor version cycles
4. Remove in next major version

---

## 10. Versioning

### 10.1 Semantic Versioning

- MAJOR: Breaking changes
- MINOR: New features (backward compatible)
- PATCH: Bug fixes

### 10.2 Go Module Versioning

- v0.x.x: Initial development (API not stable)
- v1.x.x: Stable API
- v2+: Major version in import path

---

## Appendix A: Error Handling Specification

```go
type Error struct {
    Code       ErrorCode    // Structured error code
    Message    string       // Human-readable message
    Cause      error        // Underlying error
    Provider   string       // Provider identifier
    ModelID    string       // Model identifier
    RetryAfter time.Duration // Retry delay (rate limits)
    StatusCode int          // HTTP status code
}

// Error matching
errors.Is(err, ErrRateLimited)           // true if rate limited
errors.As(err, &lamarError)              // extract structured error
lamar.IsRateLimited(err)                 // helper
lamar.ErrorCodeOf(err)                   // get error code
lamar.RetryAfter(err)                    // get retry duration
```

---

## Appendix B: Context Propagation

### Default Timeouts

| Operation | Default Timeout |
|-----------|-----------------|
| Generate | 30 seconds |
| Stream | 2 minutes |
| Embed | 10 seconds |
| EmbedBatch | 5 minutes |

### Context Values

| Key | Type | Purpose |
|-----|------|---------|
| request-id | string | Request tracing |
| trace-id | string | OpenTelemetry trace ID |
| user-id | string | User identification |

---

## Appendix C: Observability Interfaces

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}

type MetricsCollector interface {
    RecordRequest(ctx context.Context, provider, model string, duration time.Duration, err error)
    RecordTokens(ctx context.Context, provider, model string, prompt, completion int)
    RecordStreamStart(ctx context.Context, provider, model string)
    RecordStreamEvent(ctx context.Context, provider, model, eventType string)
    RecordStreamEnd(ctx context.Context, provider, model string, duration time.Duration)
}
```