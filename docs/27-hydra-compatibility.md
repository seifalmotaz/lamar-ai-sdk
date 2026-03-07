# Hydra Framework: Feature Compatibility Analysis

## Executive Summary

This document compares the proposed **Hydra Agent Framework** features from `docs/26-agent-framework-summary.md` against the existing **Lamar SDK** implementation to identify:
1. What's already implemented
2. What can be extended from existing code
3. What needs to be built from scratch
4. Potential compatibility issues

---

## Architecture Overview

### Existing Lamar SDK Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         LAMAR SDK (Current)                        │
├─────────────────────────────────────────────────────────────────────┤
│                                                                      │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐        │
│  │   Provider     │ │   Generate     │ │    Stream      │        │
│  │   Interface    │ │   Function     │ │   Function      │        │
│  │                │ │                │ │                │        │
│  │ - Model        │ │ - Generate()   │ │ - Stream()     │        │
│  │ - Generator    │ │ - Result       │ │ - Result       │        │
│  │ - Streamer      │ │ - Options      │ │ - StreamPart   │        │
│  │ - LanguageModel │ │                │ │                │        │
│  └────────────────┘ └────────────────┘ └────────────────┘        │
│                                                                      │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐        │
│  │   Tool         │ │   Middleware   │ │   Provider      │        │
│  │   System       │ │   Chain        │ │   Impl (OpenAI) │        │
│  │                │ │                │ │                │        │
│  │ - Tool[In,Out] │ │ - Handler      │ │ - Chat         │        │
│  │ - NewTool()    │ │ - Middleware   │ │ - Embed        │        │
│  │ - InputSchema  │ │ - Chain()      │ │ - Stream       │        │
│  └────────────────┘ └────────────────┘ └────────────────┘        │
│                                                                      │
│  ┌────────────────┐ ┌────────────────┐ ┌────────────────┐        │
│  │   Embed        │ │   Image        │ │   Transcription│        │
│  │   Function     │ │   Function     │ │   Function      │        │
│  └────────────────┘ └────────────────┘ └────────────────┘        │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────────┐ │
│  │                     INTERNAL PACKAGES                         │ │
│  │  - schema (jsonschema extraction)                            │ │
│  │  - httpx (HTTP client wrapper)                               │ │
│  │  - sse (Server-Sent Events parser)                           │ │
│  │  - ctxutil (timeout handling)                                │ │
│  │  - contract (testing contracts)                              │ │
│  └──────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Feature-by-Feature Analysis

### Layer 1: Graph (LangGraph-inspired)

| Feature | Proposed | Existing Lamar | Status | Notes |
|---------|----------|----------------|--------|-------|
| **StateAnnotation[T] with Reducers** | `WithReducer(fn)`, `WithDefault(fn)` | ❌ None | 🔴 BUILD | Need new package: `graph` |
| **StateGraph with nodes/edges** | `NewGraph().AddNode().AddEdge().Compile()` | ❌ None | 🔴 BUILD | Core of Hydra |
| **Graph Compilation** | Validate structure, create executable | ❌ None | 🔴 BUILD | Compile() validates |
| **Channel Types** | `LastValue`, `BinaryOperatorAggregate` | ❌ None | 🔴 BUILD | State management |
| **Command Pattern** | `Command{Goto, Resume, Update}` | ❌ None | 🔴 BUILD | Flow control |
| **Send Pattern** | `Send{Node, Args}` for parallel | ❌ None | 🔴 BUILD | Parallel execution |
| **Conditional Edges** | `AddConditionalEdges("node", router)` | ❌ None | 🔴 BUILD | Dynamic routing |
| **Interrupt/Resume** | `runtime.Interrupt()`, `Command{Resume}` | ❌ None | 🔴 BUILD | Human-in-the-loop |
| **Subgraph Namespacing** | `CheckpointNS` for nested graphs | ❌ None | 🔴 BUILD | Multi-agent support |

