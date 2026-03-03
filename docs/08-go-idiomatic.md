# Go Idiomatic Adaptations

This document outlines changes to make the AI SDK more Go-idiomatic and aligned with the Go ecosystem.

---

## 1. Package Structure & Naming

### TypeScript Pattern
```
packages/
├── ai/                    # Main package
├── @ai-sdk/provider/      # Scoped package
├── @ai-sdk/openai/        # Provider package
```

### Go Pattern
```
github.com/yourorg/lamar-sdk/
├── provider/              # Interface definitions (like io package)
├── openai/                # Provider implementation
├── anthropic/
├── google/
├── gemini/
├── aisdk/                 # High-level functions (or just root package)
├── internal/              # Private implementation details
└── examples/
```

### Naming Conventions

| TypeScript | Go | Reason |
|------------|-----|--------|
| `generateText` | `Generate` or `Complete` | Go prefers short, clear names |
| `streamText` | `Stream` or `CompleteStream` | Simplified |
| `embedMany` | `EmbedBatch` or just `Embed` | More explicit |
| `LanguageModelV3` | `LanguageModel` | Remove version from type name |
| `doGenerate` | `Generate` | Remove "do" prefix |
| `maxOutputTokens` | `MaxTokens` | Shorter |
| `finishReason` | `FinishReason` | Already good |

### Export Rules
- Go uses **PascalCase** for exported names
- Go uses **camelCase** for unexported names
- No explicit `export` keyword - capitalization determines visibility

```go
// TypeScript
export interface GenerateTextResult { ... }

// Go
type GenerateResult struct { ... }  // Exported (PascalCase)
type generateOptions struct { ... }  // Unexported (camelCase)
```

---

## 2. Error Handling

### TypeScript Pattern
```typescript
try {
  const result = await generateText({ ... });
} catch (error) {
  if (APICallError.isInstance(error)) {
    // handle API error
  }
}
```

### Go Pattern
```go
result, err := Generate(ctx, opts)
if err != nil {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        // handle API error
    }
    return err
}
```

### Error Types

```go
// errors.go
package aisdk

import (
    "errors"
    "fmt"
)

// Base error type
type Error struct {
    Code    string
    Message string
    Cause   error
}

func (e *Error) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error {
    return e.Cause
}

// Specific error types
type APIError struct {
    Error
    StatusCode int
}

type InvalidPromptError struct {
    Error
}

type NoContentGeneratedError struct {
    Error
}

// Constructors
func NewAPIError(statusCode int, message string, cause error) *APIError {
    return &APIError{
        Error: Error{
            Code:    "API_ERROR",
            Message: message,
            Cause:   cause,
        },
        StatusCode: statusCode,
    }
}

// Type checking (replaces .isInstance())
func IsAPIError(err error) bool {
    var apiErr *APIError
    return errors.As(err, &apiErr)
}

func IsInvalidPrompt(err error) bool {
    var invalidPrompt *InvalidPromptError
    return errors.As(err, &invalidPrompt)
}
```

---

## 3. Interfaces

### TypeScript Pattern
```typescript
interface LanguageModelV3 {
  readonly specificationVersion: 'v3';
  readonly provider: string;
  readonly modelId: string;
  doGenerate(options: LanguageModelV3CallOptions): Promise<LanguageModelV3GenerateResult>;
}
```

### Go Pattern
```go
// provider/language_model.go
package provider

import "context"

// LanguageModel represents a text generation model.
// It follows Go's implicit interface implementation pattern.
type LanguageModel interface {
    // Metadata
    Provider() string
    ModelID() string
    
    // Generation
    Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error)
    Stream(ctx context.Context, opts GenerateOptions) (*StreamResult, error)
}

// EmbeddingModel represents an embedding model.
type EmbeddingModel interface {
    Provider() string
    ModelID() string
    MaxEmbeddingsPerCall() int
    
    Embed(ctx context.Context, opts EmbedOptions) (*EmbedResult, error)
}
```

### Key Differences

