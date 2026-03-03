# Go SDK Implementation Plan

This document outlines a phased approach to building the Go equivalent of the Vercel AI SDK, starting from the simplest features and gradually increasing complexity.

---

## Project Structure

```
github.com/yourorg/lamar-sdk/
├── provider/              # Interface specifications (Layer 1)
│   ├── language_model.go
│   ├── embedding_model.go
│   ├── image_model.go
│   ├── speech_model.go
│   ├── transcription_model.go
│   ├── video_model.go
│   ├── reranking_model.go
│   ├── types.go
│   └── errors.go
├── providerutils/         # Shared utilities (Layer 2)
│   ├── api.go
│   ├── schema.go
│   ├── json.go
│   ├── headers.go
│   └── id.go
├── openai/                # OpenAI provider (Layer 3)
│   ├── provider.go
│   ├── chat.go
│   ├── embedding.go
│   ├── image.go
│   ├── audio.go
│   └── tools.go
├── anthropic/             # Anthropic provider
├── google/                # Google provider
├── aisdk/                 # Core functionality (Layer 4)
│   ├── generate.go
│   ├── stream.go
│   ├── embed.go
│   ├── object.go
│   ├── tools.go
│   └── registry.go
├── internal/              # Internal utilities
│   └── ...
├── examples/              # Example applications
│   ├── generate-text/
│   ├── stream-text/
│   └── embed/
├── go.mod
├── go.sum
└── README.md
```

---

## Phase 1: Foundation (Week 1-2)

**Goal**: Establish core abstractions and basic embedding functionality.

### Milestone 1.1: Provider Interfaces

**Files to create**:
- `provider/types.go` - Core types (Message, Content types, Usage, etc.)
- `provider/errors.go` - Error types with codes
- `provider/provider.go` - Model, Generator, Streamer interfaces
- `provider/capability.go` - Capability types

**Key Interfaces**:

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
type Streamer interface {
    Model
    Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)
}

// LanguageModel supports both generation modes.
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
```

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

// ToolCallContent represents a tool/function call.
type ToolCallContent struct {
    ID    string
    Name  string
    Input json.RawMessage
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

// Content helpers
func Text(s string) TextContent { return TextContent{Text: s} }
func Image(data []byte, mediaType string) ImageContent {
    return ImageContent{Data: data, MediaType: mediaType}
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
    FinishReasonOther         FinishReason = "other"
)
```

type LanguageModelUsage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

type EmbeddingModelUsage struct {
    Tokens int
}

type Warning struct {
    Type    string
    Setting string
    Details string
}
```

```go
// provider/errors.go
package provider

type AISDKError struct {
    Name    string
    Message string
    Cause   error
}

func (e *AISDKError) Error() string {
    if e.Cause != nil {
        return e.Message + ": " + e.Cause.Error()
    }
    return e.Message
}

func (e *AISDKError) Unwrap() error {
    return e.Cause
}

// Common errors
type APICallError struct {
    AISDKError
    StatusCode int
}

type InvalidPromptError struct {
    AISDKError
}

type NoContentGeneratedError struct {
    AISDKError
}
```

```go
// provider/embedding_model.go
package provider

type EmbeddingCallOptions struct {
    Values         []string
    AbortSignal    context.Context
    Headers        map[string]string
    ProviderOptions map[string]interface{}
}

type EmbeddingResult struct {
    Embeddings      [][]float64
    Usage           EmbeddingModelUsage
    Warnings        []Warning
    ProviderMetadata map[string]interface{}
    Response        *ResponseMetadata
}

type EmbeddingModel interface {
    SpecificationVersion() string
    Provider() string
    ModelID() string
    MaxEmbeddingsPerCall(ctx context.Context) (int, error)
    SupportsParallelCalls(ctx context.Context) (bool, error)
    
    DoEmbed(ctx context.Context, opts EmbeddingCallOptions) (*EmbeddingResult, error)
}
```

### Milestone 1.2: Provider Utilities

**Files to create**:
- `providerutils/api.go` - HTTP helpers
- `providerutils/json.go` - JSON parsing utilities
- `providerutils/headers.go` - Header utilities
- `providerutils/id.go` - ID generation

```go
// providerutils/api.go
package providerutils

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
)

type APIRequest struct {
    URL     string
    Headers map[string]string
    Body    interface{}
}