**Recommendation:** Build new `graph` package from scratch following LangGraph patterns.

---

### Layer 2: Runnable (LangChain-inspired)

| Feature | Proposed | Existing Lamar | Status | Notes |
|---------|----------|----------------|--------|-------|
| **Runnable[In, Out] interface** | `Invoke`, `Stream`, `Batch`, `Pipe` | ⚠️ Partial | 🟡 EXTEND | Provider interfaces similar |
| **Sequence[In, Out]** | Sequential composition | ❌ None | 🔴 BUILD | New composition pattern |
| **Map[In]** | Parallel execution with key-values | ❌ None | 🔴 BUILD | Parallel execution |
| **Lambda[In, Out]** | Wrap function as Runnable | ⚠️ Similar | 🟡 EXTEND | `tool.NewTool` similar pattern |
| **Pipe[Next]** | Chain composition | ❌ None | 🔴 BUILD | Functional composition |
| **WithRetry** | Retry logic wrapper | ✅ Exists | 🟢 REUSE | `middleware.Retry` |
| **WithFallback** | Fallback chain pattern | ⚠️ Partial | 🟡 EXTEND | Need `RunnableWithFallbacks` |
| **WithConfig** | Merge configuration | ⚠️ Partial | 🟡 EXTEND | Existing Option pattern similar |
| **Passthrough/Assign** | Input augmentation | ❌ None | 🔴 BUILD | RAG pattern |
| **Batch invocation** | Parallel invoke with concurrency | ❌ None | 🔴 BUILD | `Runnable.Batch()` |

**Recommendation:** Create new `runnable` package that wraps existing functionality. The Runnable interface can wrap existing Provider interfaces.

---

### Layer 3: Tools (Best of All)

| Feature | Proposed | Existing Lamar | Status | Notes |
|---------|----------|----------------|--------|-------|
| **Tool[In, Out]** | Generic tool with schema | ✅ Exists | 🟢 REUSE | `tool.NewTool[In, Out]()` |
| **InputSchema extraction** | Auto from struct tags | ✅ Exists | 🟢 REUSE | `schema.FromStruct()` |
| **Execute function** | `Execute(ctx, input)` | ✅ Exists | 🟢 REUSE | Same pattern |
| **ToolRuntime injection** | `State, Context, Store, Writer, Interrupt` | ❌ None | 🔴 BUILD | Major new feature |
| **Streaming tools** | `Stream(ctx, input) (<-chan Out, error)` | ❌ None | 🔴 BUILD | New streaming pattern |
| **OnInputStart callback** | Lifecycle hook | ❌ None | 🔴 BUILD | New hook |
| **OnInputAvailable callback** | Lifecycle hook | ❌ None | 🔴 BUILD | New hook |
| **ReturnDirect** | Skip LLM after execution | ❌ None | 🔴 BUILD | Short-circuit pattern |
| **NeedsApproval** | Human-in-the-loop flag | ❌ None | 🔴 BUILD | Approval pattern |
| **ToolDefinition conversion** | Convert to provider.ToolDefinition | ✅ Exists | 🟢 REUSE | `tool.ToDefinition()` |
| **Tool execution in parallel** | Execute all tool calls simultaneously | ❌ None | 🔴 BUILD | Parallel execution |

**Recommendation:** Extend existing `tool` package with ToolRuntime and lifecycle hooks.

---

### Layer 4: Agents (Vercel-inspired)

