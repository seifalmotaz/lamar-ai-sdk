# Provider Implementation Guide

This document explains how to implement a provider for the AI SDK, using OpenAI as a reference example.

---

## Provider Structure

```
packages/openai/
├── package.json
├── src/
│   ├── index.ts                    # Public exports
│   ├── internal/index.ts           # Internal exports for other packages
│   ├── openai-provider.ts          # Main provider factory
│   ├── openai-config.ts            # Configuration type
│   ├── openai-error.ts             # Error handling
│   ├── openai-language-model-capabilities.ts
│   ├── openai-tools.ts             # Provider-specific tools export
│   ├── version.ts
│   ├── chat/                       # Chat completion models
│   │   ├── openai-chat-language-model.ts
│   │   ├── openai-chat-options.ts
│   │   └── convert-to-openai-chat-messages.ts
│   ├── completion/                 # Legacy completion models
│   ├── embedding/                  # Embedding models
│   ├── image/                      # Image generation models
│   ├── transcription/              # Audio transcription models
│   ├── speech/                     # Text-to-speech models
│   ├── responses/                  # OpenAI Responses API models
│   └── tool/                       # Provider-specific tools
```

---

## Provider Factory Pattern

The provider is a **callable function** with attached properties:

```typescript
// openai-provider.ts
import { loadApiKey, loadSetting } from '@ai-sdk/provider-utils';

export function createOpenAI(options: OpenAIProviderOptions = {}): OpenAIProvider {
  const baseURL = options.baseURL ?? loadSetting({ settingName: 'OPENAI_BASE_URL' }) ?? 'https://api.openai.com/v1';
  
  const getHeaders = () => ({
    Authorization: `Bearer ${loadApiKey({
      apiKey: options.apiKey,
      environmentVariableName: 'OPENAI_API_KEY',
    })}`,
    ...(options.organization && { 'OpenAI-Organization': options.organization }),
    ...(options.project && { 'OpenAI-Project': options.project }),
  });

  const config: OpenAIConfig = {
    provider: 'openai',
    url: ({ path }) => `${baseURL}${path}`,
    headers: getHeaders,
    fetch: options.fetch,
  };

  // Create callable function
  const provider = function (modelId: OpenAIResponsesModelId): LanguageModelV3 {
    return createChatModel(modelId, config, options);
  };

  // Attach properties
  provider.specificationVersion = 'v3' as const;
  provider.provider = 'openai';
  provider.languageModel = (modelId) => createChatModel(modelId, config, options);
  provider.chat = (modelId) => createChatModel(modelId, config, options);
  provider.embedding = (modelId) => createEmbeddingModel(modelId, config, options);
  provider.image = (modelId) => createImageModel(modelId, config, options);
  provider.transcription = (modelId) => createTranscriptionModel(modelId, config, options);
  provider.speech = (modelId) => createSpeechModel(modelId, config, options);
  provider.tools = openaiTools;

  return provider as OpenAIProvider;
}

// Default export
export const openai = createOpenAI();
```

---

## Model Implementation

### Chat Model Class