func PostJSON(ctx context.Context, client *http.Client, req APIRequest) ([]byte, *http.Response, error) {
    body, err := json.Marshal(req.Body)
    if err != nil {
        return nil, nil, fmt.Errorf("marshal request: %w", err)
    }
    
    httpReq, err := http.NewRequestWithContext(ctx, "POST", req.URL, bytes.NewReader(body))
    if err != nil {
        return nil, nil, fmt.Errorf("create request: %w", err)
    }
    
    for k, v := range req.Headers {
        httpReq.Header.Set(k, v)
    }
    httpReq.Header.Set("Content-Type", "application/json")
    
    resp, err := client.Do(httpReq)
    if err != nil {
        return nil, nil, fmt.Errorf("do request: %w", err)
    }
    defer resp.Body.Close()
    
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, nil, fmt.Errorf("read response: %w", err)
    }
    
    return respBody, resp, nil
}

func CombineHeaders(headers ...map[string]string) map[string]string {
    result := make(map[string]string)
    for _, h := range headers {
        for k, v := range h {
            result[k] = v
        }
    }
    return result
}
```

### Milestone 1.3: Basic Embed Function

**Files to create**:
- `aisdk/embed.go`

```go
// aisdk/embed.go
package aisdk

import (
    "context"
    
    "github.com/yourorg/lamar-sdk/provider"
)

type EmbedOptions struct {
    Model           provider.EmbeddingModel
    Value           string
    MaxRetries      int
    Headers         map[string]string
    ProviderOptions map[string]interface{}
}

type EmbedResult struct {
    Value           string
    Embedding       []float64
    Usage           provider.EmbeddingModelUsage
    Warnings        []provider.Warning
    ProviderMetadata map[string]interface{}
}

func Embed(ctx context.Context, opts EmbedOptions) (*EmbedResult, error) {
    if opts.MaxRetries == 0 {
        opts.MaxRetries = 2
    }
    
    result, err := opts.Model.DoEmbed(ctx, provider.EmbeddingCallOptions{
        Values:         []string{opts.Value},
        Headers:        opts.Headers,
        ProviderOptions: opts.ProviderOptions,
    })
    if err != nil {
        return nil, err
    }
    
    return &EmbedResult{
        Value:           opts.Value,
        Embedding:       result.Embeddings[0],
        Usage:           result.Usage,
        Warnings:        result.Warnings,
        ProviderMetadata: result.ProviderMetadata,
    }, nil
}
```

### Milestone 1.4: OpenAI Embedding Provider

**Files to create**:
- `openai/provider.go`
- `openai/embedding.go`

```go
// openai/embedding.go
package openai

import (
    "context"
    
    "github.com/yourorg/lamar-sdk/provider"
    "github.com/yourorg/lamar-sdk/providerutils"
)

type EmbeddingModel struct {
    modelID  string
    provider *Provider
}

var _ provider.EmbeddingModel = (*EmbeddingModel)(nil)

func (m *EmbeddingModel) SpecificationVersion() string { return "v3" }
func (m *EmbeddingModel) Provider() string            { return "openai" }
func (m *EmbeddingModel) ModelID() string             { return m.modelID }
func (m *EmbeddingModel) MaxEmbeddingsPerCall(ctx context.Context) (int, error) {
    return 2048, nil // OpenAI limit
}
func (m *EmbeddingModel) SupportsParallelCalls(ctx context.Context) (bool, error) {
    return true, nil
}

func (m *EmbeddingModel) DoEmbed(ctx context.Context, opts provider.EmbeddingCallOptions) (*provider.EmbeddingResult, error) {
    req := map[string]interface{}{
        "model": m.modelID,
        "input": opts.Values,
    }
    
    body, _, err := providerutils.PostJSON(ctx, m.provider.httpClient, providerutils.APIRequest{
        URL:     m.provider.baseURL + "/embeddings",
        Headers: providerutils.CombineHeaders(m.provider.headers(), opts.Headers),
        Body:    req,
    })
    if err != nil {
        return nil, err
    }
    
    // Parse response
    var resp EmbeddingResponse
    if err := json.Unmarshal(body, &resp); err != nil {
        return nil, fmt.Errorf("parse response: %w", err)
    }
    
    // Transform to SDK format
    embeddings := make([][]float64, len(resp.Data))
    for i, d := range resp.Data {
        embeddings[i] = d.Embedding
    }
    
    return &provider.EmbeddingResult{
        Embeddings: embeddings,
        Usage: provider.EmbeddingModelUsage{
            Tokens: resp.Usage.TotalTokens,
        },
    }, nil
}
```

---

## Phase 2: Text Generation (Week 3-4)

**Goal**: Implement `generateText` and a working chat completion provider.

### Milestone 2.1: LanguageModel Interface

**Files to create**:
- `provider/language_model.go`

```go
// provider/language_model.go
package provider

