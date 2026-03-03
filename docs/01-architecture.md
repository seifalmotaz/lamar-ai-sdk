# Provider Abstraction Architecture

The AI SDK follows a **layered provider architecture** using the **adapter pattern**. This is the central backbone of the project, enabling a single unified interface across multiple AI providers.

---

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Main Package (ai)                        │
│         generateText, streamText, generateObject, etc.      │
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

---

## Four-Layer Structure

### Layer 1: Specifications (`@ai-sdk/provider`)

**Purpose**: Defines interface specifications for AI models.

**Key Interfaces**:
- `LanguageModelV3` - Text generation models
- `EmbeddingModelV3` - Embedding models
- `ImageModelV3` - Image generation models
- `VideoModelV3` - Video generation models
- `SpeechModelV3` - Text-to-speech models
- `TranscriptionModelV3` - Audio transcription models
- `RerankingModelV3` - Reranking models

**Dependencies**: Only `json-schema@^0.4.0`

**Package Size**: Minimal, stable API surface

### Layer 2: Utilities (`@ai-sdk/provider-utils`)

**Purpose**: Shared utilities for implementing providers and core functionality.

**Key Utilities**:
- HTTP utilities: `postJsonToApi`, `getFromApi`, `createEventSourceResponseHandler`
- Schema utilities: `jsonSchema`, `zodSchema`, `asSchema`, `validateTypes`
- JSON parsing: `parseJSON`, `safeParseJSON`, `parseJsonEventStream`
- ID generation: `generateId`, `createIdGenerator`
- Headers: `combineHeaders`, `extractResponseHeaders`
- Misc: `loadApiKey`, `loadSetting`, `delay`, `downloadBlob`

**Dependencies**:
- `@ai-sdk/provider` (workspace)
- `@standard-schema/spec@^1.1.0`
- `eventsource-parser@^3.0.6`

### Layer 3: Provider Implementations (`@ai-sdk/<provider>`)

**Purpose**: Concrete implementations for each AI service.

**Structure** (OpenAI example):
```
packages/openai/
├── src/
│   ├── index.ts                    # Public exports
│   ├── openai-provider.ts          # Main provider factory
│   ├── openai-config.ts            # Configuration type
│   ├── chat/                       # Chat completion models
│   ├── embedding/                  # Embedding models
│   ├── image/                      # Image generation models
│   ├── speech/                     # Text-to-speech models
│   ├── transcription/              # Audio transcription models
│   └── tool/                       # Provider-specific tools
└── package.json
```

### Layer 4: Core (`ai`)

**Purpose**: High-level functions that developers use.

**Key Functions**:
- `generateText`, `streamText`
- `generateObject`, `streamObject`
- `embed`, `embedMany`
- `generateImage`, `generateSpeech`, `generateVideo`
- `transcribe`, `rerank`

**Dependencies**:
- `@ai-sdk/provider` (workspace)
- `@ai-sdk/provider-utils` (workspace)
- `@ai-sdk/gateway` (workspace)
- `@opentelemetry/api`

---

## Key Architectural Patterns

### 1. Versioned Interfaces

All model interfaces use `specificationVersion` for backward compatibility:

```typescript
type LanguageModelV3 = {
  readonly specificationVersion: 'v3';
  // ...
}

type LanguageModelV2 = {
  readonly specificationVersion: 'v2';
  // ...
}
```

This allows the SDK to support multiple interface versions simultaneously.

### 2. "do" Prefix Pattern

Method names use `do` prefix (`doGenerate`, `doStream`, `doEmbed`) to prevent direct user access:

```typescript
// Users call:
generateText({ model: openai('gpt-4'), prompt: '...' })

// Internally, the SDK calls:
model.doGenerate({ prompt, ... })
```

### 3. Provider Options Pattern

`providerOptions` allows provider-specific settings without changing core interfaces:

```typescript
// User can pass provider-specific options
generateText({
  model: openai('gpt-4'),
  prompt: '...',
  providerOptions: {
    openai: {
      logitBias: { '123': -100 },
      reasoningEffort: 'high',
    }
  }
})
```

### 4. Provider Metadata Pattern

`providerMetadata` enables provider-specific response data:

