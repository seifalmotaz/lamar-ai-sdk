# Ollama Provider

Ollama provider for Lamar AI SDK. Supports local LLM inference via Ollama.

## Installation

```go
import "github.com/seifalmotaz/lamar-ai-sdk/providers/ollama"
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/seifalmotaz/lamar-ai-sdk/generate"
    "github.com/seifalmotaz/lamar-ai-sdk/providers/ollama"
)

func main() {
    // Create provider (connects to http://127.0.0.1:11434 by default)
    client := ollama.NewProvider()
    
    // Create model
    model := client.Llama32()
    
    // Generate text
    ctx := context.Background()
    result, err := generate.Generate(ctx, model, "Hello, world!")
    if err != nil {
        panic(err)
    }
    
    fmt.Println(result.Text)
}
```

## Configuration

### Base URL

Use a custom Ollama server:

```go
client := ollama.NewProvider(
    ollama.BaseURL("http://localhost:8080"),
)
```

Or via environment variable:

```bash
export OLLAMA_HOST=http://localhost:8080
```

### Custom HTTP Client

```go
import "net/http"

client := ollama.NewProvider(
    ollama.HTTPClient(&http.Client{
        Timeout: 60 * time.Second,
    }),
)
```

### Middleware

```go
import "github.com/seifalmotaz/lamar-ai-sdk/middleware"

client := ollama.NewProvider(
    ollama.WithMiddleware(
        middleware.TimeoutWithDefault(30 * time.Second),
        middleware.Logging(logger),
    ),
)
```

## Supported Models

### Chat Models

| Method | Model ID | Description |
|--------|----------|-------------|
| `Llama32()` | llama3.2 | Llama 3.2 |
| `Llama3()` | llama3 | Llama 3 |
| `Llama31()` | llama3.1 | Llama 3.1 |
| `Llama31_8B()` | llama3.1:8b | Llama 3.1 8B |
| `Llama31_70B()` | llama3.1:70b | Llama 3.1 70B |
| `Llama33()` | llama3.3 | Llama 3.3 |
| `Llama33_70B()` | llama3.3:70b | Llama 3.3 70B |
| `Qwen25()` | qwen2.5 | Qwen 2.5 |
| `Qwen25_7B()` | qwen2.5:7b | Qwen 2.5 7B |
| `Qwen25_14B()` | qwen2.5:14b | Qwen 2.5 14B |
| `Qwen3()` | qwen3 | Qwen 3 |
| `DeepSeekR1()` | deepseek-r1 | DeepSeek R1 (reasoning) |
| `DeepSeekV3()` | deepseek-v3 | DeepSeek V3 |
| `Phi3()` | phi3 | Phi 3 |
| `Phi3Mini()` | phi3:mini | Phi 3 Mini |
| `Phi35()` | phi3.5 | Phi 3.5 |
| `Mistral()` | mistral | Mistral |
| `Mistral7B()` | mistral:7b | Mistral 7B |
| `Mixtral()` | mixtral | Mixtral |
| `Codellama()` | codellama | Code Llama |
| `Llava()` | llava | LLaVA (vision) |
| `Gemma2()` | gemma2 | Gemma 2 |
| `Gemma2_9B()` | gemma2:9b | Gemma 2 9B |
| `CommandR()` | command-r | Command R |

### Embedding Models

| Method | Model ID | Description |
|--------|----------|-------------|
| `NomicEmbedText()` | nomic-embed-text | Nomic Embed Text |
| `MxbaiEmbedLarge()` | mxbai-embed-large | Mxbai Embed Large |
| `AllMinilm()` | all-minilm | All MiniLM |

### Custom Model

Use any Ollama model:

```go
model := client.StreamingModel("your-model-name")
embedModel := client.Embedding("your-embedding-model")
```

## Usage Examples

### Basic Generation

```go
result, err := generate.Generate(ctx, model, "What is Go?")
```

### Streaming

```go
import "github.com/seifalmotaz/lamar-ai-sdk/stream"

result := stream.Stream(ctx, model, "Tell me a story")

for part := range result.Stream() {
    switch p := part.(type) {
    case provider.StreamTextPart:
        fmt.Print(p.Delta)
    case provider.StreamFinishPart:
        fmt.Println("\nDone!")
    }
}

// Or get final text
text, _ := result.Text()
fmt.Println(text)
```