type LanguageModelCallOptions struct {
    Messages        []Message
    MaxOutputTokens *int
    Temperature     *float64
    TopP            *float64
    TopK            *int
    StopSequences   []string
    Seed            *int
    
    // Tools
    Tools       []Tool
    ToolChoice  ToolChoice
    
    // Response format
    ResponseFormat *ResponseFormat
    
    // Execution
    Headers         map[string]string
    ProviderOptions map[string]interface{}
}

type Tool struct {
    Type          string // "function"
    Name          string
    Description   string
    InputSchema   interface{} // JSON Schema
    OutputSchema  interface{}
}

type ToolChoice struct {
    Type     string // "auto", "none", "required", "tool"
    ToolName string
}

type ResponseFormat struct {
    Type        string // "text", "json"
    Schema      interface{}
    Name        string
    Description string
}

type GenerateResult struct {
    Content        []ContentPart
    FinishReason   FinishReason
    Usage          LanguageModelUsage
    Warnings       []Warning
    ProviderMetadata map[string]interface{}
    Response       *ResponseMetadata
}

type StreamResult struct {
    Stream   <-chan StreamPart
    Response *ResponseMetadata
}

type StreamPart struct {
    Type string // "text-start", "text-delta", "text-end", "finish", etc.
    ID   string
    
    // Text fields
    Delta string
    
    // Tool fields
    ToolCallID string
    ToolName   string
    Input      interface{}
    Result     interface{}
    
    // Finish fields
    FinishReason FinishReason
    Usage        LanguageModelUsage
}

type LanguageModel interface {
    SpecificationVersion() string
    Provider() string
    ModelID() string
    SupportedURLs() (map[string][]string, error)
    
    DoGenerate(ctx context.Context, opts LanguageModelCallOptions) (*GenerateResult, error)
    DoStream(ctx context.Context, opts LanguageModelCallOptions) (*StreamResult, error)
}
```

### Milestone 2.2: generateText Function

**Files to create**:
- `aisdk/generate.go`

```go
// aisdk/generate.go
package aisdk

type GenerateTextOptions struct {
    Model       provider.LanguageModel
    Prompt      string
    System      string
    Messages    []provider.Message
    MaxTokens   *int
    Temperature *float64
    MaxRetries  int
    
    Tools       map[string]Tool
    ToolChoice  provider.ToolChoice
    
    Headers         map[string]string
    ProviderOptions map[string]interface{}
}

type GenerateTextResult struct {
    Text          string
    Content       []provider.ContentPart
    ToolCalls     []ToolCall
    ToolResults   []ToolResult
    FinishReason  provider.FinishReason
    Usage         provider.LanguageModelUsage
    TotalUsage    provider.LanguageModelUsage
    Steps         []StepResult
}

func GenerateText(ctx context.Context, opts GenerateTextOptions) (*GenerateTextResult, error) {
    // 1. Standardize prompt
    messages := standardizePrompt(opts)
    
    // 2. Prepare tools
    tools := prepareTools(opts.Tools)
    
    // 3. Execute with retries
    var lastResult *GenerateTextResult
    var err error
    
    for retry := 0; retry <= opts.MaxRetries; retry++ {
        lastResult, err = executeGenerate(ctx, opts.Model, messages, tools, opts)
        if err == nil {
            break
        }
        if !isRetryable(err) {
            return nil, err
        }
    }
    
    return lastResult, err
}
```

### Milestone 2.3: OpenAI Chat Model

**Files to create**:
- `openai/chat.go`
- `openai/messages.go`

```go
// openai/chat.go
package openai

type ChatModel struct {
    modelID  string
    provider *Provider
    settings ChatSettings
}

