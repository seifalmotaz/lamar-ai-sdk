# Lamar AI SDK - Agent Instructions

# IMPORTANT

## When creating a plan

- ALWAYS ask questions if you are not sure about anything.
- ALWAYS ask for clarification if you are not sure about anything.
- ALWAYS ask for feedback if you are not sure about anything.
- ALWAYS ask for suggestions if you are not sure about anything.
- ALWAYS iterate at asking question until you have a clear understanding of what you need to do.

## Project Overview

Lamar SDK is a Go alternative to Vercel's TypeScript AI SDK. It provides a unified, type-safe interface for building AI applications with LLMs.

## Critical Architecture Decisions

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

## Package Layout

- `provider/` - interfaces and types (public)
- `generate/`, `stream/`, `embed/`, `tool/` - core APIs
- `internal/` - hidden implementation (sse, httpx, schema)
- `providers/openai/` - OpenAI implementation

## Must Remember

1. Never break backward compatibility (semver)
2. API keys never logged or in error strings
3. All exported symbols need godoc
4. Contract tests required for every provider
5. Go 1.23+ required

## Current Phase

**Phase 1 Complete** - All core functionality implemented with OpenAI provider.

### Phase 1 Accomplishments

**Weeks 1-2:** Core interfaces, types, errors, options pattern, middleware interface
- Provider package with interfaces (`Model`, `Generator`, `Streamer`, `LanguageModel`, `EmbeddingModel`)
- Typed errors with codes (`provider.Error`)
- Content types as interface sum type
- Generate and Embed packages with functional options
- Middleware handler chain pattern

**Weeks 3-4:** Streaming, tools, structured output, observability
- Stream package with `Stream()` and `StreamObject[T]()`
- SSE parser for streaming responses
- Tool package with `NewTool[In, Out]()`
- Schema extraction from Go structs
- GenerateObject[T]() and StreamObject[T]()
- OpenAI streaming implementation
- Tracing and retry middleware

### What's Implemented

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

## Key Files

- [docs/10-final-architecture.md](./docs/10-final-architecture.md) - Complete API design
- [docs/12-middleware-pattern.md](./docs/12-middleware-pattern.md) - Middleware pattern
- [docs/11-api-stability.md](./docs/11-api-stability.md) - API guarantees
- [PRD.md](./PRD.md) - Product requirements
- [SRS.md](./SRS.md) - Technical specifications
