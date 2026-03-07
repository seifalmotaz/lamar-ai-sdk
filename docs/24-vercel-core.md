# Vercel AI SDK Core Functions

## Executive Summary

The Vercel AI SDK provides streamlined functions for LLM interactions with strong TypeScript typing, streaming support, and comprehensive tool integration. The three core functions are `generateText`, `streamText`, and `generateObject` (deprecated in favor of `generateText` with `output` parameter).

---

## 1. generateText Function

### Function Signature

```typescript
async function generateText<TOOLS extends ToolSet, OUTPUT extends Output>(
  options: CallSettings & Prompt & {
    model: LanguageModel;
    tools?: TOOLS;
    toolChoice?: ToolChoice<NoInfer<TOOLS>>;
    stopWhen?: StopCondition | Array<StopCondition>;
    
    // Structured output
    output?: OUTPUT;
    
    // Lifecycle callbacks
    experimental_onStart?: OnStartCallback;
    experimental_onStepStart?: OnStepStartCallback;
    onStepFinish?: OnStepFinishCallback;
    experimental_onToolCallStart?: OnToolCallStartCallback;
    experimental_onToolCallFinish?: OnToolCallFinishCallback;
    onFinish?: OnFinishCallback;
    
    // Advanced
    prepareStep?: PrepareStepFunction;
    experimental_context?: unknown;
    experimental_repairToolCall?: ToolCallRepairFunction;
  }
): Promise<GenerateTextResult<TOOLS, OUTPUT>>
```

### Go Equivalent

```go
func GenerateText[T, U any](ctx context.Context, opts GenerateTextOptions) (*GenerateTextResult[T], error)

type GenerateTextOptions struct {
    Model       LanguageModel
    Prompt      []Message
    
    // Tools
    Tools       []Tool
    ToolChoice  ToolChoice
    StopWhen    []StopCondition
    
    // Structured output
    Output      Output
    
    // Lifecycle
    OnStart     func(OnStartEvent)
    OnStepStart func(OnStepStartEvent)
    OnStepFinish func(OnStepFinishEvent)
    OnToolCallStart func(OnToolCallStartEvent)
    OnToolCallFinish func(OnToolCallFinishEvent)
    OnFinish    func(OnFinishEvent)
    
    // Advanced
    PrepareStep func(PrepareStepEvent) (*PrepareStepResult, error)
    Context     any
    MaxTokens   int
    Temperature float64
    // ...
}

type GenerateTextResult[T any] struct {
    Text        string
    Steps       []StepResult
    ToolCalls   []ToolCall
    ToolResults []ToolResult
    FinishReason FinishReason
    Usage       Usage
    Warnings    []Warning
}
```

---

## 2. Multi-Step Execution (Tool Loop)

### Execution Flow

```go
func GenerateText(ctx context.Context, opts GenerateTextOptions) (*GenerateTextResult, error) {
    steps := []StepResult{}
    messages := opts.Prompt
    
    for {
        // 1. Prepare step (allows dynamic config)
        if opts.PrepareStep != nil {
            stepConfig, err := opts.PrepareStep(PrepareStepEvent{
                Steps: steps,
                StepNumber: len(steps) + 1,
            })
            if err != nil {
                return nil, err
            }
            // Apply step config (model change, tool change, etc.)
            opts = applyStepConfig(opts, stepConfig)
        }
        
        // 2. Call model
        opts.OnStepStart(OnStepStartEvent{StepNumber: len(steps) + 1, Messages: messages})
        
        response, err := opts.Model.DoGenerate(ctx, DoGenerateOptions{
            Prompt: messages,
            Tools: opts.Tools,
            ToolChoice: opts.ToolChoice,
        })
        if err != nil {
            return nil, err
        }
        
        // 3. Parse response
        step := StepResult{
            StepNumber: len(steps) + 1,
            Text: response.Text,
            ToolCalls: response.ToolCalls,
            FinishReason: response.FinishReason,
            Usage: response.Usage,
        }
        
        // 4. Handle tool calls
        if len(response.ToolCalls) > 0 {
            // Execute tools
            toolResults := []ToolResult{}
            for _, call := range response.ToolCalls {
                opts.OnToolCallStart(OnToolCallStartEvent{ToolCall: call})
                
                result, err := executeTool(ctx, call, opts.Tools, call.Input)
                
                opts.OnToolCallFinish(OnToolCallFinishEvent{
                    ToolCall: call,
                    Success: err == nil,
                    Output: result,
                    Error: err,
                })
                
                toolResults = append(toolResults, ToolResult{
                    CallID: call.ID,
                    Name: call.Name,
                    Result: result,
                })
            }
            
            // Add to messages for next iteration
            messages = append(messages, AIMessage{ToolCalls: response.ToolCalls})
            messages = append(messages, ToolMessage{ToolResults: toolResults})
            
            step.ToolResults = toolResults
        }
        
        steps = append(steps, step)
        opts.OnStepFinish(step)
        
        // 5. Check stop condition
        if shouldStop(opts.StopWhen, steps) {
            break
        }
        
        // Check if model wants to stop
        if response.FinishReason != "tool-calls" {
            break
        }
        
        // Check if tool has no execute function (end)
        if hasNonExecutableTools(response.ToolCalls, opts.Tools) {
            break
        }
    }
    
    opts.OnFinish(OnFinishEvent{
        Steps: steps,
        TotalUsage: calculateTotalUsage(steps),
    })
    
    return &GenerateTextResult{
        Text: steps[len(steps) - 1].Text,
        Steps: steps,
        FinishReason: steps[len(steps) - 1].FinishReason,
        Usage: calculateTotalUsage(steps),
    }, nil
}

// Stop conditions
func stepCountIs(n int) StopCondition {
    return func(opts StopConditionOptions) bool {
        return len(opts.Steps) >= n
    }
}

func hasToolCall(name string) StopCondition {
    return func(opts StopConditionOptions) bool {
        for _, step := range opts.Steps {
            for _, call := range step.ToolCalls {
                if call.Name == name {
                    return true
                }
            }
        }
        return false
    }
}
```

