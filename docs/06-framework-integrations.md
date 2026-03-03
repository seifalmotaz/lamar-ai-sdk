# Framework Integrations

The AI SDK provides UI framework integrations for **React**, **Vue**, **Svelte**, and **Angular**. Each integration follows a consistent architecture while leveraging framework-specific reactivity primitives.

---

## Architecture Overview

**Base Pattern**: All frameworks implement a class that extends `AbstractChat<UI_MESSAGE>` from the `ai` package.

```
ai (core package)
    ├── AbstractChat (abstract class)
    ├── ChatState (interface)
    └── ChatStatus (type: 'submitted' | 'streaming' | 'ready' | 'error')

Framework packages extend AbstractChat:
    ├── @ai-sdk/react → Chat extends AbstractChat
    ├── @ai-sdk/vue → Chat extends AbstractChat
    ├── @ai-sdk/svelte → Chat extends AbstractChat
    └── @ai-sdk/angular → Chat extends AbstractChat
```

---

## Core Abstractions (`ai` package)

### AbstractChat

```typescript
// packages/ai/src/ui/chat.ts
export abstract class AbstractChat<UI_MESSAGE extends UIMessage> {
  protected readonly state: ChatState<UI_MESSAGE>;
  
  abstract addMessage(message: UI_MESSAGE): void;
  abstract sendMessage(message: UI_MESSAGE): Promise<void>;
  abstract regenerate(options?: { messageId?: string }): Promise<void>;
  abstract stop(): void;
  abstract clearError(): void;
  
  // Properties
  abstract get messages(): UI_MESSAGE[];
  abstract get status(): ChatStatus;
  abstract get error(): Error | undefined;
}
```

### ChatState Interface

```typescript
// packages/ai/src/ui/chat-state.ts
export interface ChatState<UI_MESSAGE extends UIMessage> {
  status: ChatStatus;
  error: Error | undefined;
  messages: UI_MESSAGE[];
  
  pushMessage(message: UI_MESSAGE): void;
  popMessage(): void;
  replaceMessage(index: number, message: UI_MESSAGE): void;
  snapshot<T>(thing: T): T;
}
```

### UIMessage Type

```typescript
// packages/ai/src/ui/ui-messages.ts
export interface UIMessage {
  id: string;
  role: 'user' | 'assistant';
  parts: UIMessagePart[];
  createdAt?: Date;
  
  // Optional fields
  content?: string;  // Fallback for simple text
}

export type UIMessagePart =
  | { type: 'text'; text: string }
  | { type: 'image'; image: string | Uint8Array }
  | { type: 'tool-call'; ... }
  | { type: 'tool-result'; ... }
  | { type: 'reasoning'; text: string }
  | { type: 'file'; ... }
  | { type: 'source'; ... }
```

---

## React Integration

### Package: `@ai-sdk/react`

### Key Exports

```typescript
// useChat hook
export function useChat(options?: {
  id?: string;
  initialMessages?: UIMessage[];
  onToolCall?: (toolCall: ToolCall) => void;
  onFinish?: (message: UIMessage) => void;
  onError?: (error: Error) => void;
  maxSteps?: number;
  experimental_throttle?: number;
}): {
  messages: UIMessage[];
  input: string;
  setInput: (value: string) => void;
  handleSubmit: (e?: React.FormEvent) => void;
  append: (message: UIMessage) => Promise<void>;
  reload: () => Promise<void>;
  stop: () => void;
  status: ChatStatus;
  error: Error | undefined;
};

// useCompletion hook
export function useCompletion(options?: {
  api?: string;
  id?: string;
  initialCompletion?: string;
}): {
  completion: string;
  complete: (prompt: string, options?: CompletionOptions) => Promise<void>;
  error: Error | undefined;
  setCompletion: (value: string) => void;
  stop: () => void;
  input: string;
  setInput: (value: string) => void;
  handleSubmit: (e?: React.FormEvent) => void;
  isLoading: boolean;
};

// experimental_useObject hook
export function experimental_useObject<T>(options: {
  api: string;
  schema: Schema<T>;
  onFinish?: (object: T) => void;
}): {
  submit: (input: string) => void;
  object: DeepPartial<T> | undefined;
  error: Error | undefined;
  isLoading: boolean;
  stop: () => void;
  clear: () => void;
};
```

### Usage Example

