# Provider Interface Specification

The `@ai-sdk/provider` package defines interface specifications for AI models. This is the foundational layer of the SDK.

---

## LanguageModelV3

The core interface for text generation models.

### Interface Definition

```typescript
type LanguageModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;
  readonly supportedUrls:
    | PromiseLike<Record<string, RegExp[]>>
    | Record<string, RegExp[]>;

  doGenerate(
    options: LanguageModelV3CallOptions,
  ): PromiseLike<LanguageModelV3GenerateResult>;
  doStream(
    options: LanguageModelV3CallOptions,
  ): PromiseLike<LanguageModelV3StreamResult>;
};
```

### Call Options

```typescript
type LanguageModelV3CallOptions = {
  prompt: LanguageModelV3Prompt;

  // Generation settings
  maxOutputTokens?: number;
  temperature?: number;
  topP?: number;
  topK?: number;
  stopSequences?: string[];
  presencePenalty?: number;
  frequencyPenalty?: number;
  seed?: number;

  // Response format
  responseFormat?:
    | { type: "text" }
    | {
        type: "json";
        schema?: JSONSchema7;
        name?: string;
        description?: string;
      };

  // Tools
  tools?: Array<LanguageModelV3FunctionTool | LanguageModelV3ProviderTool>;
  toolChoice?: LanguageModelV3ToolChoice;

  // Execution
  abortSignal?: AbortSignal;
  headers?: Record<string, string | undefined>;

  // Provider-specific
  providerOptions?: SharedV3ProviderOptions;
  includeRawChunks?: boolean;
};
```

### Tool Types

```typescript
type LanguageModelV3FunctionTool = {
  type: "function";
  name: string;
  description: string;
  inputSchema: JSONSchema7;
  outputSchema?: JSONSchema7;
};

type LanguageModelV3ProviderTool = {
  type: "provider";
  id: string;
  name: string;
  args: Record<string, JSONValue>;
};

type LanguageModelV3ToolChoice =
  | { type: "auto" }
  | { type: "none" }
  | { type: "required" }
  | { type: "tool"; toolName: string };
```

### Prompt Structure

```typescript
type LanguageModelV3Prompt = Array<LanguageModelV3Message>;

type LanguageModelV3Message =
  | { role: "system"; content: string }
  | {
      role: "user";
      content: Array<LanguageModelV3TextPart | LanguageModelV3FilePart>;
    }
  | {
      role: "assistant";
      content: Array<
        | LanguageModelV3TextPart
        | LanguageModelV3FilePart
        | LanguageModelV3ReasoningPart
        | LanguageModelV3ToolCallPart
        | LanguageModelV3ToolResultPart
      >;
    }
  | {
      role: "tool";
      content: Array<
        LanguageModelV3ToolResultPart | LanguageModelV3ToolApprovalResponsePart
      >;
    };

type LanguageModelV3TextPart = {
  type: "text";
  text: string;
  providerMetadata?: SharedV3ProviderMetadata;
};

type LanguageModelV3FilePart = {
  type: "file";
  data: string | Uint8Array;
  mediaType: string;
  providerMetadata?: SharedV3ProviderMetadata;
};

type LanguageModelV3ReasoningPart = {
  type: "reasoning";
  text: string;
  providerMetadata?: SharedV3ProviderMetadata;
};

type LanguageModelV3ToolCallPart = {
  type: "tool-call";
  toolCallId: string;
  toolName: string;
  input: JSONValue;
  providerMetadata?: SharedV3ProviderMetadata;
};

type LanguageModelV3ToolResultPart = {
  type: "tool-result";
  toolCallId: string;
  toolName: string;
  result: JSONValue;
  isError?: boolean;
  providerMetadata?: SharedV3ProviderMetadata;
};
```

### Generate Result