---

## 3. streamText Function

### Function Signature

```go
func StreamText(ctx context.Context, opts StreamTextOptions) (*StreamTextResult, error)

type StreamTextResult struct {
    // AsyncIterable/ReadableStream-like
    TextStream   <-chan TextStreamPart
    FullStream   <-chan StreamPart
    
    // Promises (resolve when stream completes)
    Text         Promise[string]
    Usage        Promise[Usage]
    Steps        Promise[[]StepResult]
    
    // Control
    Cancel       func()
}

type StreamPart interface {
    Type() string
}

type TextStreamPart struct {
    Type   string // "text-delta" | "text-start" | "text-end"
    ID     string
    Delta  string  // for text-delta
}

type ToolCallStreamPart struct {
    Type      string // "tool-input-start" | "tool-input-delta" | "tool-input-end" | "tool-call"
    ID        string
    ToolName  string
    Delta     string  // for tool-input-delta
    Input     any     // for tool-call
}

type FinishStreamPart struct {
    Type         string // "finish"
    FinishReason string
    Usage        Usage
}
```

### Streaming Implementation

```go
func StreamText(ctx context.Context, opts StreamTextOptions) (*StreamTextResult, error) {
    textStream := make(chan StreamPart, 100)
    
    textPromise := NewDelayedPromise[string]()
    usagePromise := NewDelayedPromise[Usage]()
    stepsPromise := NewDelayedPromise[[]StepResult]()
    
    go func() {
        defer close(textStream)
        
        // Main streaming loop
        for {
            // 1. Stream model response
            modelStream, err := opts.Model.DoStream(ctx, DoStreamOptions{
                Prompt: messages,
                Tools: opts.Tools,
            })
            if err != nil {
                textStream <- ErrorStreamPart{Error: err}
                return
            }
            
            currentText := strings.Builder{}
            currentToolCalls := make(map[string]*ToolCallBuilder)
            
            // Process model stream
            for part := range modelStream {
                switch p := part.(type) {
                case TextDeltaPart:
                    currentText.WriteString(p.Delta)
                    textStream <- TextStreamPart{Type: "text-delta", Delta: p.Delta}
                    
                case ToolInputStartPart:
                    currentToolCalls[p.ID] = &ToolCallBuilder{
                        ID: p.ID,
                        Name: p.ToolName,
                    }
                    textStream <- ToolCallStreamPart{Type: "tool-input-start", ID: p.ID, ToolName: p.ToolName}
                    
                case ToolInputDeltaPart:
                    if builder, ok := currentToolCalls[p.ID]; ok {
                        builder.AppendInput(p.Delta)
                        textStream <- ToolCallStreamPart{Type: "tool-input-delta", ID: p.ID, Delta: p.Delta}
                    }
                    
                case ToolCallPart:
                    // Complete tool call
                    textStream <- ToolCallStreamPart{Type: "tool-call", ID: p.ID, Name: p.Name, Input: p.Input}
                    toolCalls = append(toolCalls, p.ToolCall)
                }
            }
            
            // 2. Execute tools if present
            if len(toolCalls) > 0 {
                // Execute in parallel
                results := make([]ToolResult, len(toolCalls))
                for i, call := range toolCalls {
                    go func(idx int, tc ToolCall) {
                        textStream <- ToolCallStreamPart{Type: "tool-execution-start", ID: tc.ID}
                        result, err := executeTool(ctx, tc, opts.Tools, tc.Input)
                        textStream <- ToolResultStreamPart{Type: "tool-result", ID: tc.ID, Result: result, Error: err}
                        results[idx] = result
                    }(i, call)
                }
                
                // Wait for all
                // ...
                
                // Add to messages
                messages = append(messages, AIMessage{ToolCalls: toolCalls})
                messages = append(messages, ToolMessage{ToolResults: results})
            }
            
            // 3. Check for continuation
            if shouldStop(opts.StopWhen, steps) || finishReason != "tool-calls" {
                textStream <- FinishStreamPart{Type: "finish", FinishReason: finishReason, Usage: usage}
                break
            }
        }
    }()
    
    return &StreamTextResult{
        TextStream: textStream,
        Text:       textPromise,
        Usage:      usagePromise,
        Steps:      stepsPromise,
    }, nil
}
```