```typescript
// chat/openai-chat-language-model.ts
import {
  LanguageModelV3,
  LanguageModelV3CallOptions,
  LanguageModelV3GenerateResult,
  LanguageModelV3StreamResult,
} from '@ai-sdk/provider';
import {
  postJsonToApi,
  createJsonResponseHandler,
  createEventSourceResponseHandler,
  combineHeaders,
} from '@ai-sdk/provider-utils';

export class OpenAIChatLanguageModel implements LanguageModelV3 {
  readonly specificationVersion = 'v3' as const;
  readonly modelId: OpenAIChatModelId;
  readonly supportedUrls = { 'image/*': [/^https?:\/\/.*$/] };
  
  private readonly config: OpenAIConfig;
  private readonly settings: OpenAIChatSettings;

  constructor(
    modelId: OpenAIChatModelId,
    config: OpenAIConfig,
    settings: OpenAIChatSettings = {},
  ) {
    this.modelId = modelId;
    this.config = config;
    this.settings = settings;
  }

  get provider(): string {
    return this.config.provider;
  }

  async doGenerate(
    options: LanguageModelV3CallOptions,
  ): Promise<LanguageModelV3GenerateResult> {
    // 1. Convert prompt to provider format
    const messages = await convertToOpenAIChatMessages(options.prompt);
    
    // 2. Build request body
    const body = {
      model: this.modelId,
      messages,
      temperature: options.temperature ?? this.settings.temperature,
      max_tokens: options.maxOutputTokens ?? this.settings.maxTokens,
      top_p: options.topP ?? this.settings.topP,
      stop: options.stopSequences ?? this.settings.stopSequences,
      // ... more options
    };

    // 3. Make API call
    const { responseHeaders, value } = await postJsonToApi({
      url: this.config.url({ path: '/chat/completions', modelId: this.modelId }),
      headers: combineHeaders(this.config.headers(), options.headers),
      body,
      failedResponseHandler: openaiFailedResponseHandler,
      successfulResponseHandler: createJsonResponseHandler(openaiChatResponseSchema),
      abortSignal: options.abortSignal,
      fetch: this.config.fetch,
    });

    // 4. Transform response to SDK format
    return {
      content: transformContent(value.choices[0].message),
      finishReason: mapFinishReason(value.choices[0].finish_reason),
      usage: {
        promptTokens: value.usage.prompt_tokens,
        completionTokens: value.usage.completion_tokens,
        totalTokens: value.usage.total_tokens,
      },
      warnings: [],
      response: {
        id: value.id,
        timestamp: new Date(),
        modelId: value.model,
        headers: responseHeaders,
      },
    };
  }

  async doStream(
    options: LanguageModelV3CallOptions,
  ): Promise<LanguageModelV3StreamResult> {
    // Similar to doGenerate but with streaming
    const body = {
      // ... same as doGenerate
      stream: true,
      stream_options: { include_usage: true },
    };

    const { responseHeaders, value } = await postJsonToApi({
      url: this.config.url({ path: '/chat/completions', modelId: this.modelId }),
      headers: combineHeaders(this.config.headers(), options.headers),
      body,
      successfulResponseHandler: createEventSourceResponseHandler(openaiChatChunkSchema),
      // ...
    });

    // Transform SSE stream to SDK stream parts
    return {
      stream: value.pipeThrough(new TransformStream({
        start(controller) {
          controller.enqueue({ type: 'stream-start', warnings: [] });
        },
        transform(chunk, controller) {
          // Handle each chunk type
          if (chunk.choices?.[0]?.delta?.content) {
            controller.enqueue({
              type: 'text-delta',
              id: '0',
              delta: chunk.choices[0].delta.content,
            });
          }
          // ... handle other chunk types
        },
        flush(controller) {
          controller.enqueue({
            type: 'finish',
            finishReason: 'stop',
            usage: { /* ... */ },
          });
        },
      })),
      response: { headers: responseHeaders },
    };
  }
}
```

### Model Factory Function

```typescript
// chat/openai-chat-model.ts
export function createChatModel(
  modelId: OpenAIChatModelId,
  config: OpenAIConfig,
  settings: OpenAIChatSettings = {},
): LanguageModelV3 {
  return new OpenAIChatLanguageModel(modelId, config, settings);
}
```

---

## Configuration

### Config Type

```typescript
// openai-config.ts
export type OpenAIConfig = {
  provider: string;
  url: (options: { modelId: string; path: string }) => string;
  headers: () => Record<string, string | undefined>;
  fetch?: FetchFunction;
  generateId?: () => string;
  fileIdPrefixes?: readonly string[];
};
```

### Provider Options

```typescript
// openai-provider-options.ts
import { z } from 'zod';

export type OpenAIProviderOptions = {
  baseURL?: string;
  apiKey?: string;
  organization?: string;
  project?: string;
  fetch?: FetchFunction;
};
```

---

## Provider-Specific Options

Each model can have provider-specific options using Zod schemas:

```typescript
// chat/openai-chat-options.ts
import { z } from 'zod';
import { lazySchema, zodSchema } from '@ai-sdk/provider-utils';

// Model IDs
export type OpenAIChatModelId =
  | 'gpt-4o'
  | 'gpt-4o-mini'
  | 'o1'
  | 'o3-mini'
  | 'gpt-5'
  | (string & {});  // Allows any string

// Provider-specific options schema
export const openaiLanguageModelChatOptions = lazySchema(() =>
  zodSchema(
    z.object({
      logitBias: z.record(z.coerce.number(), z.number()).optional(),
      reasoningEffort: z.enum(['none', 'minimal', 'low', 'medium', 'high', 'xhigh']).optional(),
      user: z.string().optional(),
      // ... more OpenAI-specific options
    }),
  ),
);
```

---

## Error Handling

### Error Schema

```typescript
// openai-error.ts
import { z } from 'zod';

export const openaiErrorDataSchema = z.object({
  error: z.object({
    message: z.string(),
    type: z.string().optional(),
    param: z.string().optional(),
    code: z.string().optional(),
  }),
});

export const openaiFailedResponseHandler = createJsonErrorResponseHandler({
  errorSchema: openaiErrorDataSchema,
  errorToMessage: data => data.error.message,
  isRetryable: (response, error) => 
    response.status === 429 || 
    (response.status >= 500 && response.status < 600),
});
```

---

## Provider-Specific Tools

OpenAI provides built-in tools that can be used with function calling:

```typescript
// tool/web-search.ts
import { z } from 'zod';
import { createProviderToolFactoryWithOutputSchema, zodSchema } from '@ai-sdk/provider-utils';

const webSearchInputSchema = z.object({
  search_context_size: z.enum(['low', 'medium', 'high']).optional(),
  user_location: z.object({
    type: z.literal('approximate'),
    approximate: z.object({
      country: z.string().optional(),
      city: z.string().optional(),
      region: z.string().optional(),
    }),
  }).optional(),
});

const webSearchOutputSchema = z.object({
  status: z.enum(['success', 'error']),
  // ... more fields
});

export const webSearchToolFactory = createProviderToolFactoryWithOutputSchema({
  id: 'openai.web_search',
  inputSchema: zodSchema(webSearchInputSchema),
  outputSchema: zodSchema(webSearchOutputSchema),
});

export const webSearch = (args = {}) => webSearchToolFactory(args);
```

---

## API Schemas

Use Zod to define request/response schemas:

```typescript
// chat/openai-chat-api.ts
import { z } from 'zod';

export const openaiChatResponseSchema = z.object({
  id: z.string(),
  object: z.literal('chat.completion'),
  created: z.number(),
  model: z.string(),
  choices: z.array(
    z.object({
      index: z.number(),
      message: z.object({
        role: z.literal('assistant'),
        content: z.string().nullable(),
        tool_calls: z.array(
          z.object({
            id: z.string(),
            type: z.literal('function'),
            function: z.object({
              name: z.string(),
              arguments: z.string(),
            }),
          }),
        ).optional(),
      }),
      finish_reason: z.string().nullable(),
    }),
  ),
  usage: z.object({
    prompt_tokens: z.number(),
    completion_tokens: z.number(),
    total_tokens: z.number(),
  }),
});

export const openaiChatChunkSchema = z.object({
  id: z.string(),
  object: z.literal('chat.completion.chunk'),
  created: z.number(),
  model: z.string(),
  choices: z.array(
    z.object({
      index: z.number(),
      delta: z.object({
        role: z.literal('assistant').optional(),
        content: z.string().nullable().optional(),
        tool_calls: z.array(
          z.object({
            index: z.number(),
            id: z.string().optional(),
            type: z.literal('function').optional(),
            function: z.object({
              name: z.string().optional(),
              arguments: z.string().optional(),
            }).optional(),
          }),
        ).optional(),
      }),
      finish_reason: z.string().nullable(),
    }),
  ),
  usage: z.object({
    prompt_tokens: z.number(),
    completion_tokens: z.number(),
    total_tokens: z.number(),
  }).optional(),
});
```

---

## Go Implementation Pattern