| Aspect | TypeScript | Go |
|--------|------------|-----|
| Interface versioning | `specificationVersion: 'v3'` | Separate interfaces or optional methods |
| Promise-based | `Promise<Result>` | `(result, error)` tuple |
| Readonly properties | `readonly` | No setter methods |
| Optional methods | All required | Can use optional interface pattern |

### Optional Interface Pattern (Go)

```go
// For optional capabilities
type StreamingModel interface {
    LanguageModel
    IsStreaming() bool
}

// Check if model supports streaming
if sm, ok := model.(StreamingModel); ok {
    // Model supports streaming
}
```

---

## 4. Configuration

### TypeScript Pattern
```typescript
await generateText({
  model: openai('gpt-4'),
  prompt: 'Hello',
  maxTokens: 100,
  temperature: 0.7,
});
```

### Go Pattern (Functional Options)

```go
// options.go
package aisdk

type GenerateOption func(*GenerateConfig)

type GenerateConfig struct {
    Model          LanguageModel
    System         string
    MaxTokens      int
    Temperature    float64
    TopP           float64
    StopSequences  []string
    Tools          []Tool
    Headers        map[string]string
    ProviderOpts   map[string]any
}

func WithSystem(system string) GenerateOption {
    return func(c *GenerateConfig) {
        c.System = system
    }
}

func WithMaxTokens(n int) GenerateOption {
    return func(c *GenerateConfig) {
        c.MaxTokens = n
    }
}

func WithTemperature(t float64) GenerateOption {
    return func(c *GenerateConfig) {
        c.Temperature = t
    }
}

func WithTools(tools ...Tool) GenerateOption {
    return func(c *GenerateConfig) {
        c.Tools = append(c.Tools, tools...)
    }
}

// Usage
result, err := Generate(ctx, model, "Hello",
    WithMaxTokens(100),
    WithTemperature(0.7),
)
```

### Alternative: Config Struct

```go
// Simpler alternative for complex configs
type GenerateRequest struct {
    Model         LanguageModel
    Prompt        string
    System        string
    MaxTokens     int
    Temperature   float64
    Tools         []Tool
}

result, err := Generate(ctx, GenerateRequest{
    Model:       openai.Model("gpt-4"),
    Prompt:      "Hello",
    MaxTokens:   100,
    Temperature: 0.7,
})
```

### Provider Configuration

```go
// openai/provider.go
package openai

import (
    "os"
    
    "github.com/yourorg/lamar-sdk/provider"
)

type Provider struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
    orgID      string
}

type ProviderOption func(*Provider)

func WithAPIKey(key string) ProviderOption {
    return func(p *Provider) {
        p.apiKey = key
    }
}

func WithBaseURL(url string) ProviderOption {
    return func(p *Provider) {
        p.baseURL = url
    }
}

func WithHTTPClient(client *http.Client) ProviderOption {
    return func(p *Provider) {
        p.httpClient = client
    }
}

func WithOrgID(orgID string) ProviderOption {
    return func(p *Provider) {
        p.orgID = orgID
    }
}

func NewProvider(opts ...ProviderOption) *Provider {
    p := &Provider{
        baseURL:    "https://api.openai.com/v1",
        httpClient: http.DefaultClient,
    }
    
    for _, opt := range opts {
        opt(p)
    }
    
    // Environment variable fallback
    if p.apiKey == "" {
        p.apiKey = os.Getenv("OPENAI_API_KEY")
    }
    
    return p
}

// Usage
client := openai.NewProvider(
    openai.WithAPIKey("sk-..."),
    openai.WithOrgID("org-..."),
)
```

---

## 5. Streaming

### TypeScript Pattern
```typescript
const result = streamText({ model, prompt });
for await (const part of result.fullStream) {
  console.log(part);
}
const text = await result.text;
```

### Go Pattern

```go
// Using channels
result := Stream(ctx, model, "Hello")

// Iterate over stream
for part := range result.Stream() {
    fmt.Println(part)
}

// Get final text (blocks until complete)
text, err := result.Text()
```

### Stream Implementation

