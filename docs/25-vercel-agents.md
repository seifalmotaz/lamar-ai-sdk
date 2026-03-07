# Vercel AI SDK Agent Patterns

## Executive Summary

The Vercel AI SDK implements agents through the `ToolLoopAgent` class, which provides multi-turn LLM conversations with automatic tool execution loops, lifecycle callbacks, and advanced control flow patterns.

---

## 1. ToolLoopAgent Architecture

### Core Class

```go
type ToolLoopAgent[CallOptions, Tools ToolSet, Output any] struct {
    version     string // "agent-v1"
    id          string
    tools       Tools
    
    // Configuration
    instructions string
    model        LanguageModel
    toolChoice   ToolChoice
    stopWhen     []StopCondition
    output       Output
    
    // Advanced
    prepareStep        PrepareStepFunc
    context            any
    
    // Lifecycle
    onStart            func(OnStartEvent)
    onStepStart        func(OnStepStartEvent)
    onStepFinish       func(OnStepFinishEvent)
    onToolCallStart    func(OnToolCallStartEvent)
    onToolCallFinish   func(OnToolCallFinishEvent)
    onFinish           func(OnFinishEvent)
    
    // Custom options
    callOptionsSchema  Schema
    prepareCall        func(opts CallOptions) (*PrepareCallResult, error)
}

func NewAgent[CallOptions, Tools ToolSet, Output any](opts AgentOptions[CallOptions, Tools, Output]) *ToolLoopAgent[CallOptions, Tools, Output] {
    return &ToolLoopAgent[CallOptions, Tools, Output]{
        version: "agent-v1",
        // ... apply options
    }
}
```

---

## 2. Stop Conditions

### Condition Pattern

```go
type StopCondition func(opts StopConditionOptions) bool

type StopConditionOptions struct {
    Steps []StepResult
}

// Built-in conditions
func StepCountIs(n int) StopCondition {
    return func(opts StopConditionOptions) bool {
        return len(opts.Steps) >= n
    }
}

func HasToolCall(name string) StopCondition {
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

func HasToolOutput(name string, matcher func(output any) bool) StopCondition {
    return func(opts StopConditionOptions) bool {
        for _, step := range opts.Steps {
            for _, result := range step.ToolResults {
                if result.Name == name && matcher(result.Output) {
                    return true
                }
            }
        }
        return false
    }
}

// Cost-based condition
func CostExceeds(limit float64, costFn func(Usage) float64) StopCondition {
    return func(opts StopConditionOptions) bool {
        var total float64
        for _, step := range opts.Steps {
            total += costFn(step.Usage)
        }
        return total >= limit
    }
}

// Token limit condition
func TokenLimitExceeds(limit int) StopCondition {
    return func(opts StopConditionOptions) bool {
        var total int
        for _, step := range opts.Steps {
            total += step.Usage.TotalTokens
        }
        return total >= limit
    }
}

// Combined conditions (AND/OR)
func AllConditions(conditions ...StopCondition) StopCondition {
    return func(opts StopConditionOptions) bool {
        for _, cond := range conditions {
            if !cond(opts) {
                return false
            }
        }
        return true
    }
}

func AnyCondition(conditions ...StopCondition) StopCondition {
    return func(opts StopConditionOptions) bool {
        for _, cond := range conditions {
            if cond(opts) {
                return true
            }
        }
        return false
    }
}
```

### Usage

```go
agent := NewAgent(AgentOptions{
    Model:    model,
    Tools:    tools,
    StopWhen: []StopCondition{
        StepCountIs(10),
        HasToolCall("done"),
        CostExceeds(1.0, calculateCost),
    },
})
```

---

## 3. Prepare Step (Dynamic Configuration)

### PrepareStep Function

```go
type PrepareStepFunc func(event PrepareStepEvent) (*PrepareStepResult, error)

type PrepareStepEvent struct {
    Steps      []StepResult
    StepNumber int
    Model      LanguageModel
    Messages   []Message
    Context    any
}

type PrepareStepResult struct {
    Model           LanguageModel         // Switch models
    ToolChoice      ToolChoice            // Force specific tool
    ActiveTools     []string              // Limit available tools
    System          string                // Override system prompt
    Messages        []Message             // Modify message history
    Context         any                   // Update flowing context
    ProviderOptions map[string]any        // Provider-specific config
}

// Usage patterns
prepareStep := func(event PrepareStepEvent) (*PrepareStepResult, error) {
    // Pattern 1: Switch models after N steps
    if len(event.Steps) > 2 {
        return &PrepareStepResult{
            Model: strongModel,  // Use stronger model for final steps
        }, nil
    }
    return nil, nil
}

prepareStep := func(event PrepareStepEvent) (*PrepareStepResult, error) {
    // Pattern 2: Phase-based tool activation
    stepNum := event.StepNumber
    
    switch {
    case stepNum <= 3:
        return &PrepareStepResult{
            ActiveTools: []string{"search"},
            ToolChoice:  Required(),
        }, nil
    case stepNum <= 7:
        return &PrepareStepResult{
            ActiveTools: []string{"analyze", "summarize"},
        }, nil
    default:
        return &PrepareStepResult{
            ActiveTools: []string{"answer"},
            ToolChoice:  Required(),
        }, nil
    }
}

prepareStep := func(event PrepareStepEvent) (*PrepareStepResult, error) {
    // Pattern 3: Trim message history
    if len(event.Messages) > MAX_CONTEXT {
        return &PrepareStepResult{
            Messages: append(
                []Message{event.Messages[0]},  // Keep system
                event.Messages[len(event.Messages)-RECENT:]...,  // Keep recent
            ),
        }, nil
    }
    return nil, nil
}
```

