# Testing Patterns

The AI SDK uses **Vitest** for testing with comprehensive mocking utilities.

---

## Test Framework

- **Framework**: Vitest
- **Test Location**: Co-located with source files (`*.test.ts`)
- **Type Tests**: `*.test-d.ts`
- **UI Tests**: `*.ui.test.ts`
- **E2E Tests**: `*.e2e.test.ts`

---

## Test Configuration

### Separate Node and Edge Configs

Each package has two vitest configs:

**Node** (`vitest.node.config.js`):
```javascript
export default defineConfig({
  define: { __PACKAGE_VERSION__: JSON.stringify(version) },
  test: {
    environment: 'node',
    include: ['**/*.test.ts{,x}'],
    exclude: ['**/*.ui.test.ts{,x}', '**/*.e2e.test.ts{,x}'],
    typecheck: { enabled: true },
  },
});
```

**Edge** (`vitest.edge.config.js`):
```javascript
export default defineConfig({
  test: {
    environment: 'edge-runtime',  // uses @edge-runtime/vm
    // ...
  },
});
```

---

## Mock Implementations

### MockLanguageModelV3

```typescript
// packages/ai/src/test/mock-language-model-v3.ts
import { LanguageModelV3, LanguageModelV3CallOptions } from '@ai-sdk/provider';

export class MockLanguageModelV3 implements LanguageModelV3 {
  readonly specificationVersion = 'v3';
  readonly provider: string;
  readonly modelId: string;
  
  doGenerate: LanguageModelV3['doGenerate'];
  doStream: LanguageModelV3['doStream'];
  
  doGenerateCalls: LanguageModelV3CallOptions[] = [];
  doStreamCalls: LanguageModelV3CallOptions[] = [];

  constructor({
    provider = 'mock-provider',
    modelId = 'mock-model-id',
    doGenerate,
    doStream,
  }: {
    provider?: string;
    modelId?: string;
    doGenerate?: LanguageModelV3['doGenerate'] | LanguageModelV3GenerateResult | LanguageModelV3GenerateResult[];
    doStream?: LanguageModelV3['doStream'] | LanguageModelV3StreamResult | LanguageModelV3StreamResult[];
  } = {}) {
    this.provider = provider;
    this.modelId = modelId;
    
    this.doGenerate = wrapMockFunction(doGenerate, this.doGenerateCalls);
    this.doStream = wrapMockFunction(doStream, this.doStreamCalls);
  }
}
```

### Usage

```typescript
import { MockLanguageModelV3, mockValues } from 'ai/test';

describe('generateText', () => {
  it('should call model with correct options', async () => {
    const model = new MockLanguageModelV3({
      doGenerate: mockValues(
        { content: [{ type: 'text', text: 'Hello' }], finishReason: 'stop', usage: { promptTokens: 10, completionTokens: 5 } },
      ),
    });

    await generateText({
      model,
      prompt: 'Say hello',
    });

    expect(model.doGenerateCalls).toHaveLength(1);
    expect(model.doGenerateCalls[0].prompt).toBeDefined();
  });
});
```

---

## HTTP Mocking

### Test Server Package

The SDK uses **MSW (Mock Service Worker)** via `@ai-sdk/test-server`:

```typescript
import { createTestServer } from '@ai-sdk/test-server/with-vitest';

const server = createTestServer({
  'https://api.openai.com/v1/chat/completions': {},
});

describe('OpenAI provider', () => {
  it('should call OpenAI API', async () => {
    server.urls['https://api.openai.com/v1/chat/completions'].response = {
      type: 'json-value',
      body: {
        id: 'chatcmpl-123',
        choices: [{
          message: { role: 'assistant', content: 'Hello!' },
          finish_reason: 'stop',
        }],
        usage: { prompt_tokens: 10, completion_tokens: 5, total_tokens: 15 },
      },
    };

    const result = await generateText({
      model: openai('gpt-4'),
      prompt: 'Say hello',
    });

    expect(result.text).toBe('Hello!');
    expect(server.calls[0].requestBodyJson).toMatchObject({
      model: 'gpt-4',
      messages: [{ role: 'user', content: 'Say hello' }],
    });
  });
});
```