```go
// stream.go
package aisdk

import (
    "context"
    "sync"
)

type StreamResult struct {
    stream  <-chan StreamPart
    text    string
    usage   Usage
    err     error
    
    once    sync.Once
    done    chan struct{}
}

type StreamPart struct {
    Type    string      // "text", "delta", "tool_call", "finish", "error"
    Content string      // Text content
    Delta   string      // Streaming delta
    ToolCall *ToolCall  // If type == "tool_call"
    Error   error       // If type == "error"
    Usage   *Usage      // If type == "finish"
}

func (r *StreamResult) Stream() <-chan StreamPart {
    return r.stream
}

func (r *StreamResult) Text() (string, error) {
    <-r.done
    return r.text, r.err
}

func (r *StreamResult) Usage() (Usage, error) {
    <-r.done
    return r.usage, r.err
}
```

### Alternative: io.Reader Pattern

```go
// For text-only streaming (simpler)
type TextStream struct {
    reader io.Reader
}

func (s *TextStream) Read(p []byte) (n int, err error) {
    return s.reader.Read(p)
}

// Usage
stream := StreamText(ctx, model, "Hello")
io.Copy(os.Stdout, stream)  // Stream to stdout
```

### SSE Stream Parser

```go
// internal/sse/sse.go
package sse

import (
    "bufio"
    "bytes"
    "io"
    "strings"
)

type Event struct {
    Type string
    Data []byte
}

type Decoder struct {
    scanner *bufio.Scanner
}

func NewDecoder(r io.Reader) *Decoder {
    return &Decoder{
        scanner: bufio.NewScanner(r),
    }
}

func (d *Decoder) Decode() (Event, error) {
    var event Event
    var data bytes.Buffer
    
    for d.scanner.Scan() {
        line := d.scanner.Text()
        
        if line == "" {
            if data.Len() > 0 {
                event.Data = data.Bytes()
                return event, nil
            }
            continue
        }
        
        if strings.HasPrefix(line, "event:") {
            event.Type = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
        } else if strings.HasPrefix(line, "data:") {
            data.WriteString(strings.TrimPrefix(line, "data:"))
        }
    }
    
    if err := d.scanner.Err(); err != nil {
        return Event{}, err
    }
    
    return Event{}, io.EOF
}
```

---

## 6. Tools & Function Calling

### TypeScript Pattern
```typescript
const weatherTool = tool({
  description: 'Get weather',
  inputSchema: z.object({ location: z.string() }),
  execute: async ({ location }) => ({ temp: 25 }),
});
```

### Go Pattern

```go
// tool.go
package aisdk

import (
    "context"
    "encoding/json"
)

// Tool represents a function that can be called by the model.
type Tool interface {
    Name() string
    Description() string
    InputSchema() json.RawMessage
    Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

// ToolFunc is a simple tool implementation.
type ToolFunc struct {
    name        string
    description string
    inputSchema json.RawMessage
    fn          func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

func NewTool(name, description string, schema json.RawMessage, fn func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)) *ToolFunc {
    return &ToolFunc{
        name:        name,
        description: description,
        inputSchema: schema,
        fn:          fn,
    }
}

func (t *ToolFunc) Name() string              { return t.name }
func (t *ToolFunc) Description() string       { return t.description }
func (t *ToolFunc) InputSchema() json.RawMessage { return t.inputSchema }
func (t *ToolFunc) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
    return t.fn(ctx, input)
}
```

### Type-Safe Tools with Generics

