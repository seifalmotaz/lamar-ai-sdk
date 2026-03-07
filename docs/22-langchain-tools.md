# LangChainJS Tools Architecture

## Executive Summary

LangChainJS provides a sophisticated tool system with structured input schemas, runtime context injection, middleware hooks, and seamless integration with LLMs and agents.

---

## 1. Tool Interface Hierarchy

### Core Interfaces

```typescript
// Base interface for all tools
interface StructuredToolInterface<SchemaT, SchemaInputT, ToolOutputT> {
  readonly lc_namespace: string[];
  
  // Schema definitions
  schema: SchemaT;           // Zod or JSON schema
  
  // Identity
  name: string;
  description: string;
  
  // Behavior
  returnDirect: boolean;      // Stop agent loop after execution
  responseFormat?: "content" | "content_and_artifact";
  
  // Provider-specific fields
  extras?: Record<string, unknown>;
  
  // Core method
  invoke(input: StructuredToolCallInput, config?: ToolRunnableConfig): Promise<ToolReturnType>;
}
```

### Simplified Tool Interface

```go
// Go equivalent
type Tool interface {
    // Identity
    Name() string
    Description() string
    
    // Schema
    InputSchema() Schema
    OutputSchema() Schema // Optional
    
    // Execution
    Invoke(ctx context.Context, input ToolCall, opts ...ToolOption) (ToolResult, error)
}

// Tool call structure
type ToolCall struct {
    ID      string
    Name    string
    Args    map[string]any
    Type    string // "tool_call"
}

// Tool result structure
type ToolResult struct {
    CallID    string
    Name      string
    Output    any
    Artifact  any    // Optional non-LLM-visible output
    Status    string // "success" | "error"
}

// Tool definition
type ToolDefinition struct {
    Name        string
    Description string
    InputSchema Schema
    Execute     func(ctx context.Context, input any, runtime ToolRuntime) (any, error)
    
    // Optional
    OutputSchema    Schema
    ReturnDirect    bool
    ResponseFormat  string // "content" | "content_and_artifact"
    NeedsApproval   bool
    Extras          map[string]any
}
```

---

## 2. Schema Definitions

### Zod vs JSON Schema

**LangChainJS supports both:**

```typescript
// Zod schema (type-safe)
import { z } from "zod";

const weatherSchema = z.object({
  location: z.string().describe("The city and state"),
  unit: z.enum(["celsius", "fahrenheit"]).optional(),
});

// JSON schema (runtime)
const weatherSchemaJSON = {
  type: "object",
  properties: {
    location: { type: "string", description: "The city and state" },
    unit: { type: "string", enum: ["celsius", "fahrenheit"] }
  },
  required: ["location"]
};
```

**For Lamar SDK (using Go struct tags):**

```go
// Using jsonschema struct tags
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=The city and state"`
    Unit     string `json:"unit,omitempty" jsonschema:"enum=celsius,enum=fahrenheit"`
}

// Schema extraction
func ExtractSchema[T any]() (*Schema, error) {
    var zero T
    return jsonschema.Reflect(zero), nil
}

// Usage:
func NewWeatherTool() *Tool[WeatherInput] {
    return DefineTool[WeatherInput](
        "get_weather",
        "Get current weather for a location",
        func(ctx context.Context, input WeatherInput, runtime ToolRuntime) (any, error) {
            return fetchWeather(input.Location, input.Unit)
        },
    )
}
```

---

## 3. Tool Creation Patterns

### Pattern 1: Function-Based Tools

```go
// Simple tool from function
func Tool[In any](name, description string, schema Schema, fn ToolFunc[In]) *BaseTool[In] {
    return &BaseTool[In]{
        name:        name,
        description: description,
        schema:      schema,
        execute:     fn,
    }
}

// Usage:
weatherTool := Tool(
    "get_weather",
    "Get current weather for a location",
    WeatherInputSchema,
    func(ctx context.Context, input WeatherInput) (any, error) {
        return fetchWeather(input.Location, input.Unit)
    },
)
```

### Pattern 2: Struct-Based Tools

```go
// Struct-based tool (class in TS)
type WeatherTool struct {
    apiKey string
}

func (t *WeatherTool) Name() string { return "get_weather" }
func (t *WeatherTool) Description() string { return "Get current weather" }
func (t *WeatherTool) InputSchema() Schema { return WeatherInputSchema }