---

## 4. Experimental Context

```go
// Context flows through entire generation lifecycle
type ContextualAgent struct {
    agent *ToolLoopAgent
}

func (a *ContextualAgent) Generate(ctx context.Context, input string, opts ...Option) (*GenerateTextResult, error) {
    return a.agent.Generate(ctx, GenerateTextOptions{
        Prompt:             []Message{UserMessage(input)},
        ExperimentalContext: map[string]any{
            "userId":    getUserFromContext(ctx),
            "sessionId": getSession(ctx),
            "timestamp": time.Now(),
        },
        PrepareStep: func(event PrepareStepEvent) (*PrepareStepResult, error) {
            // Access context in prepare step
            ctx := event.Context.(map[string]any)
            log.Printf("User: %s, Step: %d", ctx["userId"], event.StepNumber)
            return nil, nil
        },
    })
}

// Access context in tools
searchTool := DefineTool[SearchInput, SearchResult]("search", "Search for information", schema, 
    func(ctx context.Context, input SearchInput, runtime ToolRuntime) (SearchResult, error) {
        // Flowing context available in runtime
        context := runtime.Context.(map[string]any)
        userId := context["userId"].(string)
        
        // Use context for filtering, logging, etc.
        return searchAPI(input.Query, userId)
    },
)
```

---

## 5. Lifecycle Callbacks

### Event Types

```go
type OnStartEvent struct {
    Model          LanguageModel
    System         string
    Prompt         string
    Messages       []Message
    Tools          ToolSet
    ToolChoice     ToolChoice
    ActiveTools    []string
    StopWhen       []StopCondition
    Output         Output
    Context        any
}

type OnStepStartEvent struct {
    StepNumber int
    Model      LanguageModel
    Messages   []Message
    Steps      []StepResult
    Context    any
}

type OnToolCallStartEvent struct {
    StepNumber int
    Model      LanguageModel
    ToolCall   ToolCall
    Context    any
}

type OnToolCallFinishEvent struct {
    StepNumber int
    ToolCall   ToolCall
    Duration   time.Duration
    Success    bool
    Output     any
    Error      error
    Context    any
}

type OnStepFinishEvent struct {
    StepResult
}

type OnFinishEvent struct {
    Steps      []StepResult
    TotalUsage Usage
    Context    any
}
```

### Non-Blocking Callback Execution

```go
// Fire callbacks in parallel (don't block main execution)
func notify[T any](event T, callbacks []func(T)) {
    if len(callbacks) == 0 {
        return
    }
    
    for _, cb := range callbacks {
        go cb(event)  // Non-blocking
    }
}

// Usage in agent
func (a *ToolLoopAgent) Generate(ctx context.Context, opts GenerateTextOptions) (*GenerateTextResult, error) {
    // ... setup
    
    notify(OnStartEvent{...}, a.onStart)
    
    for {
        notify(OnStepStartEvent{...}, a.onStepStart)
        
        // ... model call
        
        notify(OnToolCallStartEvent{...}, a.onToolCallStart)
        // ... execute tool
        notify(OnToolCallFinishEvent{...}, a.onToolCallFinish)
        
        notify(OnStepFinishEvent{...}, a.onStepFinish)
        
        // ... check stop condition
    }
    
    notify(OnFinishEvent{...}, a.onFinish)
}
```

---

## 6. Agent Patterns

### Pattern 1: Termination Tool

```go
// Agent must call 'done' tool to finish
doneTool := DefineTool[DoneInput, any]("done", "Signal completion", Schema{
    Type: "object",
    Properties: map[string]Schema{
        "answer": {Type: "string", Description: "Final answer"},
    },
    Required: []string{"answer"},
}, func(ctx context.Context, input DoneInput, runtime ToolRuntime) (any, error) {
    return input, nil  // No execution needed
})

agent := NewAgent(AgentOptions{
    Model:      model,
    Tools:      append(tools, doneTool),
    ToolChoice: Required(),  // Force tool use every step
    StopWhen:   []StopCondition{HasToolCall("done")},
})

// Result is in last step's tool call
result, _ := agent.Generate(ctx, "What is the capital of France?")
lastStep := result.Steps[len(result.Steps) - 1]
answer := lastStep.ToolResults[0].Output.(DoneInput).Answer
```

