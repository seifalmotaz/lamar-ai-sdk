# Anthropic Provider

The Anthropic provider implements support for Claude models via the Anthropic Messages API.

## Installation

```go
import "github.com/seifalmotaz/lamar-sdk/providers/anthropic"
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/seifalmotaz/lamar-sdk/generate"
    "github.com/seifalmotaz/lamar-sdk/providers/anthropic"
)

func main() {
    // Initialize provider (uses ANTHROPIC_API_KEY environment variable)
    client := anthropic.NewProvider()
    model := client.Claude45Sonnet()

    // Generate text
    ctx := context.Background()
    result, err := generate.Generate(ctx, model, "Say hello in 3 languages")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(result.Text())
    fmt.Printf("Tokens: %d\n", result.Usage().TotalTokens)
}
```

## Authentication

The provider supports two authentication methods:

### API Key (Recommended)

```go
// From environment variable ANTHROPIC_API_KEY
client := anthropic.NewProvider()

// Or explicitly
client := anthropic.NewProvider(
    anthropic.APIKey("sk-ant-..."),
)
```

### Auth Token (OAuth)

```go
// From environment variable ANTHROPIC_AUTH_TOKEN
client := anthropic.NewProvider()

// Or explicitly
client := anthropic.NewProvider(
    anthropic.AuthToken("your-oauth-token"),
)
```

## Available Models

### Convenience Methods

| Method | Model ID | Description |
|--------|----------|-------------|
| `Claude3Haiku()` | `claude-3-haiku-20240307` | Fast, cost-effective |
| `Claude3Opus()` | `claude-3-opus-20240229` | Most capable (Claude 3) |
| `Claude35Sonnet()` | `claude-3-5-sonnet-20241022` | Balanced performance |
| `Claude35Haiku()` | `claude-3-5-haiku-20241022` | Fast responses |
| `Claude4Sonnet()` | `claude-sonnet-4-20250514` | Latest Sonnet 4 |
| `Claude4Opus()` | `claude-opus-4-20250514` | Latest Opus 4 |
| `Claude45Sonnet()` | `claude-sonnet-4-5-20250929` | Sonnet 4.5 |
| `Claude45Opus()` | `claude-opus-4-5-20251101` | Opus 4.5 |
| `Claude46Sonnet()` | `claude-sonnet-4-6` | Sonnet 4.6 |
| `Claude46Opus()` | `claude-opus-4-6` | Opus 4.6 |
| `ClaudeHaiku45()` | `claude-haiku-4-5-20251001` | Haiku 4.5 |

### Custom Model

```go
model := client.Model("claude-sonnet-4-6")
```

### Streaming Models

```go
// Non-streaming only
generator := client.Model("claude-sonnet-4-6")

// Streaming support
languageModel := client.StreamingModel("claude-sonnet-4-6")
```

## Configuration Options

### Provider Options

```go
client := anthropic.NewProvider(
    anthropic.APIKey("sk-ant-..."),
    anthropic.BaseURL("https://custom-endpoint.com"),  // Custom endpoint
    anthropic.HTTPClient(customHTTPClient),            // Custom HTTP client
    anthropic.WithHeader("X-Custom", "value"),         // Custom headers
    anthropic.WithMiddleware(                          // Middleware chain
        middleware.TimeoutWithDefault(30*time.Second),
        middleware.Logging(logger),
    ),
)
```

### Chat Model Options