func (t *WeatherTool) Invoke(ctx context.Context, call ToolCall, opts ...ToolOption) (ToolResult, error) {
    var input WeatherInput
    if err := call.BindArgs(&input); err != nil {
        return ToolResult{}, err
    }
    
    result, err := t.fetchWeather(input.Location, input.Unit)
    if err != nil {
        return ToolResult{
            CallID: call.ID,
            Status: "error",
            Output: err.Error(),
        }, nil
    }
    
    return ToolResult{
        CallID: call.ID,
        Status: "success",
        Output: result,
        Name:   call.Name,
    }, nil
}
```

### Pattern 3: Dynamic Tools

```go
// Runtime-defined tools (e.g., MCP tools)
func DynamicTool(name, description string, schema Schema, fn ToolFunc[any]) *DynamicTool {
    return &DynamicTool{
        name:        name,
        description: description,
        schema:      schema,
        execute:     fn,
        // No compile-time type safety
    }
}
```

---

## 4. Runtime Context Injection

### ToolRuntime Structure

```go
// Automatically injected by agent framework
type ToolRuntime struct {
    ToolCallID  string
    State       any                 // Graph state
    Context     any                 // Read-only runtime context
    Config      RunnableConfig      // Callbacks, timeout, etc.
    Store       Store               // Persistent storage
    Writer      func(any)          // Stream writer
    Signal      context.Context     // Cancellation
    
    // Methods
    Interrupt(value any) any       // Human-in-the-loop
    Emit(event any)                 // Stream event
}

// Usage in tool:
func (t *MyTool) Invoke(ctx context.Context, call ToolCall, opts ...ToolOption) (ToolResult, error) {
    runtime := GetRuntime(opts...) // Extract from options
    
    // Access state
    state := runtime.State.(*AgentState)
    messages := state.Messages
    
    // Access tool call ID
    fmt.Printf("Tool call ID: %s\n", runtime.ToolCallID)
    
    // Stream intermediate results
    runtime.Writer("Processing step 1...")
    // ... do work
    runtime.Writer("Processing step 2...")
    
    // Use persistent storage
    memory, _ := runtime.Store.Get([]string{t.userID, "memories"}, "preferences")
    
    // Human-in-the-loop
    if needsApproval(state) {
        response := t.runtime.Interrupt("Please approve this action")
        if !response.(bool) {
            return ToolResult{Status: "error", Output: "Action rejected"}, nil
        }
    }
    
    return ToolResult{Output: result}, nil
}
```

---

## 5. Tool Execution Flow

### ToolNode Pattern

```go
// LangGraph's ToolNode pattern
type ToolNode struct {
    Tools       []Tool
    HandleErrors bool
    WrapToolCall ToolCallWrapper
}

func (n *ToolNode) Run(state State, config RunnableConfig) (State, error) {
    // Get pending tool calls from last AI message
    messages := state["messages"].([]Message)
    lastAIMsg := findLastAIMessage(messages)
    
    if lastAIMsg == nil || len(lastAIMsg.ToolCalls) == 0 {
        return state, nil
    }
    
    // Execute tools
    toolResults := make([]ToolResult, 0, len(lastAIMsg.ToolCalls))
    
    for _, call := range lastAIMsg.ToolCalls {
        // Find tool
        tool := n.findTool(call.Name)
        if tool == nil {
            toolResults = append(toolResults, ToolResult{
                CallID: call.ID,
                Name:   call.Name,
                Status: "error",
                Output: fmt.Sprintf("Tool %s not found", call.Name),
            })
            continue
        }
        
        // Execute tool
        result, err := n.executeTool(tool, call, state, config)
        if err != nil && !n.HandleErrors {
            return nil, err
        }
        
        toolResults = append(toolResults, result)
    }
    
    // Convert to messages
    messages = append(messages, toolResultsToMessages(toolResults)...)
    return State{"messages": messages}, nil
}

func (n *ToolNode) executeTool(tool Tool, call ToolCall, state State, config RunnableConfig) (ToolResult, error) {
    // Build runtime
    runtime := ToolRuntime{
        ToolCallID: call.ID,
        State:      state,
        Config:     config,
        // ...
    }
    
    // Wrap with middleware if present
    var handler ToolHandler = func(req ToolCallRequest) (ToolResult, error) {
        return tool.Invoke(context.Background(), call, WithRuntime(runtime))
    }
    
    if n.WrapToolCall != nil {
        handler = n.WrapToolCall(handler)
    }
    
    return handler(ToolCallRequest{
        ToolCall: call,
        Tool:     tool,
        State:    state,
        Runtime:  runtime,
    })
}
```

---

## 6. Middleware Pattern

### Tool Middleware Hooks

```go
type ToolMiddleware interface {
    // Wrap tool execution
    WrapToolCall(handler ToolHandler) ToolHandler
}