### Response Types

```typescript
type ResponseType = 
  | { type: 'json-value'; body: any; headers?: Record<string, string> }
  | { type: 'stream-chunks'; chunks: string[]; headers?: Record<string, string> }
  | { type: 'binary'; body: Uint8Array; headers?: Record<string, string> }
  | { type: 'error'; status: number; body: any }
  | { type: 'empty' }
  | { type: 'controlled-stream'; headers?: Record<string, string> };
```

---

## Testing Utilities

### From `@ai-sdk/provider-utils/test`

```typescript
// Mock ID generator
mockId() // Returns deterministic ID generator

// Stream conversions
convertArrayToReadableStream(values: T[]) // Array -> ReadableStream
convertReadableStreamToArray(stream: ReadableStream) // ReadableStream -> Array
convertArrayToAsyncIterable(values: T[]) // Array -> AsyncIterable
convertAsyncIterableToArray(iterable: AsyncIterable) // AsyncIterable -> Array
```

### From `ai/test`

```typescript
// Mock models
MockLanguageModelV3
MockEmbeddingModelV3
MockImageModelV3
MockSpeechModelV3
MockTranscriptionModelV3
MockRerankingModelV3
MockProviderV3

// Utilities
mockValues(...values: T[]) // Returns values in sequence
```

---

## Testing Patterns

### 1. Testing Generate Functions

```typescript
import { describe, expect, it, vi } from 'vitest';
import { generateText } from './generate-text';
import { MockLanguageModelV3 } from '../test/mock-language-model-v3';

describe('generateText', () => {
  it('should generate text from model', async () => {
    const model = new MockLanguageModelV3({
      doGenerate: {
        content: [{ type: 'text', text: 'Hello, world!' }],
        finishReason: 'stop',
        usage: { promptTokens: 10, completionTokens: 20, totalTokens: 30 },
      },
    });

    const result = await generateText({
      model,
      prompt: 'Say hello',
    });

    expect(result.text).toBe('Hello, world!');
    expect(result.finishReason).toBe('stop');
    expect(result.usage.totalTokens).toBe(30);
  });

  it('should handle tool calls', async () => {
    const model = new MockLanguageModelV3({
      doGenerate: mockValues(
        {
          content: [{
            type: 'tool-call',
            toolCallId: 'call-1',
            toolName: 'getWeather',
            input: { location: 'Tokyo' },
          }],
          finishReason: 'tool-calls',
          usage: { promptTokens: 20, completionTokens: 10, totalTokens: 30 },
        },
        {
          content: [{ type: 'text', text: 'The weather in Tokyo is sunny.' }],
          finishReason: 'stop',
          usage: { promptTokens: 30, completionTokens: 10, totalTokens: 40 },
        },
      ),
    });

    const weatherTool = tool({
      description: 'Get weather',
      inputSchema: z.object({ location: z.string() }),
      execute: async ({ location }) => ({ temperature: 25, condition: 'sunny' }),
    });

    const result = await generateText({
      model,
      prompt: 'What is the weather in Tokyo?',
      tools: { getWeather: weatherTool },
    });

    expect(result.toolCalls).toHaveLength(1);
    expect(result.text).toBe('The weather in Tokyo is sunny.');
  });
});
```

### 2. Testing Streaming

