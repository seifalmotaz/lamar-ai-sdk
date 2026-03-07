# LangChainJS Core Architecture (Runnable System)

## Executive Summary

LangChainJS uses the **Runnable** interface as its core abstraction for composable workflows. Everything is a Runnable - prompts, LLMs, output parsers, tools, chains - and they can all be connected using the LCEL (LangChain Expression Language) pipe operator.

---

## 1. Runnable Interface

### Core Interface Definition

```typescript
interface RunnableInterface<RunInput, RunOutput, CallOptions> {
  // Core invocation method
  invoke(input: RunInput, options?: Partial<CallOptions>): Promise<RunOutput>;
  
  // Batch execution with concurrency control
  batch(inputs: RunInput[], options?: Partial<CallOptions>[], batchOptions?: RunnableBatchOptions): 
    Promise<RunOutput[] | (RunOutput | Error)[]>;
  
  // Streaming support
  stream(input: RunInput, options?: Partial<CallOptions>): Promise<IterableReadableStream<RunOutput>>;
  
  // Transform for streaming pipelines
  transform(generator: AsyncGenerator<RunInput>, options: Partial<CallOptions>): 
    AsyncGenerator<RunOutput>;
  
  // Configuration binding
  getName(suffix?: string): string;
}
```

### RunnableConfig

```go
type RunnableConfig struct {
    Configurable   map[string]any  // Runtime configurable values
    Tags           []string        // Execution tags
    Metadata       map[string]any  // Execution metadata
    RecursionLimit int             // Default 25
    MaxConcurrency int             // Batch parallelism limit
    Timeout        time.Duration   // Timeout per invoke
    Signal         context.Context // Cancellation signal
    Callbacks      []Callback      // Lifecycle callbacks
}
```

**For Lamar SDK:**

```go
type Runnable[In, Out any] interface {
    Invoke(ctx context.Context, input In, opts ...Option) (Out, error)
    Batch(ctx context.Context, inputs []In, opts ...Option) ([]Out, error)
    Stream(ctx context.Context, input In, opts ...Option) (<-chan StreamChunk[Out], error)
    Transform(ctx context.Context, input <-chan In, opts ...Option) (<-chan Out, error)
    
    // Composition
    Pipe[NewOut any](next Runnable[Out, NewOut]) Runnable[In, NewOut]
    WithConfig(opts ...Option) Runnable[In, Out]
}
```

---

## 2. Runnable Composition Patterns

### RunnableSequence (Pipe Chain)

```go
// Connect runnables in sequence
type RunnableSequence[In, Out any] struct {
    First  Runnable[In, any]
    Middle []Runnable[any, any]
    Last   Runnable[any, Out]
}

func (s *RunnableSequence[In, Out]) Invoke(ctx context.Context, input In, opts ...Option) (Out, error) {
    var next any = input
    
    // Execute first
    next, err = s.First.Invoke(ctx, next, opts...)
    if err != nil { return zero[Out](), err }
    
    // Execute middle
    for _, step := range s.Middle {
        next, err = step.Invoke(ctx, next, opts...)
        if err != nil { return zero[Out](), err }
    }
    
    // Execute last
    return s.Last.Invoke(ctx, next, opts...)
}

// Static factory:
func Sequence[In, Out any](runnables ...Runnable[any, any]) Runnable[In, Out]

// LCEL pipe operator equivalent:
chain := prompt.Pipe(llm).Pipe(parser)
// or:
chain := Sequence[any, string](prompt, llm, parser)
```

### RunnableMap (Parallel Execution)