```go
// typed_tool.go
package aisdk

import (
    "context"
    "encoding/json"
    "reflect"
)

// TypedTool provides type-safe tool execution.
type TypedTool[In, Out any] struct {
    name        string
    description string
    inputSchema json.RawMessage
    fn          func(ctx context.Context, input In) (Out, error)
}

func NewTypedTool[In, Out any](name, description string, schema json.RawMessage, fn func(ctx context.Context, input In) (Out, error)) *TypedTool[In, Out] {
    return &TypedTool[In, Out]{
        name:        name,
        description: description,
        inputSchema: schema,
        fn:          fn,
    }
}

func (t *TypedTool[In, Out]) Name() string              { return t.name }
func (t *TypedTool[In, Out]) Description() string       { return t.description }
func (t *TypedTool[In, Out]) InputSchema() json.RawMessage { return t.inputSchema }

func (t *TypedTool[In, Out]) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
    var in In
    if err := json.Unmarshal(input, &in); err != nil {
        return nil, fmt.Errorf("parse input: %w", err)
    }
    
    out, err := t.fn(ctx, in)
    if err != nil {
        return nil, err
    }
    
    return json.Marshal(out)
}

// Usage
type WeatherInput struct {
    Location string `json:"location"`
}

type WeatherOutput struct {
    Temperature int    `json:"temperature"`
    Condition   string `json:"condition"`
}

weatherSchema := json.RawMessage(`{
    "type": "object",
    "properties": {
        "location": {"type": "string"}
    },
    "required": ["location"]
}`)

weatherTool := NewTypedTool[WeatherInput, WeatherOutput](
    "get_weather",
    "Get the current weather in a location",
    weatherSchema,
    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
        // Implementation
        return WeatherOutput{Temperature: 25, Condition: "sunny"}, nil
    },
)
```

---

## 7. Schema & Validation

### TypeScript Pattern
```typescript
import { z } from 'zod';

const schema = z.object({
  name: z.string(),
  age: z.number(),
});
```

### Go Pattern

**Option 1: JSON Schema + go-playground/validator**

```go
import (
    "github.com/go-playground/validator/v10"
)

type Person struct {
    Name string `json:"name" validate:"required"`
    Age  int    `json:"age" validate:"required,min=0"`
}

var validate = validator.New()

func Validate(v interface{}) error {
    return validate.Struct(v)
}
```

**Option 2: JSON Schema Definition**

```go
// schema.go
package aisdk

import (
    "encoding/json"
)

// Schema represents a JSON Schema.
type Schema map[string]any

// ObjectSchema creates an object schema from a definition.
func ObjectSchema(properties map[string]Schema, required []string) Schema {
    return Schema{
        "type":       "object",
        "properties": properties,
        "required":   required,
    }
}

// StringSchema creates a string schema.
func StringSchema(description string) Schema {
    s := Schema{"type": "string"}
    if description != "" {
        s["description"] = description
    }
    return s
}

// NumberSchema creates a number schema.
func NumberSchema(description string) Schema {
    s := Schema{"type": "number"}
    if description != "" {
        s["description"] = description
    }
    return s
}

// Usage
personSchema := ObjectSchema(
    map[string]Schema{
        "name": StringSchema("The person's name"),
        "age":  NumberSchema("The person's age"),
    },
    []string{"name", "age"},
)
```

**Option 3: Use Existing Go Schema Libraries**

```go
// Using github.com/invopop/jsonschema
import "github.com/invopop/jsonschema"

type Person struct {
    Name string `json:"name" jsonschema:"required,description=The person's name"`
    Age  int    `json:"age" jsonschema:"required,minimum=0,description=The person's age"`
}

// Generate schema
schema := jsonschema.Reflect(&Person{})
```

---

## 8. JSON Handling

### TypeScript Pattern
```typescript
const result = JSON.parse(text);
const safeResult = safeParseJSON({ text, schema });
```

### Go Pattern

```go
// json.go
package aisdk

import (
    "encoding/json"
    "io"
)

// ParseJSON parses JSON with proper error handling.
func ParseJSON[T any](data []byte) (T, error) {
    var result T
    if err := json.Unmarshal(data, &result); err != nil {
        return result, &ParseError{
            Input: string(data),
            Err:   err,
        }
    }
    return result, nil
}

// ParseJSONFromReader parses JSON from an io.Reader.
func ParseJSONFromReader[T any](r io.Reader) (T, error) {
    var result T
    decoder := json.NewDecoder(r)
    if err := decoder.Decode(&result); err != nil {
        return result, &ParseError{
            Err: err,
        }
    }
    return result, nil
}

// SafeParseJSON returns result or error without panic.
func SafeParseJSON[T any](data []byte) (result T, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("JSON parse panic: %v", r)
        }
    }()
    return ParseJSON[T](data)
}
```

