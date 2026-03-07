# Agent Framework Inspiration Summary: Best Features Cocktail

## Executive Summary

After exploring LangGraphJS, LangChainJS, and the Vercel AI SDK, this document synthesizes the best architectural patterns and features for building a production-ready agent framework called **Hydra** - named after the mythical multi-headed creature, representing multiple agents/tools working in coordination.

---

## Architecture Comparison

| Aspect | LangGraphJS | LangChainJS | Vercel AI SDK |
|--------|-------------|--------------|---------------|
| **Core Abstraction** | StateGraph + Pregel | Runnable + LCEL | generateText/streamText |
| **State Management** | Channel-based reducers | Memory types | Thread Checkpointer |
| **Persistence** | SQLite/Postgres/Redis | ChatHistory backends | Thread-scoped |
| **Tool Execution** | ToolNode (parallel) | StructuredTool | Tool loop + callbacks |
| **Human-in-the-loop** | `interrupt()` + `Command` | Not built-in | `needsApproval` |
| **Streaming** | Async generators | Transform pipelines | ReadableStream parts |
| **Graph Flow** | Explicit nodes/edges | Runnable sequences | Implicit loop + StopCondition |

---

## Key Features Cocktail

### From LangGraphJS

| Feature | Pattern |
|---------|---------|
| **State Channels with Reducers** | `StateAnnotation[T]` with `WithReducer` for append/overwrite semantics |
| **Graph Compilation** | `NewGraph().Compile()` validates structure, creates executable Pregel |
| **Checkpoint Persistence** | `Checkpointer` interface, thread-scoped, supports time-travel |
| **Human-in-the-Loop** | `runtime.Interrupt()` in tools, `Command{Resume: value}` to continue |
| **Command Pattern** | Return `Command{Goto: "node"}` for explicit flow control |
| **Send Pattern** | Parallel execution via `Send{Node: "name", Args: state}` |
| **Subgraph Namespacing** | `CheckpointNS` for nested graphs |

### From LangChainJS

| Feature | Pattern |
|---------|---------|
| **Runnable Composition** | `Sequence`, `Map`, `Lambda`, `Pipe`, `Passthrough`, `Assign` |
| **Retry/Fallback** | `WithRetry(policy)`, `WithFallback(fallbacks...)` |
| **Configuration Propagation** | `RunnableConfig` merges through chain, uses context |
| **Structured Tool Input** | `Tool[In, Out]` with `InputSchema` + Zod/JSON Schema |
| **Tool Runtime Injection** | `ToolRuntime{State, Context, Store, Writer, Interrupt}` |
| **Memory Types** | Buffer, Window, Summary, TokenBuffer, Vector, Combined |
| **Document Model** | `Document{Content, Metadata, ID}` simple interface |

### From Vercel AI SDK

| Feature | Pattern |
|---------|---------|
| **Stop Conditions** | `StepCountIs(n)`, `HasToolCall(name)`, `CostExceeds(limit)` |
| **Prepare Step** | Dynamic model/tool/message config between steps |
| **Lifecycle Callbacks** | `OnStart`, `OnStepStart/Finish`, `OnToolCallStart/Finish`, `OnFinish` |
| **Flowing Context** | `ExperimentalContext` passes through entire execution |
| **Stream Parts** | Unified `TextPart`, `ReasoningPart`, `ToolCallPart`, `FinishPart` |
| **Delayed Promises** | `Promise[T]` resolves when stream completes |
| **Provider Tools** | `type: 'provider'` for server-executed tools |
| **Structured Output** | `Output` interface with parseComplete/parsePartial |
| **Agent Encapsulation** | `ToolLoopAgent` class wraps model + tools + callbacks |

---

## Recommended Hydra Architecture

### Layer 1: Graph (LangGraph-inspired)

```go
// State annotation pattern
type StateAnnotation[T any] struct {
    Reducer func(a, b T) T
    Default func() T
}

func WithReducer[T any](fn func(a, b T) T) AnnotationOption[T]
func WithDefault[T any](fn func() T) AnnotationOption[T]

// Graph definition
state := NewGraph[AgentState](
    WithAnnotation("messages",
        WithReducer(MessageReducer),  // append
        WithDefault(func() []Message { return []Message{} }),
    ),
    WithAnnotation("context",
        WithReducer(Overwrite),  // replace
        WithDefault(func() string { return "" }),
    ),
)

// Graph construction
graph := NewGraph[AgentState](state,
    AddNode("agent", agentNode),
    AddNode("tools", toolNode),
    AddEdge(Start, "agent"),
    AddConditionalEdges("agent", shouldContinue),
    AddEdge("tools", "agent"),
).Compile(
    WithCheckpointer(postgresCheckpoint),
    WithInterruptBefore("human_approval"),
)
```