func (m *ChatModel) DoGenerate(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.GenerateResult, error) {
    // Convert messages
    openaiMessages := convertToOpenAIMessages(opts.Messages)
    
    // Build request
    req := map[string]interface{}{
        "model":    m.modelID,
        "messages": openaiMessages,
    }
    
    if opts.MaxOutputTokens != nil {
        req["max_tokens"] = *opts.MaxOutputTokens
    }
    if opts.Temperature != nil {
        req["temperature"] = *opts.Temperature
    }
    if len(opts.Tools) > 0 {
        req["tools"] = convertTools(opts.Tools)
    }
    if opts.ResponseFormat != nil {
        req["response_format"] = map[string]string{"type": "json_object"}
    }
    
    // Make API call
    body, _, err := providerutils.PostJSON(ctx, m.provider.httpClient, providerutils.APIRequest{
        URL:     m.provider.baseURL + "/chat/completions",
        Headers: providerutils.CombineHeaders(m.provider.headers(), opts.Headers),
        Body:    req,
    })
    if err != nil {
        return nil, err
    }
    
    // Parse and transform response
    return parseChatResponse(body)
}
```

### Milestone 2.4: Tool Support

**Files to create**:
- `aisdk/tools.go`

```go
// aisdk/tools.go
package aisdk

type Tool interface {
    ID() string
    Description() string
    InputSchema() interface{}
    OutputSchema() interface{}
    Execute(ctx context.Context, input interface{}) (interface{}, error)
}

type ToolDefinition[I, O any] struct {
    ID            string
    Description   string
    InputSchema   interface{}
    OutputSchema  interface{}
    ExecuteFunc   func(ctx context.Context, input I) (O, error)
}

func (t *ToolDefinition[I, O]) Execute(ctx context.Context, input interface{}) (interface{}, error) {
    typedInput, ok := input.(I)
    if !ok {
        return nil, fmt.Errorf("invalid input type")
    }
    return t.ExecuteFunc(ctx, typedInput)
}
```

---

## Phase 3: Streaming (Week 5-6)

**Goal**: Implement `streamText` with proper streaming support.

### Milestone 3.1: Stream Infrastructure

**Files to create**:
- `providerutils/stream.go`

```go
// providerutils/stream.go
package providerutils

import (
    "bufio"
    "bytes"
    "encoding/json"
    "io"
)

type SSEParser struct {
    reader *bufio.Reader
}

func NewSSEParser(r io.Reader) *SSEParser {
    return &SSEParser{reader: bufio.NewReader(r)}
}

func (p *SSEParser) Next() (string, []byte, error) {
    var eventType string
    var data bytes.Buffer
    
    for {
        line, err := p.reader.ReadString('\n')
        if err != nil {
            return "", nil, err
        }
        
        line = strings.TrimSuffix(line, "\n")
        line = strings.TrimSuffix(line, "\r")
        
        if line == "" {
            if data.Len() > 0 {
                return eventType, data.Bytes(), nil
            }
            continue
        }
        
        if strings.HasPrefix(line, "event:") {
            eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
        } else if strings.HasPrefix(line, "data:") {
            data.WriteString(strings.TrimPrefix(line, "data:"))
        }
    }
}

func ParseSSEStream(r io.Reader) <-chan json.RawMessage {
    ch := make(chan json.RawMessage, 100)
    
    go func() {
        defer close(ch)
        parser := NewSSEParser(r)
        
        for {
            _, data, err := parser.Next()
            if err != nil {
                return
            }
            if string(data) == "[DONE]" {
                return
            }
            ch <- data
        }
    }()
    
    return ch
}
```

### Milestone 3.2: streamText Function

**Files to create**:
- `aisdk/stream.go`

```go
// aisdk/stream.go
package aisdk

type StreamTextOptions struct {
    // Same as GenerateTextOptions
}

type StreamTextResult struct {
    fullStream <-chan provider.StreamPart
    textStream <-chan string
    
    // Promise-like fields (resolved when stream completes)
    textPromise       *promise.Promise[string]
    usagePromise      *promise.Promise[provider.LanguageModelUsage]
    finishPromise     *promise.Promise[provider.FinishReason]
}

func (r *StreamTextResult) FullStream() <-chan provider.StreamPart {
    return r.fullStream
}

func (r *StreamTextResult) TextStream() <-chan string {
    return r.textStream
}