---

## 9. HTTP Client

### TypeScript Pattern
```typescript
const response = await fetch(url, { ... });
```

### Go Pattern

```go
// client.go
package providerutils

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
)

// HTTPClient interface allows custom clients.
type HTTPClient interface {
    Do(req *http.Request) (*http.Response, error)
}

// APIClient wraps HTTP operations.
type APIClient struct {
    client  HTTPClient
    headers map[string]string
}

func NewAPIClient(client HTTPClient, headers map[string]string) *APIClient {
    if client == nil {
        client = http.DefaultClient
    }
    return &APIClient{
        client:  client,
        headers: headers,
    }
}

func (c *APIClient) PostJSON(ctx context.Context, url string, body any) ([]byte, *http.Response, error) {
    data, err := json.Marshal(body)
    if err != nil {
        return nil, nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
    if err != nil {
        return nil, nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    for k, v := range c.headers {
        req.Header.Set(k, v)
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, nil, err
    }
    defer resp.Body.Close()
    
    respBody, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, resp, err
    }
    
    return respBody, resp, nil
}

func (c *APIClient) PostStream(ctx context.Context, url string, body any) (io.ReadCloser, error) {
    data, err := json.Marshal(body)
    if err != nil {
        return nil, err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("Accept", "text/event-stream")
    for k, v := range c.headers {
        req.Header.Set(k, v)
    }
    
    resp, err := c.client.Do(req)
    if err != nil {
        return nil, err
    }
    
    if resp.StatusCode >= 400 {
        defer resp.Body.Close()
        body, _ := io.ReadAll(resp.Body)
        return nil, &APIError{
            StatusCode: resp.StatusCode,
            Body:       string(body),
        }
    }
    
    return resp.Body, nil
}
```

---

## 10. Context & Cancellation

### TypeScript Pattern
```typescript
const controller = new AbortController();
await generateText({ ..., abortSignal: controller.signal });
controller.abort();
```

### Go Pattern

```go
// Context is the standard way in Go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := Generate(ctx, model, "Hello")
```

```go
// provider/openai/chat.go
func (m *ChatModel) Generate(ctx context.Context, opts GenerateOptions) (*GenerateResult, error) {
    // Check context before starting
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    
    // Pass context to HTTP request
    req, err := http.NewRequestWithContext(ctx, "POST", url, body)
    // ...
}
```

---

## 11. Concurrency Patterns

### Parallel Embeddings

```go
// embed.go
func EmbedBatch(ctx context.Context, model EmbeddingModel, texts []string) ([][]float64, error) {
    maxPerCall := model.MaxEmbeddingsPerCall()
    
    // If single call
    if maxPerCall <= 0 || maxPerCall >= len(texts) {
        result, err := model.Embed(ctx, EmbedOptions{
            Texts: texts,
        })
        return result.Embeddings, err
    }
    
    // Split into batches
    batches := chunk(texts, maxPerCall)
    results := make([][]float64, len(texts))
    
    // Process in parallel
    var wg sync.WaitGroup
    var mu sync.Mutex
    var firstErr error
    
    for i, batch := range batches {
        wg.Add(1)
        go func(batchIdx int, texts []string) {
            defer wg.Done()
            
            result, err := model.Embed(ctx, EmbedOptions{
                Texts: texts,
            })
            if err != nil {
                mu.Lock()
                if firstErr == nil {
                    firstErr = err
                }
                mu.Unlock()
                return
            }
            
            // Copy results to correct positions
            mu.Lock()
            startIdx := batchIdx * maxPerCall
            for j, emb := range result.Embeddings {
                results[startIdx+j] = emb
            }
            mu.Unlock()
        }(i, batch)
    }
    
    wg.Wait()
    return results, firstErr
}
```

---

## 12. Testing

### Go Testing Pattern

