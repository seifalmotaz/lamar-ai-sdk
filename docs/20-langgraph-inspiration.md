# LangGraphJS Architecture Inspiration

## Executive Summary

LangGraphJS is a framework for building stateful, multi-actor applications with LLMs. It implements a message-passing graph computation model inspired by Google's Pregel system, with built-in persistence, checkpointing, and state management.

---

## 1. Core Architecture: StateGraph

### Graph Structure

The primary class for building agent workflows is `StateGraph`:

```typescript
// State is defined using Annotation.Root with reducer functions
const StateAnnotation = Annotation.Root({
  messages: Annotation<BaseMessage[]>({
    reducer: (a, b) => a.concat(b),
    default: () => [],
  }),
  context: Annotation<string>({
    reducer: (_, b) => b,  // Overwrite
    default: () => "",
  }),
});

// Create and compile graph
const graph = new StateGraph(StateAnnotation)
  .addNode("agent", agentNode)
  .addNode("tools", toolNode)
  .addEdge(START, "agent")
  .addConditionalEdges("agent", shouldContinue)
  .addEdge("tools", "agent")
  .compile({ checkpointer });
```

**Key Concepts:**

- **Nodes**: JavaScript/TypeScript functions that receive state and return partial updates
- **Edges**: Unconditional connections between nodes
- **Conditional Edges**: Functions that determine routing based on state
- **Compilation**: Validates structure and creates executable Pregel instance

### Annotation System

State is managed through "channels" defined by annotations:

```typescript
// Annotation creates channels with reducers
const Annotation = {
  Root: <S extends StateDefinition>(spec: S) => spec,
};

// Channel types:
// 1. LastValue<T> - Overwrites on each update (no reducer)
// 2. BinaryOperatorAggregate<T> - Applies reducer to combine values

const countAnnotation = Annotation<number>({
  reducer: (a, b) => a + b,  // Accumulate
  default: () => 0,
});
```

**For Lamar SDK:**

```go
// Go equivalent using functional options
type StateAnnotation[T any] struct {
    Reducer func(a, b T) T
    Default func() T
}

func Annotation[T any](opts ...AnnotationOption[T]) StateAnnotation[T] {
    // Build annotation from options
}

// Usage:
type AgentState struct {
    Messages []Message
    Context  string
}

state := NewGraph(
    WithState(MessagesAnnotation(
        WithReducer(MessageReducer),
        WithDefault(func() []Message { return []Message{} }),
    )),
)
```

---

## 2. Pregel Execution Engine

### Execution Model

The Pregel class implements a **superstep-based** computation model:

```
┌─────────────────────────────────────────────────────────────────────┐
│                           PREGEL EXECUTION                          │
├─────────────────────────────────────────────────────────────────────┤
│  INPUT                                                               │
│    │                                                                 │
│    ▼                                                                 │
│  ┌────────────────┐                                                 │
│  │  Checkpoint    │ ← Load previous state or create empty           │
│  │  Channels      │ ← Apply input writes                            │
│  └────────────────┘                                                 │
│    │                                                                 │
│    ▼                                                                 │
│  ┌─────────────────────────────────────────────────────────┐        │
│  │                   SUPERSTEP LOOP                         │        │
│  │                                                          │        │
│  │   1. Prepare Tasks:                                      │        │
│  │      - PUSH from Topic<Send> (pending Send packets)     │        │
│  │      - PULL from channel versions (nodes with new input) │        │
│  │                                                          │        │
│  │   2. Execute Tasks (can be parallel):                    │        │
│  │      - Read state from channels                          │        │
│  │      - Call node function                                │        │
│  │      - Collect writes (state updates, Send packets)       │        │
│  │                                                          │        │
│  │   3. Apply Writes:                                        │        │
│  │      - Group by channel                                   │        │
│  │      - Apply to channels (call channel.update())         │        │
│  │      - Update channel_versions                           │        │
│  │                                                          │        │
│  │   4. Check Termination:                                  │        │
│  │      - Any tasks generated? Continue loop                 │        │
│  │      - No tasks? Done                                    │        │
│  │      - Interrupt detected? Pause and save                │        │
│  │                                                          │        │
│  │   5. Save Checkpoint:                                    │        │
│  │      - Create snapshot of channel_values                  │        │
│  │      - Save pending_writes                                │        │
│  │      - Persist via checkpointer                          │        │
│  │                                                          │        │
│  └─────────────────────────────────────────────────────────┘        │
│                                                                      │
│    ▼                                                                 │
│  OUTPUT                                                              │
│    - Read final values from output_channels                          │
│    - Return or stream to caller                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Key Implementation Patterns

```go
// Go implementation pattern
type Pregel struct {
    nodes       map[string]PregelNode
    channels    map[string]Channel
    checkpointer Checkpointer
    interruptBefore []string
    interruptAfter  []string
}