```go
// Extended thinking (reasoning)
model := client.Claude46Sonnet(
    anthropic.ThinkingEnabled(2048),  // Enable with budget tokens
)

// Adaptive thinking
model := client.Claude46Sonnet(
    anthropic.ThinkingAdaptive(),  // Let model decide
)

// Disable thinking
model := client.Claude45Sonnet(
    anthropic.ThinkingDisabled(),
)

// Send reasoning content back to model
model := client.Claude45Sonnet(
    anthropic.SendReasoning(true),
)

// Disable parallel tool use
model := client.Claude45Sonnet(
    anthropic.DisableParallelToolUse(),
)

// Structured output mode
model := client.Claude45Sonnet(
    anthropic.StructuredOutputMode("outputFormat"),  // Native JSON schema
)
model := client.Claude45Sonnet(
    anthropic.StructuredOutputMode("jsonTool"),  // Fallback tool
)

// Prompt caching
model := client.Claude45Sonnet(
    anthropic.CacheControl("5m"),  // 5 minutes TTL
)
model := client.Claude45Sonnet(
    anthropic.CacheControl("1h"),  // 1 hour TTL
)

// Performance options
model := client.Claude46Opus(
    anthropic.Speed("fast"),      // Fast mode (Opus 4.6 only)
)
model := client.Claude45Sonnet(
    anthropic.Effort("high"),     // Reasoning effort: low, medium, high, max
)

// Tool streaming
model := client.Claude45Sonnet(
    anthropic.ToolStreaming(true),  // Fine-grained tool streaming
)
```

## Streaming

```go
import "github.com/seifalmotaz/lamar-sdk/stream"

func main() {
    client := anthropic.NewProvider()
    model := client.Claude45Sonnet()

    ctx := context.Background()
    result := stream.Stream(ctx, model, "Tell me a story")

    for part := range result.Stream() {
        switch p := part.(type) {
        case provider.StreamTextPart:
            fmt.Print(p.Delta)
        case provider.StreamToolCallPart:
            fmt.Printf("\nTool call: %s\n", p.ToolCall.Name)
        case provider.StreamErrorPart:
            fmt.Printf("\nError: %v\n", p.Error)
        case provider.StreamFinishPart:
            fmt.Println("\n--- Done ---")
        }
    }

    // Blocking access to final result
    text, _ := result.Text()
    usage, _ := result.Usage()
    fmt.Printf("Tokens: %d\n", usage.TotalTokens)
}
```

## Tool Calling

### Basic Tool Use

```go
import (
    "github.com/seifalmotaz/lamar-sdk/generate"
    "github.com/seifalmotaz/lamar-sdk/tool"
)

type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
    Unit     string `json:"unit" jsonschema:"enum=celsius,enum=fahrenheit"`
}

weatherTool := tool.NewTool(
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput) (map[string]any, error) {
        return map[string]any{
            "temperature": 22.5,
            "condition":   "sunny",
        }, nil
    },
)

toolDefs := tool.ToDefinitions(weatherTool)

result, err := generate.Generate(ctx, model, "What's the weather in Tokyo?",
    generate.Tools(toolDefs...),
    generate.ToolChoice(provider.ToolChoiceAuto()),
)
```

### Tool Choice Options

```go
// Let model decide (default)
generate.ToolChoice(provider.ToolChoiceAuto())

// Force at least one tool call
generate.ToolChoice(provider.ToolChoiceRequired())

// Force specific tool
generate.ToolChoice(provider.ToolChoiceNamed("get_weather"))

// Force single tool per response (Anthropic-specific)
client := anthropic.NewProvider()
model := client.Claude45Sonnet(
    anthropic.DisableParallelToolUse(),
)
```

### Multi-Turn Tool Calling

```go
messages := []provider.Message{
    provider.UserMessage("What's the weather in Tokyo?"),
}

for {
    result, err := generate.Generate(ctx, model, "",
        generate.Messages(messages...),
        generate.Tools(toolDefs...),
    )
    if err != nil {
        panic(err)
    }

    // Check for tool calls
    if len(result.ToolCalls()) == 0 {
        fmt.Println(result.Text())
        break
    }

    // Add assistant message with tool calls
    messages = append(messages, result.Message())

    // Execute tools and add results
    for _, tc := range result.ToolCalls() {
        output, err := weatherTool.Execute(ctx, tc.Input)
        if err != nil {
            panic(err)
        }

        messages = append(messages, provider.ToolResultMessage(
            provider.NewToolResultContentFromJSON(tc.ID, tc.Name, output, false),
        ))
    }
}
```

## Multimodal Content

### Vision (Images)

