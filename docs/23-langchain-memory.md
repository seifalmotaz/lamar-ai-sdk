# LangChainJS Memory & Conversation History

## Executive Summary

LangChainJS provides a flexible memory system for maintaining conversation state across turns. The architecture separates **storage** (ChatMessageHistory) from **strategy** (Memory types), allowing for composable memory approaches.

---

## 1. Message Type System

### Message Hierarchy

```go
// Base message interface
type Message interface {
    GetType() string    // "human" | "ai" | "system" | "tool"
    GetContent() Content
    GetID() string
    GetName() string
}

// Content can be string or multimodal blocks
type Content interface{}
type ContentBlock interface {
    GetType() string    // "text" | "image" | "tool_call" | ...
}

// Specific message types
type HumanMessage struct {
    Content []ContentBlock
    ID      string
    Name    string
}

type AIMessage struct {
    Content     []ContentBlock
    ToolCalls   []ToolCall
    ID          string
    Name        string
}
```

---

## 2. Chat History Storage

### In-Memory Implementation

```go
type InMemoryHistory struct {
    messages []Message
    mu       sync.RWMutex
}

func (h *InMemoryHistory) GetMessages(ctx context.Context) ([]Message, error) {
    h.mu.RLock()
    defer h.mu.RUnlock()
    return append([]Message{}, h.messages...), nil
}

func (h *InMemoryHistory) AddMessage(ctx context.Context, message Message) error {
    h.mu.Lock()
    defer h.mu.Unlock()
    h.messages = append(h.messages, message)
    return nil
}
```

### PostgreSQL Implementation

```go
type PostgresHistory struct {
    Pool      *pgxpool.Pool
    SessionID string
    TableName string
}

func (h *PostgresHistory) GetMessages(ctx context.Context) ([]Message, error) {
    query := fmt.Sprintf("SELECT message FROM %s WHERE session_id = $1 ORDER BY id", h.TableName)
    rows, err := h.Pool.Query(ctx, query, h.SessionID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    var messages []Message
    for rows.Next() {
        var stored StoredMessage
        if err := rows.Scan(&stored); err != nil {
            return nil, err
        }
        messages = append(messages, stored.ToMessage())
    }
    return messages, nil
}
```

---

## 3. Memory Types

### BufferMemory

```go
// Simple buffer - stores all messages
type BufferMemory struct {
    BaseChatMemory
    HumanPrefix string
    AIPrefix    string
    MemoryKey   string
}

func (m *BufferMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    messages, err := m.History.GetMessages(ctx)
    if err != nil {
        return nil, err
    }
    
    if m.ReturnMessages {
        return map[string]any{m.MemoryKey: messages}, nil
    }
    
    // Convert to string format
    return map[string]any{m.MemoryKey: getBufferString(messages, m.HumanPrefix, m.AIPrefix)}, nil
}
```

### BufferWindowMemory

```go
// Keep only last K interactions
type BufferWindowMemory struct {
    BaseChatMemory
    K           int    // Number of interactions (each = 2 messages)
    MemoryKey   string
}

func (m *BufferWindowMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    messages, err := m.History.GetMessages(ctx)
    if err != nil {
        return nil, err
    }
    
    // Keep last K * 2 messages
    windowMessages := messages
    if len(messages) > m.K * 2 {
        windowMessages = messages[len(messages) - m.K * 2:]
    }
    
    if m.ReturnMessages {
        return map[string]any{m.MemoryKey: windowMessages}, nil
    }
    return map[string]any{m.MemoryKey: getBufferString(windowMessages)}, nil
}
```

### ConversationSummaryMemory

```go
// Compress history into summary using LLM
type ConversationSummaryMemory struct {
    BaseChatMemory
    LLM              LanguageModel
    Buffer           string    // Current summary
    MemoryKey        string
}

func (m *ConversationSummaryMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
    // 1. Save to history
    if err := m.BaseChatMemory.SaveContext(ctx, inputs, outputs); err != nil {
        return err
    }
    
    // 2. Get last 2 messages (new interaction)
    messages, err := m.History.GetMessages(ctx)
    if err != nil {
        return err
    }
    newLine := getBufferString(messages[len(messages) - 2:])
    
    // 3. Generate new summary
    newSummary, err := m.LLM.Invoke(ctx, prompt)
    if err != nil {
        return err
    }
    
    m.Buffer = newSummary
    return nil
}
```

### CombinedMemory

```go
// Combine multiple memory types
type CombinedMemory struct {
    Memories []Memory
}

func (m *CombinedMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
    result := make(map[string]any)
    for _, mem := range m.Memories {
        vars, err := mem.LoadMemoryVariables(ctx, inputs)
        if err != nil {
            return nil, err
        }
        for k, v := range vars {
            result[k] = v
        }
    }
    return result, nil
}
```

---

## 4. Recommended Lamar SDK Memory API

```go
// Core interfaces
type Memory interface {
    MemoryKeys() []string
    Load(ctx context.Context, inputs map[string]any) (map[string]any, error)
    Save(ctx context.Context, inputs, outputs map[string]any) error
    Clear(ctx context.Context) error
}

type ChatHistory interface {
    Messages(ctx context.Context) ([]Message, error)
    Add(ctx context.Context, message Message) error
    Clear(ctx context.Context) error
}

// Implementations
func NewBufferMemory(history ChatHistory, opts ...MemoryOption) Memory
func NewWindowMemory(history ChatHistory, k int, opts ...MemoryOption) Memory
func NewSummaryMemory(history ChatHistory, llm LanguageModel, opts ...MemoryOption) Memory
func NewCombinedMemory(memories ...Memory) Memory
```