### With Options

```go
temp := 0.7
ctxWindow := 8192

config := &ollama.ChatConfig{
    Temperature: &temp,
    NumCtx:      &ctxWindow,
}

model := client.ModelWithConfig("llama3.2", config)
```

### Functional Options

```go
config := ollama.newChatConfig(
    ollama.ChatTemperature(0.7),
    ollama.ChatNumCtx(8192),
    ollama.ChatRepeatPenalty(1.1),
)

model := client.ModelWithConfig("llama3.2", config)
```

### Vision (Multimodal)

```go
imageData, _ := os.ReadFile("image.png")

msg := provider.UserMessageWithContent(
    provider.Text("What's in this image?"),
    provider.Image(imageData, "image/png"),
)

result, err := generate.Generate(ctx, model, "",
    generate.Messages(msg),
)
```

### Tool Calling

```go
import (
    "github.com/seifalmotaz/lamar-ai-sdk/generate"
    "github.com/seifalmotaz/lamar-ai-sdk/tool"
)

// Define tool
weatherTool := tool.NewTool("get_weather", "Get weather for location", 
    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
        return WeatherOutput{Temperature: 22, Condition: "sunny"}, nil
    })

toolDefs := tool.ToDefinitions(weatherTool)

result, err := generate.Generate(ctx, model, "What's the weather in Tokyo?",
    generate.Tools(toolDefs...),
    generate.ToolChoice(provider.ToolChoiceAuto()),
)

// Handle tool calls
for _, tc := range result.ToolCalls() {
    fmt.Printf("Tool: %s, Input: %s\n", tc.Name, string(tc.Input))
}
```

### Embeddings

```go
import "github.com/seifalmotaz/lamar-ai-sdk/embed"

embedModel := client.NomicEmbedText()

result, err := embed.Embed(ctx, embedModel, "Hello world")
fmt.Printf("Embedding: %d dimensions\n", len(result.Embedding))

// Batch embeddings
batchResult, err := embed.EmbedBatch(ctx, embedModel, []string{
    "hello",
    "world",
})
```

## Ollama-Specific Options

### Reasoning Mode (DeepSeek-R1, Qwen3)

Enable thinking/reasoning for supported models:

```go
think := true
config := &ollama.ChatConfig{
    Think: &think,
}
```

### Keep-Alive

Keep model loaded in memory:

```go
config := &ollama.ChatConfig{
    KeepAlive: "10m", // Keep loaded for 10 minutes
}
```

### Native Sampling Options

```go
mirostat := 2
repeatPenalty := 1.1

config := &ollama.ChatConfig{
    Mirostat:      &mirostat,
    RepeatPenalty: &repeatPenalty,
    NumCtx:        &ctxWindow,
    NumPredict:    &maxTokens,
    TFSZ:          &tfsz,
    MinP:          &minP,
}
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OLLAMA_HOST` | Ollama server URL | `http://127.0.0.1:11434` |

## Error Handling

```go
import "errors"
import "github.com/seifalmotaz/lamar-ai-sdk/provider"

result, err := generate.Generate(ctx, model, "prompt")
if err != nil {
    var pErr *provider.Error
    if errors.As(err, &pErr) {
        switch pErr.Code {
        case provider.CodeModelNotFound:
            fmt.Println("Model not found")
        case provider.CodeAPITimeout:
            fmt.Println("Timeout - is Ollama running?")
        default:
            fmt.Printf("Error: %s\n", pErr.Message)
        }
    }
}
```

## API Compatibility

| Feature | Support |
|---------|---------|
| Text Generation | ✅ |
| Streaming | ✅ |
| Tool Calling | ✅ |
| Vision | ✅ |
| Embeddings | ✅ |
| Structured Output | ✅ (JSON mode) |
| Reasoning/Thinking | ✅ |

## Requirements

- Go 1.23 or later
- Ollama running locally or accessible via network

## Installation of Ollama

```bash
# macOS/Linux
curl -fsSL https://ollama.com/install.sh | sh

# Pull a model
ollama pull llama3.2

# Run server (usually starts automatically)
ollama serve
```

## License

MIT