```tsx
import { useChat } from '@ai-sdk/react';
import { openai } from '@ai-sdk/openai';

function Chat() {
  const { messages, input, setInput, handleSubmit, status, error } = useChat({
    api: '/api/chat',
    maxSteps: 5,
    onToolCall: async ({ toolName, args }) => {
      if (toolName === 'getWeather') {
        return await getWeather(args.location);
      }
    },
  });

  return (
    <div>
      {messages.map(m => (
        <div key={m.id}>
          <strong>{m.role}:</strong>
          {m.parts.map((part, i) => {
            if (part.type === 'text') return <span key={i}>{part.text}</span>;
            if (part.type === 'tool-call') return <ToolCall key={i} {...part} />;
          })}
        </div>
      ))}
      
      <form onSubmit={handleSubmit}>
        <input value={input} onChange={e => setInput(e.target.value)} />
        <button type="submit" disabled={status === 'streaming'}>
          Send
        </button>
      </form>
      
      {error && <div className="error">{error.message}</div>}
    </div>
  );
}
```

### React State Management

```typescript
// packages/react/src/chat.react.ts
import { AbstractChat, ChatState, ChatStatus } from 'ai';
import { useSyncExternalStore } from 'react';

class ReactChatState<UI_MESSAGE extends UIMessage> implements ChatState<UI_MESSAGE> {
  #messages: UI_MESSAGE[] = [];
  #status: ChatStatus = 'ready';
  #error: Error | undefined;
  
  #messagesCallbacks = new Set<() => void>();
  #statusCallbacks = new Set<() => void>();
  #errorCallbacks = new Set<() => void>();

  get messages() { return this.#messages; }
  get status() { return this.#status; }
  get error() { return this.#error; }

  pushMessage(message: UI_MESSAGE) {
    this.#messages.push(message);
    this.#messagesCallbacks.forEach(cb => cb());
  }

  // ... more methods
}

export class Chat<UI_MESSAGE extends UIMessage> extends AbstractChat<UI_MESSAGE> {
  constructor(options: ChatOptions<UI_MESSAGE>) {
    super(new ReactChatState<UI_MESSAGE>());
  }
}
```

---

## Vue Integration

### Package: `@ai-sdk/vue`

### Key Exports

```typescript
// useCompletion composable
export function useCompletion(options?: {
  api?: string;
  id?: string;
}): {
  completion: Ref<string>;
  complete: (prompt: string) => Promise<void>;
  error: Ref<Error | undefined>;
  isLoading: Ref<boolean>;
  stop: () => void;
};

// experimental_useObject composable
export function experimental_useObject<T>(options: {
  api: string;
  schema: Schema<T>;
}): {
  submit: (input: string) => void;
  object: Ref<DeepPartial<T> | undefined>;
  error: Ref<Error | undefined>;
  isLoading: Ref<boolean>;
};
```

### Usage Example

```vue
<script setup lang="ts">
import { useCompletion } from '@ai-sdk/vue';

const { completion, complete, isLoading, error } = useCompletion({
  api: '/api/completion',
});

async function handleSubmit() {
  await complete('Write a poem about the sea');
}
</script>

<template>
  <div>
    <button @click="handleSubmit" :disabled="isLoading">
      Generate
    </button>
    <p v-if="error">{{ error.message }}</p>
    <p>{{ completion }}</p>
  </div>
</template>
```

### Vue State Management

```typescript
// packages/vue/src/chat.vue.ts
import { ref, Ref } from 'vue';
import { AbstractChat, ChatState, ChatStatus, UIMessage } from 'ai';

class VueChatState<UI_MESSAGE extends UIMessage> implements ChatState<UI_MESSAGE> {
  messagesRef: Ref<UI_MESSAGE[]> = ref([]);
  statusRef: Ref<ChatStatus> = ref('ready');
  errorRef: Ref<Error | undefined> = ref(undefined);

  get messages() { return this.messagesRef.value; }
  get status() { return this.statusRef.value; }
  get error() { return this.errorRef.value; }

  pushMessage(message: UI_MESSAGE) {
    this.messagesRef.value = [...this.messagesRef.value, message];
  }

  // ... more methods
}

export class Chat<UI_MESSAGE extends UIMessage> extends AbstractChat<UI_MESSAGE> {
  constructor(options: ChatOptions<UI_MESSAGE>) {
    super(new VueChatState<UI_MESSAGE>());
  }
}
```

---

## Svelte Integration

### Package: `@ai-sdk/svelte`

### Key Exports

