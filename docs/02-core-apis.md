# Core APIs

The `ai` package exports high-level functions that developers use to interact with AI models.

---

## generateText

Generate text completion (non-streaming).

### Function Signature

```typescript
async function generateText<
  TOOLS extends ToolSet,
  OUTPUT extends Output = Output<string, string>,
>(options: {
  model: LanguageModel;
  prompt?: string;
  system?: string;
  messages?: Array<ModelMessage>;
  maxTokens?: number;
  temperature?: number;
  topP?: number;
  topK?: number;
  stopSequences?: string[];
  tools?: TOOLS;
  toolChoice?: ToolChoice<TOOLS>;
  maxRetries?: number;
  abortSignal?: AbortSignal;
  headers?: Record<string, string>;
  // ... more options
}): Promise<GenerateTextResult<TOOLS, OUTPUT>>
```

### Result Type

```typescript
interface GenerateTextResult<TOOLS, OUTPUT> {
  readonly content: Array<ContentPart<TOOLS>>;
  readonly text: string;
  readonly reasoning: Array<ReasoningOutput>;
  readonly files: Array<GeneratedFile>;
  readonly sources: Array<Source>;
  readonly toolCalls: Array<TypedToolCall<TOOLS>>;
  readonly toolResults: Array<TypedToolResult<TOOLS>>;
  readonly finishReason: FinishReason;
  readonly usage: LanguageModelUsage;
  readonly totalUsage: LanguageModelUsage;
  readonly warnings: CallWarning[];
  readonly steps: Array<StepResult<TOOLS>>;
  readonly output: InferCompleteOutput<OUTPUT>;
  readonly request: LanguageModelRequestMetadata;
  readonly response: LanguageModelResponseMetadata;
}
```

### Implementation Flow

1. **Initialization Phase**:
   - Resolves the language model via `resolveLanguageModel(modelArg)`
   - Prepares abort signals with timeout support
   - Prepares call settings and retries
   - Standardizes the prompt

2. **Main Execution Loop**:
   - Uses a `do-while` loop for multi-step generation (tool calling)
   - Each step calls `model.doGenerate()` (LanguageModelV3 interface)
   - Continues when tool calls exist and need execution
   - Stops when stop condition is met

3. **Result Aggregation**:
   - Combines content, tool calls, usage across steps
   - Calculates total usage
   - Returns final result

### Go Equivalent

```go
func GenerateText(ctx context.Context, opts GenerateTextOptions) (*GenerateTextResult, error) {
    // Resolve model
    model := ResolveLanguageModel(opts.Model)
    
    // Prepare settings
    settings := PrepareCallSettings(opts)
    
    // Standardize prompt
    prompt := StandardizePrompt(opts)
    
    // Execute with retries
    var result *GenerateTextResult
    for retry := 0; retry <= opts.MaxRetries; retry++ {
        var err error
        result, err = executeGeneration(ctx, model, prompt, settings)
        if err == nil {
            break
        }
        if !isRetryable(err) {
            return nil, err
        }
    }
    
    return result, nil
}

type GenerateTextResult struct {
    Text          string
    Reasoning     []ReasoningOutput
    ToolCalls     []TypedToolCall
    ToolResults   []TypedToolResult
    FinishReason  FinishReason
    Usage         LanguageModelUsage
    TotalUsage    LanguageModelUsage
    Warnings      []Warning
    Steps         []StepResult
    Output        any
}
```

---

## streamText

Stream text completion.

### Function Signature

```typescript
function streamText<
  TOOLS extends ToolSet,
  OUTPUT extends Output = Output<string, string>,
>(options: {
  // Same as generateText
}): StreamTextResult<TOOLS, OUTPUT>
```

### Result Type