type PregelNode struct {
    triggers   []string          // Channels that trigger this node
    channels   []string          // State keys to read
    writers    []ChannelWrite    // How to write output
    runnable   NodeFunc         // The actual function
    retry      *RetryPolicy
    cache      *CachePolicy
}

func (p *Pregel) Invoke(ctx context.Context, input State) (State, error) {
    // 1. Load checkpoint or create empty
    checkpoint := p.checkpointer.Get(config)
    
    // 2. Apply input to channels
    for name, value := range input {
        p.channels[name].Update([]any{value})
    }
    
    // 3. Superstep loop
    for {
        tasks := p.prepareTasks()
        if len(tasks) == 0 {
            break
        }
        
        // Execute tasks in parallel
        p.executeTasks(ctx, tasks)
        
        // Apply writes
        p.applyWrites(tasks)
        
        // Save checkpoint
        if p.checkpointer != nil {
            p.checkpointer.Put(config, checkpoint)
        }
    }
    
    // 4. Return output channels
    return p.readOutput(), nil
}
```

---

## 3. Persistence & Checkpointing

### Checkpointer Interface

```typescript
abstract class BaseCheckpointSaver<V = number> {
  // Get single checkpoint
  abstract getTuple(config): Promise<CheckpointTuple | undefined>;
  
  // List all checkpoints for a thread
  abstract list(config, options?): AsyncGenerator<CheckpointTuple>;
  
  // Save checkpoint
  abstract put(config, checkpoint, metadata, newVersions): Promise<RunnableConfig>;
  
  // Store intermediate writes
  abstract putWrites(config, writes, taskId): Promise<void>;
  
  // Delete thread
  abstract deleteThread(threadId): Promise<void>;
}
```

### Checkpoint Structure

```go
type Checkpoint struct {
    Version         int                        // Currently 4
    ID              string                     // UUID6
    Timestamp       time.Time
    ChannelValues   map[string]any             // Actual state
    ChannelVersions map[string]int             // Version per channel
    VersionsSeen    map[string]map[string]int  // Per-node tracking
}

type CheckpointTuple struct {
    Config         RunnableConfig
    Checkpoint     Checkpoint
    Metadata       CheckpointMetadata
    ParentConfig   *RunnableConfig
    PendingWrites  []PendingWrite
}

type CheckpointMetadata struct {
    Source  string            // "input" | "loop" | "update" | "fork"
    Step    int               // -1 for input, 0+ for loop steps
    Parents map[string]string // Checkpoint namespace -> parent ID
}
```

### Thread Management

```go
type Config struct {
    ThreadID      string            // Required: unique conversation identifier
    CheckpointNS   string           // Optional: namespace for sub-graphs
    CheckpointID   string           // Optional: specific checkpoint (time-travel)
    // ... custom configurable values
}
```

**For Lamar SDK:**

```go
// Thread state management
type ThreadStore interface {
    Get(threadID string) (*Checkpoint, error)
    Put(threadID string, checkpoint *Checkpoint) error
    List(threadID string, opts ListOptions) ([]CheckpointTuple, error)
    Delete(threadID string) error
}

// Implementations:
// - MemoryStore (testing)
// - SQLiteStore (development)
// - PostgresStore (production)
// - RedisStore (distributed)
```

### Time-Travel Debugging

```go
// Get all checkpoints for a thread
history, _ := graph.GetStateHistory(ctx, config)
for _, snapshot := range history {
    fmt.Printf("Step %d: %s\n", snapshot.Metadata.Step, snapshot.Values)
}

// Replay from specific checkpoint
config.CheckpointID = "1ef663ba-28fe-6528-8002-5a559208592c"
result, _ := graph.Invoke(ctx, input, config)

