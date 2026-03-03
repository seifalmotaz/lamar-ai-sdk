# Lamar AI SDK

A Go SDK for building AI-powered applications with Large Language Models. Unified interface for multiple AI providers.

## Install

```bash
go get github.com/yourorg/lamar
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/yourorg/lamar"
    "github.com/yourorg/lamar/providers/openai"
)

func main() {
    client := openai.NewProvider() // Uses OPENAI_API_KEY env var
    
    // Generate text
    result, err := lamar.Generate(
        context.Background(),
        client.GPT4oMini(),
        "Say hello in 5 languages",
    )
    if err != nil {
        panic(err)
    }
    fmt.Println(result.Text)
}
```

## Features

- **Text Generation**: Streaming and non-streaming
- **Structured Output**: Type-safe JSON responses with schema validation
- **Embeddings**: Single and batch text embeddings
- **Tool Calling**: Type-safe function calling with generics
- **Multiple Providers**: OpenAI, Anthropic, Google (more coming)
- **Middleware**: Extensible request/response pipeline

## Documentation

See [docs/](./docs/) for detailed architecture and implementation guides.

## Status

🚧 **In Development** - See [PRD.md](./PRD.md) and [SRS.md](./SRS.md) for roadmap.

## License

MIT