```typescript
interface StreamTextResult<TOOLS, OUTPUT> {
  // Promise-based properties (resolve when streaming completes)
  readonly text: PromiseLike<string>;
  readonly usage: PromiseLike<LanguageModelUsage>;
  readonly finishReason: PromiseLike<FinishReason>;
  readonly steps: PromiseLike<Array<StepResult<TOOLS>>>;
  
  // Stream types
  readonly textStream: AsyncIterableStream<string>;
  readonly fullStream: AsyncIterableStream<TextStreamPart<TOOLS>>;
  readonly partialOutputStream: AsyncIterableStream<InferPartialOutput<OUTPUT>>;
  
  // Response methods
  toTextStreamResponse(init?: ResponseInit): Response;
  toUIMessageStream<UI_MESSAGE>(options?): AsyncIterableStream<...>;
  consumeStream(): PromiseLike<void>;
}
```

### Stream Part Types

```typescript
type TextStreamPart<TOOLS> =
  | { type: 'text-start'; id: string }
  | { type: 'text-delta'; id: string; text: string }
  | { type: 'text-end'; id: string }
  | { type: 'reasoning-start'; id: string }
  | { type: 'reasoning-delta'; id: string; delta: string }
  | { type: 'reasoning-end'; id: string }
  | { type: 'tool-call' } & TypedToolCall<TOOLS>
  | { type: 'tool-result' } & TypedToolResult<TOOLS>
  | { type: 'source'; source: Source }
  | { type: 'file'; file: GeneratedFile }
  | { type: 'start-step'; stepNumber: number }
  | { type: 'finish-step'; stepNumber: number; usage: LanguageModelUsage }
  | { type: 'start' }
  | { type: 'finish'; finishReason: FinishReason; usage: LanguageModelUsage }
  | { type: 'error'; error: Error }
  | { type: 'abort' }
```

### Implementation Flow

1. **Immediate Return**: Returns a `StreamTextResult` immediately (deferred execution)

2. **Stream Architecture**:
   - `createStitchableStream()` - allows adding multiple streams dynamically
   - Event processor `TransformStream` for recording content and triggering callbacks
   - `runToolsTransformation` for executing tools in parallel

3. **Step Streaming**:
   - Calls `model.doStream()` (LanguageModelV3 interface)
   - Processes chunks via `TransformStream` transformer
   - Tracks active text/reasoning content by ID

4. **Tool Execution**:
   - Tools execute in parallel while stream continues
   - Stream closes only when all tool results are received

### Go Equivalent

```go
func StreamText(ctx context.Context, opts StreamTextOptions) *StreamTextResult {
    result := &StreamTextResult{
        textStream: make(chan string, 100),
        fullStream: make(chan TextStreamPart, 100),
        done:       make(chan struct{}),
    }
    
    go func() {
        defer close(result.done)
        defer close(result.textStream)
        defer close(result.fullStream)
        
        model := ResolveLanguageModel(opts.Model)
        stream, err := model.DoStream(ctx, opts)
        if err != nil {
            result.fullStream <- TextStreamPart{Type: "error", Error: err}
            return
        }
        
        for part := range stream {
            // Process and forward parts
            result.fullStream <- part
            if part.Type == "text-delta" {
                result.textStream <- part.Text
            }
        }
    }()
    
    return result
}

type StreamTextResult struct {
    textStream <-chan string
    fullStream <-chan TextStreamPart
    done       <-chan struct{}
    
    // Methods
    Text() (string, error)          // Blocks until complete
    Usage() (LanguageModelUsage, error)
    FinishReason() (FinishReason, error)
    ConsumeStream() error
}
```

---

## embed / embedMany

Generate embeddings.

### embed Function

```typescript
async function embed(options: {
  model: EmbeddingModel;
  value: string;
  maxRetries?: number;
  abortSignal?: AbortSignal;
  headers?: Record<string, string>;
  providerOptions?: ProviderOptions;
}): Promise<EmbedResult>

interface EmbedResult {
  readonly value: string;
  readonly embedding: Array<number>;
  readonly usage: EmbeddingModelUsage;  // { tokens: number }
  readonly warnings: Array<Warning>;
  readonly providerMetadata?: ProviderMetadata;
}
```

### embedMany Function