| Feature | Proposed | Existing Lamar | Status | Notes |
|---------|----------|----------------|--------|-------|
| **ToolLoopAgent** | Multi-step tool execution | ⚠️ Partial | 🟡 EXTEND | Need loop logic |
| **StopCondition** | `StepCountIs(n)`, `HasToolCall(name)` | ❌ None | 🔴 BUILD | New concept |
| **PrepareStep** | Dynamic config per step | ❌ None | 🔴 BUILD | New concept |
| **OnStart callback** | Lifecycle event | ❌ None | 🔴 BUILD | New hook |
| **OnStepStart/Finish callbacks** | Lifecycle events | ❌ None | 🔴 BUILD | New hooks |
| **OnToolCallStart/Finish callbacks** | Lifecycle events | ❌ None | 🔴 BUILD | New hooks |
| **OnFinish callback** | Completion event | ❌ None | 🔴 BUILD | New hook |
| **Flowing Context** | `ExperimentalContext` passes through | ⚠️ Partial | 🟡 EXTEND | Context helpers exist |
| **Tool calling loop** | Call tools until stop condition | ❌ None | 🔴 BUILD | Core agent loop |
| **Multi-step tracking** | Accumulate `StepResult[]` | ❌ None | 🔴 BUILD | Step history |

**Recommendation:** Build `agent` package with ToolLoopAgent wrapping existing Provider.

**Current Gap:** The existing `generate.Generate()` and `stream.Stream()` functions make a single call to the model. They don't handle tool result cycling. This needs to be added.

---

### Layer 5: Persistence (LangGraph-inspired)

| Feature | Proposed | Existing Lamar | Status | Notes |
|---------|----------|----------------|--------|-------|
| **Checkpointer interface** | `GetTuple`, `Put`, `List`, `DeleteThread` | ❌ None | 🔴 BUILD | New interface |
| **ThreadConfig** | `ThreadID`, `CheckpointNS`, `CheckpointID` | ❌ None | 🔴 BUILD | Thread scoping |
| **Checkpoint structure** | ChannelVersions, VersionsSeen | ❌ None | 🔴 BUILD | State versioning |
| **MemoryCheckpointer** | In-memory implementation | ❌ None | 🔴 BUILD | For testing |
| **SQLiteCheckpointer** | SQLite implementation | ❌ None | 🔴 BUILD | Development |
| **PostgresCheckpointer** | PostgreSQL implementation | ❌ None | 🔴 BUILD | Production |
| **Time-travel** | `GetStateHistory`, `UpdateState` | ❌ None | 🔴 BUILD | Debugging |
| **Fork from checkpoint** | Create divergent paths | ❌ None | 🔴 BUILD | Experimentation |
| **Cross-thread Store** | Persist across conversations | ❌ None | 🔴 BUILD | Memory layer |

**Recommendation:** Build new `persistence` package with Checkpointer interface.

---

### Layer 6: Streaming (Vercel-inspired)

| Feature | Proposed | Existing Lamar | Status | Notes |
|---------|----------|----------------|--------|-------|
| **StreamPart interface** | Polymorphic stream parts | ✅ Exists | 🟢 REUSE | `provider.StreamPart` |
| **StreamTextPart** | Text delta | ✅ Exists | 🟢 REUSE | `provider.StreamTextPart{Delta}` |
| **StreamToolCallPart** | Tool call in stream | ✅ Exists | 🟢 REUSE | `provider.StreamToolCallPart{ToolCall}` |
| **StreamFinishPart** | Finish reason + usage | ✅ Exists | 🟢 REUSE | `provider.StreamFinishPart{FinishReason, Usage}` |
| **StreamErrorPart** | Error in stream | ✅ Exists | 🟢 REUSE | `provider.StreamErrorPart{Error}` |
| **Delayed Promise pattern** | `Text()`, `Usage()` block | ✅ Exists | 🟢 REUSE | `stream.Result.Text()` etc. |
| **Thread-safe Result** | `atomic.Pointer[streamData]` | ⚠️ Similar | 🟡 EXTEND | Uses `sync.RWMutex` |
| **ReasoningPart** | O1-style reasoning content | ✅ Exists | 🟢 REUSE | `provider.ReasoningContent` |
| **Stream chunk accumulation** | Build text from deltas | ✅ Exists | 🟢 REUSE | `strings.Builder` in stream.go |

**Compatibility:** The existing streaming implementation is well-aligned with Vercel patterns. Minor extensions needed for agent streaming.

