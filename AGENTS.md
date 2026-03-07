# Lamar AI SDK - Agent Instructions

## IMPORTANT

### When creating a plan

- ALWAYS ask questions if you are not sure about anything.
- ALWAYS ask for clarification if you are not sure about anything.
- ALWAYS ask for feedback if you are not sure about anything.
- ALWAYS ask for suggestions if you are not sure about anything.
- ALWAYS iterate at asking question until you have a clear understanding of what you need to do.

---

## Project Overview

Lamar SDK is a Go alternative to Vercel's TypeScript AI SDK. It provides a unified, type-safe interface for building AI applications with LLMs.

**Full documentation is available in [README.md](./README.md)** - use it as the authoritative reference for:
- All API usage patterns
- Feature documentation
- Provider development guide
- Type system reference
- Best practices

---

## Critical Architecture Decisions

### Interface Segregation

```
Model (base)
    ├── Generator (non-streaming)
    │       └── LanguageModel (Generator + Streamer)
    ├── Streamer (streaming)
    │       └── LanguageModel (Generator + Streamer)
    ├── EmbeddingModel
    ├── ImageModel
    ├── TranscriptionModel
    └── SpeechModel
```

Models implement ONLY what they support. Use capability checks:

```go
provider.CanGenerate(m)      // Generator interface
provider.CanStream(m)        // Streamer interface
provider.CanEmbed(m)         // EmbeddingModel interface
provider.IsLanguageModel(m)  // LanguageModel interface
provider.CanGenerateImage(m) // ImageModel interface
provider.CanTranscribe(m)    // TranscriptionModel interface
provider.CanSynthesize(m)    // SpeechModel interface
provider.HasCapability(m, provider.CapVision) // Generic capability
```

### Thread-Safe Streaming

`stream.Result` uses channels and synchronization for concurrent access. Never access result fields directly—use methods:

```go
result := stream.Stream(ctx, model, "prompt")

// Consume stream
for part := range result.Stream() { }

// Blocking methods (wait for completion)
text, err := result.Text()
usage, _ := result.Usage()
finishReason, _ := result.FinishReason()
```

### Type-Safe Content

Content is an interface sum type. Use type assertion or type switch:

```go
type Content interface { content() }

// Implementations:
// - TextContent
// - ImageContent
// - AudioContent
// - ToolCallContent
// - ToolResultContent
// - ReasoningContent

// Type switch
switch c := content.(type) {
case provider.TextContent:
    fmt.Println(c.Text)
case provider.ImageContent:
    fmt.Printf("%d bytes, %s\n", len(c.Data), c.MediaType)
}
```

### Structured Errors

All errors use `*Error` with structured codes:

```go
type Error struct {
    Code       ErrorCode
    Message    string
    Cause      error
    Provider   string
    ModelID    string
    RetryAfter time.Duration
    StatusCode int
}

// Use helper functions:
provider.IsRateLimited(err)
provider.IsTimeout(err)
provider.IsContextCanceled(err)
provider.IsAuthenticationError(err)
provider.IsNotFoundError(err)
provider.IsInvalidInput(err)
provider.IsContentFiltered(err)
provider.RetryAfter(err)
provider.ErrorCodeOf(err)
```

### Zero-Handwritten Schemas

Schema extraction via `jsonschema` struct tags. Never manually write JSON schemas:

```go
type Person struct {
    Name string `json:"name" jsonschema:"required,description=Full name"`
    Age  int    `json:"age" jsonschema:"required,minimum=0,maximum=150"`
}

result, _ := generate.GenerateObject[Person](ctx, model, "prompt")
```

---

## Key Patterns

### Functional Options

All public APIs use `opts ...Option` pattern. No config structs passed directly:

```go
// Provider
client := openai.NewProvider(
    openai.APIKey("key"),
    openai.WithMiddleware(middleware.TimeoutWithDefault(30*time.Second)),
)

// Generate
result, _ := generate.Generate(ctx, model, "prompt",
    generate.MaxTokens(100),
    generate.Temperature(0.7),
)

// Embed
result, _ := embed.Embed(ctx, model, "text",
    embed.WithTimeout(30*time.Second),
)
```

### Context-First

Every public function takes `context.Context` as first parameter:

```go
generate.Generate(ctx, model, "prompt")
stream.Stream(ctx, model, "prompt")
embed.Embed(ctx, model, "text")
image.Generate(ctx, model, "prompt")
speech.Synthesize(ctx, model, "text")
transcription.Transcribe(ctx, model, audioData, "audio/mp3")
```

### Validation on Entry

Fail fast with sentinel errors:

- nil model → `ErrInvalidModel`
- empty prompt → `ErrInvalidPrompt`
- empty input → `ErrInvalidInput`
- nil data with media content → `ErrInvalidMediaType`

### Middleware Handler Chain

```go
type Handler interface {
    Handle(ctx context.Context, req Request) (Response, error)
}

type Middleware func(Handler) Handler

// Chain middleware
chain := middleware.Chain(
    middleware.Recover(),
    middleware.Logging(logger),
    middleware.TimeoutWithDefault(30*time.Second),
)
```

---

## Agent Framework

The `agent/` package provides multi-step LLM tool-calling loops. Use it when:

- You need the model to execute tools and continue the conversation
- You need automatic retries on transient errors
- You need stop conditions (max steps, specific tool calls, etc.)
- You need observability into the loop (callbacks, streaming events)