```go
// Execute multiple runnables in parallel
type RunnableMap[In any, Out map[string]any] struct {
    Steps map[string]Runnable[In, any]
}

func (m *RunnableMap[In, Out]) Invoke(ctx context.Context, input In, opts ...Option) (Out, error) {
    result := make(Out)
    
    var wg sync.WaitGroup
    var mu sync.Mutex
    errors := make([]error, 0, len(m.Steps))
    
    for name, runnable := range m.Steps {
        wg.Add(1)
        go func(name string, r Runnable[In, any]) {
            defer wg.Done()
            output, err := r.Invoke(ctx, input, opts...)
            if err != nil {
                mu.Lock()
                errors = append(errors, err)
                mu.Unlock()
                return
            }
            mu.Lock()
            result[name] = output
            mu.Unlock()
        }(name, runnable)
    }
    
    wg.Wait()
    if len(errors) > 0 {
        return result, errors[0] // Return first error
    }
    return result, nil
}

// Usage:
chain := Sequence(
    RunnableMap[any, map[string]any]{
        "context": retriever,
        "question": RunnableLambda[any, any](func(ctx context.Context, input any) (any, error) {
            return input, nil // Passthrough
        }),
    },
    prompt,
    llm,
)
```

### RunnableLambda (Function Wrapper)

```go
// Wrap functions as runnables
type RunnableLambda[In, Out any] struct {
    Func func(ctx context.Context, input In, opts ...Option) (Out, error)
}

func Lambda[In, Out any](fn func(ctx context.Context, input In, opts ...Option) (Out, error)) Runnable[In, Out] {
    return &RunnableLambda[In, Out]{Func: fn}
}

// Usage in chain:
chain := prompt.Pipe(
    llm,
).Pipe(
    Lambda[string, string](func(ctx context.Context, input string, opts ...Option) (string, error) {
        // Extract text from AIMessage
        return input, nil
    }),
)
```

### RunnablePassthrough (Identity/Augment)

```go
// Pass through input and optionally augment with computed values
type RunnablePassthrough[In any] struct {
    Func func(ctx context.Context, input In) error // Optional side effect
}

func (p *RunnablePassthrough[In]) Invoke(ctx context.Context, input In, opts ...Option) (In, error) {
    if p.Func != nil {
        if err := p.Func(ctx, input); err != nil {
            return zero[In](), err
        }
    }
    return input, nil
}

// Assign pattern (augment input)
func Assign[In map[string]any](mapping map[string]Runnable[In, any]) Runnable[In, In] {
    return Sequence(
        RunnableMap[In, map[string]any](mapping),
        Lambda[map[string]any, In](func(ctx context.Context, combined map[string]any, opts ...Option) (In, error) {
            // Merge original input with computed values
            result := make(In)
            for k, v := range combined {
                result[k] = v
            }
            return result, nil
        }),
    )
}

// Usage:
chain := Assign[map[string]any](map[string]Runnable[map[string]any, any]{
    "context": retriever,
}).Pipe(prompt).Pipe(llm)
```

---

## 3. RunnablePassthrough with Assign

The `assign` pattern is fundamental for RAG:

```go
// Input: {question: "What is the capital of France?"}
// Output: {question: "...", context: "relevant documents..."}

func RetrievalChain(retriever Runnable[string, []Document], llm Runnable[string, string]) Runnable[map[string]any, string] {
    return Sequence(
        // Augment input with context
        RunnableMap[map[string]any, map[string]any]{
            "question": Lambda[map[string]any, string](func(ctx context.Context, input map[string]any, opts ...Option) (string, error) {
                return input["question"].(string), nil
            }),
            "context": Sequence(
                Lambda[map[string]any, string](func(ctx context.Context, input map[string]any, opts ...Option) (string, error) {
                    return input["question"].(string), nil
                }),
                retriever,
                Lambda[[]Document, string](func(ctx context.Context, docs []Document, opts ...Option) (string, error) {
                    // Format documents
                    texts := make([]string, len(docs))
                    for i, doc := range docs {
                        texts[i] = doc.Content
                    }
                    return strings.Join(texts, "\n\n"), nil
                }),
            ),
        },
        // Format prompt
        prompt,
        // Invoke LLM
        llm,
        // Parse output
        StringParser(),
    )
}
```

---

## 4. Error Handling and Retry

### RunnableWithFallbacks

