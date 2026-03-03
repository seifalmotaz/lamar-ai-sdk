# Lamar AI SDK

A Go SDK for building AI-powered applications with Large Language Models. 
Unified, type-safe interface for multiple AI providers.

[![Go 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

## Installation

```bash
go get github.com/seifalmotaz/lamar-sdk
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/seifalmotaz/lamar-sdk/generate"
    "github.com/seifalmotaz/lamar-sdk/providers/openai"
)

func main() {
    // Initialize provider (uses OPENAI_API_KEY env var)
    client := openai.NewProvider()
    model := client.GPT4o()

    // Generate text
    result, err := generate.Generate(context.Background(), model, 
        "Say hello in 3 languages",
    )
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(result.Text())
    fmt.Printf("Tokens: %d\n", result.Usage().TotalTokens)
}
```

## Features

### Text Generation

Non-streaming and streaming text generation:

```go
// Non-streaming
result, err := generate.Generate(ctx, model, "Hello")

// Streaming
streamResult := stream.Stream(ctx, model, "Tell me a story")
for part := range streamResult.Stream() {
    if text, ok := part.(provider.StreamTextPart); ok {
        fmt.Print(text.Delta)
    }
}
text, _ := streamResult.Text()
```

### Structured Output

Type-safe JSON responses with automatic schema extraction:

```go
type Person struct {
    Name string `json:"name" jsonschema:"required"`
    Age  int    `json:"age" jsonschema:"required,minimum=0"`
}

result, err := generate.GenerateObject[Person](ctx, model, 
    "Generate a random person",
)
fmt.Printf("Name: %s, Age: %d\n", result.Object.Name, result.Object.Age)
```

### Embeddings

Single and batch embeddings:

```go
// Single
result, err := embed.Embed(ctx, model, "Hello world")

// Batch
texts := []string{"Hello", "World"}
batch, err := embed.EmbedBatch(ctx, model, texts)
```

### Tool Calling

Type-safe function calling:

```go
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
}

weatherTool := tool.NewTool("get_weather", "Get weather",
    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
        return WeatherOutput{Temperature: 22.5, Condition: "sunny"}, nil
    },
)

result, _ := generate.Generate(ctx, model, "Weather in Tokyo?",
    generate.Tools(tool.ToDefinition(weatherTool)),
)
```

### Middleware

Extensible request/response pipeline:

```go
// With logging and metrics
result, err := generate.Generate(ctx, model, "Hello",
    generate.WithLogger(&myLogger{}),
    generate.WithMetrics(&myMetrics{}),
)

// Custom retry
cfg := middleware.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: time.Second,
}
handler := middleware.Retry(cfg)(baseHandler)
```

## API Reference

### Core Packages

| Package | Description |
|---------|-------------|
| `generate` | Text generation with `Generate()` and `GenerateObject[T]()` |
| `stream` | Streaming text with `Stream()` and `StreamObject[T]()` |
| `embed` | Embeddings with `Embed()` and `EmbedBatch()` |
| `tool` | Type-safe tools with `NewTool[In, Out]()` |
| `provider` | Interfaces, types, and errors |
| `middleware` | Request/response middleware |

### Provider Packages

| Provider | Package | Support |
|----------|---------|---------|
| OpenAI | `providers/openai` | Generate, Stream, Embed |

## Examples

See [examples/](./examples/) for complete working examples:

- **[chat](./examples/openai/chat/)** - Basic text generation
- **[stream](./examples/openai/stream/)** - Streaming generation
- **[embed](./examples/openai/embed/)** - Embedding generation
- **[tools](./examples/openai/tools/)** - Tool calling
- **[structured](./examples/openai/structured/)** - Structured output
- **[middleware](./examples/openai/middleware/)** - Middleware usage

## Architecture

### Interface Segregation

```
Model (base) → Generator (non-streaming) → LanguageModel (both)
            → Streamer (streaming)       →
```

Models implement only what they support. Use capability checks:
- `provider.CanGenerate(m)` - Check if model supports generation
- `provider.CanStream(m)` - Check if model supports streaming

### Thread-Safe Streaming

Stream results use mutex synchronization for concurrent access:

```go
result := stream.Stream(ctx, model, "prompt")

// Consume in real-time
for part := range result.Stream() { }

// Or wait and get final result
text, err := result.Text()
```

### Typed Errors

All errors use structured codes:

```go
var providerErr *provider.Error
if errors.As(err, &providerErr) {
    switch providerErr.Code {
    case provider.CodeRateLimited:
        // Handle rate limit
    case provider.CodeAPITimeout:
        // Handle timeout
    }
}
```

## Configuration

### Functional Options

All public APIs use the functional options pattern:

```go
result, err := generate.Generate(ctx, model, "prompt",
    generate.MaxTokens(100),
    generate.Temperature(0.7),
    generate.System("You are helpful."),
    generate.WithTimeout(30*time.Second),
)
```

### Context Cancellation

All functions support context cancellation:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := generate.Generate(ctx, model, "prompt")
```

## Testing

Run unit tests:

```bash
go test ./...
```

Run integration tests (requires `OPENAI_API_KEY`):

```bash
OPENAI_API_KEY=your-key go test ./tests/integration/...
```

## Project Structure

```
lamar-sdk/
├── provider/          # Interfaces and types
├── generate/          # Generate API
├── stream/            # Streaming API
├── embed/             # Embedding API
├── tool/              # Tool definitions
├── middleware/        # Middleware chain
├── internal/           # Private implementation
│   ├── schema/        # JSON schema extraction
│   ├── sse/           # SSE parsing
│   ├── httpx/         # HTTP utilities
│   └── contract/      # Contract tests
├── providers/         # Provider implementations
│   └── openai/        # OpenAI provider
├── examples/          # Example applications
└── tests/             # Integration tests
```

## License

MIT License - see [LICENSE](LICENSE) for details.

## Status

**Phase 1 Complete** - Core functionality implemented with OpenAI provider.

See [PRD.md](./PRD.md) and [SRS.md](./SRS.md) for roadmap.