func (r *StreamTextResult) Text() (string, error) {
    return r.textPromise.Get()
}

func (r *StreamTextResult) Usage() (provider.LanguageModelUsage, error) {
    return r.usagePromise.Get()
}

func StreamText(ctx context.Context, opts StreamTextOptions) *StreamTextResult {
    fullStream := make(chan provider.StreamPart, 100)
    textStream := make(chan string, 100)
    
    result := &StreamTextResult{
        fullStream:   fullStream,
        textStream:   textStream,
        textPromise:  promise.New[string](),
        usagePromise: promise.New[provider.LanguageModelUsage](),
        finishPromise: promise.New[provider.FinishReason](),
    }
    
    go func() {
        defer close(fullStream)
        defer close(textStream)
        
        // Execute stream
        streamResult, err := opts.Model.DoStream(ctx, prepareOptions(opts))
        if err != nil {
            fullStream <- provider.StreamPart{Type: "error", Error: err}
            return
        }
        
        var textBuilder strings.Builder
        var usage provider.LanguageModelUsage
        var finishReason provider.FinishReason
        
        for part := range streamResult.Stream {
            fullStream <- part
            
            if part.Type == "text-delta" {
                textBuilder.WriteString(part.Delta)
                textStream <- part.Delta
            }
            
            if part.Type == "finish" {
                usage = part.Usage
                finishReason = part.FinishReason
            }
        }
        
        result.textPromise.Resolve(textBuilder.String())
        result.usagePromise.Resolve(usage)
        result.finishPromise.Resolve(finishReason)
    }()
    
    return result
}
```

### Milestone 3.3: OpenAI Streaming

```go
// openai/chat.go
func (m *ChatModel) DoStream(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.StreamResult, error) {
    stream := make(chan provider.StreamPart, 100)
    
    // Build streaming request
    req := map[string]interface{}{
        "model":    m.modelID,
        "messages": convertToOpenAIMessages(opts.Messages),
        "stream":   true,
        "stream_options": map[string]bool{
            "include_usage": true,
        },
    }
    
    // Make streaming request
    resp, err := m.doStreamRequest(ctx, req, opts.Headers)
    if err != nil {
        close(stream)
        return nil, err
    }
    
    go func() {
        defer close(stream)
        
        stream <- provider.StreamPart{Type: "stream-start"}
        
        eventStream := providerutils.ParseSSEStream(resp.Body)
        for data := range eventStream {
            chunk := parseOpenAIChunk(data)
            parts := m.transformChunkToParts(chunk)
            for _, part := range parts {
                stream <- part
            }
        }
    }()
    
    return &provider.StreamResult{Stream: stream}, nil
}
```

---

## Phase 4: Object Generation (Week 7-8)

**Goal**: Support structured output with schema validation.

### Milestone 4.1: Schema Utilities

**Files to create**:
- `providerutils/schema.go`

```go
// providerutils/schema.go
package providerutils

import (
    "encoding/json"
    "reflect"
)

type Schema interface {
    JSONSchema() map[string]interface{}
    Validate(value interface{}) error
}

// JSONSchema wraps a raw JSON Schema
type JSONSchema struct {
    schema map[string]interface{}
}

func NewJSONSchema(schema map[string]interface{}) *JSONSchema {
    return &JSONSchema{schema: schema}
}

func (s *JSONSchema) JSONSchema() map[string]interface{} {
    return s.schema
}
```

### Milestone 4.2: generateObject Function

```go
// aisdk/object.go
package aisdk

type GenerateObjectOptions struct {
    Model           provider.LanguageModel
    Prompt          string
    System          string
    Messages        []provider.Message
    Schema          providerutils.Schema
    Mode            string // "auto", "json", "tool"
}