```go
type RunnableWithFallbacks[In, Out any] struct {
    Primary   Runnable[In, Out]
    Fallbacks []Runnable[In, Out]
}

func (r *RunnableWithFallbacks[In, Out]) Invoke(ctx context.Context, input In, opts ...Option) (Out, error) {
    var lastErr error
    
    // Try primary
    result, err := r.Primary.Invoke(ctx, input, opts...)
    if err == nil {
        return result, nil
    }
    lastErr = err
    
    // Try fallbacks in order
    for _, fallback := range r.Fallbacks {
        result, err = fallback.Invoke(ctx, input, opts...)
        if err == nil {
            return result, nil
        }
        lastErr = err
    }
    
    return zero[Out](), lastErr
}

// Usage:
chain := llm.WithFallbacks(
    FallbackChain(llm2, llm3),
).Pipe(parser)
```

### RunnableRetry

```go
type RetryPolicy struct {
    MaxAttempts    int
    InitialDelay   time.Duration
    BackoffMult    float64
    MaxDelay       time.Duration
    RetryOn        func(error) bool
}

func (r *RunnableRetry[In, Out]) Invoke(ctx context.Context, input In, opts ...Option) (Out, error) {
    var lastErr error
    delay := r.Policy.InitialDelay
    
    for attempt := 1; attempt <= r.Policy.MaxAttempts; attempt++ {
        result, err := r.Bound.Invoke(ctx, input, opts...)
        if err == nil {
            return result, nil
        }
        
        lastErr = err
        if !r.Policy.RetryOn(err) {
            return zero[Out](), err
        }
        
        if attempt < r.Policy.MaxAttempts {
            time.Sleep(delay)
            delay = time.Duration(float64(delay) * r.Policy.BackoffMult)
            if delay > r.Policy.MaxDelay {
                delay = r.Policy.MaxDelay
            }
        }
    }
    
    return zero[Out](), lastErr
}

// Usage:
chain := llm.WithRetry(RetryPolicy{
    MaxAttempts:  3,
    InitialDelay: 1 * time.Second,
    BackoffMult:  2.0,
    MaxDelay:     10 * time.Second,
    RetryOn: func(err error) bool {
        // Retry on rate limits
        var apiErr *APIError
        if errors.As(err, &apiErr) {
            return apiErr.StatusCode == 429
        }
        return false
    },
})
```

---

## 5. Streaming Implementation

### Stream Interface

```go
func (r *RunnableSequence[In, Out]) Stream(ctx context.Context, input In, opts ...Option) (<-chan StreamChunk[Out], error) {
    out := make(chan StreamChunk[Out])
    
    go func() {
        defer close(out)
        
        // Create input generator
        inputChan := make(chan StreamChunk[In])
        go func() {
            defer close(inputChan)
            inputChan <- StreamChunk[In]{Value: input, Done: false}
            inputChan <- StreamChunk[In]{Done: true}
        }()
        
        // Transform through each step
        var currentChan <-chan StreamChunk[any] = transform(inputChan, r.First)
        for _, step := range r.Middle {
            currentChan = transform(currentChan, step)
        }
        currentChan = transform(currentChan, r.Last)
        
        // Pipe to output
        for chunk := range currentChan {
            out <- StreamChunk[Out]{Value: chunk.Value.(Out), Done: chunk.Done}
        }
    }()
    
    return out, nil
}

func transform[In, Out any](input <-chan StreamChunk[In], r Runnable[In, Out]) <-chan StreamChunk[Out] {
    out := make(chan StreamChunk[Out])
    go func() {
        defer close(out)
        for chunk := range input {
            if chunk.Done {
                out <- StreamChunk[Out]{Done: true}
                return
            }
            result, err := r.Invoke(context.Background(), chunk.Value)
            if err != nil {
                out <- StreamChunk[Out]{Error: err}
                return
            }
            out <- StreamChunk[Out]{Value: result}
        }
    }()
    return out
}
```

### IterableReadableStream Pattern

```go
type StreamChunk[T any] struct {
    Value T
    Error error
    Done  bool
}

type Stream[T any] interface {
    Next() (StreamChunk[T], error)
    Close() error
}

// Reading from stream:
for {
    chunk := <-stream
    if chunk.Done {
        break
    }
    if chunk.Error != nil {
        return chunk.Error
    }
    fmt.Println(chunk.Value)
}
```