type ToolHandler func(req ToolCallRequest) (ToolResult, error)
type ToolCallRequest struct {
    ToolCall ToolCall
    Tool     Tool
    State    any
    Runtime  ToolRuntime
}

// Example: Caching middleware
type CachingMiddleware struct {
    Cache Cache
}

func (m *CachingMiddleware) WrapToolCall(handler ToolHandler) ToolHandler {
    return func(req ToolCallRequest) (ToolResult, error) {
        // Check cache
        key := fmt.Sprintf("%s:%s", req.ToolCall.Name, req.ToolCall.Args)
        if cached, ok := m.Cache.Get(key); ok {
            return cached.(ToolResult), nil
        }
        
        // Execute and cache
        result, err := handler(req)
        if err == nil {
            m.Cache.Set(key, result)
        }
        return result, err
    }
}

// Example: Logging middleware
type LoggingMiddleware struct {
    Logger Logger
}

func (m *LoggingMiddleware) WrapToolCall(handler ToolHandler) ToolHandler {
    return func(req ToolCallRequest) (ToolResult, error) {
        start := time.Now()
        m.Logger.Info("tool_call_start", "tool", req.ToolCall.Name, "args", req.ToolCall.Args)
        
        result, err := handler(req)
        
        m.Logger.Info("tool_call_finish", "tool", req.ToolCall.Name, "duration", time.Since(start))
        return result, err
    }
}

// Example: Rate limiting middleware
type RateLimitMiddleware struct {
    Limiter RateLimiter
}

func (m *RateLimitMiddleware) WrapToolCall(handler ToolHandler) ToolHandler {
    return func(req ToolCallRequest) (ToolResult, error) {
        if !m.Limiter.Allow() {
            return ToolResult{
                CallID: req.ToolCall.ID,
                Status: "error",
                Output: "Rate limit exceeded",
            }, nil
        }
        return handler(req)
    }
}

// Usage:
toolNode := NewToolNode(tools,
    WithMiddleware(&CachingMiddleware{Cache: cache}),
    WithMiddleware(&LoggingMiddleware{Logger: logger}),
)
```

---

## 7. Response Formats

### Content vs Content+Artifact

```go
// Format 1: Content only (default)
result, err := tool.Invoke(ctx, call)
// result.Output = "The weather is 72°F" (sent to LLM)

// Format 2: Content + Artifact (separate LLM-visible and internal data)
func (t *SearchTool) Invoke(ctx context.Context, call ToolCall, opts ...ToolOption) (ToolResult, error) {
    results := t.search(call.Args["query"])
    
    // Content: LLM-visible summary
    content := formatResultsForLLM(results)
    
    // Artifact: Full result objects (not sent to LLM)
    artifact := results
    
    return ToolResult{
        CallID:   call.ID,
        Status:   "success",
        Output:   content,
        Artifact: artifact,
    }, nil
}

// Handling in ToolNode:
func toolResultsToMessages(results []ToolResult) []Message {
    messages := make([]Message, len(results))
    for i, result := range results {
        messages[i] = ToolMessage{
            ToolCallID: result.CallID,
            Name:       result.Name,
            Content:    result.Output,
            Artifact:   result.Artifact, // Preserved but not in content
        }
    }
    return messages
}
```

---

## 8. Return Direct Pattern

```go
// Tool returns final answer directly (skip LLM processing)
func NewCalculatorTool() *Tool {
    return &Tool{
        Name:        "calculate",
        Description: "Perform math calculations",
        ReturnDirect: true, // Stop agent loop after this tool
        Execute: func(ctx context.Context, call ToolCall, runtime ToolRuntime) (any, error) {
            // ... compute result
            return result, nil
        },
    }
}

// In agent loop:
func shouldReturnDirect(toolResults []ToolResult) bool {
    for _, result := range toolResults {
        if result.ReturnDirect {
            return true
        }
    }
    return false
}