func GenerateObject[T any](ctx context.Context, opts GenerateObjectOptions) (*GenerateObjectResult[T], error) {
    // Set response format
    responseFormat := &provider.ResponseFormat{
        Type:   "json",
        Schema: opts.Schema.JSONSchema(),
    }
    
    // Generate text with JSON format
    result, err := GenerateText(ctx, GenerateTextOptions{
        Model:          opts.Model,
        Prompt:         opts.Prompt,
        System:         opts.System,
        ResponseFormat: responseFormat,
    })
    if err != nil {
        return nil, err
    }
    
    // Parse JSON
    var obj T
    if err := json.Unmarshal([]byte(result.Text), &obj); err != nil {
        return nil, fmt.Errorf("parse object: %w", err)
    }
    
    // Validate against schema
    if err := opts.Schema.Validate(obj); err != nil {
        return nil, fmt.Errorf("validate object: %w", err)
    }
    
    return &GenerateObjectResult[T]{
        Object:       obj,
        FinishReason: result.FinishReason,
        Usage:        result.Usage,
    }, nil
}
```

---

## Phase 5: Additional Providers (Week 9-10)

**Goal**: Add Anthropic and Google providers.

### Milestone 5.1: Anthropic Provider

**Files to create**:
- `anthropic/provider.go`
- `anthropic/chat.go`
- `anthropic/messages.go`

### Milestone 5.2: Google Provider

**Files to create**:
- `google/provider.go`
- `google/chat.go`
- `google/embedding.go`

---

## Phase 6: Additional Features (Week 11-12)

**Goal**: Complete remaining features.

### Milestone 6.1: Image Generation

- `provider/image_model.go`
- `aisdk/generate_image.go`
- `openai/image.go`

### Milestone 6.2: Audio Transcription & Speech

- `provider/transcription_model.go`
- `provider/speech_model.go`
- `aisdk/transcribe.go`
- `aisdk/generate_speech.go`
- `openai/audio.go`

### Milestone 6.3: Reranking

- `provider/reranking_model.go`
- `aisdk/rerank.go`
- `cohere/reranking.go`

---

## Testing Strategy

### Unit Tests

```go
// embed/embed_test.go
package embed_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/lamar/embed"
    "github.com/yourorg/lamar/provider"
)

// MockEmbeddingModel for testing
type MockEmbeddingModel struct {
    embeddings         [][]float64
    maxEmbeddingsPerCall int
    err                error
}

func (m *MockEmbeddingModel) Provider() string  { return "mock" }
func (m *MockEmbeddingModel) ModelID() string   { return "mock-embedding" }
func (m *MockEmbeddingModel) MaxEmbeddingsPerCall() int {
    if m.maxEmbeddingsPerCall == 0 {
        return 2048
    }
    return m.maxEmbeddingsPerCall
}

func (m *MockEmbeddingModel) Embed(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
    if m.err != nil {
        return nil, m.err
    }
    return &provider.EmbedResult{
        Embeddings: m.embeddings,
    }, nil
}

func TestEmbed(t *testing.T) {
    model := &MockEmbeddingModel{
        embeddings: [][]float64{{0.1, 0.2, 0.3}},
    }
    
    result, err := embed.Do(context.Background(), model, "Hello, world!")
    
    require.NoError(t, err)
    assert.Equal(t, []float64{0.1, 0.2, 0.3}, result.Embedding)
}
```

### Integration Tests

```go
// providers/openai/chat_test.go
//go:build integration

package openai_test

import (
    "context"
    "os"
    "testing"
    
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/lamar/providers/openai"
)

func TestChatGenerate(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    
    client := openai.NewProvider(openai.APIKey(apiKey))
    
    result, err := client.GPT4oMini().Generate(context.Background(), &provider.GenerateRequest{
        Prompt: "Say hello",
    })
    
    require.NoError(t, err)
    require.NotEmpty(t, result.Text)
    require.Contains(t, []provider.FinishReason{
        provider.FinishReasonStop,
        provider.FinishReasonLength,
    }, result.FinishReason)
}
```

### Contract Testing

Contract tests verify that provider implementations correctly implement the interfaces.

```go
// internal/contract/model_test.go
package contract_test

import (
    "context"
    "errors"
    "strings"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/lamar/provider"
)

// ModelContract tests LanguageModel implementations.
type ModelContract struct {
    Model         provider.Generator
    SkipStreaming bool
}

func (c *ModelContract) TestBasicGeneration(t *testing.T) {
    ctx := context.Background()
    result, err := c.Model.Generate(ctx, &provider.GenerateRequest{
        Prompt: "Say 'hello'",
    })
    
    require.NoError(t, err)
    assert.NotEmpty(t, result.Text)
    assert.Contains(t, []provider.FinishReason{
        provider.FinishReasonStop,
        provider.FinishReasonLength,
    }, result.FinishReason)
    assert.Greater(t, result.Usage.TotalTokens, 0)
}