```typescript
import { convertReadableStreamToArray } from '@ai-sdk/provider-utils/test';

describe('streamText', () => {
  it('should stream text parts', async () => {
    const model = new MockLanguageModelV3({
      doStream: {
        stream: convertArrayToReadableStream([
          { type: 'text-start', id: '0' },
          { type: 'text-delta', id: '0', delta: 'Hello' },
          { type: 'text-delta', id: '0', delta: ' world' },
          { type: 'text-end', id: '0' },
          { type: 'finish', finishReason: 'stop', usage: { promptTokens: 10, completionTokens: 5, totalTokens: 15 } },
        ]),
      },
    });

    const result = streamText({
      model,
      prompt: 'Say hello',
    });

    const parts = await convertReadableStreamToArray(result.fullStream);
    
    expect(parts.filter(p => p.type === 'text-delta').map(p => p.delta).join('')).toBe('Hello world');
    expect(await result.text).toBe('Hello world');
  });
});
```

### 3. Testing Embeddings

```typescript
describe('embed', () => {
  it('should generate single embedding', async () => {
    const model = new MockEmbeddingModelV3({
      doEmbed: {
        embeddings: [[0.1, 0.2, 0.3]],
        usage: { tokens: 10 },
      },
    });

    const result = await embed({
      model,
      value: 'Hello, world!',
    });

    expect(result.embedding).toEqual([0.1, 0.2, 0.3]);
    expect(result.usage.tokens).toBe(10);
  });
});
```

### 4. Testing with HTTP Mocks

```typescript
import { createTestServer } from '@ai-sdk/test-server/with-vitest';

const server = createTestServer({
  'https://api.openai.com/v1/embeddings': {},
});

describe('OpenAI embedding', () => {
  beforeEach(() => {
    server.calls = [];
  });

  it('should send correct request', async () => {
    server.urls['https://api.openai.com/v1/embeddings'].response = {
      type: 'json-value',
      body: {
        data: [{ embedding: [0.1, 0.2, 0.3], index: 0 }],
        usage: { total_tokens: 10 },
      },
    };

    const result = await embed({
      model: openai.embedding('text-embedding-3-small'),
      value: 'Hello',
    });

    expect(result.embedding).toEqual([0.1, 0.2, 0.3]);
    expect(server.calls[0].requestBodyJson).toMatchObject({
      input: ['Hello'],
      model: 'text-embedding-3-small',
    });
  });
});
```

---

## Go Testing Patterns

### Testing with Mock Models

```go
package aisdk_test

import (
    "context"
    "testing"
    
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    
    "github.com/yourorg/aisdk"
    "github.com/yourorg/aisdk/provider"
)

// MockLanguageModel implements provider.LanguageModelV3 for testing
type MockLanguageModel struct {
    provider      string
    modelID       string
    generateFunc  func(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.GenerateResult, error)
    streamFunc    func(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.StreamResult, error)
    generateCalls []provider.LanguageModelCallOptions
}

func (m *MockLanguageModel) SpecificationVersion() string { return "v3" }
func (m *MockLanguageModel) Provider() string            { return m.provider }
func (m *MockLanguageModel) ModelID() string             { return m.modelID }

func (m *MockLanguageModel) DoGenerate(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.GenerateResult, error) {
    m.generateCalls = append(m.generateCalls, opts)
    if m.generateFunc != nil {
        return m.generateFunc(ctx, opts)
    }
    return &provider.GenerateResult{
        Content: []provider.ContentPart{
            {Type: "text", Text: "Hello, world!"},
        },
        FinishReason: provider.FinishReasonStop,
        Usage:        provider.LanguageModelUsage{TotalTokens: 30},
    }, nil
}

func (m *MockLanguageModel) DoStream(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.StreamResult, error) {
    if m.streamFunc != nil {
        return m.streamFunc(ctx, opts)
    }
    // Return mock stream
    stream := make(chan provider.StreamPart, 3)
    stream <- provider.StreamPart{Type: "text-delta", Delta: "Hello"}
    stream <- provider.StreamPart{Type: "text-delta", Delta: " world"}
    close(stream)
    return &provider.StreamResult{Stream: stream}, nil
}

func TestGenerateText(t *testing.T) {
    model := &MockLanguageModel{
        modelID: "test-model",
    }

    result, err := aisdk.GenerateText(context.Background(), aisdk.GenerateTextOptions{
        Model:  model,
        Prompt: "Say hello",
    })

    require.NoError(t, err)
    assert.Equal(t, "Hello, world!", result.Text)
    assert.Equal(t, provider.FinishReasonStop, result.FinishReason)
    assert.Len(t, model.generateCalls, 1)
}
```