func (a *Agent) Step(ctx context.Context, state State) (State, error) {
    // ... execute tools
    
    results := toolNode.Run(state, config)
    state = mergeToolResults(state, results)
    
    if shouldReturnDirect(results) {
        return state, Done // End loop
    }
    
    // ... continue to LLM
}
```

---

## 9. Streaming Tool Execution

```go
// Tool that streams intermediate results
func (t *LongRunningTool) Invoke(ctx context.Context, call ToolCall, opts ...ToolOption) (ToolResult, error) {
    runtime := GetRuntime(opts...)
    
    // Stream progress via runtime.Writer
    runtime.Writer(ProgressUpdate{Stage: "Starting", Percent: 0})
    
    for i := 0; i < 10; i++ {
        time.Sleep(100 * time.Millisecond)
        runtime.Writer(ProgressUpdate{Stage: "Processing", Percent: i * 10})
    }
    
    runtime.Writer(ProgressUpdate{Stage: "Complete", Percent: 100})
    
    return ToolResult{
        CallID: call.ID,
        Output: result,
    }, nil
}

// Client handling:
for chunk := range tool.Stream(...) {
    switch v := chunk.(type) {
    case ProgressUpdate:
        fmt.Printf("Progress: %s - %d%%\n", v.Stage, v.Percent)
    case PreliminaryResult:
        fmt.Printf("Preliminary: %v\n", v.Output)
    case FinalResult:
        fmt.Printf("Final: %v\n", v.Output)
    }
}
```

---

## 10. Tool Approval (Human-in-the-Loop)

```go
// Define tool requiring approval
func NewDeleteFileTool() *Tool {
    return &Tool{
        Name:          "delete_file",
        Description:   "Delete a file from the filesystem",
        NeedsApproval: true, // Requires user confirmation
        Execute: func(ctx context.Context, call ToolCall, runtime ToolRuntime) (any, error) {
            return os.Remove(call.Args["path"])
        },
    }
}

// Agent handling:
func (a *Agent) executeTools(ctx context.Context, toolCalls []ToolCall) ([]ToolResult, error) {
    var results []ToolResult
    
    for _, call := range toolCalls {
        tool := a.findTool(call.Name)
        
        // Check if needs approval
        if tool.NeedsApproval {
            // Create approval request
            approval := ToolApprovalRequest{
                ApprovalID: uuid.New(),
                ToolCall:   call,
                Message:    "This action requires approval",
            }
            
            // Emit approval request (pauses execution)
            a.Emit(ApprovalRequested{Approval: approval})
            
            // Wait for response (via interrupt or external API)
            response := a.WaitForApproval(ctx, approval.ApprovalID)
            
            if !response.Approved {
                results = append(results, ToolResult{
                    CallID: call.ID,
                    Status: "error",
                    Output: "Action rejected by user",
                })
                continue
            }
        }
        
        // Execute tool
        result, err := tool.Execute(ctx, call)
        results = append(results, result)
    }
    
    return results, nil
}
```

---

## Summary: Tool System Architecture

| Component            | Purpose                              | Go Equivalent                         |
| -------------------- | ------------------------------------ | ------------------------------------- |
| `StructuredToolInterface` | Type-safe tool definition        | `Tool[In, Out]` interface             |
| `DynamicTool`       | Runtime-defined tools                | `DynamicTool` struct                  |
| `ToolRuntime`        | Injected context (state, config)     | `ToolRuntime` struct                   |
| `ToolNode`           | Batch tool execution in graph        | `ToolNode` struct                      |
| `returnDirect`       | Short-circuit agent loop              | `ReturnDirect bool` field              |
| `needsApproval`      | Human-in-the-loop approval           | `NeedsApproval bool` field            |
| `responseFormat`     | Separate content and artifact         | `Output` and `Artifact` fields        |
| Middleware           | Wrap tool execution (logging, cache)  | `WrapToolCall(handler) handler` pattern |

---

## Recommended Lamar SDK Tool API

```go
// Core interface
type Tool interface {
    Name() string
    Description() string
    InputSchema() Schema
    Invoke(ctx context.Context, call ToolCall, opts ...ToolOption) (ToolResult, error)
}

// Construction
func DefineTool[In any](name, desc string, fn func(ctx context.Context, input In, runtime ToolRuntime) (any, error)) *TypedTool[In]

// Options
func WithReturnDirect(t *TypedTool[In]) *TypedTool[In]
func WithNeedsApproval(t *TypedTool[In]) *TypedTool[In]
func WithArtifact[In, Out any](t *TypedTool[In]) *TypedToolWithArtifact[In, Out]

// Middleware
type ToolMiddleware interface {
    WrapCall(handler ToolHandler) ToolHandler
}

// Execution
type ToolNode struct {
    Tools       []Tool
    Middleware  []ToolMiddleware
    HandleErrors bool
}

func (n *ToolNode) Execute(ctx context.Context, calls []ToolCall, state State) ([]ToolResult, error)