func (c *ModelContract) TestStreaming(t *testing.T) {
    if c.SkipStreaming {
        t.Skip("streaming not supported")
    }
    
    streamer, ok := c.Model.(provider.Streamer)
    if !ok {
        t.Skip("model doesn't implement Streamer")
    }
    
    ctx := context.Background()
    result := streamer.Stream(ctx, &provider.GenerateRequest{
        Prompt: "Count from 1 to 5",
    })
    
    var parts []string
    for part := range result.Stream() {
        if textPart, ok := part.(provider.TextPart); ok {
            parts = append(parts, textPart.Delta)
        }
    }
    
    fullText, err := result.Text()
    require.NoError(t, err)
    assert.NotEmpty(t, fullText)
    assert.Equal(t, strings.Join(parts, ""), fullText)
}

func (c *ModelContract) TestContextCancellation(t *testing.T) {
    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately
    
    _, err := c.Model.Generate(ctx, &provider.GenerateRequest{
        Prompt: "Hello",
    })
    
    require.Error(t, err)
    assert.True(t,
        errors.Is(err, context.Canceled) ||
        provider.ErrorCodeOf(err) == provider.CodeContextCanceled,
    )
}

func (c *ModelContract) TestEmptyPrompt(t *testing.T) {
    ctx := context.Background()
    _, err := c.Model.Generate(ctx, &provider.GenerateRequest{
        Prompt: "",
    })
    
    require.Error(t, err)
    assert.True(t,
        provider.ErrorCodeOf(err) == provider.CodeInvalidPrompt ||
        errors.Is(err, provider.ErrInvalidPrompt),
    )
}

func (c *ModelContract) TestMaxTokens(t *testing.T) {
    ctx := context.Background()
    result, err := c.Model.Generate(ctx, &provider.GenerateRequest{
        Prompt: "Write a long story about a dragon",
        Config: &provider.GenerateConfig{
            MaxTokens: 10,
        },
    })
    
    require.NoError(t, err)
    // Response should be truncated
    assert.LessOrEqual(t, result.Usage.CompletionTokens, 15) // Allow some slack
}

func (c *ModelContract) TestTools(t *testing.T) {
    if !provider.HasCapability(c.Model, provider.CapTools) {
        t.Skip("model doesn't support tools")
    }
    
    // Tool test implementation
}
```

### Provider Contract Test Example

```go
// providers/openai/contract_test.go
//go:build integration

package openai_test

import (
    "os"
    "testing"
    
    "github.com/yourorg/lamar/internal/contract"
    "github.com/yourorg/lamar/providers/openai"
)

func TestOpenAIContract(t *testing.T) {
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    
    client := openai.NewProvider(openai.APIKey(apiKey))
    
    t.Run("GPT4oMini", func(t *testing.T) {
        c := &contract.ModelContract{
            Model: client.GPT4oMini(),
        }
        t.Run("BasicGeneration", c.TestBasicGeneration)
        t.Run("Streaming", c.TestStreaming)
        t.Run("ContextCancellation", c.TestContextCancellation)
        t.Run("EmptyPrompt", c.TestEmptyPrompt)
        t.Run("MaxTokens", c.TestMaxTokens)
        t.Run("Tools", c.TestTools)
    })
    
    t.Run("GPT4o", func(t *testing.T) {
        c := &contract.ModelContract{
            Model: client.GPT4o(),
        }
        t.Run("BasicGeneration", c.TestBasicGeneration)
        t.Run("Streaming", c.TestStreaming)
    })
}
```

### Embedding Contract Test

```go
// internal/contract/embedding_test.go
package contract_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/lamar/provider"
)

// EmbeddingContract tests EmbeddingModel implementations.
type EmbeddingContract struct {
    Model provider.EmbeddingModel
}

func (c *EmbeddingContract) TestBasicEmbed(t *testing.T) {
    ctx := context.Background()
    result, err := c.Model.Embed(ctx, &provider.EmbedRequest{
        Texts: []string{"Hello, world!"},
    })
    
    require.NoError(t, err)
    require.Len(t, result.Embeddings, 1)
    assert.NotEmpty(t, result.Embeddings[0])
    assert.Greater(t, result.Usage.TotalTokens, 0)
}