```typescript
async function embedMany(options: {
  model: EmbeddingModel;
  values: Array<string>;
  maxRetries?: number;
  maxParallelCalls?: number;  // default: Infinity
  abortSignal?: AbortSignal;
  headers?: Record<string, string>;
  providerOptions?: ProviderOptions;
}): Promise<EmbedManyResult>

interface EmbedManyResult {
  readonly values: Array<string>;
  readonly embeddings: Array<Array<number>>;
  readonly usage: EmbeddingModelUsage;
  readonly warnings: Array<Warning>;
}
```

### Batching Logic

1. Check `model.maxEmbeddingsPerCall` and `model.supportsParallelCalls`
2. **Single-call path**: If no limit, process all in one call
3. **Batched path**:
   - Split values into chunks using `splitArray(values, maxEmbeddingsPerCall)`
   - Split chunks into parallel groups
   - Execute calls and aggregate results

### Go Equivalent

```go
func Embed(ctx context.Context, opts EmbedOptions) (*EmbedResult, error) {
    model := ResolveEmbeddingModel(opts.Model)
    
    resp, err := model.DoEmbed(ctx, EmbeddingModelCallOptions{
        Values:   []string{opts.Value},
        Headers:  opts.Headers,
        // ...
    })
    if err != nil {
        return nil, err
    }
    
    return &EmbedResult{
        Value:     opts.Value,
        Embedding: resp.Embeddings[0],
        Usage:     resp.Usage,
    }, nil
}

func EmbedMany(ctx context.Context, opts EmbedManyOptions) (*EmbedManyResult, error) {
    model := ResolveEmbeddingModel(opts.Model)
    
    maxPerCall, _ := model.MaxEmbeddingsPerCall(ctx)
    supportsParallel, _ := model.SupportsParallelCalls(ctx)
    
    if maxPerCall == 0 || maxPerCall >= len(opts.Values) {
        // Single call
        resp, err := model.DoEmbed(ctx, EmbeddingModelCallOptions{
            Values: opts.Values,
        })
        if err != nil {
            return nil, err
        }
        return &EmbedManyResult{
            Values:     opts.Values,
            Embeddings: resp.Embeddings,
            Usage:      resp.Usage,
        }, nil
    }
    
    // Batched calls
    chunks := splitSlice(opts.Values, maxPerCall)
    results := make([][]float64, len(opts.Values))
    var totalTokens int
    
    for i, chunk := range chunks {
        resp, err := model.DoEmbed(ctx, EmbeddingModelCallOptions{
            Values: chunk,
        })
        if err != nil {
            return nil, err
        }
        // Map embeddings back to original indices
        for j, emb := range resp.Embeddings {
            results[i*maxPerCall+j] = emb
        }
        totalTokens += resp.Usage.Tokens
    }
    
    return &EmbedManyResult{
        Values:     opts.Values,
        Embeddings: results,
        Usage:      EmbeddingModelUsage{Tokens: totalTokens},
    }, nil
}
```

---

## generateObject / streamObject

Generate structured output with schema validation.

### Function Signature

```typescript
async function generateObject<T>(options: {
  model: LanguageModel;
  prompt?: string;
  messages?: Array<ModelMessage>;
  schema: Schema<T>;
  mode?: 'auto' | 'json' | 'tool';
  output?: 'object' | 'array' | 'enum';
  // ... other options
}): Promise<GenerateObjectResult<T>>

interface GenerateObjectResult<T> {
  readonly object: T;
  readonly finishReason: FinishReason;
  readonly usage: LanguageModelUsage;
  readonly warnings: Array<Warning>;
  readonly text: string;
}
```

### Schema Definition

```typescript
import { zodSchema } from 'ai';
import { z } from 'zod';

const schema = z.object({
  name: z.string(),
  age: z.number(),
  email: z.string().email(),
});

const result = await generateObject({
  model: openai('gpt-4'),
  prompt: 'Generate a user profile',
  schema: zodSchema(schema),
});

console.log(result.object); // { name: "...", age: 25, email: "..." }
```