---

## 6. Configuration Propagation

### Merge Algorithm

```go
func MergeConfigs(base RunnableConfig, overrides ...RunnableConfig) RunnableConfig {
    result := base.Clone()
    
    for _, override := range overrides {
        // Merge tags (union)
        result.Tags = append(result.Tags, override.Tags...)
        result.Tags = unique(result.Tags)
        
        // Merge metadata (shallow merge)
        result.Metadata = mergeMaps(result.Metadata, override.Metadata)
        
        // Merge configurable (shallow merge)
        result.Configurable = mergeMaps(result.Configurable, override.Configurable)
        
        // Take minimum timeout
        if override.Timeout > 0 {
            if result.Timeout == 0 || override.Timeout < result.Timeout {
                result.Timeout = override.Timeout
            }
        }
        
        // Merge signals using context cancellation
        if override.Signal != nil {
            result.Signal = mergeContexts(result.Signal, override.Signal)
        }
        
        // Merge callbacks
        result.Callbacks = append(result.Callbacks, override.Callbacks...)
        
        // Take lower recursion limit
        if override.RecursionLimit > 0 && override.RecursionLimit < result.RecursionLimit {
            result.RecursionLimit = override.RecursionLimit
        }
    }
    
    return result
}
```

### Context Propagation Pattern

```go
// Go uses context for propagation (implicit)
func (r *Runnable[In, Out]) Invoke(ctx context.Context, input In, opts ...Option) (Out, error) {
    // Extract config from context
    config := GetConfigFromContext(ctx)
    
    // Merge with options
    mergedConfig := MergeConfigs(config, opts...)
    
    // Propagate to children
    childCtx := SetConfigInContext(ctx, mergedConfig)
    
    return r.runnable.Invoke(childCtx, input)
}
```

---

## 7. Binding Pattern

### WithConfig

```go
func (r *baseRunnable[In, Out]) WithConfig(opts ...Option) Runnable[In, Out] {
    return &RunnableBinding[In, Out]{
        Bound:  r,
        Config: NewConfig(opts...),
    }
}

type RunnableBinding[In, Out any] struct {
    Bound  Runnable[In, Out]
    Config RunnableConfig
    Kwargs map[string]any
}

func (b *RunnableBinding[In, Out]) Invoke(ctx context.Context, input In, opts ...Option) (Out, error) {
    mergedConfig := MergeConfigs(b.Config, NewConfig(opts...))
    return b.Bound.Invoke(ctx, input, WithConfig(mergedConfig))
}

// Usage:
boundChain := chain.WithConfig(WithMaxTokens(1000), WithTemperature(0.7))
result, err := boundChain.Invoke(ctx, input)
```

### WithRetry / WithFallbacks

```go
// Fluent interface pattern
chain := llm.
    WithRetry(RetryPolicy{MaxAttempts: 3}).
    WithFallbacks(FallbackChain(llm2)).
    WithConfig(WithTimeout(30 * time.Second)).
    Pipe(parser)

// Equivalent to:
chain := Sequence(
    &RunnableRetry[string, AIMessage]{
        Bound: &RunnableWithFallbacks[string, AIMessage]{
            Primary:   llm,
            Fallbacks: []Runnable[string, AIMessage]{llm2},
            Config: RunnableConfig{Timeout: 30 * time.Second},
        },
        Policy: RetryPolicy{MaxAttempts: 3},
    },
    parser,
)
```

---

## 8. Key Patterns for Lamar SDK

### Pattern 1: Base Runnable with Defaults

