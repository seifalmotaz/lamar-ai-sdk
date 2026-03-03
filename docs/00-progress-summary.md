# Lamar AI SDK - Progress Summary

## Goal

Build **Lamar AI SDK** - A Go-based AI SDK that provides a unified, type-safe interface for building AI-powered applications with Large Language Models (LLMs). It serves as the Go alternative to Vercel's TypeScript AI SDK.

## Architecture Decisions

### Interface Segregation

```
Model (base) → Generator (non-streaming) → LanguageModel (both)
            → Streamer (streaming)       →
```

Models implement ONLY what they support. Use `provider.CanGenerate(m)`, `provider.CanStream(m)` for capability checks.

### Thread-Safe Streaming

`stream.Result` uses `atomic.Pointer[streamData]` for concurrent access. Never access result fields directly—use `result.Text()`, `result.Usage()` methods which block until completion.

### Type-Safe Content

Content is an interface sum type: `TextContent`, `ImageContent`, `AudioContent`, `ToolCallContent`, `ToolResultContent`, `ReasoningContent`. Use type assertion to determine type.

### Structured Errors

All errors use `*Error` with `Code`, `Message`, `Provider`, `ModelID`, `RetryAfter`. Use `errors.As()` and helper functions like `IsRateLimited(err)`.

### Zero-Handwritten Schemas

Schema extraction via `jsonschema` struct tags. Never manually write JSON schemas.

## Key Patterns

### Functional Options

All public APIs use `opts ...Option` pattern. No config structs passed directly.

### Context-First

Every public function takes `context.Context` as first parameter.

### Validation on Entry

Fail fast: nil model → `ErrInvalidModel`, empty prompt → `ErrInvalidPrompt`.

### Middleware Handler Chain

```go
type Handler interface { Handle(ctx, req) (resp, error) }
type Middleware func(Handler) Handler
```

## Discoveries

- **Module path**: `github.com/seifalmotaz/lamar-sdk`
- **Go version**: 1.23+
- **Key dependencies**: `github.com/invopop/jsonschema` for schema extraction, `go.opentelemetry.io/otel` for tracing
- **Interface pattern**: `LanguageModel` interface extends both `Generator` and `Streamer` - use when a model supports both
- **Result types**: Generate returns `*Result` with methods `Text()`, `Usage()`, `FinishReason()` - NOT direct field access
- **Streaming**: Uses channel-based pattern with `<-chan StreamPart` and thread-safe accessor methods
- **Tool calling**: Uses `NewTool[In, Out]()` generic function with jsonschema tags for schema extraction
- **Batch embedding bug fix**: Fixed indexing bug for uneven batch sizes - use cumulative index tracking
- **Context cancellation**: Must check at function entry with `select { case <-ctx.Done(): ... default: }`
- **Default timeout**: 2 minutes for streaming operations

## Phase 1 Accomplishments (COMPLETE)

### Week 1-2: Core Interfaces and Types
- ✅ `provider/` package - interfaces, types, errors, capabilities
- ✅ `generate/` package - `Generate()`, `GenerateObject[T]()`, options, result
- ✅ `embed/` package - `Embed()`, `EmbedBatch()`, with batch handling
- ✅ `middleware/` package - Handler/Middleware chain, logging, metrics, recover, retry, tracing

### Week 3: Streaming & Tools
- ✅ `internal/schema/` - JSON schema extraction from Go structs
- ✅ `internal/sse/` - Server-Sent Events parser
- ✅ `stream/` package - `Stream()`, `StreamObject[T]()`
- ✅ `tool/` package - `NewTool[In, Out]()` with type-safe execution
- ✅ OpenAI streaming implementation with tool call aggregation

### Week 4: Polish & Testing
- ✅ `GenerateObject[T]()` and `StreamObject[T]()` for structured output
- ✅ Tracing middleware (OpenTelemetry)
- ✅ Retry middleware with exponential backoff
- ✅ Integration tests in `tests/integration/`
- ✅ Examples in `examples/openai/`
- ✅ README.md and AGENTS.md updated