---

## 4. Tool Definition API

### Tool Structure

```go
type Tool[In, Out any] struct {
    Name         string
    Description  string
    InputSchema  Schema  // Zod-style schema
    
    // Execution
    Execute      func(ctx context.Context, input In, runtime ToolRuntime) (Out, error)
    
    // Streaming
    Stream       func(ctx context.Context, input In, runtime ToolRuntime) (<-chan Out, error)  // Optional
    
    // Lifecycle hooks
    OnInputStart    func(runtime ToolRuntime)
    OnInputDelta   func(delta string, runtime ToolRuntime)
    OnInputAvailable func(input In, runtime ToolRuntime)
    
    // Control
    ReturnDirect   bool    // Skip LLM after execution
    NeedsApproval  bool    // Human-in-the-loop
}

// Construction
func DefineTool[In, Out any](name, description string, schema Schema, fn func(ctx context.Context, input In, runtime ToolRuntime) (Out, error)) *Tool[In, Out]

// Dynamic tools
func DynamicTool(name, description string, schema Schema, fn func(ctx context.Context, input any, runtime ToolRuntime) (any, error)) *DynamicTool

// Provider-executed tools
func ProviderTool(name string, id string, args map[string]any, supportsDeferred bool) *ProviderTool
```

---

## 5. Output Specification

### Structured Output

```go
type Output interface {
    Name() string
    ResponseFormat() ResponseFormat
    ParseComplete(text string) (any, error)
    ParsePartial(text string) (any, error)  // For streaming
}

// Output types
func ObjectOutput[T any](schema Schema, name string) Output
func ArrayOutput[T any](elementSchema Schema, name string) Output
func TextOutput() Output
func EnumOutput[T ~string](enums []T) Output
func JSONOutput() Output
```

### Generate with Output

```go
result, err := GenerateText(ctx, GenerateTextOptions{
    Model: model,
    Prompt: []Message{UserMessage("List 5 countries")},
    Output: ObjectOutput[CountryList](Schema{
        Type: "object",
        Properties: map[string]Schema{
            "countries": {
                Type: "array",
                Items: Schema{
                    Type: "object",
                    Properties: map[string]Schema{
                        "name": {Type: "string"},
                        "capital": {Type: "string"},
                    },
                },
            },
        },
    }, "country_list"),
})
// result.Typed = CountryList{Countries: [...]}

// Streaming with output
stream, _ := StreamText(ctx, StreamTextOptions{
    Model: model,
    Prompt: []Message{UserMessage("List 5 countries")},
    Output: ObjectOutput[CountryList](schema, "country_list"),
})

for part := range stream.FullStream {
    if part.Type == "partial-object" {
        fmt.Printf("Partial: %+v\n", part.Value)
    }
}
```

---

## 6. Key Patterns

### Pattern 1: Delayed Promise

For stream results where text/usage/steps are available as promises:

```go
type Promise[T any] interface {
    Await() (T, error)
}

type DelayedPromise[T any] struct {
    value    T
    err      error
    done     chan struct{}
}

func (p *DelayedPromise[T]) Await() (T, error) {
    <-p.done
    return p.value, p.err
}

func (p *DelayedPromise[T]) Resolve(value T) {
    p.value = value
    close(p.done)
}

func (p *DelayedPromise[T]) Reject(err error) {
    p.err = err
    close(p.done)
}
```

### Pattern 2: Lifecycle Callbacks

```go
// Non-blocking callback execution
func notify[T any](event T, callbacks []func(T)) {
    for _, cb := range callbacks {
        go cb(event)  // Fire and forget
    }
}

// Usage
for _, cb := range opts.OnStepFinish {
    go cb(step)  // Don't block execution
}
```

### Pattern 3: Tool Streaming

```go
// Tool can stream intermediate results
func SearchTool[In, Out any]() *Tool[In, Out] {
    return DefineTool[In, Out]("search", "...", schema, func(ctx context.Context, input In, runtime ToolRuntime) (<-chan Out, error) {
        out := make(chan Out)
        
        go func() {
            defer close(out)
            
            // Stream results
            results := searchAPI(input.Query)
            for i, result := range results {
                // Stream preliminary result
                out <- PreliminaryResult{Result: result, Index: i}
            }
            
            // Final result
            out <- FinalResult{Results: results}
        }()
        
        return out, nil
    })
}
```

### Pattern 4: Prepare Step (Dynamic Config)

```go
// Switch models mid-execution
result, _ := GenerateText(ctx, GenerateTextOptions{
    Model: fastModel,
    Tools: tools,
    PrepareStep: func(event PrepareStepEvent) (*PrepareStepResult, error) {
        // After 2 steps, switch to stronger model
        if len(event.Steps) > 2 {
            return &PrepareStepResult{
                Model: strongModel,
            }, nil
        }
        return nil, nil
    },
})

// Phase-based tool activation
PrepareStep: func(event PrepareStepEvent) (*PrepareStepResult, error) {
    stepNum := len(event.Steps) + 1
    
    switch {
    case stepNum <= 2:
        return &PrepareStepResult{
            ActiveTools: []string{"search"},
            ToolChoice: Required(),
        }, nil
    case stepNum <= 5:
        return &PrepareStepResult{
            ActiveTools: []string{"analyze"},
        }, nil
    default:
        return &PrepareStepResult{
            ActiveTools: []string{"summarize"},
            ToolChoice: Required(),
        }, nil
    }
}
```

---

## 7. Recommended Lamar SDK API

```go
// Core functions
func GenerateText(ctx context.Context, opts GenerateTextOptions) (*GenerateTextResult, error)
func StreamText(ctx context.Context, opts StreamTextOptions) (*StreamTextResult, error)

// Options
type GenerateTextOptions struct {
    Model       LanguageModel
    Prompt      []Message
    Tools       []Tool
    ToolChoice  ToolChoice
    StopWhen    []StopCondition
    Output      Output
    
    OnStepStart, OnStepFinish func(StepEvent)
    OnToolCallStart, OnToolCallFinish func(ToolCallEvent)
    OnFinish func(FinishEvent)
    
    PrepareStep func(PrepareStepEvent) (*PrepareStepResult, error)
    Context     any
    
    // Standard settings
    MaxTokens   int
    Temperature float64
    // ...
}

// Results
type GenerateTextResult struct {
    Text   string
    Steps  []StepResult
    Usage  Usage
    // ... async promises for stream
}

type StreamTextResult struct {
    TextStream <-chan TextPart
    FullStream <-chan StreamPart
    
    Text   Promise[string]
    Usage  Promise[Usage]
    Steps  Promise[[]StepResult]
}

// Stop conditions
func StepCountIs(n int) StopCondition
func HasToolCall(name string) StopCondition
func CostExceeds(limit float64) StopCondition
func CustomCondition(fn func([]StepResult) bool) StopCondition

// Tool construction
func DefineTool[In, Out any](name, description string, schema Schema, fn func(context.Context, In, ToolRuntime) (Out, error)) *Tool[In, Out]
func WithReturnDirect(t *Tool[In, Out]) *Tool[In, Out]
func WithNeedsApproval(t *Tool[In, Out]) *Tool[In, Out]

// Output construction
func ObjectOutput[T any](schema Schema, name string) Output
func ArrayOutput[T any](elementSchema Schema, name string) Output
```