```go
type BaseRunnable[In, Out any] struct {
    // Provide default implementations for batch/stream based on invoke
}

func (r *BaseRunnable[In, Out]) Batch(ctx context.Context, inputs []In, opts ...Option) ([]Out, error) {
    results := make([]Out, len(inputs))
    
    // Use configured concurrency
    concurrency := GetConfig(opts...).MaxConcurrency
    if concurrency == 0 {
        concurrency = 1
    }
    
    sem := make(chan struct{}, concurrency)
    var wg sync.WaitGroup
    errors := make([]error, len(inputs))
    
    for i, input := range inputs {
        wg.Add(1)
        go func(idx int, in In) {
            defer wg.Done()
            sem <- struct{}{}
            result, err := r.Invoke(ctx, in, opts...)
            <-sem
            results[idx] = result
            errors[idx] = err
        }(i, input)
    }
    
    wg.Wait()
    
    // Return first error or results
    for _, err := range errors {
        if err != nil {
            return nil, err
        }
    }
    return results, nil
}
```

### Pattern 2: Coercion Pattern

```go
// Everything is coercible to a Runnable
func CoerceToRunnable[In, Out any](thing any) Runnable[In, Out] {
    switch v := thing.(type) {
    case Runnable[In, Out]:
        return v
    case func(context.Context, In) (Out, error):
        return Lambda(v)
    case func(In) (Out, error):
        return Lambda(func(ctx context.Context, input In) (Out, error) {
            return v(input)
        })
    default:
        panic(fmt.Sprintf("Cannot coerce %T to Runnable", thing))
    }
}
```

### Pattern 3: Callback Hierarchy

```go
type CallbackManager struct {
    handlers []CallbackHandler
}

func (m *CallbackManager) Invoke(ctx context.Context, input any, runnable Runnable[any, any], opts ...Option) (any, error) {
    // Start callback
    runID := uuid.New()
    for _, handler := range m.handlers {
        handler.OnChainStart(ctx, ChainStartEvent{
            RunID:  runID,
            Name:   runnable.GetName(),
            Input:  input,
            Tags:   GetConfig(opts...).Tags,
        })
    }
    
    result, err := runnable.Invoke(ctx, input, opts...)
    
    // End callback
    for _, handler := range m.handlers {
        if err != nil {
            handler.OnChainError(ctx, ChainErrorEvent{
                RunID: runID,
                Error: err,
            })
        } else {
            handler.OnChainEnd(ctx, ChainEndEvent{
                RunID:  runID,
                Output: result,
            })
        }
    }
    
    return result, err
}
```

---

## Summary Table: Runnable Types

| Runnable Type          | Purpose                          | Go Implementation                          |
| ---------------------- | -------------------------------- | ------------------------------------------ |
| `RunnableSequence`     | Sequential execution             | `Sequence(runnables...)`                  |
| `RunnableMap`         | Parallel execution               | `Map{"key": runnable}`                     |
| `RunnableLambda`      | Function wrapper                 | `Lambda(fn)`                               |
| `RunnablePassthrough` | Identity/augment                  | `Passthrough()` or `Assign(mapping)`       |
| `RunnableWithFallbacks` | Error recovery                 | `RunWithFallbacks(fallbacks...)`           |
| `RunnableRetry`       | Retry logic                      | `WithRetry(policy)`                        |
| `RunnableBinding`     | Config binding                   | `WithConfig(opts...)`                      |

---

## Recommended Lamar SDK API

```go
// Core interface
type Runnable[In, Out any] interface {
    Invoke(ctx context.Context, input In, opts ...Option) (Out, error)
    Batch(ctx context.Context, inputs []In, opts ...Option) ([]Out, error)
    Stream(ctx context.Context, input In, opts ...Option) (<-chan StreamChunk[Out], error)
    
    // Composition
    Pipe[Next any](next Runnable[Out, Next]) Runnable[In, Next]
    WithConfig(opts ...Option) Runnable[In, Out]
    WithRetry(policy RetryPolicy) Runnable[In, Out]
    WithFallbacks(fallbacks ...Runnable[In, Out]) Runnable[In, Out]
}

// Construction functions
func Sequence[In, Out any](runnables ...any) Runnable[In, Out]
func Map[In any](steps map[string]any) Runnable[In, map[string]any]
func Lambda[In, Out any](fn func(context.Context, In) (Out, error)) Runnable[In, Out]
func Passthrough[In any]() Runnable[In, In]
func Assign[In map[string]any](steps map[string]any) Runnable[In, In]
```