```go
imageData, _ := os.ReadFile("photo.jpg")
msg := provider.UserMessageWithContent(
    provider.Text("Describe this image in detail."),
    provider.Image(imageData, "image/jpeg"),
)

result, err := generate.Generate(ctx, model, "",
    generate.Messages(msg),
)
```

### URL-based Images

```go
msg := provider.UserMessageWithContent(
    provider.Text("What's in this image?"),
    provider.ImageFromURL("https://example.com/image.png"),
)
```

### Reasoning Content

```go
model := client.Claude46Sonnet(
    anthropic.ThinkingEnabled(2048),
)

result, err := generate.Generate(ctx, model, "Solve this complex problem...")

// Access reasoning content
for _, content := range result.Content() {
    if reasoning, ok := content.(provider.ReasoningContent); ok {
        fmt.Printf("Reasoning: %s\n", reasoning.Text)
    }
}
```

## Structured Output

### Using JSON Schema

```go
type Person struct {
    Name string `json:"name" jsonschema:"required,description=Full name"`
    Age  int    `json:"age" jsonschema:"required,minimum=0,maximum=150"`
}

result, err := generate.GenerateObject[Person](ctx, model,
    "Generate a random person",
)
if err != nil {
    panic(err)
}

fmt.Printf("Name: %s, Age: %d\n", result.Object.Name, result.Object.Age)
```

### Native Structured Output

```go
model := client.Claude45Sonnet(
    anthropic.StructuredOutputMode("outputFormat"),
)

// Uses native JSON schema output (requires compatible model)
result, err := generate.GenerateObject[Person](ctx, model, "Generate a person")
```

## System Prompts

```go
result, err := generate.Generate(ctx, model, "Hello",
    generate.System("You are a helpful coding assistant specialized in Go."),
)
```

### Multi-turn with System Prompt

```go
messages := []provider.Message{
    provider.UserMessage("Write a function to reverse a string."),
    provider.AssistantMessage("Here's a function..."),
    provider.UserMessage("Now add error handling."),
}

result, err := generate.Generate(ctx, model, "",
    generate.System("You are a Go expert."),
    generate.Messages(messages...),
)
```

## Generation Parameters

```go
result, err := generate.Generate(ctx, model, "prompt",
    generate.MaxTokens(1000),
    generate.Temperature(0.7),   // 0.0 to 1.0 for Claude
    generate.TopP(0.9),
    generate.TopK(50),
    generate.StopSequences("END", "STOP"),
    generate.System("You are helpful."),
)
```

## Middleware

```go
import "github.com/seifalmotaz/lamar-sdk/middleware"

client := anthropic.NewProvider(
    anthropic.APIKey("sk-ant-..."),
    anthropic.WithMiddleware(
        middleware.Chain(
            middleware.Recover(),
            middleware.Logging(logger),
            middleware.TimeoutWithDefault(60*time.Second),
            middleware.Retry(middleware.RetryConfig{
                MaxAttempts:  3,
                InitialDelay: time.Second,
                MaxDelay:     30 * time.Second,
                Multiplier:   2.0,
            }),
        ),
    ),
)
```

## Error Handling

```go
result, err := generate.Generate(ctx, model, "prompt")
if err != nil {
    var providerErr *provider.Error
    if errors.As(err, &providerErr) {
        switch providerErr.Code {
        case provider.CodeRateLimited:
            retryAfter := providerErr.RetryAfter
            fmt.Printf("Rate limited, retry after: %v\n", retryAfter)
        case provider.CodeAuthenticationFailed:
            fmt.Println("Invalid API key")
        case provider.CodeModelNotFound:
            fmt.Println("Model not found")
        case provider.CodeInvalidRequest:
            fmt.Printf("Invalid request: %s\n", providerErr.Message)
        case provider.CodeAPITimeout:
            fmt.Println("Request timed out")
        }
    }
    return
}

// Or use helper functions
if provider.IsRateLimited(err) {
    retryAfter := provider.RetryAfter(err)
    // Wait and retry
}
```

## Advanced Features

### Prompt Caching

```go
// Enable caching for system prompt
model := client.Claude45Sonnet(
    anthropic.CacheControl("1h"),
)

result, err := generate.Generate(ctx, model, "Long document...",
    generate.System("You are analyzing documents."),
)
```

