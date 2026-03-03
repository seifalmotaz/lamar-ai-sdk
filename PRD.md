# Product Requirements Document (PRD)
# Lamar AI SDK

## 1. Product Overview

### 1.1 Vision
Lamar SDK is a Go-based AI SDK that provides a unified, type-safe interface for building AI-powered applications with Large Language Models (LLMs). It serves as the Go alternative to Vercel's TypeScript AI SDK.

### 1.2 Problem Statement
- Go developers lack a unified, well-designed SDK for AI provider integration
- Existing solutions are provider-specific with inconsistent APIs
- No idiomatic Go patterns for streaming, tool calling, and structured outputs
- Manual schema generation and provider-specific code is error-prone

### 1.3 Target Users
- Backend Go developers building AI applications
- Teams migrating from TypeScript/Node.js to Go
- DevOps/Platform engineers deploying AI services
- Companies requiring type-safe, compiled AI integrations

---

## 2. Key Features

### 2.1 Core Capabilities (P0 - Must Have)

| Feature | Description | Priority |
|---------|-------------|----------|
| Text Generation | Non-streaming text completion | P0 |
| Streaming | Real-time streaming generation | P0 |
| Structured Output | JSON Schema validated responses | P0 |
| Embeddings | Single and batch text embeddings | P0 |
| Tool Calling | Function calling with type-safe tools | P0 |
| OpenAI Provider | Full OpenAI API support | P0 |

### 2.2 Extended Capabilities (P1 - Should Have)

| Feature | Description | Priority |
|---------|-------------|----------|
| Multiple Providers | Anthropic, Google, Azure support | P1 |
| Vision Support | Image understanding in messages | P1 |
| Audio Support | Audio input/output | P1 |
| Middleware | Request/response interceptors | P1 |
| Observability | Logging, metrics, tracing | P1 |

### 2.3 Future Capabilities (P2 - Nice to Have)

| Feature | Description | Priority |
|---------|-------------|----------|
| Image Generation | DALL-E, Stable Diffusion | P2 |
| Speech Synthesis | TTS capabilities | P2 |
| Video Generation | Video model support | P2 |
| Reranking | Search result reranking | P2 |
| Caching | Response caching layer | P2 |

---

## 3. Functional Requirements

### 3.1 API Design

| ID | Requirement |
|----|-------------|
| FR-001 | All public functions accept `context.Context` as first parameter |
| FR-002 | Functional options pattern for all configuration |
| FR-003 | Zero-value validation returns explicit errors |
| FR-004 | Streaming via channels with thread-safe result access |
| FR-005 | Structured errors with error codes and provider context |
| FR-006 | Interface segregation: `Model`, `Generator`, `Streamer`, `LanguageModel` |
| FR-007 | Type-safe tools via generics |
| FR-008 | Auto-generate JSON schemas from struct tags |

### 3.2 Provider Support

| ID | Requirement |
|----|-------------|
| FR-010 | OpenAI GPT-4, GPT-4o, GPT-3.5, O1 models |
| FR-011 | OpenAI text-embedding-3-small/large, ada-002 |
| FR-012 | OpenAI streaming with SSE parsing |
| FR-013 | OpenAI function calling with parallel calls |
| FR-014 | Provider-specific options (logit_bias, reasoning_effort) |

### 3.3 Streaming Requirements

| ID | Requirement |
|----|-------------|
| FR-020 | Channel-based streaming: `<-chan Part` |
| FR-021 | Thread-safe result accumulation via `atomic.Pointer` |
| FR-022 | Non-blocking stream consumption |
| FR-023 | Context cancellation support |
| FR-024 | Graceful stream termination on error |

---

## 4. Non-Functional Requirements

### 4.1 Performance

| ID | Requirement | Target |
|----|-------------|--------|
| NFR-001 | Memory allocation | < 1KB per request overhead |
| NFR-002 | Streaming latency | < 10ms to first token |
| NFR-003 | Goroutine safety | All public APIs thread-safe |
| NFR-004 | Connection pooling | Reuse HTTP connections |

### 4.2 Reliability

| ID | Requirement |
|----|-------------|
| NFR-010 | Graceful degradation on provider errors |
| NFR-011 | Automatic retry on transient failures |
| NFR-012 | Rate limit handling with `Retry-After` |
| NFR-013 | Timeout enforcement via context |

### 4.3 Usability

| ID | Requirement |
|----|-------------|
| NFR-020 | Clear, idiomatic Go API |
| NFR-021 | Comprehensive godoc documentation |
| NFR-022 | Working examples for all features |
| NFR-023 | Clear error messages with actionable guidance |

### 4.4 Maintainability

| ID | Requirement |
|----|-------------|
| NFR-030 | Semantic versioning |
| NFR-031 | No breaking changes in minor versions |
| NFR-032 | Deprecation policy (2 version cycles) |
| NFR-033 | Contract tests for provider implementations |

---

## 5. Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Test Coverage | > 90% | go tool cover |
| Provider Compatibility | 100% OpenAI API parity | Contract tests |
| Documentation Coverage | 100% exported symbols | godoc |
| User Adoption | 1000+ GitHub stars (Year 1) | GitHub |
| Issue Response | < 48h initial response | GitHub Issues |

---

## 6. Timeline

### Phase 1: Foundation (Weeks 1-2)
- Core interfaces and types
- Error handling
- Options pattern
- OpenAI embeddings

### Phase 2: Generation (Weeks 3-4)
- `Generate()` function
- Message types
- OpenAI chat completions
- Tool calling

### Phase 3: Streaming (Weeks 5-6)
- `Stream()` function
- SSE parsing
- Thread-safe result
- Stream cancellation

### Phase 4: Structured Output (Weeks 7-8)
- `GenerateObject[T]()`
- `StreamObject[T]()`
- Schema extraction
- Validation

### Phase 5: Polish (Weeks 9-10)
- Middleware pattern
- Observability interfaces
- Documentation
- Examples

### Phase 6: Extended Providers (Weeks 11-12)
- Anthropic provider
- Google provider
- Additional model types

---

## 7. Out of Scope

- Client-side JavaScript/React hooks
- Edge runtime optimization
- Server-side rendering utilities
- Image generation (Phase 1-4)
- Audio transcription (Phase 1-4)
- Video generation (Phase 1-4)

---

## 8. Dependencies

| Dependency | Purpose | License |
|------------|---------|---------|
| invopop/jsonschema | JSON Schema from structs | MIT |
| go-playground/validator | Input validation | MIT |
| golang.org/x/time | Rate limiting | BSD-3 |
| go.opentelemetry.io/otel | Tracing (optional) | Apache-2.0 |

---

## 9. Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Provider API changes | High | Contract tests, version pinning |
| Streaming complexity | Medium | Extensive testing, clear docs |
| Schema extraction edge cases | Medium | Comprehensive test suite |
| Generics learning curve | Low | Clear examples, documentation |