**When to use `generate` vs `agent`:**

| Use Case | Package | Reason |
|----------|---------|--------|
| Single text generation | `generate` | Simpler API, no loop needed |
| Single tool call | `generate` | Tool results handled manually |
| Multi-step tool loops | `agent` | Automatic tool execution |
| Need callbacks/events | `agent` | Built-in observability |
| Need stop conditions | `agent` | Customizable termination |

**See [README.md](./README.md#agent-framework) for full documentation.**

---

## Package Layout

```
lamar-sdk/
├── provider/              # Interfaces and types (PUBLIC API)
│   ├── provider.go        # Core interfaces (Model, Generator, Streamer, etc.)
│   ├── types.go           # Content types, Messages, Usage, etc.
│   ├── errors.go          # Structured errors with codes
│   └── capability.go      # Capability system
│
├── generate/              # Non-streaming generation API
├── stream/                # Streaming generation API
├── embed/                 # Text embeddings API
├── tool/                  # Type-safe tool definitions
├── middleware/            # Request/response middleware
├── image/                 # Image generation API
├── speech/                # Text-to-speech API
├── transcription/         # Audio transcription API
├── agent/                 # Multi-step tool-calling agent framework
│
├── internal/              # Private implementation (DO NOT IMPORT)
│   ├── schema/            # JSON schema extraction
│   ├── sse/               # Server-sent events parsing
│   ├── httpx/             # HTTP client utilities
│   └── contract/          # Contract test helpers
│
├── providers/             # Provider implementations
│   └── openai/            # OpenAI provider
│
├── examples/              # Example applications
│   └── openai/            # OpenAI examples
│
├── tests/                 # Integration tests
│   └── integration/
│
├── docs/                  # Architecture documentation
├── README.md              # Comprehensive documentation
├── PRD.md                 # Product requirements
├── SRS.md                 # Technical specifications
└── AGENTS.md              # This file
```

---

## Must Remember

1. **Never break backward compatibility (semver)** - Deprecated features must work for one minor version
2. **API keys never logged or in error strings** - Security requirement
3. **All exported symbols need godoc** - Documentation requirement
4. **Contract tests required for every provider** - Test requirement
5. **Go 1.23+ required** - Language version requirement
6. **No code comments** - Keep code clean, explain in godoc only

---

## Feature Status

| Feature | Package | Status |
|---------|---------|--------|
| Core interfaces | `provider/` | ✅ Complete |
| Generate | `generate/` | ✅ Complete |
| Stream | `stream/` | ✅ Complete |
| Embed | `embed/` | ✅ Complete |
| Tools | `tool/` | ✅ Complete |
| Structured output | `generate/`, `stream/` | ✅ Complete |
| Middleware | `middleware/` | ✅ Complete |
| Image generation | `image/` | ✅ Complete |
| Speech synthesis | `speech/` | ✅ Complete |
| Audio transcription | `transcription/` | ✅ Complete |
| Agent framework | `agent/` | ✅ Complete |
| OpenAI provider | `providers/openai/` | ✅ Complete |
| Schema extraction | `internal/schema/` | ✅ Complete |
| SSE parsing | `internal/sse/` | ✅ Complete |
| Examples | `examples/` | ✅ Complete |
| Integration tests | `tests/integration/` | ✅ Complete |
| Documentation | `README.md` | ✅ Complete |

---

## When Adding New Providers

Follow the pattern in `providers/openai/`. Required files:

1. `provider.go` - Provider struct, factory function, model factory methods
2. `config.go` - Provider-specific options (optional)
3. `types.go` - API request/response types
4. `chat.go` - ChatModel implementing Generator/LanguageModel
5. `chat_stream.go` - Streaming implementation (if supported)
6. `embedding.go` - EmbeddingModel (if supported)
7. `image.go` - ImageModel (if supported)
8. `speech.go` - SpeechModel (if supported)
9. `transcription.go` - TranscriptionModel (if supported)
10. `*_test.go` - Unit and integration tests

Implementation checklist:
- [ ] Implement required interfaces with compile-time verification
- [ ] Convert SDK types to/from API types
- [ ] Map errors to `provider.Error` with appropriate codes
- [ ] Support middleware wrapping (`wrapGenerate`, `wrapStream`, etc.)
- [ ] Use internal `httpx.Client` for HTTP operations
- [ ] Use internal `sse.Reader` for streaming
- [ ] Write unit tests with mock HTTP server
- [ ] Write integration tests

---

## Key Files for Reference

| File | Purpose |
|------|---------|
| [README.md](./README.md) | **Authoritative documentation** |
| [provider/provider.go](./provider/provider.go) | Core interfaces |
| [provider/types.go](./provider/types.go) | Content types, Messages |
| [provider/errors.go](./provider/errors.go) | Error codes and helpers |
| [provider/capability.go](./provider/capability.go) | Capability system |
| [providers/openai/provider.go](./providers/openai/provider.go) | Provider implementation example |
| [providers/openai/chat.go](./providers/openai/chat.go) | Chat model implementation |
| [docs/10-final-architecture.md](./docs/10-final-architecture.md) | Architecture decisions |
| [docs/12-middleware-pattern.md](./docs/12-middleware-pattern.md) | Middleware pattern |
| [PRD.md](./PRD.md) | Product requirements |
| [SRS.md](./SRS.md) | Technical specifications |