---

## Content Types Compatibility

| Feature | Status | Location |
|---------|--------|----------|
| **TextContent** | 🟢 REUSE | `provider.TextContent` |
| **ImageContent** | 🟢 REUSE | `provider.ImageContent` |
| **AudioContent** | 🟢 REUSE | `provider.AudioContent` |
| **ToolCallContent** | 🟢 REUSE | `provider.ToolCallContent` |
| **ToolResultContent** | 🟢 REUSE | `provider.ToolResultContent` |
| **ReasoningContent** | 🟢 REUSE | `provider.ReasoningContent` |
| **Content interface** | 🟢 REUSE | `Content interface { content() }` |
| **Helper constructors** | 🟢 REUSE | `provider.Text()`, `Image()` etc. |

---

## Middleware Compatibility

| Feature | Status | Location |
|---------|--------|----------|
| **Handler interface** | 🟢 REUSE | `middleware.Handler` |
| **Middleware function** | 🟢 REUSE | `middleware.Middleware` |
| **Chain** | 🟢 REUSE | `middleware.Chain()` |
| **Logging middleware** | 🟢 REUSE | `middleware.Logging()` |
| **Metrics middleware** | 🟢 REUSE | `middleware.Metrics()` |
| **Timeout middleware** | 🟢 REUSE | `middleware.Timeout()` |
| **Retry middleware** | 🟢 REUSE | `middleware.Retry()` |
| **Recover middleware** | 🟢 REUSE | `middleware.Recover()` |
| **Tracing middleware** | 🟢 REUSE | `middleware.Tracing()` |
| **Provider-level middleware** | 🟢 REUSE | `openai.WithMiddleware()` |

---

## Provider Interface Compatibility

| Feature | Status | Location |
|---------|--------|----------|
| **Model interface** | 🟢 REUSE | `provider.Model` |
| **Generator interface** | 🟢 REUSE | `provider.Generator` |
| **Streamer interface** | 🟢 REUSE | `provider.Streamer` |
| **LanguageModel** | 🟢 REUSE | `provider.LanguageModel` |
| **EmbeddingModel** | 🟢 REUSE | `provider.EmbeddingModel` |
| **ImageModel** | 🟢 REUSE | `provider.ImageModel` |
| **TranscriptionModel** | 🟢 REUSE | `provider.TranscriptionModel` |
| **SpeechModel** | 🟢 REUSE | `provider.SpeechModel` |
| **Capability checks** | 🟢 REUSE | `provider.CanGenerate()` etc. |
| **Config propagation** | 🟢 REUSE | Functional options |

---

## New Packages Required

### 1. `graph` Package (NEW - Required)

```
graph/
├── annotation.go      # StateAnnotation[T], WithReducer, WithDefault
├── channel.go         # LastValue, BinaryOperatorAggregate, Topic
├── graph.go           # StateGraph, AddNode, AddEdge, Compile
├── pregel.go          # Execution engine, superstep loop
├── checkpointer.go    # Checkpointer interface
├── checkpoint.go      # Checkpoint, CheckpointTuple
├── command.go         # Command, Send
├── interrupt.go       # Interrupt, Resume
├── doc.go
```

### 2. `runnable` Package (NEW - Optional, for composition)

```
runnable/
├── runnable.go        # Runnable[In, Out] interface
├── sequence.go        # Sequence[In, Out]
├── map.go             # Map[In] for parallel
├── lambda.go          # Lambda[In, Out]
├── passthrough.go     # Passthrough, Assign
├── retry.go           # WithRetry
├── fallback.go        # WithFallback
├── doc.go
```

### 3. `agent` Package (NEW - Required)

```
agent/
├── agent.go           # Agent interface, ToolLoopAgent
├── options.go         # AgentOptions, StopCondition, PrepareStepFunc
├── generate.go        # Generate() with tool loop
├── stream.go          # Stream() with tool loop
├── step.go            # StepResult, Step accumulation
├── callbacks.go       # OnStartEvent, OnStepEvent, etc.
├── doc.go
```