### Layer 2: Runnable (LangChain-inspired)

```go
// Core interface
type Runnable[In, Out any] interface {
    Invoke(ctx context.Context, input In, opts ...Option) (Out, error)
    Stream(ctx context.Context, input In, opts ...Option) (<-chan StreamPart[Out], error)
    Batch(ctx context.Context, inputs []In, opts ...Option) ([]Out, error)
    
    Pipe[Next any](next Runnable[Out, Next]) Runnable[In, Next]
    WithConfig(opts ...Option) Runnable[In, Out]
    WithRetry(policy RetryPolicy) Runnable[In, Out]
    WithFallback(fallbacks ...Runnable[In, Out]) Runnable[In, Out]
}

// Composition
func Sequence[In, Out any](runnables ...any) Runnable[In, Out]
func Map[In any](steps map[string]Runnable[In, any]) Runnable[In, map[string]any]
func Lambda[In, Out any](fn func(ctx context.Context, input In) (Out, error)) Runnable[In, Out]
func Assign[In map[string]any](steps map[string]Runnable[In, any]) Runnable[In, In]
```

### Layer 3: Tools (Best of All)

```go
// Tool definition with streaming and lifecycle
type Tool[In, Out any] struct {
    Name        string
    Description string
    InputSchema Schema
    
    Execute func(ctx context.Context, input In, runtime ToolRuntime) (Out, error)
    Stream  func(ctx context.Context, input In, runtime ToolRuntime) (<-chan Out, error)
    
    OnInputStart     func(runtime ToolRuntime)
    OnInputAvailable func(input In, runtime ToolRuntime)
    
    ReturnDirect  bool
    NeedsApproval bool
}

// ToolRuntime injection
type ToolRuntime struct {
    ToolCallID string
    ThreadID   string
    State      any
    Context    any
    Store      Store
    Writer     func(any)
    Interrupt  func(value any) any
}

// Tool construction
func DefineTool[In, Out any](name, desc string, schema Schema, 
    fn func(ctx context.Context, input In, runtime ToolRuntime) (Out, error)) *Tool[In, Out]
func WithReturnDirect() ToolOption
func WithNeedsApproval() ToolOption
```

### Layer 4: Agents (Vercel-inspired)

```go
// ToolLoopAgent
type ToolLoopAgent[Tools ToolSet] struct {
    Model       LanguageModel
    Tools       Tools
    StopWhen    []StopCondition
    PrepareStep PrepareStepFunc
    Context     any
    
    OnStart, OnStepStart, OnStepFinish func(Event)
    OnToolCallStart, OnToolCallFinish  func(Event)
    OnFinish func(Event)
}

func NewToolLoopAgent[Tools ToolSet](opts AgentOptions) *ToolLoopAgent[Tools]

// Stop conditions
func StepCountIs(n int) StopCondition
func HasToolCall(name string) StopCondition
func CostExceeds(limit float64, costFn func(Usage) float64) StopCondition

// Prepare step for dynamic config
type PrepareStepFunc func(event PrepareStepEvent) (*PrepareStepResult, error)
```

### Layer 5: Persistence (LangGraph-inspired)

```go
type Checkpointer interface {
    GetTuple(ctx context.Context, config ThreadConfig) (*CheckpointTuple, error)
    List(ctx context.Context, config ThreadConfig) ([]CheckpointTuple, error)
    Put(ctx context.Context, config ThreadConfig, checkpoint *Checkpoint) error
    DeleteThread(ctx context.Context, threadID string) error
}

type ThreadConfig struct {
    ThreadID     string
    CheckpointNS string
    CheckpointID string  // for time-travel
}

func NewMemoryCheckpointer() Checkpointer
func NewSQLiteCheckpointer(db *sql.DB) Checkpointer
func NewPostgresCheckpointer(pool *pgxpool.Pool) Checkpointer
```

### Layer 6: Streaming (Vercel-inspired)

```go
type StreamPart interface {
    Type() string
}

type TextPart struct {
    Type  string // "text-delta"
    Delta string
}

type ToolCallPart struct {
    Type   string // "tool-call"
    ID     string
    Name   string
    Input  any
}

type FinishPart struct {
    Type       string // "finish"
    Usage      Usage
    StopReason string
}

type StreamTextResult struct {
    TextStream <-chan StreamPart
    FullStream <-chan StreamPart
    
    Text   Promise[string]
    Usage  Promise[Usage]
    Steps  Promise[[]StepResult]
}
```