### Modes

- **`auto`**: Let the SDK choose the best mode
- **`json`**: Use JSON response format
- **`tool`**: Use function calling
- **`enum`**: For single enum value output

### Go Equivalent

```go
func GenerateObject[T any](ctx context.Context, opts GenerateObjectOptions) (*GenerateObjectResult[T], error) {
    // Validate schema is provided
    if opts.Schema == nil {
        return nil, errors.New("schema is required")
    }
    
    model := ResolveLanguageModel(opts.Model)
    
    // Determine mode
    mode := opts.Mode
    if mode == "" {
        mode = "auto"
    }
    
    // Convert schema to JSON schema
    jsonSchema := opts.Schema.JSONSchema()
    
    // Configure response format
    responseFormat := &ResponseFormat{
        Type: "json",
        Schema: jsonSchema,
    }
    
    // Call model
    result, err := model.DoGenerate(ctx, LanguageModelCallOptions{
        Prompt:         prompt,
        ResponseFormat: responseFormat,
    })
    if err != nil {
        return nil, err
    }
    
    // Parse and validate
    var obj T
    if err := json.Unmarshal([]byte(result.Text), &obj); err != nil {
        return nil, fmt.Errorf("parse object: %w", err)
    }
    
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

## Tool Definition

Define tools for function calling.

### Function Signature

```typescript
function tool<TInput, TOutput>(options: {
  id?: string;
  description: string;
  inputSchema: Schema<TInput>;
  outputSchema?: Schema<TOutput>;
  execute: (input: TInput, context: ToolExecuteOptions) => Promise<TOutput>;
}): Tool<TInput, TOutput>
```

### Usage Example

```typescript
import { generateText, tool } from 'ai';
import { z } from 'zod';

const weatherTool = tool({
  description: 'Get the current weather in a location',
  inputSchema: z.object({
    location: z.string().describe('The location to get the weather for'),
  }),
  outputSchema: z.object({
    temperature: z.number(),
    condition: z.string(),
  }),
  execute: async ({ location }) => {
    const response = await fetch(`/api/weather?location=${location}`);
    return response.json();
  },
});

const result = await generateText({
  model: openai('gpt-4'),
  prompt: 'What is the weather in Tokyo?',
  tools: { weather: weatherTool },
});
```

### Go Equivalent

```go
type Tool interface {
    ID() string
    Description() string
    InputSchema() Schema
    OutputSchema() Schema
    Execute(ctx context.Context, input any) (any, error)
}

type ToolDefinition[I, O any] struct {
    id          string
    description string
    inputSchema Schema
    outputSchema Schema
    execute     func(ctx context.Context, input I) (O, error)
}

func NewTool[I, O any](opts ToolOptions[I, O]) *ToolDefinition[I, O] {
    return &ToolDefinition[I, O]{
        id:          opts.ID,
        description: opts.Description,
        inputSchema: opts.InputSchema,
        outputSchema: opts.OutputSchema,
        execute:     opts.Execute,
    }
}

// Usage
weatherTool := NewTool(ToolOptions[WeatherInput, WeatherOutput]{
    Description: "Get the current weather in a location",
    InputSchema: NewZodSchema(z.Object{
        "location": z.String().Describe("The location"),
    }),
    Execute: func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
        return getWeather(input.Location)
    },
})
```

---

## Summary

| Function | Purpose | Complexity |
|----------|---------|------------|
| `embed` | Generate single embedding | Low |
| `embedMany` | Generate multiple embeddings | Low-Medium |
| `generateText` | Non-streaming text generation | Medium |
| `generateObject` | Structured output | Medium |
| `streamText` | Streaming text generation | High |
| `streamObject` | Streaming structured output | High |
| `tool` | Define tools | Low |

**Implementation Order for Go SDK**:
1. `embed` (simplest, good starting point)
2. `generateText` (core functionality)
3. `tool` + tool execution
4. `generateObject`
5. `streamText` (requires streaming infrastructure)
6. `streamObject`