```go
// openai/provider.go
package openai

import (
    "context"
    "net/http"
)

type Provider struct {
    baseURL    string
    apiKey     string
    httpClient *http.Client
    headers    func() map[string]string
}

type ProviderOptions struct {
    BaseURL string
    APIKey  string
    Organization string
    Project string
    HTTPClient *http.Client
}

func NewProvider(opts ProviderOptions) *Provider {
    if opts.BaseURL == "" {
        opts.BaseURL = "https://api.openai.com/v1"
    }
    if opts.APIKey == "" {
        opts.APIKey = os.Getenv("OPENAI_API_KEY")
    }
    
    httpClient := opts.HTTPClient
    if httpClient == nil {
        httpClient = http.DefaultClient
    }
    
    return &Provider{
        baseURL:    opts.BaseURL,
        apiKey:     opts.APIKey,
        httpClient: httpClient,
        headers: func() map[string]string {
            h := map[string]string{
                "Authorization": "Bearer " + opts.APIKey,
            }
            if opts.Organization != "" {
                h["OpenAI-Organization"] = opts.Organization
            }
            return h
        },
    }
}

// Model returns a chat completion model
func (p *Provider) Model(modelID string) *ChatModel {
    return NewChatModel(modelID, p)
}

// Embedding returns an embedding model
func (p *Provider) Embedding(modelID string) *EmbeddingModel {
    return NewEmbeddingModel(modelID, p)
}

// ChatModel implementation
type ChatModel struct {
    modelID  string
    provider *Provider
    settings ChatSettings
}

var _ provider.LanguageModelV3 = (*ChatModel)(nil)

func (m *ChatModel) SpecificationVersion() string { return "v3" }
func (m *ChatModel) Provider() string            { return "openai" }
func (m *ChatModel) ModelID() string             { return m.modelID }

func (m *ChatModel) DoGenerate(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.GenerateResult, error) {
    // 1. Convert prompt to OpenAI format
    messages, err := ConvertToOpenAIMessages(opts.Prompt)
    if err != nil {
        return nil, err
    }
    
    // 2. Build request
    req := &ChatCompletionRequest{
        Model:    m.modelID,
        Messages: messages,
        // ... apply settings
    }
    
    // 3. Make API call
    resp, err := m.provider.doRequest(ctx, "/chat/completions", req)
    if err != nil {
        return nil, err
    }
    
    // 4. Transform to SDK format
    return transformResponse(resp), nil
}

func (m *ChatModel) DoStream(ctx context.Context, opts provider.LanguageModelCallOptions) (*provider.StreamResult, error) {
    // Streaming implementation
    req := &ChatCompletionRequest{
        Model:    m.modelID,
        Messages: messages,
        Stream:   true,
        StreamOptions: &StreamOptions{
            IncludeUsage: true,
        },
    }
    
    stream, err := m.provider.doStreamRequest(ctx, "/chat/completions", req)
    if err != nil {
        return nil, err
    }
    
    // Return channel-based stream
    return &provider.StreamResult{
        Stream: transformStream(stream),
    }, nil
}
```

---

## Key Implementation Checklist

1. **Provider Factory**:
   - [ ] Create callable function with attached model factories
   - [ ] Handle API key loading (env vars, params)
   - [ ] Handle base URL configuration
   - [ ] Support custom headers

2. **Model Implementation**:
   - [ ] Implement `LanguageModelV3` interface
   - [ ] Convert prompts to provider format
   - [ ] Handle `doGenerate` (non-streaming)
   - [ ] Handle `doStream` (streaming)
   - [ ] Map finish reasons
   - [ ] Transform usage stats

3. **Error Handling**:
   - [ ] Define error schema
   - [ ] Create error response handler
   - [ ] Map provider errors to SDK errors

4. **Provider-Specific Options**:
   - [ ] Define Zod schemas for options
   - [ ] Handle `providerOptions` in call options

5. **Tools** (optional):
   - [ ] Define provider-specific tools
   - [ ] Create tool factory functions

6. **Testing**:
   - [ ] Mock HTTP responses
   - [ ] Test all model types
   - [ ] Test edge cases