## Bug Fixes Applied

- Fixed batch embedding indexing bug
- Added `BatchError` type for partial batch failures
- Added context cancellation checks at function entry
- Added default streaming timeout (2 minutes)
- Changed `GPT4o()` and `GPT4oMini()` to return `LanguageModel`
- Added `IsError` helper to `provider/errors.go`

## Project Structure

```
/Volumes/MacExtend/Peronal/lamar-ai-sdk/
├── AGENTS.md                          # Agent instructions - Phase 1 complete
├── README.md                          # Full documentation
├── go.mod                             # Module definition
├── go.sum                             # Dependency checksums
├── provider/
│   ├── provider.go                    # Core interfaces
│   ├── types.go                       # Content types, Message, Usage
│   ├── errors.go                      # Typed errors with codes
│   └── capability.go                  # Capability constants
├── generate/
│   ├── generate.go                    # Generate() function
│   ├── object.go                      # GenerateObject[T]() function
│   ├── options.go                     # Functional options
│   └── result.go                      # Result type with accessor methods
├── stream/
│   ├── stream.go                      # Stream() function
│   ├── object.go                      # StreamObject[T]() function
│   ├── result.go                      # Result type with thread-safe access
│   └── options.go                     # Functional options
├── embed/
│   ├── embed.go                       # Embed() and EmbedBatch()
│   ├── options.go
│   └── result.go
├── tool/
│   └── tool.go                        # NewTool[In, Out]()
├── middleware/
│   ├── middleware.go                  # Handler/Middleware interfaces
│   ├── logging.go
│   ├── metrics.go
│   ├── recover.go
│   ├── retry.go                       # Exponential backoff retry
│   └── tracing.go                     # OpenTelemetry tracing
├── internal/
│   ├── schema/schema.go               # JSON schema extraction
│   ├── sse/reader.go                  # SSE parser
│   ├── httpx/client.go                # HTTP client with streaming
│   └── contract/contract.go           # Contract test helpers
├── providers/openai/
│   ├── provider.go                    # NewProvider(), GPT4o()
│   ├── chat.go                        # Generate implementation
│   ├── chat_stream.go                 # Stream implementation
│   ├── embedding.go                   # Embed implementation
│   └── types.go                       # OpenAI API types
├── examples/
│   └── openai/
│       ├── chat/main.go
│       ├── stream/main.go
│       ├── embed/main.go
│       ├── tools/main.go
│       ├── structured/main.go
│       └── middleware/main.go
├── tests/integration/
│   └── openai_test.go                 # Integration tests
└── docs/
    ├── 10-final-architecture.md       # API design decisions
    ├── 11-api-stability.md            # API guarantees
    └── 12-middleware-pattern.md       # Middleware pattern
```

## Feature Matrix

| Feature | Package | Status |
|---------|---------|--------|
| Core interfaces | `provider/` | ✅ |
| Generate | `generate/` | ✅ |
| Stream | `stream/` | ✅ |
| Embed | `embed/` | ✅ |
| Tools | `tool/` | ✅ |
| Structured output | `generate/`, `stream/` | ✅ |
| Middleware | `middleware/` | ✅ |
| OpenAI provider | `providers/openai/` | ✅ |
| Schema extraction | `internal/schema/` | ✅ |
| SSE parsing | `internal/sse/` | ✅ |
| Examples | `examples/` | ✅ |
| Integration tests | `tests/integration/` | ✅ |

## Next Steps

Phase 1 is complete. Potential future work:

1. **Additional Providers**
   - Anthropic (Claude)
   - Google (Gemini)
   - Azure OpenAI
   - Mistral

2. **Enhanced Middleware**
   - Caching layer
   - Rate limiting
   - Request/response logging with redaction

3. **Advanced Features**
   - Multi-modal support (images, audio)
   - Function calling improvements
   - Conversation management
   - Prompt templates

4. **Developer Experience**
   - More comprehensive examples
   - Tutorial documentation
   - Error handling guides

5. **Performance**
   - Connection pooling
   - Request batching
   - Response caching