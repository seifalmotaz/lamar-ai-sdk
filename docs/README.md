# Lamar AI SDK - Documentation Index

A comprehensive guide to building a Go SDK alternative to the Vercel AI SDK.

---

## Documentation Files

| File | Description |
|------|-------------|
| [00-overview.md](./00-overview.md) | Project overview, repository structure, core philosophies |
| [01-architecture.md](./01-architecture.md) | Provider abstraction architecture, layered design |
| [02-core-apis.md](./02-core-apis.md) | Core APIs: generateText, streamText, embed, etc. |
| [03-provider-interface.md](./03-provider-interface.md) | Provider interface specifications |
| [04-provider-implementation.md](./04-provider-implementation.md) | How to implement a provider (OpenAI example) |
| [05-testing.md](./05-testing.md) | Testing patterns and utilities |
| [06-framework-integrations.md](./06-framework-integrations.md) | UI framework integrations (React, Vue, Svelte, Angular) |
| [07-phased-implementation.md](./07-phased-implementation.md) | Phased implementation plan for Go SDK |
| [08-go-idiomatic.md](./08-go-idiomatic.md) | Go idiomatic adaptations and patterns |
| [09-go-ecosystem.md](./09-go-ecosystem.md) | Go ecosystem libraries and recommendations |
| [**10-final-architecture.md**](./10-final-architecture.md) | **Final architecture decisions and API design** |
| [**11-api-stability.md**](./11-api-stability.md) | **API stability guarantees and versioning policy** |
| [**12-middleware-pattern.md**](./12-middleware-pattern.md) | **Middleware pattern for extensibility** |

---

## Architecture Decision Records

Key architectural decisions captured in this documentation:

| Decision | Location |
|----------|----------|
| Interface segregation (`Model`, `Generator`, `Streamer`) | [10-final-architecture.md](./10-final-architecture.md#core-interfaces) |
| Thread-safe streaming with `atomic.Pointer` | [10-final-architecture.md](./10-final-architecture.md#stream) |
| Type-safe content via interface sum types | [10-final-architecture.md](./10-final-architecture.md#core-types) |
| Structured errors with error codes | [10-final-architecture.md](./10-final-architecture.md#errors) |
| Capability discovery system | [10-final-architecture.md](./10-final-architecture.md#model-capabilities) |
| Middleware pattern for extensibility | [12-middleware-pattern.md](./12-middleware-pattern.md) |
| API stability and versioning | [11-api-stability.md](./11-api-stability.md) |

---

## Quick Reference

### Architecture Layers

```
┌─────────────────────────────────────────────────────────────┐
│                    Main Package (ai)                        │
│         generateText, streamText, embed, etc.               │
└─────────────────────────┬───────────────────────────────────┘
                          │ uses
                          ▼
┌─────────────────────────────────────────────────────────────┐
│              Specifications (@ai-sdk/provider)               │
│         LanguageModelV3, EmbeddingModelV3, etc.              │
└─────────────────────────┬───────────────────────────────────┘
                          │ implements
                          ▼
┌─────────────────────────────────────────────────────────────┐
│         Shared Utilities (@ai-sdk/provider-utils)            │
└─────────────────────────┬───────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│         Provider Implementations (@ai-sdk/<provider>)        │
│      OpenAI, Anthropic, Google, Azure, Amazon Bedrock       │
└─────────────────────────────────────────────────────────────┘
```

### Core APIs

| Function | Purpose | Complexity |
|----------|---------|------------|
| `embed` | Generate single embedding | Low |
| `embedMany` | Generate multiple embeddings | Low-Medium |
| `generateText` | Non-streaming text generation | Medium |
| `generateObject` | Structured output | Medium |
| `streamText` | Streaming text generation | High |
| `streamObject` | Streaming structured output | High |

### Model Types

| Model Type | Interface | Functions |
|------------|-----------|-----------|
| Language | `LanguageModelV3` | generateText, streamText, generateObject, streamObject |
| Embedding | `EmbeddingModelV3` | embed, embedMany |
| Image | `ImageModelV3` | generateImage |
| Speech | `SpeechModelV3` | generateSpeech |
| Transcription | `TranscriptionModelV3` | transcribe |
| Video | `VideoModelV3` | generateVideo |
| Reranking | `RerankingModelV3` | rerank |

### Implementation Phases

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| Phase 1 | Week 1-2 | Provider interfaces, embed function, OpenAI embeddings |
| Phase 2 | Week 3-4 | generateText, tools, OpenAI chat completions |
| Phase 3 | Week 5-6 | streamText, streaming infrastructure |
| Phase 4 | Week 7-8 | generateObject, schema validation |
| Phase 5 | Week 9-10 | Anthropic & Google providers |
| Phase 6 | Week 11-12 | Image, audio, reranking |

---

## Key Insights for Go Implementation

### 1. Interface Design
- Use Go interfaces with type parameters for generics
- Version interfaces (`SpecificationVersion() string`)
- Use channels for streaming (`<-chan StreamPart`)

### 2. Error Handling
- Create custom error types with `error` interface
- Use `errors.As()` for type checking
- Wrap errors with context

### 3. Configuration
- Functional options pattern
- Environment variable fallbacks
- Provider-specific options via `map[string]interface{}`

### 4. Streaming
- Use channels for stream parts
- Context cancellation support
- Promise-like patterns for resolved values

### 5. Testing
- Mock interfaces for unit testing
- Test server for integration testing
- Table-driven tests for multiple cases

---

## Resources

- **Vercel AI SDK**: https://github.com/vercel/ai
- **Documentation**: https://ai-sdk.dev/docs
- **License**: Apache-2.0