```typescript
export class Chat<UI_MESSAGE extends UIMessage> {
  messages = $state<UI_MESSAGE[]>([]);
  status = $state<ChatStatus>('ready');
  error = $state<Error | undefined>(undefined);
  input = $state<string>('');
  
  async sendMessage(message: UI_MESSAGE): Promise<void>;
  async regenerate(options?: { messageId?: string }): Promise<void>;
  stop(): void;
}

export class Completion {
  completion = $state<string>('');
  isLoading = $state<boolean>(false);
  error = $state<Error | undefined>(undefined);
  
  async complete(prompt: string): Promise<void>;
  stop(): void;
}
```

### Usage Example

```svelte
<script lang="ts">
  import { Chat } from '@ai-sdk/svelte';
  
  const chat = new Chat({
    api: '/api/chat',
  });
  
  async function handleSubmit() {
    await chat.sendMessage({
      id: crypto.randomUUID(),
      role: 'user',
      parts: [{ type: 'text', text: chat.input }],
    });
  }
</script>

<div>
  {#each chat.messages as message}
    <div>
      {message.role}: 
      {#each message.parts as part}
        {#if part.type === 'text'}
          {part.text}
        {/if}
      {/each}
    </div>
  {/each}
  
  <input bind:value={chat.input} />
  <button onclick={handleSubmit} disabled={chat.status === 'streaming'}>
    Send
  </button>
</div>
```

### Svelte State Management

```typescript
// packages/svelte/src/chat.svelte.ts
import { AbstractChat, ChatState, ChatStatus, UIMessage } from 'ai';

class SvelteChatState<UI_MESSAGE extends UIMessage> implements ChatState<UI_MESSAGE> {
  messages = $state<UI_MESSAGE[]>([]);
  status = $state<ChatStatus>('ready');
  error = $state<Error | undefined>(undefined);

  pushMessage(message: UI_MESSAGE) {
    this.messages = [...this.messages, message];
  }

  snapshot<T>(thing: T): T {
    return $state.snapshot(thing);
  }
}

export class Chat<UI_MESSAGE extends UIMessage> extends AbstractChat<UI_MESSAGE> {
  constructor(options: ChatOptions<UI_MESSAGE>) {
    super(new SvelteChatState<UI_MESSAGE>());
  }
}
```

---

## Angular Integration

### Package: `@ai-sdk/angular`

### Key Exports

```typescript
export class Chat<UI_MESSAGE extends UIMessage> {
  readonly messages = signal<UI_MESSAGE[]>([]);
  readonly status = signal<ChatStatus>('ready');
  readonly error = signal<Error | undefined>(undefined);
  
  async sendMessage(message: UI_MESSAGE): Promise<void>;
  async regenerate(): Promise<void>;
  stop(): void;
}

export class Completion {
  readonly completion = signal<string>('');
  readonly isLoading = signal<boolean>(false);
  readonly error = signal<Error | undefined>(undefined);
  
  async complete(prompt: string): Promise<void>;
}
```

### Usage Example

```typescript
// chat.component.ts
import { Component } from '@angular/core';
import { Chat } from '@ai-sdk/angular';

@Component({
  selector: 'app-chat',
  template: `
    <div *ngFor="let message of chat.messages()">
      <strong>{{ message.role }}:</strong>
      {{ message.parts[0].text }}
    </div>
    
    <input [(ngModel)]="input" />
    <button (click)="handleSubmit()" [disabled]="chat.status() === 'streaming'">
      Send
    </button>
  `,
})
export class ChatComponent {
  chat = new Chat({ api: '/api/chat' });
  input = '';

  async handleSubmit() {
    await this.chat.sendMessage({
      id: crypto.randomUUID(),
      role: 'user',
      parts: [{ type: 'text', text: this.input }],
    });
    this.input = '';
  }
}
```

### Angular State Management

```typescript
// packages/angular/src/lib/chat.ng.ts
import { signal, Signal } from '@angular/core';
import { AbstractChat, ChatState, ChatStatus, UIMessage } from 'ai';

class AngularChatState<UI_MESSAGE extends UIMessage> implements ChatState<UI_MESSAGE> {
  #messages = signal<UI_MESSAGE[]>([]);
  #status = signal<ChatStatus>('ready');
  #error = signal<Error | undefined>(undefined);

  get messages() { return this.#messages(); }
  get status() { return this.#status(); }
  get error() { return this.#error(); }

  pushMessage(message: UI_MESSAGE) {
    this.#messages.update(msgs => [...msgs, message]);
  }

  replaceMessage(index: number, message: UI_MESSAGE) {
    this.#messages.update(msgs => {
      const updated = [...msgs];
      updated[index] = message;
      return updated;
    });
  }

  snapshot<T>(thing: T): T {
    return structuredClone(thing);
  }
}
```