### HTTP Mocking in Go

```go
package openai_test

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"
    
    "github.com/stretchr/testify/assert"
    
    "github.com/yourorg/aisdk/openai"
)

func TestOpenAIGenerate(t *testing.T) {
    // Create test server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        assert.Equal(t, "/chat/completions", r.URL.Path)
        assert.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
        
        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        w.Write([]byte(`{
            "id": "chatcmpl-123",
            "choices": [{
                "message": {"role": "assistant", "content": "Hello!"},
                "finish_reason": "stop"
            }],
            "usage": {"prompt_tokens": 10, "completion_tokens": 5, "total_tokens": 15}
        }`))
    }))
    defer server.Close()

    provider := openai.NewProvider(openai.ProviderOptions{
        BaseURL: server.URL,
        APIKey:  "test-key",
    })

    result, err := provider.Model("gpt-4").DoGenerate(context.Background(), provider.LanguageModelCallOptions{
        Prompt: []provider.Message{
            {Role: "user", Content: "Say hello"},
        },
    })

    assert.NoError(t, err)
    assert.Equal(t, "Hello!", result.Content[0].Text)
}

// Using tableau for table-driven tests
func TestFinishReasonMapping(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected provider.FinishReason
    }{
        {"stop", "stop", provider.FinishReasonStop},
        {"length", "length", provider.FinishReasonLength},
        {"tool_calls", "tool_calls", provider.FinishReasonToolCalls},
        {"content_filter", "content_filter", provider.FinishReasonContentFilter},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := mapFinishReason(tt.input)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Testing Streaming in Go

```go
func TestStreamText(t *testing.T) {
    model := &MockLanguageModel{
        streamFunc: func(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.StreamResult, error) {
            stream := make(chan provider.StreamPart, 5)
            go func() {
                defer close(stream)
                stream <- provider.StreamPart{Type: "text-delta", ID: "0", Delta: "Hello"}
                stream <- provider.StreamPart{Type: "text-delta", ID: "0", Delta: " world"}
                stream <- provider.StreamPart{
                    Type:         "finish",
                    FinishReason: provider.FinishReasonStop,
                    Usage:        provider.LanguageModelUsage{TotalTokens: 15},
                }
            }()
            return &provider.StreamResult{Stream: stream}, nil
        },
    }

    result := aisdk.StreamText(context.Background(), aisdk.StreamTextOptions{
        Model:  model,
        Prompt: "Say hello",
    })

    var textParts []string
    for part := range result.FullStream() {
        if part.Type == "text-delta" {
            textParts = append(textParts, part.Delta)
        }
    }

    assert.Equal(t, []string{"Hello", " world"}, textParts)
    
    text, err := result.Text()
    require.NoError(t, err)
    assert.Equal(t, "Hello world", text)
}
```

---

## Test File Organization

```
packages/ai/src/
├── generate-text/
│   ├── generate-text.ts
│   ├── generate-text.test.ts        # Unit tests
│   ├── generate-text.test-d.ts      # Type tests
│   ├── __snapshots__/
│   │   └── generate-text.test.ts.snap
│   └── __fixtures__/
│       └── example-response.json
```

---

## Key Testing Takeaways

1. **Co-locate tests** with source files
2. **Use mocks** for models and HTTP
3. **Test both happy and error paths**
4. **Verify API contracts** with schemas
5. **Test streaming** with channels/async iterators
6. **Use table-driven tests** for multiple cases
7. **Type tests** ensure type inference works correctly