```go
// generate_test.go
package aisdk_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/lamar-sdk/aisdk"
    "github.com/yourorg/lamar-sdk/provider"
)

// Mock model for testing
type MockModel struct {
    generateFunc func(ctx context.Context, opts provider.GenerateOptions) (*provider.GenerateResult, error)
    streamFunc   func(ctx context.Context, opts provider.GenerateOptions) (*provider.StreamResult, error)
}

func (m *MockModel) Provider() string { return "mock" }
func (m *MockModel) ModelID() string  { return "mock-model" }

func (m *MockModel) Generate(ctx context.Context, opts provider.GenerateOptions) (*provider.GenerateResult, error) {
    if m.generateFunc != nil {
        return m.generateFunc(ctx, opts)
    }
    return &provider.GenerateResult{
        Content: []provider.ContentPart{
            {Type: "text", Text: "Hello!"},
        },
        FinishReason: "stop",
    }, nil
}

// Test with table-driven approach
func TestGenerate(t *testing.T) {
    tests := []struct {
        name    string
        model   *MockModel
        prompt  string
        want    string
        wantErr bool
    }{
        {
            name:   "simple text",
            model:  &MockModel{},
            prompt: "Say hello",
            want:   "Hello!",
        },
        {
            name: "custom response",
            model: &MockModel{
                generateFunc: func(ctx context.Context, opts provider.GenerateOptions) (*provider.GenerateResult, error) {
                    return &provider.GenerateResult{
                        Content: []provider.ContentPart{
                            {Type: "text", Text: "Custom response"},
                        },
                        FinishReason: "stop",
                    }, nil
                },
            },
            prompt: "Test",
            want:   "Custom response",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := aisdk.Generate(context.Background(), tt.model, tt.prompt)
            
            if tt.wantErr {
                require.Error(t, err)
                return
            }
            
            require.NoError(t, err)
            assert.Equal(t, tt.want, result.Text)
        })
    }
}

// Integration test (skipped without API key)
func TestOpenAIGenerate(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        t.Skip("OPENAI_API_KEY not set")
    }
    
    client := openai.NewProvider(openai.WithAPIKey(apiKey))
    model := client.Model("gpt-4o-mini")
    
    result, err := aisdk.Generate(context.Background(), model, "Say hello")
    
    require.NoError(t, err)
    assert.NotEmpty(t, result.Text)
}
```

---

## 13. Module Organization

### go.mod

```go
module github.com/yourorg/lamar-sdk

go 1.22

require (
    github.com/go-playground/validator/v10 v10.19.0
    github.com/stretchr/testify v1.9.0
)
```

### Package Exports

```go
// aisdk/aisdk.go (or root package)
package aisdk

// Core types
type GenerateResult struct { ... }
type StreamResult struct { ... }
type Tool interface { ... }

// Core functions
func Generate(ctx context.Context, model LanguageModel, prompt string, opts ...GenerateOption) (*GenerateResult, error)
func Stream(ctx context.Context, model LanguageModel, prompt string, opts ...GenerateOption) *StreamResult
func Embed(ctx context.Context, model EmbeddingModel, text string) (*EmbedResult, error)
func EmbedBatch(ctx context.Context, model EmbeddingModel, texts []string) ([][]float64, error)

// Re-export provider types for convenience
type LanguageModel = provider.LanguageModel
type EmbeddingModel = provider.EmbeddingModel
type GenerateOptions = provider.GenerateOptions
```

---

## Summary: Key Changes

| Aspect | TypeScript | Go |
|--------|------------|-----|
| **Errors** | try/catch, `.isInstance()` | Multiple returns, `errors.As()` |
| **Interfaces** | Explicit, versioned | Implicit, simple |
| **Promises** | `Promise<T>`, `await` | Channels, blocking calls |
| **Streams** | `ReadableStream`, `AsyncIterable` | `<-chan T`, `io.Reader` |
| **Config** | Object literals | Functional options, config structs |
| **Generics** | Full type inference | Type parameters required |
| **JSON** | `JSON.parse()`, Zod | `encoding/json`, struct tags |
| **HTTP** | `fetch` API | `net/http.Client` |
| **Context** | `AbortSignal` | `context.Context` |
| **Testing** | Vitest, mocks | `testing` package, interfaces |
| **Exports** | `export` keyword | PascalCase naming |
| **Module** | npm packages | Go modules |