// Fork from past checkpoint
graph.UpdateState(ctx, config, StateUpdate{
    Values: map[string]any{"messages": []Message{newMessage}},
    AsNode: "human",
})
```

---

## 4. Control Flow Patterns

### Command Pattern

Nodes can return `Command` objects to control flow:

```typescript
class Command<Resume, Update, Nodes> {
  resume?: Resume;        // Resume after interrupt
  update?: Update;         // State update
  goto?: Nodes | Send;     // Next node(s)
  graph?: string;          // Target graph (PARENT for subgraphs)
}

// Usage in node:
return new Command({
  update: { messages: [new AIMessage("Done")] },
  goto: "nextNode",
});
```

**For Lamar SDK:**

```go
type Command struct {
    Resume any
    Update map[string]any
    Goto   []string
    Graph  string  // "" for current, "PARENT" for parent graph
}

func (c *Command) Apply(state State) State {
    // Merge updates into state
    for key, value := range c.Update {
        state[key] = value
    }
    return state
}
```

### Send Pattern (Parallel Execution)

```go
type Send struct {
    Node string
    Args State
}

// Use for parallel tool execution:
func agentNode(state State) []Send {
    calls := state["tool_calls"].([]ToolCall)
    sends := make([]Send, len(calls))
    for i, call := range calls {
        sends[i] = Send{Node: "tools", Args: State{"tool_call": call}}
    }
    return sends
}
```

### Interrupt Pattern (Human-in-the-Loop)

```go
func interrupt[T any](value T) T {
    scratchpad := getScratchpad()
    scratchpad.InterruptCounter++
    
    idx := scratchpad.InterruptCounter
    
    // Check for resume value
    if len(scratchpad.Resume) > idx {
        return scratchpad.Resume[idx].(T)
    }
    
    // No resume - throw interrupt
    panic(GraphInterrupt{ID: generateID(), Value: value})
}

// Resume with Command:
// graph.Invoke(ctx, Command{Resume: "User input"}, config)
```

---

## 5. Prebuilt Agents

### createReactAgent

The primary prebuilt agent:

```go
type ReactAgentConfig struct {
    Model           LanguageModel
    Tools           []Tool
    Checkpointer    Checkpointer
    StateSchema     Annotation    // Custom state
    Prompt          string
    PreModelHook    NodeFunc
    PostModelHook   NodeFunc
    InterruptBefore []string
    InterruptAfter  []string
    Name            string
}

func NewReactAgent(config ReactAgentConfig) *CompiledGraph {
    // 1. Build state schema
    if config.StateSchema == nil {
        config.StateSchema = MessagesAnnotation
    }
    
    // 2. Create workflow
    workflow := NewStateGraph(config.StateSchema).
        AddNode("agent", callModel).
        AddEdge(START, "agent")
    
    // 3. Add tools if present
    if len(config.Tools) > 0 {
        toolNode := NewToolNode(config.Tools)
        workflow.AddNode("tools", toolNode).
            AddConditionalEdges("agent", toolsCondition).
            AddEdge("tools", "agent")
    }
    
    // 4. Compile
    return workflow.Compile(WithCheckpointer(config.Checkpointer))
}

func toolsCondition(state State) string {
    lastMessage := state["messages"].([]Message)[len(state["messages"])-1]
    if isAIMessage(lastMessage) && len(lastMessage.ToolCalls) > 0 {
        return "tools"
    }
    return END
}
```

### ToolNode Execution

```go
type ToolNode struct {
    Tools          []Tool
    HandleErrors   bool
    WrapToolCall   ToolCallWrapper
}

func (n *ToolNode) Run(state State, config RunnableConfig) (State, error) {
    // Get pending tool calls
    messages := state["messages"].([]Message)
    lastAIMsg := findLastAIMessage(messages)
    
    // Execute tools in parallel
    outputs := make([]ToolMessage, len(lastAIMsg.ToolCalls))
    for i, call := range lastAIMsg.ToolCalls {
        output, err := n.runTool(call, config, state)
        if err != nil && !n.HandleErrors {
            return nil, err
        }
        outputs[i] = output
    }
    
    return State{"messages": outputs}, nil
}
```

---

## 6. Key Architectural Lessons

### 1. State Channel Pattern

State is managed through typed channels with reducers:

```go
type Channel interface {
    Update(values []any) bool
    Get() any
    Checkpoint() []byte
    Restore(data []byte) error
}