func (c *EmbeddingContract) TestBatchEmbed(t *testing.T) {
    ctx := context.Background()
    texts := []string{"Hello", "World", "Test"}
    
    result, err := c.Model.Embed(ctx, &provider.EmbedRequest{
        Texts: texts,
    })
    
    require.NoError(t, err)
    require.Len(t, result.Embeddings, len(texts))
    
    // All embeddings should have the same dimension
    dim := len(result.Embeddings[0])
    for i, emb := range result.Embeddings {
        assert.Len(t, emb, dim, "embedding %d has wrong dimension", i)
    }
}

func (c *EmbeddingContract) TestMaxEmbeddingsPerCall(t *testing.T) {
    max := c.Model.MaxEmbeddingsPerCall()
    assert.Greater(t, max, 0)
}
```

### Mock Models for Testing

```go
// internal/mock/model.go
package mock

import (
    "context"
    
    "github.com/yourorg/lamar/provider"
)

// Ensure MockModel implements interfaces
var (
    _ provider.Model         = (*MockModel)(nil)
    _ provider.Generator     = (*MockModel)(nil)
    _ provider.Streamer      = (*MockModel)(nil)
    _ provider.LanguageModel = (*MockModel)(nil)
    _ provider.ModelWithInfo = (*MockModel)(nil)
)

type MockModel struct {
    ProviderName string
    ModelIDValue string
    
    GenerateFunc func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error)
    StreamFunc   func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error)
    
    GenerateCalls []provider.GenerateRequest
    StreamCalls   []provider.GenerateRequest
}

func (m *MockModel) Provider() string { return m.ProviderName }
func (m *MockModel) ModelID() string  { return m.ModelIDValue }

func (m *MockModel) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
    m.GenerateCalls = append(m.GenerateCalls, *req)
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, req)
    }
    return &provider.GenerateResult{
        Text:         "mock response",
        FinishReason: provider.FinishReasonStop,
        Usage:        provider.Usage{TotalTokens: 10},
    }, nil
}

func (m *MockModel) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
    m.StreamCalls = append(m.StreamCalls, *req)
    if m.StreamFunc != nil {
        return m.StreamFunc(ctx, req)
    }
    // Default mock stream
    stream := make(chan provider.Part, 3)
    go func() {
        defer close(stream)
        stream <- provider.TextPart{Delta: "mock"}
        stream <- provider.FinishPart{
            FinishReason: provider.FinishReasonStop,
            Usage:        provider.Usage{TotalTokens: 5},
        }
    }()
    return &provider.StreamResult{Stream: stream}, nil
}

func (m *MockModel) Info() provider.ModelInfo {
    return provider.ModelInfo{
        Provider:     m.ProviderName,
        ModelID:      m.ModelIDValue,
        Capabilities: []provider.Capability{
            provider.CapStreaming,
            provider.CapTools,
        },
        MaxTokens: 4096,
    }
}

func (m *MockModel) HasCapability(cap provider.Capability) bool {
    for _, c := range m.Info().Capabilities {
        if c == cap {
            return true
        }
    }
    return false
}
```

---

## Go Module Structure

```go
// go.mod
module github.com/yourorg/lamar-sdk

go 1.22

require (
    github.com/stretchr/testify v1.9.0
)

// go.sum
```

---

## Summary Timeline

| Phase | Duration | Key Deliverables |
|-------|----------|------------------|
| Phase 1 | Week 1-2 | Provider interfaces, embed function, OpenAI embeddings |
| Phase 2 | Week 3-4 | generateText, tools, OpenAI chat completions |
| Phase 3 | Week 5-6 | streamText, streaming infrastructure |
| Phase 4 | Week 7-8 | generateObject, schema validation |
| Phase 5 | Week 9-10 | Anthropic & Google providers |
| Phase 6 | Week 11-12 | Image, audio, reranking |

---

## Priority Order (Least to Most Complex)

1. **Embeddings** - Simplest: single input → vector output
2. **generateText** - Core functionality, no streaming
3. **Tools** - Function calling integration
4. **generateObject** - Structured output with validation
5. **streamText** - Requires streaming infrastructure
6. **streamObject** - Streaming + validation
7. **Image Generation** - New model type
8. **Audio** - Transcription & speech generation
9. **Reranking** - New model type
10. **Video** - Most complex, potentially long-running