### MCP Servers

```go
model := client.Claude45Sonnet(
    anthropic.WithMCPServers([]anthropic.MCPServerConfig{
        {
            Type:  "url",
            Name:  "my-server",
            URL:   "https://api.example.com/mcp",
            AuthorizationToken: "token",
        },
    }),
)
```

### Container/Code Execution

```go
model := client.Claude45Sonnet(
    anthropic.WithContainer("container-id", []anthropic.ContainerSkill{
        {
            Type:    "anthropic",
            SkillID: "code_execution",
        },
    }),
)
```

## Model Capabilities

All Claude models support:

- ✅ Text generation (non-streaming)
- ✅ Streaming generation
- ✅ Vision (images)
- ✅ Tool calling
- ❌ Embeddings (not supported by Anthropic API)
- ❌ Image generation (not supported by Anthropic API)
- ❌ Speech synthesis (not supported by Anthropic API)
- ❌ Audio transcription (not supported by Anthropic API)

### Capability Checking

```go
model := client.Claude45Sonnet()

if provider.CanGenerate(model) {
    // Supports Generate()
}

if provider.CanStream(model) {
    // Supports Stream()
}

if provider.IsLanguageModel(model) {
    // Supports both Generate() and Stream()
}
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/seifalmotaz/lamar-sdk/generate"
    "github.com/seifalmotaz/lamar-sdk/middleware"
    "github.com/seifalmotaz/lamar-sdk/providers/anthropic"
    "github.com/seifalmotaz/lamar-sdk/stream"
    "github.com/seifalmotaz/lamar-sdk/tool"
)

func main() {
    // Initialize with options
    client := anthropic.NewProvider(
        anthropic.APIKey(os.Getenv("ANTHROPIC_API_KEY")),
        anthropic.WithMiddleware(
            middleware.TimeoutWithDefault(60*time.Second),
        ),
    )

    // Create model with thinking enabled
    model := client.Claude46Sonnet(
        anthropic.ThinkingEnabled(2048),
        anthropic.Effort("high"),
    )

    ctx := context.Background()

    // Define a tool
    weatherTool := tool.NewTool(
        "get_weather",
        "Get current weather",
        func(ctx context.Context, input struct {
            Location string `json:"location"`
        }) (map[string]any, error) {
            return map[string]any{
                "temperature": 22.5,
                "condition":   "sunny",
            }, nil
        },
    )

    // Generate with tools
    result, err := generate.Generate(ctx, model,
        "What's the weather like in Tokyo?",
        generate.Tools(tool.ToDefinitions(weatherTool)...),
        generate.MaxTokens(1000),
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.Text())

    // Stream with system prompt
    streamResult := stream.Stream(ctx, model,
        "Explain quantum computing simply",
        stream.System("You are a science teacher."),
    )

    for part := range streamResult.Stream() {
        if text, ok := part.(provider.StreamTextPart); ok {
            fmt.Print(text.Delta)
        }
    }
}
```

## Differences from OpenAI Provider

| Feature | Anthropic | OpenAI |
|---------|-----------|--------|
| Embeddings | ❌ Not supported | ✅ Supported |
| Image generation | ❌ Not supported | ✅ DALL-E |
| Speech synthesis | ❌ Not supported | ✅ TTS |
| Audio transcription | ❌ Not supported | ✅ Whisper |
| Extended thinking | ✅ `ThinkingEnabled()` | ❌ Not supported |
| Prompt caching | ✅ `CacheControl()` | ❌ Not supported |
| MCP servers | ✅ `WithMCPServers()` | ❌ Not supported |
| Parameter restrictions | Temperature ignored when thinking enabled | All parameters work |

## See Also

- [Main README](../../README.md) - Full SDK documentation
- [Provider Interface Reference](../../README.md#provider-interface-reference) - Interface documentation
- [Tool Calling](../../README.md#toolfunction-calling) - Tool calling guide
- [Middleware System](../../README.md#middleware-system) - Middleware documentation