// Implementations:
type LastValueChannel[T any] struct { value T }
type AccumulateChannel[T any] struct { values []T }
func (c *AccumulateChannel[T]) Update(values []any) bool {
    for _, v := range values {
        c.values = append(c.values, v.(T))
    }
    return true
}
```

### 2. Graph Compilation Validates Structure

```go
func (g *StateGraph) Compile(opts ...CompileOption) (*CompiledGraph, error) {
    // Validation:
    // - No orphaned nodes
    // - All nodes reachable
    // - No cycles in unconditional edges
    // - Valid interrupt configuration
    
    // Create executable Pregel instance
    compiled := &CompiledGraph{
        builder: g,
        nodes:   make(map[string]PregelNode),
        channels: g.channels.Clone(),
    }
    
    // Attach nodes and edges
    for name, node := range g.nodes {
        compiled.attachNode(name, node)
    }
    for from, to := range g.edges {
        compiled.attachEdge(from, to)
    }
    for from, branch := range g.branches {
        compiled.attachBranch(from, branch)
    }
    
    return compiled.validate(), nil
}
```

### 3. Checkpoint at Every Step

Critical for:
- Resumption after interrupt
- Time-travel debugging
- Human-in-the-loop workflows

```go
func (p *Pregel) tick(ctx context.Context, tasks []Task) error {
    // Execute all tasks
    for _, task := range tasks {
        result := task.Runnable.Invoke(ctx, task.Input)
        p.applyWrites(task)
    }
    
    // Always checkpoint after step
    if p.checkpointer != nil {
        p.saveCheckpoint()
    }
    
    return nil
}
```

### 4. Namespace for Subgraphs

```go
// Subgraphs have their own checkpoint namespace
config.CheckpointNS = "agent:123|subgraph:456"

// Hierarchy preserved in checkpoint structure
// Parent checkpoint ID stored in metadata
```

### 5. Versioned State Updates

Every state update increments channel versions:

```go
type Checkpoint struct {
    ChannelVersions map[string]int
    VersionsSeen    map[string]map[string]int
}

// Prevents re-processing of already-seen state
// Enables efficient checkpoint comparison
```

---

## 7. Recommended Patterns for Lamar SDK

### Pattern 1: Interface Segregation

```go
// Separate interfaces for different capabilities
type Graph interface {
    Invoke(ctx context.Context, input State) (State, error)
    Stream(ctx context.Context, input State) <-chan StreamEvent
}

type CheckpointableGraph interface {
    Graph
    GetState(config Config) (*StateSnapshot, error)
    GetStateHistory(config Config) ([]StateSnapshot, error)
    UpdateState(config Config, update StateUpdate) error
}

type InterruptableGraph interface {
    Graph
    Interrupts() []Interrupt
    Resume(ctx context.Context, values []any) error
}
```

### Pattern 2: Functional Options

```go
func NewGraph[S any](opts ...GraphOption[S]) *StateGraph[S] {
    g := &StateGraph[S]{}
    for _, opt := range opts {
        opt(g)
    }
    return g
}

type GraphOption[S any] func(*StateGraph[S])

func WithCheckpointer(c Checkpointer) GraphOption[S] {
    return func(g *StateGraph[S]) { g.checkpointer = c }
}
```

### Pattern 3: Channel Reducers

```go
type AnnotationOption[T any] func(*StateAnnotation[T])

func WithReducer[T any](fn func(a, b T) T) AnnotationOption[T] {
    return func(a *StateAnnotation[T]) { a.Reducer = fn }
}

func WithDefault[T any](fn func() T) AnnotationOption[T] {
    return func(a *StateAnnotation[T]) { a.Default = fn }
}

// Usage:
state := NewGraph[AgentState](
    WithAnnotation("messages", 
        WithReducer(func(a, b []Message) []Message { return append(a, b...) }),
        WithDefault(func() []Message { return []Message{} }),
    ),
)
```

### Pattern 4: Context Propagation

```go
type ContextKey string

const (
    ThreadIDKey     ContextKey = "thread_id"
    CheckpointKey   ContextKey = "checkpoint"
    InterruptsKey    ContextKey = "interrupts"
    ResumeKey       ContextKey = "resume_values"
)

// Use context for implicit parameter passing
func (n *Node) Invoke(ctx context.Context, state State) State {
    // Access checkpointer via context
    checkpoint := ctx.Value(CheckpointKey).(*Checkpoint)
    // ...
}
```