```typescript
type LanguageModelV3GenerateResult = {
  content: Array<LanguageModelV3Content>;
  finishReason: LanguageModelV3FinishReason;
  usage: LanguageModelV3Usage;
  warnings: Array<LanguageModelV3Warning>;
  providerMetadata?: SharedV3ProviderMetadata;
  request?: { body?: unknown };
  response: {
    id?: string;
    timestamp: Date;
    modelId: string;
    headers?: Record<string, string>;
    body?: unknown;
  };
};

type LanguageModelV3Content =
  | LanguageModelV3TextPart
  | LanguageModelV3ReasoningPart
  | LanguageModelV3FilePart
  | LanguageModelV3ToolCallPart
  | LanguageModelV3ToolResultPart;

type LanguageModelV3FinishReason =
  | "stop"
  | "length"
  | "content-filter"
  | "tool-calls"
  | "error"
  | "other";

type LanguageModelV3Usage = {
  promptTokens: number;
  completionTokens: number;
  totalTokens: number;
};

type LanguageModelV3Warning = {
  type: "unsupported-setting" | "unsupported-content" | "other";
  setting?: string;
  details?: string;
  content?: LanguageModelV3Content;
};
```

### Stream Result

```typescript
type LanguageModelV3StreamResult = {
  stream: ReadableStream<LanguageModelV3StreamPart>;
  request?: { body?: unknown };
  response: { headers?: Record<string, string> };
};

type LanguageModelV3StreamPart =
  // Text
  | {
      type: "text-start";
      id: string;
      providerMetadata?: SharedV3ProviderMetadata;
    }
  | { type: "text-delta"; id: string; delta: string }
  | { type: "text-end"; id: string }

  // Reasoning
  | {
      type: "reasoning-start";
      id: string;
      providerMetadata?: SharedV3ProviderMetadata;
    }
  | { type: "reasoning-delta"; id: string; delta: string }
  | { type: "reasoning-end"; id: string }

  // Files
  | {
      type: "file";
      id: string;
      data: string;
      mediaType: string;
      providerMetadata?: SharedV3ProviderMetadata;
    }

  // Sources
  | {
      type: "source";
      id: number;
      sourceType: "url";
      url: string;
      title?: string;
      providerMetadata?: SharedV3ProviderMetadata;
    }

  // Tools
  | {
      type: "tool-input-start";
      id: string;
      toolName: string;
      providerMetadata?: SharedV3ProviderMetadata;
    }
  | { type: "tool-input-delta"; id: string; delta: string }
  | { type: "tool-input-end"; id: string }
  | {
      type: "tool-call";
      id: string;
      toolCallId: string;
      toolName: string;
      input: JSONValue;
      providerMetadata?: SharedV3ProviderMetadata;
    }
  | {
      type: "tool-result";
      id: string;
      toolCallId: string;
      toolName: string;
      result: JSONValue;
      isError?: boolean;
      providerMetadata?: SharedV3ProviderMetadata;
    }
  | {
      type: "tool-approval-request";
      toolCallId: string;
      toolName: string;
      args: JSONValue;
    }

  // Stream control
  | { type: "stream-start"; warnings: Array<LanguageModelV3Warning> }
  | {
      type: "response-metadata";
      id?: string;
      timestamp?: Date;
      modelId?: string;
      headers?: Record<string, string>;
    }
  | {
      type: "finish";
      finishReason: LanguageModelV3FinishReason;
      usage: LanguageModelV3Usage;
      providerMetadata?: SharedV3ProviderMetadata;
    }
  | { type: "raw"; rawValue: unknown }
  | { type: "error"; error: Error };
```

---

## EmbeddingModelV3

Interface for embedding models.

### Interface Definition

```typescript
type EmbeddingModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;
  readonly maxEmbeddingsPerCall:
    | PromiseLike<number | undefined>
    | number
    | undefined;
  readonly supportsParallelCalls: PromiseLike<boolean> | boolean;

  doEmbed(
    options: EmbeddingModelV3CallOptions,
  ): PromiseLike<EmbeddingModelV3Result>;
};
```

### Call Options

```typescript
type EmbeddingModelV3CallOptions = {
  values: Array<string>;
  abortSignal?: AbortSignal;
  headers?: Record<string, string | undefined>;
  providerOptions?: SharedV3ProviderOptions;
};
```

### Result

```typescript
type EmbeddingModelV3Result = {
  embeddings: Array<Array<number>>;
  usage: EmbeddingModelV3Usage;
  warnings?: Array<SharedV3Warning>;
  providerMetadata?: SharedV3ProviderMetadata;
  response?: { headers?: Record<string, string>; body?: unknown };
};

type EmbeddingModelV3Usage = {
  tokens: number;
};
```

---

## ImageModelV3

