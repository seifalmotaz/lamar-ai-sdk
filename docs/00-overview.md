# AI SDK Overview & Go SDK Plan

## What is the Vercel AI SDK?

The Vercel AI SDK is a TypeScript/JavaScript SDK for building AI-powered applications with Large Language Models (LLMs). It provides a **unified interface** for multiple AI providers.

- **Repository**: https://github.com/vercel/ai
- **Documentation**: https://ai-sdk.dev/docs
- **License**: Apache-2.0

---

## Repository Structure

This is a **monorepo** using pnpm workspaces and Turborepo.

### Key Directories

| Directory                 | Description                                                                          |
| ------------------------- | ------------------------------------------------------------------------------------ |
| `packages/ai`             | Main SDK package (`ai` on npm)                                                       |
| `packages/provider`       | Provider interface specifications (`@ai-sdk/provider`)                               |
| `packages/provider-utils` | Shared utilities for providers and core (`@ai-sdk/provider-utils`)                   |
| `packages/<provider>`     | AI provider implementations (openai, anthropic, google, azure, amazon-bedrock, etc.) |
| `packages/<framework>`    | UI framework integrations (react, vue, svelte, angular, rsc)                         |
| `packages/codemod`        | Automated migrations for major releases                                              |
| `examples/`               | Example applications (ai-functions, next-openai, etc.)                               |
| `content/`                | Documentation source files (MDX)                                                     |
| `contributing/`           | Contributor guides and documentation                                                 |
| `tools/`                  | Internal tooling (eslint-config, tsconfig)                                           |

---

## Core Package Dependencies

```
ai ─────────────────┬──▶ @ai-sdk/provider-utils ──▶ @ai-sdk/provider
                    │
@ai-sdk/<provider> ─┴──▶ @ai-sdk/provider-utils ──▶ @ai-sdk/provider
```

### Package Purposes

1. **`@ai-sdk/provider`** - Defines interface specifications (LanguageModelV3, EmbeddingModelV3, etc.)
2. **`@ai-sdk/provider-utils`** - Shared utilities for implementing providers
3. **`@ai-sdk/<provider>`** - Concrete implementations for each AI service
4. **`ai`** - High-level functions like `generateText`, `streamText`, `generateObject`

---

## Core APIs

| Function                   | Purpose                    | Package |
| -------------------------- | -------------------------- | ------- |
| `generateText`             | Generate text completion   | `ai`    |
| `streamText`               | Stream text completion     | `ai`    |
| `generateObject`           | Generate structured output | `ai`    |
| `streamObject`             | Stream structured output   | `ai`    |
| `embed` / `embedMany`      | Generate embeddings        | `ai`    |
| `generateImage`            | Generate images            | `ai`    |
| `generateSpeech`           | Generate speech/audio      | `ai`    |
| `generateVideo`            | Generate video             | `ai`    |
| `transcribe`               | Transcribe audio           | `ai`    |
| `tool`                     | Define a tool              | `ai`    |
| `jsonSchema` / `zodSchema` | Define schemas             | `ai`    |

---

## Model Types

| Model Type          | Interface              | AI Functions                                                   |
| ------------------- | ---------------------- | -------------------------------------------------------------- |
| Language Model      | `LanguageModelV3`      | `generateText`, `streamText`, `generateObject`, `streamObject` |
| Embedding Model     | `EmbeddingModelV3`     | `embed`, `embedMany`                                           |
| Image Model         | `ImageModelV3`         | `generateImage`                                                |
| Reranking Model     | `RerankingModelV3`     | `rerank`                                                       |
| Transcription Model | `TranscriptionModelV3` | `transcribe`                                                   |
| Speech Model        | `SpeechModelV3`        | `generateSpeech`                                               |
| Video Model         | `VideoModelV3`         | `generateVideo`                                                |

---

## Core Philosophies

### 1. Unified Provider Interface

- A single, consistent API across many AI providers
- Enables community providers to be developed independently in 3rd party packages

### 2. Separated Building Blocks

- Building blocks beyond the provider abstraction layer must be cleanly architected
- Critical for tree shaking and agentic development
- Enforcing architectural boundaries reduces complexity and side effects

### 3. Lean, Focused Mission

- Keep the AI SDK centered on: provider abstraction layer + directly associated building blocks
- Be conservative about adding entirely new building blocks
- Often the better solution is a separate project built on top of the AI SDK

### 4. API Stability

- Never change the signature of existing public functions
- The only exception is a new AI SDK major release
- Keep provider option schemas as restrictive as possible

### 5. Beware Premature Abstraction

- **Rule of 3**: Wait until at least 3 providers have implemented the same concept before generalizing
- When unsure or provider-specific, prefer `providerOptions`
- Resist pressure to abstract based on one provider

---

## Available Providers (40+)

| Provider       | Package Name             |
| -------------- | ------------------------ |
| OpenAI         | `@ai-sdk/openai`         |
| Anthropic      | `@ai-sdk/anthropic`      |
| Google         | `@ai-sdk/google`         |
| Google Vertex  | `@ai-sdk/google-vertex`  |
| Azure          | `@ai-sdk/azure`          |
| Amazon Bedrock | `@ai-sdk/amazon-bedrock` |
| Groq           | `@ai-sdk/groq`           |
| Mistral        | `@ai-sdk/mistral`        |
| Cohere         | `@ai-sdk/cohere`         |
| xAI            | `@ai-sdk/xai`            |
| DeepSeek       | `@ai-sdk/deepseek`       |
| Fireworks      | `@ai-sdk/fireworks`      |
| Together AI    | `@ai-sdk/togetherai`     |
| Perplexity     | `@ai-sdk/perplexity`     |
| Cerebras       | `@ai-sdk/cerebras`       |
| Replicate      | `@ai-sdk/replicate`      |
| ...            | ...                      |

---

## Go SDK Considerations

### Key Differences from TypeScript

| Aspect            | TypeScript                             | Go                                    |
| ----------------- | -------------------------------------- | ------------------------------------- |
| Generics          | Full support                           | Type parameters (Go 1.18+)            |
| Async/Streams     | Promise, AsyncIterable, ReadableStream | goroutines, channels, io.Reader       |
| Error Handling    | try/catch, Result types                | Multiple return values, errors        |
| Schema Validation | Zod, JSON Schema                       | go-playground/validator, gojsonschema |
| HTTP Client       | fetch API                              | net/http, resty                       |
| Testing           | Vitest                                 | testing package                       |

### Go Advantages

- Single binary deployment
- Better performance for CPU-bound tasks
- Strong typing with compile-time checks
- Simpler dependency management
- Native concurrency with goroutines

### TypeScript Advantages

- Dynamic typing flexibility
- Rich ecosystem (Zod, React hooks)
- Edge runtime support
- Easier JSON handling

---

## Documentation Structure

| File                            | Description                                |
| ------------------------------- | ------------------------------------------ |
| `00-overview.md`                | This file - overview and project structure |
| `01-architecture.md`            | Provider abstraction architecture          |
| `02-core-apis.md`               | Core APIs (generateText, streamText, etc.) |
| `03-provider-interface.md`      | Provider interfaces specification          |
| `04-provider-implementation.md` | How to implement a provider                |
| `05-testing.md`                 | Testing patterns                           |
| `06-framework-integrations.md`  | UI framework integrations                  |
| `07-phased-implementation.md`   | Phased implementation plan for Go          |