### Pattern 2: Cost Control

```go
const MAX_COST = 0.10 // $0.10

prepareStep := func(event PrepareStepEvent) (*PrepareStepResult, error) {
    // Calculate cost so far
    var totalCost float64
    for _, step := range event.Steps {
        totalCost += calculateCost(step.Usage)
    }
    
    if totalCost > MAX_COST * 0.8 {  // 80% threshold
        log.Printf("Approaching cost limit: $%.4f / $%.2f", totalCost, MAX_COST)
        
        // Switch to cheaper model
        return &PrepareStepResult{
            Model: cheapModel,
        }, nil
    }
    
    return nil, nil
}

stopWhen := []StopCondition{
    CostExceeds(MAX_COST, calculateCost),
}
```

### Pattern 3: Nested Agents

```go
// Child agent for subtasks
childAgent := NewAgent(AgentOptions{
    Model:    model,
    Tools:    analysisTools,
    StopWhen: []StopCondition{StepCountIs(5)},
})

// Parent agent delegates to child
parentAgent := NewAgent(AgentOptions{
    Model: model,
    Tools: []Tool{
        DefineTool[TaskInput, string]("delegate", "Delegate task to sub-agent", schema, 
            func(ctx context.Context, input TaskInput, runtime ToolRuntime) (string, error) {
                result, err := childAgent.Stream(ctx, StreamTextOptions{
                    Prompt: []Message{UserMessage(input.Task)},
                })
                if err != nil {
                    return "", err
                }
                return await(result.Text), nil
            },
        ),
    },
})
```

### Pattern 4: Human Approval Workflow

```go
// Tool requiring approval
sensitiveTool := DefineTool[DeleteInput, any]("delete_file", "Delete a file", schema,
    func(ctx context.Context, input DeleteInput, runtime ToolRuntime) (any, error) {
        return deleteFile(input.Path)
    },
)
sensitiveTool.NeedsApproval = true

// Agent handles approval requests in stream
stream, _ := agent.Stream(ctx, StreamTextOptions{
    Prompt: []Message{UserMessage("Delete unnecessary files")},
})

for part := range stream.FullStream {
    switch p := part.(type) {
    case ToolApprovalRequestPart:
        // Emit approval request to UI
        approval := waitForUserApproval(p.Request)
        
        if approval.Approved {
            stream.Resume(approval.Value)
        } else {
            stream.Resume(Rejected)
        }
        
    case TextPart:
        fmt.Print(p.Delta)
    }
}
```

---

## 7. Implementation for Lamar SDK

```go
// Agent interface
type Agent[CallOptions, Tools ToolSet, Output any] interface {
    Version() string
    ID() string
    Tools() Tools
    
    Generate(ctx context.Context, opts AgentGenerateOptions) (*GenerateTextResult, error)
    Stream(ctx context.Context, opts AgentStreamOptions) (*StreamTextResult, error)
}

// Agent options
type AgentOptions[CallOptions, Tools ToolSet, Output any] struct {
    ID             string
    Instructions   string
    Model          LanguageModel
    Tools          Tools
    ToolChoice     ToolChoice
    StopWhen       []StopCondition
    Output         Output
    
    PrepareStep    PrepareStepFunc
    Context        any
    
    // Lifecycle
    OnStart        func(OnStartEvent)
    OnStepStart    func(OnStepStartEvent)
    OnStepFinish   func(OnStepFinishEvent)
    OnToolCallStart func(OnToolCallStartEvent)
    OnToolCallFinish func(OnToolCallFinishEvent)
    OnFinish       func(OnFinishEvent)
    
    // Custom call options
    CallOptionsSchema Schema
    PrepareCall       func(opts CallOptions) (*PrepareCallResult, error)
    
    // LLM settings
    MaxTokens   int
    Temperature float64
    // ...
}

// Construction
func NewAgent[CallOptions, Tools ToolSet, Output any](opts AgentOptions[CallOptions, Tools, Output]) Agent[CallOptions, Tools, Output]

// Stop conditions
func StepCountIs(n int) StopCondition
func HasToolCall(name string) StopCondition
func CostExceeds(limit float64) StopCondition
func AllConditions(conditions ...StopCondition) StopCondition
func AnyCondition(conditions ...StopCondition) StopCondition

// Prepare step
type PrepareStepFunc func(event PrepareStepEvent) (*PrepareStepResult, error)

// Context flow
type ToolRuntime struct {
    ToolCallID string
    State      any
    Context    any
    Config     RunnableConfig
    Store      Store
    Writer     func(any)
    Signal     context.Context
}
```