### 4. `persistence` Package (NEW - Required)

```
persistence/
├── checkpointer.go    # Checkpointer interface
├── checkpoint.go      # Checkpoint, CheckpointTuple, CheckpointMetadata
├── memory.go          # MemoryCheckpointer
├── sqlite.go          # SQLiteCheckpointer
├── postgres.go        # PostgresCheckpointer
├── thread.go          # ThreadConfig
├── store.go           # Cross-thread Store interface
├── doc.go
```

### 5. Extensions to Existing Packages

```
tool/
├── tool.go           # ADD: ExecuteWithContext(ctx, input, runtime) for ToolRuntime
├── runtime.go        # NEW: ToolRuntime struct with State, Context, Interrupt, etc.
├── streaming.go      # NEW: StreamableTool interface
├── lifecycle.go      # NEW: OnInputStart, OnInputAvailable callbacks

generate/
├── generate.go       # MODIFY: Add tool loop for multi-step

stream/
├── stream.go          # MODIFY: Add tool loop for multi-step
```

---

## Implementation Priority

### Phase 1: Core Agent (Highest Priority)

1. **Agent Loop** - Multi-step tool execution with stop conditions
2. **ToolRuntime** - State, context, and lifecycle injection into tools
3. **Step Accumulation** - Track all steps in multi-turn execution
4. **Lifecycle Callbacks** - OnStart, OnStepStart/Finish, OnToolCallStart/Finish, OnFinish

**Estimated effort:** 1-2 weeks

### Phase 2: Basic Graph (Medium Priority)

1. **StateAnnotation** - Define state with reducers
2. **StateGraph** - Nodes, edges, compilation
3. **Pregel Execution** - Superstep-based execution
4. **Channels** - LastValue, BinaryOperatorAggregate

**Estimated effort:** 2-3 weeks

### Phase 3: Persistence (Medium Priority)

1. **Checkpointer Interface** - GetTuple, Put, List, DeleteThread
2. **MemoryCheckpointer** - In-memory for testing
3. **SQLiteCheckpointer** - Development persistence
4. **PostgresCheckpointer** - Production persistence

**Estimated effort:** 2 weeks

### Phase 4: Runnable Composition (Lower Priority)

1. **Runnable Interface** - Invoke, Stream, Batch, Pipe
2. **Sequence, Map, Lambda** - Composition primitives
3. **Retry/Fallback Wrappers** - Error handling

**Estimated effort:** 1-2 weeks

---

## Compatibility Summary

| Category | Total Features | Already Exists | Can Extend | Must Build |
|----------|---------------|----------------|------------|------------|
| **Graph** | 9 | 0 | 0 | 9 |
| **Runnable** | 10 | 2 | 4 | 4 |
| **Tools** | 11 | 4 | 0 | 7 |
| **Agents** | 10 | 1 | 1 | 8 |
| **Persistence** | 9 | 0 | 0 | 9 |
| **Streaming** | 9 | 7 | 1 | 1 |
| **Content Types** | 8 | 8 | 0 | 0 |
| **Middleware** | 10 | 10 | 0 | 0 |
| **Provider** | 10 | 10 | 0 | 0 |
| **TOTAL** | **86** | **42 (49%)** | **6 (7%)** | **38 (44%)** |

---

## Conclusion

The existing Lamar SDK provides a solid foundation for building the Hydra Agent Framework:

- **49% of features** already exist (Provider, Tool, Stream, Middleware, Content Types)
- **7% can be extended** from existing patterns
- **44% must be built new** (Graph, Agent Loop, Persistence)

The recommended approach is to start with **Phase 1 (Agent Loop)** which can leverage existing `provider.LanguageModel` and `tool.Tool` interfaces, then add persistence and graph features incrementally.