Interface for image generation models.

### Interface Definition

```typescript
type ImageModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;
  readonly maxImagesPerCall: number | undefined | GetMaxImagesPerCallFunction;

  doGenerate(options: ImageModelV3CallOptions): PromiseLike<ImageModelV3Result>;
};
```

### Call Options

```typescript
type ImageModelV3CallOptions = {
  prompt: string;
  n?: number;
  size?: `${number}x${number}`;
  aspectRatio?: ImageModelV3AspectRatio;
  seed?: number;
  abortSignal?: AbortSignal;
  headers?: Record<string, string | undefined>;
  providerOptions?: SharedV3ProviderOptions;
  providerMetadata?: ImageModelV3ProviderMetadata;
};
```

### Result

```typescript
type ImageModelV3Result = {
  images: Array<string | Uint8Array>; // base64 or binary
  warnings: Array<SharedV3Warning>;
  providerMetadata?: ImageModelV3ProviderMetadata;
  response: {
    timestamp: Date;
    modelId: string;
    headers: Record<string, string> | undefined;
  };
  usage?: ImageModelV3Usage;
};

type ImageModelV3Usage = {
  imagesCreated: number;
};
```

---

## SpeechModelV3

Interface for text-to-speech models.

### Interface Definition

```typescript
type SpeechModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;

  doGenerate(
    options: SpeechModelV3CallOptions,
  ): PromiseLike<SpeechModelV3Result>;
};
```

### Call Options

```typescript
type SpeechModelV3CallOptions = {
  text: string;
  voice?: string;
  outputFormat?: string;
  abortSignal?: AbortSignal;
  headers?: Record<string, string | undefined>;
  providerOptions?: SharedV3ProviderOptions;
};
```

### Result

```typescript
type SpeechModelV3Result = {
  audio: string | Uint8Array; // base64 or binary
  warnings: Array<SharedV3Warning>;
  request?: { body?: unknown };
  response: {
    timestamp: Date;
    modelId: string;
    headers?: SharedV2Headers;
    body?: unknown;
  };
  providerMetadata?: Record<string, JSONObject>;
};
```

---

## TranscriptionModelV3

Interface for audio transcription models.

### Interface Definition

```typescript
type TranscriptionModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;

  doGenerate(
    options: TranscriptionModelV3CallOptions,
  ): PromiseLike<TranscriptionModelV3Result>;
};
```

### Call Options

```typescript
type TranscriptionModelV3CallOptions = {
  audio: string | Uint8Array; // base64 or binary
  language?: string;
  abortSignal?: AbortSignal;
  headers?: Record<string, string | undefined>;
  providerOptions?: SharedV3ProviderOptions;
};
```

### Result

```typescript
type TranscriptionModelV3Result = {
  text: string;
  segments: Array<{
    text: string;
    startSecond: number;
    endSecond: number;
  }>;
  language: string | undefined;
  durationInSeconds: number | undefined;
  warnings: Array<SharedV3Warning>;
  response: {
    timestamp: Date;
    modelId: string;
    headers?: SharedV2Headers;
    body?: unknown;
  };
  providerMetadata?: SharedV3ProviderMetadata;
};
```

---

## VideoModelV3

Interface for video generation models.

### Interface Definition

```typescript
type VideoModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;
  readonly maxVideosPerCall: number | undefined | GetMaxVideosPerCallFunction;

  doGenerate(options: VideoModelV3CallOptions): PromiseLike<VideoModelV3Result>;
};
```

### Result

```typescript
type VideoModelV3Result = {
  videos: Array<{
    type: "url" | "base64";
    data: string;
  }>;
  warnings: Array<SharedV3Warning>;
  providerMetadata?: SharedV3ProviderMetadata;
  response: {
    timestamp: Date;
    modelId: string;
    headers: Record<string, string> | undefined;
  };
};
```

---

## RerankingModelV3

Interface for reranking models.

### Interface Definition

```typescript
type RerankingModelV3 = {
  readonly specificationVersion: "v3";
  readonly provider: string;
  readonly modelId: string;

  doRerank(
    options: RerankingModelV3CallOptions,
  ): PromiseLike<RerankingModelV3Result>;
};
```

### Call Options