```typescript
interface LanguageModelV3GenerateResult {
  // Standard fields
  content: Array<LanguageModelV3Content>;
  finishReason: LanguageModelV3FinishReason;
  usage: LanguageModelV3Usage;
  warnings: Array<LanguageModelV3Warning>;
  
  // Provider-specific data
  providerMetadata?: Record<string, Record<string, JSONValue>>;
}
```

### 5. Warnings Pattern

All results include `warnings` array for unsupported features:

```typescript
const result = await generateText({
  model: openai('gpt-4'),
  prompt: '...',
});

console.log(result.warnings);
// [{ type: 'unsupported-setting', setting: 'topK', details: '...' }]
```

### 6. Streaming via ReadableStream

All streaming uses web standard `ReadableStream`:

```typescript
interface LanguageModelV3StreamResult {
  stream: ReadableStream<LanguageModelV3StreamPart>;
  request: { body?: unknown };
  response: { headers?: Record<string, string> };
}
```

### 7. Symbol-based Error Checking

`AISDKError` uses symbols for reliable `instanceof` checks across package versions:

```typescript
const name = 'AI_MyError';
const marker = `vercel.ai.error.${name}`;
const symbol = Symbol.for(marker);

export class MyError extends AISDKError {
  private readonly [symbol] = true;

  constructor({ message, cause }: { message: string; cause?: unknown }) {
    super({ name, message, cause });
  }

  static isInstance(error: unknown): error is MyError {
    return AISDKError.hasMarker(error, marker);
  }
}
```

---

## Adapter Pattern Implementation

The provider abstraction follows the adapter pattern:

```
AIFunction (generateText, streamText)
    │
    ├── uses LanguageModelV3 (interface)
    │
ProviderLanguageModelImplementation
    │
    ├── implements LanguageModelV3
    │
    └── Provider-specific classes:
        ├── OpenAIChatLanguageModel
        ├── AnthropicMessagesLanguageModel
        ├── GoogleGenerativeAILanguageModel
        └── ...
```

**Benefits**:
- Swapping providers without changing application code
- Community-developed third-party providers
- Consistent DX across different AI services
- Easy testing with mock implementations

---

## Go Translation Considerations

### Interface Design

```go
// Go equivalent of LanguageModelV3
type LanguageModelV3 interface {
    SpecificationVersion() string // "v3"
    Provider() string
    ModelID() string
    SupportedURLs() (map[string][]*regexp.Regexp, error)
    
    DoGenerate(ctx context.Context, opts LanguageModelV3CallOptions) (*LanguageModelV3GenerateResult, error)
    DoStream(ctx context.Context, opts LanguageModelV3CallOptions) (*LanguageModelV3StreamResult, error)
}
```

### Streaming in Go

```go
// Use channels instead of ReadableStream
type LanguageModelV3StreamResult struct {
    Stream <-chan LanguageModelV3StreamPart
    Request *RequestMetadata
    Response *ResponseMetadata
}

// Or use io.Reader with a custom decoder
type StreamReader interface {
    Read() (LanguageModelV3StreamPart, error)
}
```

### Error Handling

```go
// Go idiom: errors with type checking
type AISDKError struct {
    Name    string
    Message string
    Cause   error
}

func (e *AISDKError) Error() string {
    return e.Message
}

func IsAISDKError(err error) bool {
    var sdkErr *AISDKError
    return errors.As(err, &sdkErr)
}
```

### Provider Options

```go
// Use functional options or map[string]any
type GenerateTextOption func(*GenerateTextConfig)

func WithProviderOptions(opts map[string]any) GenerateTextOption {
    return func(c *GenerateTextConfig) {
        c.ProviderOptions = opts
    }
}

// Usage
result, err := GenerateText(ctx,
    openai.Model("gpt-4"),
    "Hello, world!",
    WithProviderOptions(map[string]any{
        "openai": map[string]any{
            "logitBias": map[int]float64{123: -100},
        },
    }),
)
```

---

## Key Takeaways

1. **The layered architecture (Specifications → Utilities → Providers → Core) is foundational**—never violate this boundary
2. **Versioned interfaces** allow backward compatibility and gradual migration
3. **Provider Options pattern** enables extensibility without breaking changes
4. **Warnings pattern** provides graceful degradation for unsupported features
5. **Symbol-based error checking** works across package versions
6. **Streaming abstraction** hides provider-specific implementations