---

## Common Patterns Across Frameworks

### 1. State Implementation Pattern

Each framework creates a `*ChatState` class implementing `ChatState<UI_MESSAGE>`:

| Framework | Reactivity Primitive |
|-----------|---------------------|
| React | `useState`, `useSyncExternalStore` |
| Vue | `ref()`, `reactive()` |
| Svelte | `$state` rune |
| Angular | `signal()` |

### 2. Chat Class Pattern

All extend `AbstractChat<UI_MESSAGE>` and pass framework-specific state:

```typescript
class FrameworkChat<UI_MESSAGE extends UIMessage> extends AbstractChat<UI_MESSAGE> {
  constructor(options: ChatOptions<UI_MESSAGE>) {
    super(new FrameworkChatState<UI_MESSAGE>());
  }
}
```

### 3. Shared API Surface

All frameworks expose similar functionality:

| Feature | React | Vue | Svelte | Angular |
|---------|-------|-----|--------|---------|
| Chat | `useChat()` | `Chat` class | `Chat` class | `Chat` class |
| Completion | `useCompletion()` | `useCompletion()` | `Completion` | `Completion` |
| Structured Object | `experimental_useObject()` | `experimental_useObject()` | `StructuredObject` | `StructuredObject` |

---

## Go Framework Considerations

Go doesn't have the same UI framework ecosystem, but similar patterns can be applied:

### Server-Side Chat Handler

```go
// Framework-agnostic chat handler
package aisdk

import (
    "context"
    "net/http"
)

type ChatHandler struct {
    model   LanguageModel
    options ChatHandlerOptions
}

func NewChatHandler(model LanguageModel, opts ChatHandlerOptions) *ChatHandler {
    return &ChatHandler{model: model, options: opts}
}

func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req ChatRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Stream response
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    
    result := StreamText(r.Context(), StreamTextOptions{
        Model:    h.model,
        Messages: req.Messages,
    })
    
    flusher, _ := w.(http.Flusher)
    for part := range result.FullStream() {
        data, _ := json.Marshal(part)
        fmt.Fprintf(w, "data: %s\n\n", data)
        flusher.Flush()
    }
}
```

### Gin Integration

```go
package aigin

import "github.com/gin-gonic/gin"

func ChatHandler(model LanguageModel) gin.HandlerFunc {
    return func(c *gin.Context) {
        var req ChatRequest
        if err := c.ShouldBindJSON(&req); err != nil {
            c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
            return
        }
        
        result := StreamText(c.Request.Context(), StreamTextOptions{
            Model:    model,
            Messages: req.Messages,
        })
        
        c.Stream(func(w io.Writer) bool {
            select {
            case part, ok := <-result.FullStream():
                if !ok {
                    return false
                }
                data, _ := json.Marshal(part)
                fmt.Fprintf(w, "data: %s\n\n", data)
                return true
            case <-c.Request.Context().Done():
                return false
            }
        })
    }
}
```

### Echo Integration

```go
package aiecho

import "github.com/labstack/echo/v4"

func ChatHandler(model LanguageModel) echo.HandlerFunc {
    return func(c echo.Context) error {
        var req ChatRequest
        if err := c.Bind(&req); err != nil {
            return err
        }
        
        result := StreamText(c.Request.Context(), StreamTextOptions{
            Model:    model,
            Messages: req.Messages,
        })
        
        return c.Stream(http.StatusOK, "text/event-stream", func(w io.Writer) {
            for part := range result.FullStream() {
                data, _ := json.Marshal(part)
                fmt.Fprintf(w, "data: %s\n\n", data)
            }
        })
    }
}
```

---

## Summary

| Aspect | Pattern |
|--------|---------|
| Core Abstraction | `AbstractChat` and `ChatState` in `ai` package |
| Framework Integration | Extend `AbstractChat` with framework-specific state |
| State Management | Use framework primitives (hooks, refs, signals, runes) |
| API Surface | Consistent `Chat`, `Completion`, `StructuredObject` |
| Go Approach | Server-side handlers with SSE streaming |