```typescript
type RerankingModelV3CallOptions = {
  query: string;
  documents: Array<string>;
  topN?: number;
  abortSignal?: AbortSignal;
  headers?: Record<string, string | undefined>;
  providerOptions?: SharedV3ProviderOptions;
};
```

### Result

```typescript
type RerankingModelV3Result = {
  ranking: Array<{
    index: number;
    relevanceScore: number;
  }>;
  providerMetadata?: SharedV3ProviderMetadata;
  warnings?: Array<SharedV3Warning>;
  response?: {
    id?: string;
    timestamp?: Date;
    modelId?: string;
    headers?: SharedV3Headers;
    body?: unknown;
  };
};
```

---

## Go Interface Definitions

```go
// provider/provider.go
package provider

import "context"

// Model is the base interface for all AI models.
// All model types must embed this interface.
type Model interface {
    Provider() string
    ModelID() string
}

// Generator is a model that supports non-streaming generation.
// Not all models support generation - use type assertion to check.
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

// EmbeddingModel is the interface for embedding models.
type EmbeddingModel interface {
    Model
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error)
    MaxEmbeddingsPerCall() int
}

// ImageModel is the interface for image generation models.
type ImageModel interface {
    Model
    GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResult, error)
    MaxImagesPerCall() int
}

// SpeechModel is the interface for speech generation models.
type SpeechModel interface {
    Model
    GenerateSpeech(ctx context.Context, req *SpeechRequest) (*SpeechResult, error)
}

// TranscriptionModel is the interface for audio transcription models.
type TranscriptionModel interface {
    Model
    Transcribe(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResult, error)
}

// VideoModel is the interface for video generation models.
type VideoModel interface {
    Model
    GenerateVideo(ctx context.Context, req *VideoRequest) (*VideoResult, error)
    MaxVideosPerCall() int
}

// RerankingModel is the interface for reranking models.
type RerankingModel interface {
    Model
    Rerank(ctx context.Context, req *RerankingRequest) (*RerankingResult, error)
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

func CanEmbed(m Model) bool {
    _, ok := m.(EmbeddingModel)
    return ok
}
```

### Model Info and Capabilities

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

// HasCapability checks if a model supports a capability.
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

### Content Types (Type-Safe)

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
func ToolCall(id, name string, input json.RawMessage) ToolCallContent {
    return ToolCallContent{ID: id, Name: name, Input: input}
}
```

### Stream Result (Thread-Safe)

```go
// stream/result.go
package stream

import (
    "sync/atomic"
)

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
```

---

## Key Design Principles

1. **Interface Segregation** - Models implement only what they support (`Generator`, `Streamer`)
2. **Capability Discovery** - Use `HasCapability()` or type assertion to check features
3. **Type-Safe Content** - Use interface-based sum types, not string discrimination
4. **Thread-Safe Streaming** - Use `atomic.Pointer` for concurrent access safety
5. **Provider Metadata** - Custom data via `ProviderMetadata` on results
6. **Warnings** - All results include warnings for graceful degradation
7. **Functional Options** - Configure requests via `GenerateOption` functions
8. **Context-First** - All methods take `context.Context` as first parameter after receiver

---

## Implementation Verification

Every provider implementation must verify interface compliance at compile time:

```go
// providers/openai/chat.go
package openai

import "github.com/yourorg/lamar/provider"

// Compile-time interface verification
var (
    _ provider.Model         = (*ChatModel)(nil)
    _ provider.Generator     = (*ChatModel)(nil)
    _ provider.Streamer      = (*ChatModel)(nil)
    _ provider.LanguageModel = (*ChatModel)(nil)
)
```

For capabilities:

```go
// providers/openai/chat.go
var _ provider.ModelWithInfo = (*ChatModel)(nil)

func (m *ChatModel) Info() provider.ModelInfo {
    return provider.ModelInfo{
        Provider:     "openai",
        ModelID:      m.id,
        Capabilities: []provider.Capability{
            provider.CapStreaming,
            provider.CapTools,
            provider.CapVision,
            provider.CapJSON,
        },
        MaxTokens:   m.maxTokens,
        ContextSize: m.contextSize,
    }
}

func (m *ChatModel) HasCapability(cap provider.Capability) bool {
    for _, c := range m.Info().Capabilities {
        if c == cap {
            return true
        }
    }
    return false
}
```