---

## System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────┐
│                        HYDRA AGENT FRAMEWORK                       │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐        │
│  │   Graph Layer  │ │ Runnable Layer │ │   Tool Layer   │        │
│  │                │ │                │ │                │        │
│  │ - StateGraph   │ │ - Sequence     │ │ - DefineTool   │        │
│  │ - Channels     │ │ - Map/Lambda   │ │ - ToolRuntime   │        │
│  │ - Nodes/Edges  │ │ - Pipe         │ │ - Streaming     │        │
│  │ - Checkpointer │ │ - Retry        │ │ - Approval     │        │
│  │ - Interrupt    │ │ - Fallback     │ │ - Middleware   │        │
│  └────────────────┘ └────────────────┘ └────────────────┘        │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                       AGENT LAYER                             │ │
│  │                                                               │ │
│  │  ToolLoopAgent ──── GraphAgent ──── SupervisorAgent          │ │
│  │  (StopCondition)   (StateGraph)    (Handoffs)               │ │
│  │                                                               │ │
│  │  Features:                                                    │ │
│  │  - PrepareStep (dynamic config)                              │ │
│  │  - Lifecycle Callbacks                                       │ │
│  │  - Flowing Context                                           │ │
│  │  - Structured Output                                         │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                     PERSISTENCE LAYER                         │ │
│  │                                                               │ │
│  │  MemoryCheckpointer ──── SQLiteCheckpointer ──── Postgres    │ │
│  │                                                               │ │
│  │  Features:                                                    │ │
│  │  - Thread-scoped state                                       │ │
│  │  - Time-travel debugging                                     │ │
│  │  - Subgraph namespacing                                      │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                     PROVIDER LAYER                            │ │
│  │                                                               │ │
│  │  OpenAI Provider ──── Anthropic Provider ──── Google Provider │ │
│  │                                                               │ │
│  │  (Leverages existing Lamar SDK provider interface)           │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Phases

### Phase 1: Core Abstractions (Week 1-2)
- [ ] StateAnnotation with reducers
- [ ] StateGraph with nodes/edges compilation
- [ ] Runnable interface (Invoke, Stream, Batch, Pipe)
- [ ] StreamPart types (TextPart, ToolCallPart, FinishPart)
- [ ] Delayed Promise pattern

### Phase 2: Tool System (Week 3-4)
- [ ] Tool[In, Out] definition with InputSchema
- [ ] ToolRuntime injection (State, Context, Store, Writer, Interrupt)
- [ ] Streaming tools via generator pattern
- [ ] Tool middleware (caching, logging, rate limiting)
- [ ] Tool execution with parallel mode

### Phase 3: Graph Execution (Week 5-6)
- [ ] Pregel-style superstep execution
- [ ] Channel update and version tracking
- [ ] Checkpointer interface
- [ ] MemoryCheckpointer implementation
- [ ] Interrupt/Command pattern for flow control

### Phase 4: Agents (Week 7-8)
- [ ] ToolLoopAgent with stop conditions
- [ ] PrepareStep for dynamic configuration
- [ ] Lifecycle callbacks
- [ ] Flowing context through execution
- [ ] Structured output support

### Phase 5: Persistence (Week 9-10)
- [ ] SQLiteCheckpointer implementation
- [ ] PostgresCheckpointer implementation
- [ ] Time-travel APIs (GetStateHistory, UpdateState)
- [ ] Fork from past checkpoint
- [ ] Cross-thread Store interface

### Phase 6: Polish & Framework Integration (Week 11-12)
- [ ] React SDK for useChat/useCompletion hooks
- [ ] Vue SDK for reactive wrappers
- [ ] Svelte SDK for $state runes
- [ ] Comprehensive documentation
- [ ] Integration tests

---

## Conclusion

The **Hydra Agent Framework** synthesizes the best patterns from three ecosystems:

| From | We Take |
|------|---------|
| **LangGraphJS** | State channels, checkpointers, interrupt/resume, explicit graph, time-travel |
| **LangChainJS** | Runnable composition, structured tools, retry/fallback, memory types |
| **Vercel AI SDK** | Stop conditions, prepare step, lifecycle callbacks, stream parts, agent encapsulation |

This combination provides a production-ready, extensible foundation for building sophisticated multi-agent systems in Go that supports:
- Explicit graph-based workflows
- Composable runnables
- Thread-scoped persistence
- Human-in-the-loop workflows
- Streaming execution
- Framework-agnostic core with framework-specific SDKs