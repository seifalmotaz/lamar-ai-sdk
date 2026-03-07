# Lamar AI SDK

A Go SDK for building AI-powered applications with Large Language Models.  
Unified, type-safe interface for multiple AI providers.

[![Go 1.23+](https://img.shields.io/badge/Go-1.23%2B-00ADD8?style=flat&logo=go)](https://golang.org/dl/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **⚠️ Early Development Stage**  
> This project is under heavy development and is considered a side project for a larger project that depends on it. Expect potential bugs, breaking changes, and design shifts. The API may change significantly between versions. Not recommended for production use until a stable release.

---

## Table of Contents

- [Installation](#installation)
- [Quick Start](#quick-start)
- [Core Concepts](#core-concepts)
- [Text Generation](#text-generation)
- [Streaming Generation](#streaming-generation)
- [Structured Output](#structured-output)
- [Tool/Function Calling](#toolfunction-calling)
- [Embeddings](#embeddings)
- [Multimodal Content](#multimodal-content)
- [Image Generation](#image-generation)
- [Speech Synthesis](#speech-synthesis)
- [Audio Transcription](#audio-transcription)
- [Agent Framework](#agent-framework)
- [Middleware System](#middleware-system)
- [Error Handling](#error-handling)
- [Content Types Reference](#content-types-reference)
- [Provider Interface Reference](#provider-interface-reference)
- [Creating a New Provider](#creating-a-new-provider)
- [OpenAI Provider Reference](#openai-provider-reference)
- [Type System Reference](#type-system-reference)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [API Stability](#api-stability)
- [Comparison with Vercel AI SDK](#comparison-with-vercel-ai-sdk)

---

## Installation

```bash
go get github.com/seifalmotaz/lamar-sdk
```

**Requirements:**

- Go 1.23 or later

---

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
    // Initialize provider (uses OPENAI_API_KEY environment variable)
    client := openai.NewProvider()
    model := client.GPT5Mini()

    // Generate text with context
    ctx := context.Background()
    result, err := generate.Generate(ctx, model, "Say hello in 3 languages")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }

    fmt.Println(result.Text())
    fmt.Printf("Tokens used: %d\n", result.Usage().TotalTokens)
}
```

---

## Core Concepts

### Architecture Overview

Lamar SDK follows an **interface segregation** pattern where models implement only the capabilities they support:

```
Model (base interface)
    ├── Generator (non-streaming generation)
    │       └── LanguageModel (Generator + Streamer)
    ├── Streamer (streaming generation)
    │       └── LanguageModel (Generator + Streamer)
    ├── EmbeddingModel (text embeddings)
    ├── ImageModel (image generation)
    ├── TranscriptionModel (audio transcription)
    └── SpeechModel (text-to-speech)
```

This design allows type-safe capability checking:

```go
// Check if model supports generation
if provider.CanGenerate(model) {
    result, err := generate.Generate(ctx, model, "prompt")
}

// Check if model supports streaming
if provider.CanStream(model) {
    result := stream.Stream(ctx, model, "prompt")
}

// Check if model supports embeddings
if provider.CanEmbed(model) {
    result, err := embed.Embed(ctx, model, "text")
}

// Check if model is a full LanguageModel
if provider.IsLanguageModel(model) {
    // Use for both Generate and Stream
}
```

### Capability System

Models can declare their capabilities for runtime introspection:

```go
// Check specific capabilities
if provider.HasCapability(model, provider.CapVision) {
    // Model supports image understanding
}

if provider.HasCapability(model, provider.CapTools) {
    // Model supports function calling
}

// Available capabilities
const (
    CapStreaming        Capability = "streaming"         // Supports streaming
    CapTools            Capability = "tools"             // Supports function calling
    CapVision           Capability = "vision"            // Supports image input
    CapAudio            Capability = "audio"             // Supports audio I/O
    CapJSON             Capability = "json"              // Supports structured output
    CapReasoning        Capability = "reasoning"         // Supports reasoning (O1)
    CapImageGeneration  Capability = "image_generation"  // Supports image generation
    CapTranscription    Capability = "transcription"     // Supports audio transcription
    CapSpeech           Capability = "speech"            // Supports TTS
)
```

### Error Handling Philosophy

All errors use structured types with error codes, enabling programmatic handling:

```go
import "errors"

result, err := generate.Generate(ctx, model, "prompt")
if err != nil {
    var providerErr *provider.Error
    if errors.As(err, &providerErr) {
        switch providerErr.Code {
        case provider.CodeRateLimited:
            retryAfter := providerErr.RetryAfter
            // Wait and retry
        case provider.CodeAuthenticationFailed:
            // Check API key
        case provider.CodeAPITimeout:
            // Handle timeout
        }
    }
}

// Or use helper functions
if provider.IsRateLimited(err) {
    retryAfter := provider.RetryAfter(err)
}
```

### Context-First Design

Every public function takes `context.Context` as the first parameter:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
result, err := generate.Generate(ctx, model, "prompt")

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
go func() {
    time.Sleep(5 * time.Second)
    cancel() // Cancel after 5 seconds
}()
result, err := generate.Generate(ctx, model, "prompt")
```

### Functional Options Pattern

All configuration uses functional options for a clean, extensible API:

```go
// Provider configuration
client := openai.NewProvider(
    openai.APIKey("your-api-key"),
    openai.BaseURL("https://custom-endpoint.com"),
    openai.WithMiddleware(middleware.TimeoutWithDefault(30*time.Second)),
)

// Generation configuration
result, err := generate.Generate(ctx, model, "prompt",
    generate.MaxTokens(500),
    generate.Temperature(0.7),
    generate.System("You are a helpful assistant."),
)
```

---

## Text Generation

### Basic Usage

```go
import (
    "context"
    "github.com/seifalmotaz/lamar-sdk/generate"
    "github.com/seifalmotaz/lamar-sdk/providers/openai"
)

func main() {
    client := openai.NewProvider()
    model := client.GPT5Mini()

    ctx := context.Background()
    result, err := generate.Generate(ctx, model, "What is the capital of France?")
    if err != nil {
        panic(err)
    }

    fmt.Println(result.Text())
}
```

### Configuration Options

```go
result, err := generate.Generate(ctx, model, "prompt",
    // System prompt
    generate.System("You are a helpful assistant."),

    // Model parameters
    generate.MaxTokens(1000),
    generate.Temperature(0.7),       // 0.0 to 2.0
    generate.TopP(0.9),              // 0.0 to 1.0
    generate.TopK(50),               // Top-K sampling
    generate.Seed(42),               // Deterministic sampling

    // Stop sequences
    generate.StopSequences("END", "STOP"),

    // Tools (function calling)
    generate.Tools(toolDefinitions...),
    generate.ToolChoice(provider.ToolChoiceAuto()),

    // Response format
    generate.ResponseFormat(provider.ResponseFormatJSON()),

    // Context and observability
    generate.WithTimeout(30*time.Second),
    generate.WithLogger(myLogger),
    generate.WithMetrics(myMetrics),
)
```

### Working with Messages

Build multi-turn conversations:

```go
messages := []provider.Message{
    provider.SystemMessage("You are a helpful coding assistant."),
    provider.UserMessage("Write a function to reverse a string."),
    provider.AssistantMessage("Here's a function to reverse a string..."),
    provider.UserMessage("Now make it handle Unicode correctly."),
}

result, err := generate.Generate(ctx, model, "",
    generate.Messages(messages...),
)
```

### Accessing Results

```go
result, err := generate.Generate(ctx, model, "prompt")

// Text content
text := result.Text()

// Raw content parts
content := result.Content()
for _, c := range content {
    switch v := c.(type) {
    case provider.TextContent:
        fmt.Println("Text:", v.Text)
    case provider.ToolCallContent:
        fmt.Printf("Tool call: %s\n", v.Name)
    }
}

// Tool calls (if any)
toolCalls := result.ToolCalls()
for _, tc := range toolCalls {
    fmt.Printf("Tool: %s, Input: %s\n", tc.Name, string(tc.Input))
}

// Finish reason
switch result.FinishReason() {
case provider.FinishReasonStop:
    // Normal completion
case provider.FinishReasonLength:
    // Hit max_tokens limit
case provider.FinishReasonToolCalls:
    // Model wants to call tools
}

// Token usage
usage := result.Usage()
fmt.Printf("Prompt: %d, Completion: %d, Total: %d\n",
    usage.PromptTokens,
    usage.CompletionTokens,
    usage.TotalTokens,
)
```

---

## Streaming Generation

### Basic Streaming

```go
import "github.com/seifalmotaz/lamar-sdk/stream"

func main() {
    client := openai.NewProvider()
    model := client.GPT5Mini() // LanguageModel supports streaming

    ctx := context.Background()
    result := stream.Stream(ctx, model, "Tell me a short story")

    // Consume stream in real-time
    for part := range result.Stream() {
        switch p := part.(type) {
        case provider.StreamTextPart:
            fmt.Print(p.Delta) // Print each token as it arrives
        case provider.StreamToolCallPart:
            fmt.Printf("\nTool call: %s\n", p.ToolCall.Name)
        case provider.StreamErrorPart:
            fmt.Printf("\nError: %v\n", p.Error)
        case provider.StreamFinishPart:
            fmt.Println("\n--- Stream finished ---")
        }
    }
}
```

### Thread-Safe Result Access

Stream results are thread-safe. You can access the final result even while consuming the stream:

```go
result := stream.Stream(ctx, model, "prompt")

// Start goroutine to consume stream
go func() {
    for part := range result.Stream() {
        if text, ok := part.(provider.StreamTextPart); ok {
            fmt.Print(text.Delta)
        }
    }
}()

// In another goroutine, wait for final result
go func() {
    <-result.Done() // Wait for stream to complete

    text, err := result.Text()
    if err != nil {
        panic(err)
    }
    fmt.Println("\nFinal text:", text)
}()
```

### Blocking for Final Result

```go
result := stream.Stream(ctx, model, "prompt", stream.MaxTokens(100))

// Block until stream completes, then get final values
text, err := result.Text()
if err != nil {
    panic(err)
}

usage, _ := result.Usage()
finishReason, _ := result.FinishReason()

fmt.Printf("Text: %s\n", text)
fmt.Printf("Tokens: %d\n", usage.TotalTokens)
```

### Stream Options

```go
result := stream.Stream(ctx, model, "prompt",
    stream.System("You are helpful."),
    stream.MaxTokens(500),
    stream.Temperature(0.7),
    stream.Messages(messages...),
    stream.Tools(toolDefinitions...),
    stream.WithTimeout(60*time.Second),
)
```

---

## Structured Output

### Basic Structured Output

Use `GenerateObject[T]` for type-safe JSON responses with automatic schema extraction:

```go
type Person struct {
    Name string `json:"name" jsonschema:"required,description=Full name of the person"`
    Age  int    `json:"age" jsonschema:"required,minimum=0,maximum=150,description=Age in years"`
    City string `json:"city" jsonschema:"description=City of residence"`
}

result, err := generate.GenerateObject[Person](ctx, model,
    "Generate a random person from a fictional city",
)
if err != nil {
    panic(err)
}

person := result.Object
fmt.Printf("Name: %s, Age: %d, City: %s\n", person.Name, person.Age, person.City)
```

### Schema Tags Reference

Use `jsonschema` struct tags to define constraints:

```go
type Product struct {
    Name        string  `json:"name" jsonschema:"required,description=Product name"`
    Price       float64 `json:"price" jsonschema:"required,minimum=0,description=Price in USD"`
    InStock     bool    `json:"in_stock" jsonschema:"description=Whether product is available"`
    Quantity    int     `json:"quantity" jsonschema:"minimum=0,maximum=1000"`
    Category    string  `json:"category" jsonschema:"enum=electronics,enum=clothing,enum=food"`
    Tags        []string `json:"tags" jsonschema:"minItems=1,maxItems=10"`
    Description string  `json:"description,omitempty" jsonschema:"maxLength=500"`
}
```

**Available tags:**

| Tag           | Description              | Example                             |
| ------------- | ------------------------ | ----------------------------------- |
| `required`    | Field is required        | `jsonschema:"required"`             |
| `description` | Field description        | `jsonschema:"description=The name"` |
| `minimum`     | Minimum value (numbers)  | `jsonschema:"minimum=0"`            |
| `maximum`     | Maximum value (numbers)  | `jsonschema:"maximum=100"`          |
| `minLength`   | Minimum string length    | `jsonschema:"minLength=1"`          |
| `maxLength`   | Maximum string length    | `jsonschema:"maxLength=100"`        |
| `enum`        | Enum values (repeatable) | `jsonschema:"enum=a,enum=b"`        |
| `minItems`    | Minimum array items      | `jsonschema:"minItems=1"`           |
| `maxItems`    | Maximum array items      | `jsonschema:"maxItems=10"`          |

### Complex Nested Types

```go
type Address struct {
    Street  string `json:"street" jsonschema:"required"`
    City    string `json:"city" jsonschema:"required"`
    Country string `json:"country" jsonschema:"required"`
}

type Company struct {
    Name    string  `json:"name" jsonschema:"required"`
    Address Address `json:"address" jsonschema:"required"`
    Employees int   `json:"employees" jsonschema:"minimum=1"`
}

type Employee struct {
    Name    string   `json:"name" jsonschema:"required"`
    Company Company  `json:"company" jsonschema:"required"`
    Skills  []string `json:"skills" jsonschema:"minItems=1,maxItems=10"`
}

result, err := generate.GenerateObject[Employee](ctx, model,
    "Generate an employee at a tech company",
)
```

### Streaming Structured Output

```go
result := stream.StreamObject[Person](ctx, model, "Generate a person")

for part := range result.Stream() {
    switch part.Type {
    case "partial":
        fmt.Printf("Partial: %+v\n", part.Object)
    case "complete":
        fmt.Println("Stream complete")
    }
}

// Get final object
person, err := result.Object()
if err != nil {
    panic(err)
}
```

### Structured Output Options

```go
result, err := generate.GenerateObject[Person](ctx, model, "prompt",
    generate.System("Generate realistic data."),
    generate.MaxTokens(500),
    generate.Temperature(0.8),
)
```

---

## Tool/Function Calling

### Type-Safe Tool Definition

```go
import "github.com/seifalmotaz/lamar-sdk/tool"

// Define input/output types with schema tags
type WeatherInput struct {
    Location string `json:"location" jsonschema:"required,description=City name"`
    Unit     string `json:"unit" jsonschema:"enum=celsius,enum=fahrenheit"`
}

type WeatherOutput struct {
    Temperature float64 `json:"temperature"`
    Condition   string  `json:"condition"`
    Humidity    int     `json:"humidity"`
}

// Create type-safe tool
weatherTool := tool.NewTool(
    "get_weather",
    "Get current weather for a location",
    func(ctx context.Context, input WeatherInput) (WeatherOutput, error) {
        // Your implementation here
        return WeatherOutput{
            Temperature: 22.5,
            Condition:   "sunny",
            Humidity:    45,
        }, nil
    },
)
```

### Using Tools in Generation

```go
// Convert to tool definition
toolDefs := tool.ToDefinitions(weatherTool)

result, err := generate.Generate(ctx, model, "What's the weather in Tokyo?",
    generate.Tools(toolDefs...),
    generate.ToolChoice(provider.ToolChoiceAuto()),
)
if err != nil {
    panic(err)
}

// Handle tool calls
for _, tc := range result.ToolCalls() {
    if tc.Name == "get_weather" {
        // Execute the tool
        output, err := weatherTool.Execute(ctx, tc.Input)
        if err != nil {
            panic(err)
        }

        // Create tool result message
        toolResult := provider.NewToolResultContent(
            tc.ID,
            tc.Name,
            output,
            false, // not an error
        )

        // Continue conversation with tool result
        messages := []provider.Message{
            provider.UserMessage("What's the weather in Tokyo?"),
            result.Message(), // Assistant's message with tool calls
            provider.ToolResultMessage(toolResult),
        }

        // Get final response
        finalResult, err := generate.Generate(ctx, model, "",
            generate.Messages(messages...),
        )
    }
}
```

### Tool Choice Options

```go
// Let model decide (default)
generate.ToolChoice(provider.ToolChoiceAuto())

// Force no tool calls
generate.ToolChoice(provider.ToolChoiceNone())

// Force at least one tool call
generate.ToolChoice(provider.ToolChoiceRequired())

// Force specific tool
generate.ToolChoice(provider.ToolChoiceNamed("get_weather"))
```

### Multiple Tools

```go
type SearchInput struct {
    Query string `json:"query" jsonschema:"required"`
}

type CalculatorInput struct {
    Expression string `json:"expression" jsonschema:"required"`
}

searchTool := tool.NewTool("search", "Search the web",
    func(ctx context.Context, input SearchInput) (string, error) {
        return "Search results...", nil
    },
)

calcTool := tool.NewTool("calculate", "Evaluate math expressions",
    func(ctx context.Context, input CalculatorInput) (float64, error) {
        return 42.0, nil
    },
)

toolDefs := tool.ToDefinitions(searchTool, calcTool)

result, err := generate.Generate(ctx, model, "Search for Go tutorials and calculate 5 * 10",
    generate.Tools(toolDefs...),
)
```

### Tool Result Handling

```go
// Create tool result content
toolResult := provider.NewToolResultContent(
    toolCallID,
    toolName,
    json.RawMessage(resultJSON),
    isError, // true if tool execution failed
)

// Or from any Go value (automatically JSON marshaled)
toolResult := provider.NewToolResultContentFromJSON(
    toolCallID,
    toolName,
    WeatherOutput{Temperature: 22.5, Condition: "sunny"},
    false,
)

// Add to conversation
messages = append(messages, provider.ToolResultMessage(toolResult))
```

---

## Embeddings

### Single Embedding

```go
import "github.com/seifalmotaz/lamar-sdk/embed"

func main() {
    client := openai.NewProvider()
    model := client.TextEmbedding3Small()

    ctx := context.Background()
    result, err := embed.Embed(ctx, model, "Hello, world!")
    if err != nil {
        panic(err)
    }

    fmt.Printf("Embedding dimension: %d\n", len(result.Embedding))
    fmt.Printf("First 5 values: %v\n", result.Embedding[:5])
    fmt.Printf("Tokens used: %d\n", result.Usage.TotalTokens)
}
```

### Batch Embeddings

```go
texts := []string{
    "Hello, world!",
    "Goodbye, world!",
    "The quick brown fox",
}

result, err := embed.EmbedBatch(ctx, model, texts)
if err != nil {
    panic(err)
}

fmt.Printf("Generated %d embeddings\n", len(result.Embeddings))
for i, emb := range result.Embeddings {
    fmt.Printf("Text %d: dimension %d\n", i, len(emb))
}
```

### Embedding Options

```go
result, err := embed.Embed(ctx, model, text,
    embed.WithTimeout(30*time.Second),
    embed.WithLogger(myLogger),
    embed.WithMetrics(myMetrics),
)
```

### Embedding Models

| Model                   | Dimensions | Use Case                        |
| ----------------------- | ---------- | ------------------------------- |
| `TextEmbedding3Small()` | 1536       | General purpose, cost-effective |
| `TextEmbedding3Large()` | 3072       | Higher quality, larger vectors  |
| `TextEmbeddingAda002()` | 1536       | Legacy, GPT-3 era               |

---

## Multimodal Content

### Content Types Overview

The SDK supports multiple content types through a polymorphic `Content` interface:

```go
type Content interface {
    content() // Private marker method
}

// Available content types:
// - TextContent: Plain text
// - ImageContent: Image data (base64 or URL)
// - AudioContent: Audio data
// - ToolCallContent: Tool/function call
// - ToolResultContent: Tool execution result
// - ReasoningContent: Model reasoning (O1-style)
```

### Text Content

```go
textContent := provider.Text("Hello, world!")

// Or in a message
msg := provider.UserMessage("Hello!")
```

### Image Content

```go
// From base64 data
imageData, _ := os.ReadFile("image.png")
img := provider.Image(imageData, "image/png")

// From URL
img := provider.ImageFromURL("https://example.com/image.png")

// Use in message
msg := provider.UserMessageWithContent(
    provider.Text("What's in this image?"),
    img,
)
```

### Audio Content

```go
// From base64 data
audioData, _ := os.ReadFile("audio.mp3")
audio := provider.Audio(audioData, "audio/mp3")

// From URL
audio := provider.AudioFromURL("https://example.com/audio.mp3")

// Use in message (for audio-capable models like GPT-4o-audio)
msg := provider.UserMessageWithContent(
    provider.Text("What is being said?"),
    audio,
)
```

### Vision Example

```go
client := openai.NewProvider()
model := client.GPT5Mini() // GPT-5 supports vision

imageData, _ := os.ReadFile("photo.jpg")
msg := provider.UserMessageWithContent(
    provider.Text("Describe this image in detail."),
    provider.Image(imageData, "image/jpeg"),
)

result, err := generate.Generate(ctx, model, "",
    generate.Messages(msg),
)
```

### Audio Chat Example

```go
client := openai.NewProvider()
model := client.GPT4oAudioPreview() // GPT-4o supports audio

audioData, _ := os.ReadFile("recording.mp3")
messages := []provider.Message{
    provider.UserMessageWithContent(
        provider.Audio(audioData, "audio/mp3"),
        provider.Text("Transcribe and summarize this audio."),
    ),
}

result, err := generate.Generate(ctx, model, "",
    generate.Messages(messages...),
)
```

### Type Switching on Content

```go
for _, content := range result.Content() {
    switch c := content.(type) {
    case provider.TextContent:
        fmt.Printf("Text: %s\n", c.Text)
    case provider.ImageContent:
        fmt.Printf("Image: %d bytes, type: %s\n", len(c.Data), c.MediaType)
    case provider.AudioContent:
        fmt.Printf("Audio: %d bytes, type: %s\n", len(c.Data), c.MediaType)
    case provider.ToolCallContent:
        fmt.Printf("Tool call: %s(%s)\n", c.Name, string(c.Input))
    case provider.ToolResultContent:
        fmt.Printf("Tool result: %s = %s\n", c.Name, string(c.Result))
    case provider.ReasoningContent:
        fmt.Printf("Reasoning: %s\n", c.Text)
    }
}
```

---

## Image Generation

### Basic Image Generation

```go
import "github.com/seifalmotaz/lamar-sdk/image"

func main() {
    client := openai.NewProvider()
    model := client.DALLE3()

    ctx := context.Background()
    result, err := image.Generate(ctx, model, "A serene mountain landscape at sunset")
    if err != nil {
        panic(err)
    }

    // Save the generated image
    for i, imgData := range result.Images {
        filename := fmt.Sprintf("generated_%d.png", i)
        os.WriteFile(filename, imgData, 0644)
    }
}
```

### Image Options

```go
result, err := image.Generate(ctx, model, "prompt",
    image.N(4),                    // Number of images (DALL-E 2 only)
    image.Size("1024x1024"),       // Size: 256x256, 512x512, 1024x1024
    image.Quality("hd"),           // Quality: "standard" or "hd" (DALL-E 3)
    image.Format("png"),           // Format: "png", "jpeg", "webp"
    image.WithTimeout(60*time.Second),
)
```

### Image Models

| Model       | Method        | Notes                             |
| ----------- | ------------- | --------------------------------- |
| DALL-E 2    | `DALLE2()`    | Supports N > 1, smaller sizes     |
| DALL-E 3    | `DALLE3()`    | Higher quality, HD option         |
| GPT Image 1 | `GPTImage1()` | Latest model with editing support |

### Working with Results

```go
result, err := image.Generate(ctx, model, "prompt")

// Generated images
for i, imgData := range result.Images {
    fmt.Printf("Image %d: %d bytes\n", i, len(imgData))
}

// Revised prompts (DALL-E 3)
for i, prompt := range result.RevisedPrompts {
    fmt.Printf("Prompt %d: %s\n", i, prompt)
}

// Usage information
fmt.Printf("Images generated: %d\n", len(result.Images))
```

---

## Speech Synthesis

### Basic Text-to-Speech

```go
import "github.com/seifalmotaz/lamar-sdk/speech"

func main() {
    client := openai.NewProvider()
    model := client.TTS1()

    ctx := context.Background()
    result, err := speech.Synthesize(ctx, model, "Hello, this is a test.")
    if err != nil {
        panic(err)
    }

    // Save the audio
    os.WriteFile("output.mp3", result.Audio, 0644)
}
```

### Speech Options

```go
result, err := speech.Synthesize(ctx, model, "text to speak",
    speech.Voice("nova"),           // Voice: alloy, echo, fable, onyx, nova, shimmer
    speech.Speed(1.2),              // Speed: 0.25 to 4.0
    speech.Format("mp3"),           // Format: mp3, opus, aac, flac, wav, pcm
    speech.Instructions("Speak with enthusiasm"), // GPT-4o-Mini-TTS only
    speech.WithTimeout(30*time.Second),
)
```

### Speech Models

| Model           | Method           | Notes                                  |
| --------------- | ---------------- | -------------------------------------- |
| TTS-1           | `TTS1()`         | Standard quality, faster               |
| TTS-1-HD        | `TTS1HD()`       | Higher quality                         |
| GPT-4o-Mini-TTS | `GPT4oMiniTTS()` | Latest model with instructions support |

### Available Voices

- `alloy` - Neutral, balanced
- `echo` - Warm, conversational
- `fable` - British accent
- `onyx` - Deep, authoritative
- `nova` - Friendly, upbeat
- `shimmer` - Soft, gentle

---

## Audio Transcription

### Basic Transcription

```go
import "github.com/seifalmotaz/lamar-sdk/transcription"

func main() {
    client := openai.NewProvider()
    model := client.Whisper1()

    audioData, _ := os.ReadFile("recording.mp3")

    ctx := context.Background()
    result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3")
    if err != nil {
        panic(err)
    }

    fmt.Println("Transcription:", result.Text)
    fmt.Println("Language:", result.Language)
    fmt.Println("Duration:", result.Duration)
}
```

### Transcription Options

```go
result, err := transcription.Transcribe(ctx, model, audioData, "audio/mp3",
    transcription.Language("en"),      // Hint for language
    transcription.Prompt("Context about the audio..."),
    transcription.WithTimeout(60*time.Second),
)
```

### Working with Segments

```go
for i, segment := range result.Segments {
    fmt.Printf("Segment %d [%.2f - %.2f]: %s\n",
        i,
        segment.Start,
        segment.End,
        segment.Text,
    )
}
```

---

## Middleware System

### Overview

Middleware provides a way to intercept and modify requests/responses:

```go
type Handler interface {
    Handle(ctx context.Context, req Request) (Response, error)
}

type Middleware func(Handler) Handler
```

### Available Middleware

#### Timeout Middleware

```go
import "github.com/seifalmotaz/lamar-sdk/middleware"

// Simple timeout
client := openai.NewProvider(
    openai.WithMiddleware(
        middleware.TimeoutWithDefault(30*time.Second),
    ),
)

// Advanced configuration
timeoutMW := middleware.Timeout(middleware.TimeoutConfig{
    Default: 60 * time.Second,
    PerProvider: map[string]time.Duration{
        "openai":    45 * time.Second,
        "anthropic": 90 * time.Second,
    },
    PerModel: map[string]time.Duration{
        "o1-preview": 120 * time.Second,
        "gpt-5-mini-2025-08-07": 30 * time.Second,
    },
})
```

#### Retry Middleware

```go
retryMW := middleware.Retry(middleware.RetryConfig{
    MaxAttempts:  3,
    InitialDelay: time.Second,
    MaxDelay:     30 * time.Second,
    Multiplier:   2.0,
    RetryOn: func(err error) bool {
        // Retry on rate limits and timeouts
        return provider.IsRateLimited(err) || provider.IsTimeout(err)
    },
})
```

#### Logging Middleware

```go
type MyLogger struct{}

func (l *MyLogger) Debug(msg string, args ...any) { log.Printf("[DEBUG] "+msg, args...) }
func (l *MyLogger) Info(msg string, args ...any)  { log.Printf("[INFO] "+msg, args...) }
func (l *MyLogger) Warn(msg string, args ...any)  { log.Printf("[WARN] "+msg, args...) }
func (l *MyLogger) Error(msg string, args ...any) { log.Printf("[ERROR] "+msg, args...) }

client := openai.NewProvider(
    openai.WithMiddleware(middleware.Logging(&MyLogger{})),
)
```

#### Metrics Middleware

```go
type MyMetrics struct{}

func (m *MyMetrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration, err error) {
    metrics.Counter("api_requests_total").WithLabels(provider, model).Inc()
    metrics.Histogram("api_request_duration").WithLabels(provider, model).Observe(duration.Seconds())
}

func (m *MyMetrics) RecordTokens(ctx context.Context, provider, model string, prompt, completion int) {
    metrics.Counter("api_tokens_total").WithLabels(provider, model).Add(float64(prompt + completion))
}

func (m *MyMetrics) RecordStreamStart(ctx context.Context, provider, model string) {
    metrics.Counter("api_streams_total").WithLabels(provider, model).Inc()
}

func (m *MyMetrics) RecordStreamEnd(ctx context.Context, provider, model string, duration time.Duration) {
    metrics.Histogram("api_stream_duration").WithLabels(provider, model).Observe(duration.Seconds())
}

func (m *MyMetrics) RecordStreamEvent(ctx context.Context, provider, model, eventType string) {
    metrics.Counter("api_stream_events").WithLabels(provider, model, eventType).Inc()
}

client := openai.NewProvider(
    openai.WithMiddleware(middleware.Metrics(&MyMetrics{})),
)
```

#### Tracing Middleware (OpenTelemetry)

```go
import (
    "go.opentelemetry.io/otel/trace"
    "github.com/seifalmotaz/lamar-sdk/middleware"
)

tracerProvider := trace.NewTracerProvider()

client := openai.NewProvider(
    openai.WithMiddleware(middleware.Tracing(tracerProvider)),
)
```

#### Panic Recovery Middleware

```go
recoverMW := middleware.Recover()
```

### Chaining Middleware

```go
chain := middleware.Chain(
    middleware.Recover(),
    middleware.Logging(logger),
    middleware.Metrics(metrics),
    middleware.TimeoutWithDefault(30*time.Second),
    middleware.Retry(retryConfig),
)

client := openai.NewProvider(
    openai.WithMiddleware(chain),
)
```

### Custom Middleware

```go
func CustomMiddleware(name string) middleware.Middleware {
    return func(next middleware.Handler) middleware.Handler {
        return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
            // Pre-processing
            start := time.Now()
            log.Printf("[%s] Starting request to %s/%s", name, req.Provider(), req.ModelID())

            // Call next handler
            resp, err := next.Handle(ctx, req)

            // Post-processing
            duration := time.Since(start)
            log.Printf("[%s] Completed in %v", name, duration)

            return resp, err
        })
    }
}
```

### Request/Response Types

```go
// Request interface
type Request interface {
    Provider() string
    ModelID() string
    InputCount() int  // Number of prompts/embeddings
}

// Response interface
type Response interface {
    Usage() provider.Usage
    FinishReason() provider.FinishReason
}
```

---

## Agent Framework

The `agent` package provides multi-step LLM tool-calling loops with stop conditions. It handles the cycle of: call model → execute tools → call model again until a stop condition is met.

### Basic Usage

```go
import "github.com/seifalmotaz/lamar-sdk/agent"

func main() {
    client := openai.NewProvider()
    model := client.GPT5Mini()

    // Create an agent with tools
    ag := agent.New(model,
        agent.WithTools(weatherTool, calculatorTool),
        agent.WithStopWhen(agent.StepCountIs(10)),
    )

    // Run synchronously
    result, err := ag.Invoke(ctx,
        agent.WithMessages(
            provider.UserMessage("What's the weather in Tokyo?"),
        ),
    )
    if err != nil {
        panic(err)
    }

    fmt.Println(result.FinalText)
    fmt.Printf("Steps: %d, Tokens: %d\n", len(result.Steps), result.TotalUsage.TotalTokens)
}
```

### Streaming Events

```go
stream := ag.Stream(ctx,
    agent.WithMessages(
        provider.UserMessage("Calculate 42 * 7"),
    ),
)

for event := range stream {
    switch e := event.(type) {
    case agent.StreamEventStepStart:
        fmt.Printf("Step %d starting\n", e.StepNumber)
    case agent.StreamEventContentDelta:
        fmt.Print(e.Delta)
    case agent.StreamEventToolCall:
        fmt.Printf("Tool call: %s\n", e.ToolCall.Name)
    case agent.StreamEventToolResult:
        fmt.Printf("Tool result: %v\n", e.Result.Output)
    case agent.StreamEventStepFinish:
        fmt.Printf("Step %d finished\n", e.Result.StepNumber)
    case agent.StreamEventFinish:
        fmt.Printf("Done! Final text: %s\n", e.Result.FinalText)
    case agent.StreamEventError:
        log.Printf("Error: %v", e.Error)
    }
}
```

### Stop Conditions

Control when the agent loop terminates:

```go
// Stop after N steps
agent.New(model, agent.WithStopWhen(agent.StepCountIs(5)))

// Stop when a specific tool is called
agent.New(model, agent.WithStopWhen(agent.HasToolCall("submit_answer")))

// Stop when finish reason matches
agent.New(model, agent.WithStopWhen(agent.HasFinishReason(
    provider.FinishReasonStop,
    provider.FinishReasonLength,
)))

// Stop when ANY condition is met
agent.New(model, agent.WithStopWhen(
    agent.StopWhenAny(
        agent.StepCountIs(10),
        agent.HasToolCall("finish"),
    ),
))

// Stop when ALL conditions are met
agent.New(model, agent.WithStopWhen(
    agent.StopWhenAll(
        agent.StepCountAtLeast(3),
        agent.HasToolCall("submit"),
    ),
))
```

### Agent Configuration

```go
ag := agent.New(model,
    // Tools available to the agent
    agent.WithTools(weatherTool, calculatorTool),

    // Stop conditions
    agent.WithStopWhen(agent.StepCountIs(10)),

    // System prompt
    agent.WithSystem("You are a helpful assistant."),

    // Tool choice strategy
    agent.WithToolChoice(provider.ToolChoiceAuto()),

    // Model parameters
    agent.WithTemperature(0.7),
    agent.WithMaxTokens(1000),

    // Max retries on transient errors
    agent.WithMaxRetries(3),

    // Callbacks for observability
    agent.WithCallbacks(agent.Callbacks{
        OnStepStart: func(ctx context.Context, stepNumber int, messages []provider.Message) error {
            log.Printf("Step %d starting", stepNumber)
            return nil
        },
        OnToolCallStart: func(ctx context.Context, tc provider.ToolCall) error {
            log.Printf("Calling tool: %s", tc.Name)
            return nil
        },
        OnToolCallFinish: func(ctx context.Context, result agent.ToolExecutionResult) error {
            log.Printf("Tool %s completed in %v", result.ToolName, result.Duration)
            return nil
        },
    }),
)
```

### Dynamic Configuration

Change model, tools, or settings between steps:

```go
ag := agent.New(model,
    agent.WithTools(defaultTools...),
    agent.WithPrepareStep(func(ctx context.Context, p agent.PrepareStepParams) *agent.PrepareStepResult {
        // Use cheaper model after step 3
        if p.StepNumber > 3 {
            return &agent.PrepareStepResult{
                Model: cheapModel,
                Tools: fewerTools,
            }
        }

        // Change behavior based on last tool calls
        if len(p.ToolCalls) > 0 && p.ToolCalls[0].Name == "search" {
            return &agent.PrepareStepResult{
                System: ptr("Focus on summarizing the search results."),
            }
        }

        return nil // No changes
    }),
)
```

### Callbacks

Full observability into agent execution:

```go
callbacks := agent.Callbacks{
    // Called before agent execution begins
    OnStart: func(ctx context.Context, messages []provider.Message) error {
        log.Printf("Starting with %d messages", len(messages))
        return nil
    },

    // Called before each step
    OnStepStart: func(ctx context.Context, stepNumber int, messages []provider.Message) error {
        log.Printf("Step %d starting with %d messages", stepNumber, len(messages))
        return nil
    },

    // Called after each step completes
    OnStepFinish: func(ctx context.Context, result agent.StepResult) error {
        log.Printf("Step %d: %d tool calls, %d tokens, %v",
            result.StepNumber,
            len(result.ToolCalls),
            result.Usage.TotalTokens,
            result.Duration,
        )
        return nil
    },

    // Called before tool execution
    OnToolCallStart: func(ctx context.Context, tc provider.ToolCall) error {
        log.Printf("Executing tool: %s", tc.Name)
        return nil
    },

    // Called after tool execution
    OnToolCallFinish: func(ctx context.Context, result agent.ToolExecutionResult) error {
        if result.Error != nil {
            log.Printf("Tool %s failed: %v", result.ToolName, result.Error)
        } else {
            log.Printf("Tool %s returned: %v", result.ToolName, result.Output)
        }
        return nil
    },

    // Called when agent completes successfully
    OnFinish: func(ctx context.Context, result *agent.Result) error {
        log.Printf("Agent finished: %d steps, %d total tokens",
            len(result.Steps),
            result.TotalUsage.TotalTokens,
        )
        return nil
    },

    // Called on errors - return nil to suppress and continue
    OnError: func(ctx context.Context, stepNumber int, err error) error {
        log.Printf("Step %d error: %v", stepNumber, err)
        return nil // Suppress error, continue to next step
    },
}

ag := agent.New(model, agent.WithCallbacks(callbacks))
```

### Accessing Results

```go
result, err := ag.Invoke(ctx, agent.WithMessages(messages...))

// Final text response
fmt.Println(result.FinalText)

// Final content (includes non-text parts)
for _, c := range result.FinalContent {
    switch v := c.(type) {
    case provider.TextContent:
        fmt.Println(v.Text)
    case provider.ToolCallContent:
        fmt.Printf("Tool: %s\n", v.Name)
    }
}

// Complete message history
for i, msg := range result.FinalMessages {
    fmt.Printf("Message %d [%s]: %v\n", i, msg.Role, msg.Content)
}

// Step-by-step breakdown
for _, step := range result.Steps {
    fmt.Printf("Step %d:\n", step.StepNumber)
    fmt.Printf("  Model: %s/%s\n", step.Model.Provider, step.Model.ModelID)
    fmt.Printf("  Tool calls: %d\n", len(step.ToolCalls))
    fmt.Printf("  Tokens: %d\n", step.Usage.TotalTokens)
    fmt.Printf("  Duration: %v\n", step.Duration)
    fmt.Printf("  Finish reason: %s\n", step.FinishReason)

    for _, tr := range step.ToolResults {
        fmt.Printf("  Tool %s: %v\n", tr.ToolName, tr.Output)
    }
}

// Total usage
fmt.Printf("Total tokens: %d\n", result.TotalUsage.TotalTokens)
fmt.Printf("Total duration: %v\n", result.TotalDuration)
```

### Streaming with Timeout

```go
// Stream with timeout
stream := ag.StreamWithTimeout(ctx, 60*time.Second,
    agent.WithMessages(
        provider.UserMessage("What's the weather in Tokyo?"),
    ),
)

// IMPORTANT: Drain the channel to prevent goroutine leaks
for event := range stream {
    // Handle events
}
```

### Combining with Regular Generation

```go
// Use generate package for simple single calls
result, _ := generate.Generate(ctx, model, "Hello")

// Use agent package when you need tool loops
ag := agent.New(model,
    agent.WithTools(weatherTool),
    agent.WithStopWhen(agent.StepCountIs(5)),
)
result, _ := ag.Invoke(ctx, agent.WithMessages(
    provider.UserMessage("What's the weather in Tokyo?"),
))
```

---

## Error Handling

### Error Structure

All SDK errors are structured with `provider.Error`:

```go
type Error struct {
    Code       ErrorCode     // Structured error code
    Message    string        // Human-readable message
    Cause      error         // Underlying error
    Provider   string        // Provider identifier
    ModelID    string        // Model identifier
    RetryAfter time.Duration // Retry delay (rate limits)
    StatusCode int           // HTTP status code
}
```

### Error Codes

| Code | Constant                   | Description                |
| ---- | -------------------------- | -------------------------- |
| 0    | `CodeUnknown`              | Unspecified error          |
| 1    | `CodeInvalidRequest`       | Malformed request          |
| 2    | `CodeInvalidModel`         | Nil/invalid model          |
| 3    | `CodeInvalidPrompt`        | Empty prompt               |
| 4    | `CodeInvalidInput`         | Invalid input data         |
| 5    | `CodeAuthenticationFailed` | Auth failure               |
| 6    | `CodeRateLimited`          | Rate limit exceeded        |
| 7    | `CodeModelNotFound`        | Model doesn't exist        |
| 8    | `CodeContentFiltered`      | Content filtered by safety |
| 9    | `CodeContextCanceled`      | Context canceled           |
| 10   | `CodeAPITimeout`           | API timeout                |
| 11   | `CodeParseError`           | Response parse failure     |
| 12   | `CodeUnsupportedModel`     | Unsupported model          |
| 13   | `CodeUnsupportedOperation` | Unsupported operation      |

### Error Checking

```go
import (
    "errors"
    "github.com/seifalmotaz/lamar-sdk/provider"
)

result, err := generate.Generate(ctx, model, "prompt")
if err != nil {
    // Method 1: Type assertion
    var providerErr *provider.Error
    if errors.As(err, &providerErr) {
        fmt.Printf("Code: %v\n", providerErr.Code)
        fmt.Printf("Message: %s\n", providerErr.Message)
        fmt.Printf("Provider: %s\n", providerErr.Provider)
        fmt.Printf("Model: %s\n", providerErr.ModelID)
        fmt.Printf("HTTP Status: %d\n", providerErr.StatusCode)

        if providerErr.RetryAfter > 0 {
            fmt.Printf("Retry after: %v\n", providerErr.RetryAfter)
        }

        if providerErr.Cause != nil {
            fmt.Printf("Cause: %v\n", providerErr.Cause)
        }
    }

    // Method 2: Helper functions
    if provider.IsRateLimited(err) {
        retryAfter := provider.RetryAfter(err)
        time.Sleep(retryAfter)
        // Retry the request
    }

    if provider.IsTimeout(err) {
        // Handle timeout
    }

    if provider.IsContextCanceled(err) {
        // User canceled
    }

    if provider.IsAuthenticationError(err) {
        // Check API key
    }

    if provider.IsNotFoundError(err) {
        // Model doesn't exist
    }

    if provider.IsInvalidInput(err) {
        // Invalid parameters
    }

    if provider.IsContentFiltered(err) {
        // Content was filtered
    }
}
```

### Helper Functions

| Function                     | Returns         | Description                |
| ---------------------------- | --------------- | -------------------------- |
| `IsRateLimited(err)`         | `bool`          | Check for rate limit error |
| `RetryAfter(err)`            | `time.Duration` | Get retry delay            |
| `IsTimeout(err)`             | `bool`          | Check for timeout          |
| `IsContextCanceled(err)`     | `bool`          | Check for cancellation     |
| `IsAuthenticationError(err)` | `bool`          | Check auth failure         |
| `IsNotFoundError(err)`       | `bool`          | Check model not found      |
| `IsInvalidInput(err)`        | `bool`          | Check invalid input        |
| `IsContentFiltered(err)`     | `bool`          | Check content filter       |
| `ErrorCodeOf(err)`           | `ErrorCode`     | Get error code             |

### Sentinel Errors

```go
// Pre-defined errors for common cases
var (
    ErrInvalidModel         *Error  // model parameter is nil
    ErrInvalidPrompt        *Error  // prompt is empty
    ErrInvalidInput         *Error  // input is invalid
    ErrInvalidMediaType     *Error  // media type is required
    ErrRateLimited          *Error  // rate limit exceeded
    ErrContextCanceled      *Error  // context canceled
    ErrAPITimeout           *Error  // request timed out
    ErrAuthenticationFailed *Error  // authentication failed
    ErrModelNotFound        *Error  // model not found
    ErrContentFiltered      *Error  // content filtered
    ErrUnsupportedModel     *Error  // unsupported model
    ErrUnsupportedOperation *Error  // unsupported operation
)

// Compare sentinel errors
if errors.Is(err, provider.ErrRateLimited) {
    // Handle rate limit
}
```

### Error Handling Patterns

```go
func generateWithRetry(ctx context.Context, model provider.Generator, prompt string) (*generate.Result, error) {
    maxRetries := 3
    baseDelay := time.Second

    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := generate.Generate(ctx, model, prompt)
        if err == nil {
            return result, nil
        }

        // Don't retry on non-retryable errors
        if provider.IsAuthenticationError(err) ||
           provider.IsNotFoundError(err) ||
           provider.IsInvalidInput(err) ||
           provider.IsContentFiltered(err) {
            return nil, err
        }

        // Check for rate limit
        if provider.IsRateLimited(err) {
            delay := provider.RetryAfter(err)
            if delay == 0 {
                delay = baseDelay * time.Duration(1<<attempt)
            }
            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        // Check for timeout
        if provider.IsTimeout(err) {
            if attempt < maxRetries-1 {
                continue
            }
        }

        // Check for context cancellation
        if provider.IsContextCanceled(err) {
            return nil, err
        }

        return nil, err
    }

    return nil, fmt.Errorf("max retries exceeded")
}
```

---

## Content Types Reference

### Content Interface

```go
type Content interface {
    content() // Private marker method
}
```

Content is a polymorphic type. Use type assertion or type switch to determine the specific type.

### TextContent

```go
type TextContent struct {
    Text string
}

// Constructor
text := provider.Text("Hello, world!")

// Type switch
switch c := content.(type) {
case provider.TextContent:
    fmt.Println(c.Text)
}
```

### ImageContent

```go
type ImageContent struct {
    Data      []byte
    MediaType string // "image/png", "image/jpeg", etc.
}

// Constructor from base64 data
data, _ := os.ReadFile("image.png")
img := provider.Image(data, "image/png")

// Constructor from URL
img := provider.ImageFromURL("https://example.com/image.png")

// Type switch
switch c := content.(type) {
case provider.ImageContent:
    fmt.Printf("%d bytes, type: %s\n", len(c.Data), c.MediaType)
}
```

### AudioContent

```go
type AudioContent struct {
    Data      []byte
    MediaType string // "audio/mp3", "audio/wav", etc.
}

// Constructor from base64 data
data, _ := os.ReadFile("audio.mp3")
audio := provider.Audio(data, "audio/mp3")

// Constructor from URL
audio := provider.AudioFromURL("https://example.com/audio.mp3")

// Type switch
switch c := content.(type) {
case provider.AudioContent:
    fmt.Printf("%d bytes, type: %s\n", len(c.Data), c.MediaType)
}
```

### ToolCallContent

```go
type ToolCallContent struct {
    ID    string          // Unique tool call ID
    Name  string          // Tool name
    Input json.RawMessage // Tool input as JSON
}

// Constructor from JSON
tc := provider.NewToolCallContent("call_123", "get_weather", json.RawMessage(`{"location":"Tokyo"}`))

// Constructor from any value (automatically marshaled)
tc := provider.NewToolCallContentFromJSON("call_123", "get_weather", map[string]any{
    "location": "Tokyo",
})
```

### ToolResultContent

```go
type ToolResultContent struct {
    ID      string          // Matches ToolCallContent.ID
    Name    string          // Tool name
    Result  json.RawMessage // Tool result as JSON
    IsError bool            // true if tool execution failed
}

// Constructor from JSON
tr := provider.NewToolResultContent("call_123", "get_weather", json.RawMessage(`{"temp":22}`), false)

// Constructor from any value
tr := provider.NewToolResultContentFromJSON("call_123", "get_weather",
    WeatherOutput{Temperature: 22.5}, false)
```

### ReasoningContent

```go
type ReasoningContent struct {
    Text string // Model's reasoning/thinking process
}

// Type switch
switch c := content.(type) {
case provider.ReasoningContent:
    fmt.Println("Model reasoning:", c.Text)
}
```

### Message Constructors

```go
// System message
msg := provider.SystemMessage("You are a helpful assistant.")

// User message (text only)
msg := provider.UserMessage("What is the capital of France?")

// User message (multimodal)
msg := provider.UserMessageWithContent(
    provider.Text("Describe this image"),
    provider.Image(imageData, "image/png"),
)

// Assistant message (text only)
msg := provider.AssistantMessage("The capital of France is Paris.")

// Assistant message (with tool calls)
msg := provider.AssistantMessageWithToolCalls(
    provider.NewToolCallContent("call_1", "get_weather", input),
    provider.NewToolCallContent("call_2", "search", input2),
)

// Tool result message
msg := provider.ToolResultMessage(
    provider.NewToolResultContent("call_1", "get_weather", result, false),
)
```

---

## Provider Interface Reference

### Interface Hierarchy

```
Model (base)
├── Generator (non-streaming)
│   └── LanguageModel (Generator + Streamer)
├── Streamer (streaming)
│   └── LanguageModel (Generator + Streamer)
├── EmbeddingModel
├── ImageModel
├── TranscriptionModel
└── SpeechModel
```

### Model Interface

Base interface for all models:

```go
type Model interface {
    Provider() string  // Provider name (e.g., "openai")
    ModelID() string   // Model identifier (e.g., "gpt-5-mini-2025-08-07")
}
```

### Generator Interface

For non-streaming text generation:

```go
type Generator interface {
    Model
    Generate(ctx context.Context, req *GenerateRequest) (*GenerateResult, error)
}
```

### Streamer Interface

For streaming text generation:

```go
type Streamer interface {
    Model
    Stream(ctx context.Context, req *GenerateRequest) (*StreamResult, error)
}
```

### LanguageModel Interface

Full-featured model supporting both streaming and non-streaming:

```go
type LanguageModel interface {
    Generator
    Streamer
}
```

### EmbeddingModel Interface

For text embeddings:

```go
type EmbeddingModel interface {
    Model
    Embed(ctx context.Context, req *EmbedRequest) (*EmbedResult, error)
    MaxEmbeddingsPerCall() int
}
```

### ImageModel Interface

For image generation:

```go
type ImageModel interface {
    Model
    GenerateImage(ctx context.Context, req *ImageRequest) (*ImageResult, error)
    MaxImagesPerCall() int
}
```

### TranscriptionModel Interface

For audio transcription:

```go
type TranscriptionModel interface {
    Model
    Transcribe(ctx context.Context, req *TranscriptionRequest) (*TranscriptionResult, error)
}
```

### SpeechModel Interface

For text-to-speech:

```go
type SpeechModel interface {
    Model
    Synthesize(ctx context.Context, req *SpeechRequest) (*SpeechResult, error)
}
```

### Capability Checking Functions

```go
// Check interface implementation
func CanGenerate(m Model) bool
func CanStream(m Model) bool
func CanEmbed(m Model) bool
func IsLanguageModel(m Model) bool
func CanGenerateImage(m Model) bool
func CanTranscribe(m Model) bool
func CanSynthesize(m Model) bool

// Check declared capabilities
func HasCapability(m Model, cap Capability) bool

// Get model information
func GetModelInfo(m Model) (ModelInfo, bool)
```

### Complete Example

```go
func useModel(ctx context.Context, m provider.Model) {
    fmt.Printf("Provider: %s, Model: %s\n", m.Provider(), m.ModelID())

    // Check capabilities
    if provider.CanGenerate(m) {
        gen := m.(provider.Generator)
        result, _ := gen.Generate(ctx, &provider.GenerateRequest{Prompt: "Hello"})
        fmt.Println(result.Text)
    }

    if provider.CanStream(m) {
        streamer := m.(provider.Streamer)
        result, _ := streamer.Stream(ctx, &provider.GenerateRequest{Prompt: "Hello"})
        for part := range result.Stream {
            // Consume stream
        }
    }

    if provider.IsLanguageModel(m) {
        lm := m.(provider.LanguageModel)
        // Can use both Generate and Stream
    }

    if provider.CanEmbed(m) {
        emb := m.(provider.EmbeddingModel)
        result, _ := emb.Embed(ctx, &provider.EmbedRequest{Texts: []string{"Hello"}})
        fmt.Println(result.Embeddings)
    }
}
```

---

## Creating a New Provider

This section explains how to implement a new provider for the Lamar SDK.

### Provider Architecture

Providers implement the SDK's interfaces and translate between SDK types and their API format.

**File Structure:**

```
providers/
└── yourprovider/
    ├── provider.go          # Main provider struct and factory
    ├── config.go            # Provider-specific options
    ├── types.go             # API request/response types
    ├── chat.go              # Chat model implementation
    ├── chat_stream.go       # Streaming implementation
    ├── embedding.go         # Embedding model
    ├── image.go             # Image generation model
    ├── transcription.go     # Audio transcription model
    ├── speech.go            # Text-to-speech model
    └── *_test.go            # Tests
```

### Step 1: Provider Factory

Create the main provider structure:

```go
// provider.go
package yourprovider

import (
    "context"
    "net/http"
    "os"

    "github.com/seifalmotaz/lamar-sdk/internal/httpx"
    "github.com/seifalmotaz/lamar-sdk/middleware"
    "github.com/seifalmotaz/lamar-sdk/provider"
)

const DefaultBaseURL = "https://api.yourprovider.com/v1"

type Provider struct {
    client      *httpx.Client
    baseURL     string
    middlewares []middleware.Middleware
}

type Option func(*Provider)

func APIKey(key string) Option {
    return func(p *Provider) {
        p.client.SetHeader("Authorization", "Bearer "+key)
    }
}

func BaseURL(url string) Option {
    return func(p *Provider) {
        p.baseURL = url
        p.client.SetBaseURL(url)
    }
}

func HTTPClient(client *http.Client) Option {
    return func(p *Provider) {
        p.client = httpx.NewClient(p.baseURL, client)
    }
}

func WithMiddleware(middlewares ...middleware.Middleware) Option {
    return func(p *Provider) {
        p.middlewares = append(p.middlewares, middlewares...)
    }
}

func NewProvider(opts ...Option) *Provider {
    p := &Provider{
        baseURL: DefaultBaseURL,
    }

    // Create default HTTP client
    p.client = httpx.NewClient(p.baseURL, http.DefaultClient)

    // Apply options
    for _, opt := range opts {
        opt(p)
    }

    // Set API key from environment if not provided
    if apiKey := os.Getenv("YOURPROVIDER_API_KEY"); apiKey != "" {
        p.client.SetHeader("Authorization", "Bearer "+apiKey)
    }

    return p
}
```

### Step 2: Model Factory Methods

Add factory methods for creating models:

```go
// In provider.go

// Model returns a non-streaming model
func (p *Provider) Model(id string) provider.Generator {
    return &ChatModel{id: id, provider: p}
}

// StreamingModel returns a streaming-capable model
func (p *Provider) StreamingModel(id string) provider.LanguageModel {
    return &ChatModel{id: id, provider: p}
}

// Embedding returns an embedding model
func (p *Provider) Embedding(id string) provider.EmbeddingModel {
    return &EmbeddingModel{id: id, provider: p}
}

// Convenience methods for popular models
func (p *Provider) YourModel1() provider.LanguageModel {
    return p.StreamingModel("your-model-1")
}

func (p *Provider) YourEmbeddingModel() provider.EmbeddingModel {
    return p.Embedding("your-embedding-model")
}
```

### Step 3: Implement Chat Model

```go
// chat.go
package yourprovider

import (
    "context"
    "encoding/json"

    "github.com/seifalmotaz/lamar-sdk/provider"
)

type ChatModel struct {
    id       string
    provider *Provider
}

// Compile-time interface verification
var (
    _ provider.Model         = (*ChatModel)(nil)
    _ provider.Generator     = (*ChatModel)(nil)
    _ provider.Streamer      = (*ChatModel)(nil)
    _ provider.LanguageModel = (*ChatModel)(nil)
)

func (m *ChatModel) Provider() string { return "yourprovider" }
func (m *ChatModel) ModelID() string  { return m.id }

func (m *ChatModel) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
    return m.provider.wrapGenerate(ctx, m.id, req, func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
        // 1. Convert SDK request to your API format
        apiReq, err := m.buildAPIRequest(req)
        if err != nil {
            return nil, err
        }

        // 2. Make API call
        var apiResp APIResponse
        if err := m.provider.client.Post(ctx, "/chat/completions", apiReq, &apiResp); err != nil {
            return nil, m.mapError(err)
        }

        // 3. Convert response to SDK format
        return m.buildResult(&apiResp)
    })
}

func (m *ChatModel) buildAPIRequest(req *provider.GenerateRequest) (*APIRequest, error) {
    apiReq := &APIRequest{
        Model: m.id,
    }

    // Handle prompt
    if req.Prompt != "" {
        apiReq.Messages = append(apiReq.Messages, APIMessage{
            Role:    "user",
            Content: req.Prompt,
        })
    }

    // Handle messages
    for _, msg := range req.Messages {
        apiMsg, err := m.convertMessage(msg)
        if err != nil {
            return nil, err
        }
        apiReq.Messages = append(apiReq.Messages, apiMsg)
    }

    // Handle system prompt
    if req.Config.System != "" {
        apiReq.Messages = append([]APIMessage{{
            Role:    "system",
            Content: req.Config.System,
        }}, apiReq.Messages...)
    }

    // Handle config
    if req.Config.MaxTokens > 0 {
        apiReq.MaxTokens = req.Config.MaxTokens
    }
    if req.Config.Temperature > 0 {
        apiReq.Temperature = req.Config.Temperature
    }
    if len(req.Config.Tools) > 0 {
        apiReq.Tools = m.convertTools(req.Config.Tools)
        apiReq.ToolChoice = m.convertToolChoice(req.Config.ToolChoice)
    }

    return apiReq, nil
}

func (m *ChatModel) convertMessage(msg provider.Message) (APIMessage, error) {
    apiMsg := APIMessage{Role: string(msg.Role)}

    // Handle content
    if len(msg.Content) == 1 {
        if text, ok := msg.Content[0].(provider.TextContent); ok {
            apiMsg.Content = text.Text
            return apiMsg, nil
        }
    }

    // Handle multimodal content
    var contents []APIContent
    for _, c := range msg.Content {
        switch content := c.(type) {
        case provider.TextContent:
            contents = append(contents, APIContent{
                Type: "text",
                Text: content.Text,
            })
        case provider.ImageContent:
            contents = append(contents, APIContent{
                Type: "image_url",
                ImageURL: &APIImageURL{
                    URL: "data:" + content.MediaType + ";base64," +
                         base64.StdEncoding.EncodeToString(content.Data),
                },
            })
        case provider.ToolCallContent:
            apiMsg.ToolCalls = append(apiMsg.ToolCalls, APIToolCall{
                ID:   content.ID,
                Type: "function",
                Function: APIFunction{
                    Name:      content.Name,
                    Arguments: string(content.Input),
                },
            })
        case provider.ToolResultContent:
            apiMsg.Role = "tool"
            apiMsg.ToolCallID = content.ID
            apiMsg.Content = string(content.Result)
        }
    }

    if len(contents) > 0 {
        apiMsg.Content = contents
    }

    return apiMsg, nil
}

func (m *ChatModel) buildResult(apiResp *APIResponse) (*provider.GenerateResult, error) {
    if len(apiResp.Choices) == 0 {
        return nil, provider.NewError(provider.CodeParseError, "no choices in response", nil)
    }

    choice := apiResp.Choices[0]

    result := &provider.GenerateResult{
        Text:         choice.Message.Content,
        FinishReason: m.mapFinishReason(choice.FinishReason),
        Usage: provider.Usage{
            PromptTokens:     apiResp.Usage.PromptTokens,
            CompletionTokens: apiResp.Usage.CompletionTokens,
            TotalTokens:      apiResp.Usage.TotalTokens,
        },
    }

    // Handle content
    if choice.Message.Content != "" {
        result.Content = append(result.Content, provider.Text(choice.Message.Content))
    }

    // Handle tool calls
    for _, tc := range choice.Message.ToolCalls {
        result.ToolCalls = append(result.ToolCalls, provider.ToolCall{
            ID:    tc.ID,
            Name:  tc.Function.Name,
            Input: json.RawMessage(tc.Function.Arguments),
        })
        result.Content = append(result.Content, provider.NewToolCallContent(
            tc.ID, tc.Function.Name, json.RawMessage(tc.Function.Arguments),
        ))
    }

    return result, nil
}

func (m *ChatModel) mapFinishReason(reason string) provider.FinishReason {
    switch reason {
    case "stop":
        return provider.FinishReasonStop
    case "length":
        return provider.FinishReasonLength
    case "tool_calls", "function_call":
        return provider.FinishReasonToolCalls
    case "content_filter":
        return provider.FinishReasonContentFilter
    default:
        return provider.FinishReasonError
    }
}
```

### Step 4: Implement Streaming

```go
// chat_stream.go
package yourprovider

import (
    "context"
    "encoding/json"
    "io"

    "github.com/seifalmotaz/lamar-sdk/internal/sse"
    "github.com/seifalmotaz/lamar-sdk/provider"
)

func (m *ChatModel) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
    return m.provider.wrapStream(ctx, m.id, req, func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
        // 1. Build request with streaming enabled
        apiReq, err := m.buildAPIRequest(req)
        if err != nil {
            return nil, err
        }
        apiReq.Stream = true

        // 2. Open streaming connection
        rc, err := m.provider.client.DoStream(ctx, "POST", "/chat/completions", apiReq)
        if err != nil {
            return nil, m.mapError(err)
        }

        // 3. Set up channels
        stream := make(chan provider.StreamPart, 100)
        done := make(chan struct{})

        result := &provider.StreamResult{
            Stream: stream,
            Done:   done,
        }

        // 4. Process stream in goroutine
        go func() {
            defer close(stream)
            defer close(done)
            defer rc.Close()

            reader := sse.NewReader(rc)
            var usage provider.Usage
            var finishReason provider.FinishReason

            for {
                event, err := reader.ReadEvent()
                if err != nil {
                    if err == sse.Done || err == io.EOF {
                        stream <- provider.StreamFinishPart{
                            FinishReason: finishReason,
                            Usage:        usage,
                        }
                        break
                    }
                    stream <- provider.StreamErrorPart{Error: err}
                    break
                }

                // Skip non-data events
                if event.Type != "data" {
                    continue
                }

                // Parse the event data
                var streamResp APIStreamResponse
                if err := json.Unmarshal(event.Data, &streamResp); err != nil {
                    stream <- provider.StreamErrorPart{Error: err}
                    continue
                }

                // Process choices
                for _, choice := range streamResp.Choices {
                    if choice.Delta.Content != "" {
                        stream <- provider.StreamTextPart{
                            Delta: choice.Delta.Content,
                        }
                    }
                    if choice.Delta.ToolCalls != nil {
                        for _, tc := range choice.Delta.ToolCalls {
                            stream <- provider.StreamToolCallPart{
                                ToolCall: provider.ToolCall{
                                    ID:    tc.ID,
                                    Name:  tc.Function.Name,
                                    Input: json.RawMessage(tc.Function.Arguments),
                                },
                            }
                        }
                    }
                    if choice.FinishReason != "" {
                        finishReason = m.mapFinishReason(choice.FinishReason)
                    }
                }

                // Capture usage
                if streamResp.Usage.TotalTokens > 0 {
                    usage = provider.Usage{
                        PromptTokens:     streamResp.Usage.PromptTokens,
                        CompletionTokens: streamResp.Usage.CompletionTokens,
                        TotalTokens:      streamResp.Usage.TotalTokens,
                    }
                }
            }
        }()

        return result, nil
    })
}
```

### Step 5: Error Handling

```go
// In provider.go

func (p *Provider) mapError(err error) *provider.Error {
    // Check if already a provider error
    var providerErr *provider.Error
    if errors.As(err, &providerErr) {
        return providerErr
    }

    // Check for HTTP errors
    var httpErr *httpx.Error
    if errors.As(err, &httpErr) {
        code := provider.CodeUnknown
        switch httpErr.StatusCode {
        case 400:
            code = provider.CodeInvalidRequest
        case 401, 403:
            code = provider.CodeAuthenticationFailed
        case 404:
            code = provider.CodeModelNotFound
        case 429:
            code = provider.CodeRateLimited
        case 500, 502, 503, 504:
            code = provider.CodeAPITimeout
        }

        return &provider.Error{
            Code:       code,
            Message:    httpErr.Message,
            Cause:      err,
            Provider:   "yourprovider",
            StatusCode: httpErr.StatusCode,
            RetryAfter: parseRetryAfter(httpErr.Header),
        }
    }

    return &provider.Error{
        Code:     provider.CodeUnknown,
        Message:  err.Error(),
        Cause:    err,
        Provider: "yourprovider",
    }
}

func parseRetryAfter(header http.Header) time.Duration {
    val := header.Get("Retry-After")
    if val == "" {
        return 0
    }

    // Try parsing as seconds
    if secs, err := strconv.Atoi(val); err == nil {
        return time.Duration(secs) * time.Second
    }

    // Try parsing as date
    if t, err := http.ParseTime(val); err == nil {
        return time.Until(t)
    }

    return 0
}
```

### Step 6: Middleware Integration

```go
// In provider.go

type generateHandler func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error)

func (p *Provider) wrapGenerate(ctx context.Context, modelID string, req *provider.GenerateRequest, core generateHandler) (*provider.GenerateResult, error) {
    if len(p.middlewares) == 0 {
        return core(ctx, req)
    }

    handler := middleware.Chain(p.middlewares...)(middleware.HandlerFunc(
        func(ctx context.Context, r middleware.Request) (middleware.Response, error) {
            result, err := core(ctx, req)
            if err != nil {
                return nil, err
            }
            return &middleware.GenerateResponse{
                Text:             result.Text,
                Content:          result.Content,
                ToolCalls:        result.ToolCalls,
                FinishReasonData: result.FinishReason,
                UsageData:        result.Usage,
            }, nil
        },
    ))

    mwReq := &middleware.GenerateRequest{
        ProviderName: "yourprovider",
        Model:        modelID,
        Prompt:       req.Prompt,
        Messages:     req.Messages,
        Config:       req.Config,
    }

    resp, err := handler.Handle(ctx, mwReq)
    if err != nil {
        return nil, err
    }

    genResp := resp.(*middleware.GenerateResponse)
    return &provider.GenerateResult{
        Text:         genResp.Text,
        Content:      genResp.Content,
        ToolCalls:    genResp.ToolCalls,
        FinishReason: genResp.FinishReasonData,
        Usage:        genResp.UsageData,
    }, nil
}

// Similar wrapStream and wrapEmbed methods...
```

### Step 7: Testing

```go
// provider_test.go
package yourprovider

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/seifalmotaz/lamar-sdk/provider"
)

func TestChatModelGenerate(t *testing.T) {
    // Create mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify headers
        if r.Header.Get("Authorization") != "Bearer test-key" {
            t.Error("missing authorization header")
        }

        // Return mock response
        resp := APIResponse{
            Choices: []APIChoice{{
                Message: APIMessage{
                    Role:    "assistant",
                    Content: "Hello, world!",
                },
                FinishReason: "stop",
            }},
            Usage: APIUsage{
                PromptTokens:     10,
                CompletionTokens: 5,
                TotalTokens:      15,
            },
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    }))
    defer server.Close()

    // Create provider
    p := NewProvider(
        APIKey("test-key"),
        BaseURL(server.URL),
    )
    model := p.Model("test-model")

    // Test generate
    ctx := context.Background()
    result, err := model.Generate(ctx, &provider.GenerateRequest{
        Prompt: "Hello",
    })
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if result.Text != "Hello, world!" {
        t.Errorf("Text = %q, want %q", result.Text, "Hello, world!")
    }

    if result.Usage.TotalTokens != 15 {
        t.Errorf("TotalTokens = %d, want %d", result.Usage.TotalTokens, 15)
    }
}
```

### Provider Implementation Checklist

- [ ] Provider struct with factory function `NewProvider()`
- [ ] Functional options: `APIKey()`, `BaseURL()`, `HTTPClient()`, `WithMiddleware()`
- [ ] Environment variable fallback for API keys
- [ ] HTTP client setup with authentication headers
- [ ] Model factory methods: `Model()`, `StreamingModel()`, `Embedding()`, etc.
- [ ] Implement `Model` interface: `Provider()`, `ModelID()`
- [ ] Implement `Generator` interface: `Generate()`
- [ ] Implement `Streamer` interface: `Stream()` (if supported)
- [ ] Implement `EmbeddingModel` interface (if supported)
- [ ] Request conversion: SDK types → API types
- [ ] Response conversion: API types → SDK types
- [ ] Content type handling: text, images, audio, tools
- [ ] Error mapping: HTTP errors → `provider.Error`
- [ ] Finish reason mapping
- [ ] Usage statistics
- [ ] Middleware wrapper methods
- [ ] SSE streaming support
- [ ] Unit tests with mock HTTP server
- [ ] Integration tests

---

## OpenAI Provider Reference

### Provider Initialization

```go
import "github.com/seifalmotaz/lamar-sdk/providers/openai"

// Simple initialization (uses OPENAI_API_KEY env var)
client := openai.NewProvider()

// With API key
client := openai.NewProvider(openai.APIKey("your-api-key"))

// With custom base URL (for proxies or Azure)
client := openai.NewProvider(
    openai.APIKey("your-api-key"),
    openai.BaseURL("https://your-proxy.com/v1"),
)

// With organization and project
client := openai.NewProvider(
    openai.APIKey("your-api-key"),
    openai.OrgID("org-123"),
    openai.ProjectID("proj-456"),
)

// With custom HTTP client
httpClient := &http.Client{Timeout: 60 * time.Second}
client := openai.NewProvider(
    openai.APIKey("your-api-key"),
    openai.HTTPClient(httpClient),
)

// With middleware
client := openai.NewProvider(
    openai.APIKey("your-api-key"),
    openai.WithMiddleware(
        middleware.TimeoutWithDefault(30*time.Second),
        middleware.Logging(logger),
    ),
)
```

### Chat Models

| Method                | Returns         | Model ID                | Notes                |
| --------------------- | --------------- | ----------------------- | -------------------- |
| `GPT5Mini()`          | `LanguageModel` | `gpt-5-mini-2025-08-07` | Fast, cost-effective |
| `GPT51()`             | `LanguageModel` | `gpt-5.1-2025-11-13`    | Balanced model       |
| `GPT52()`             | `LanguageModel` | `gpt-5.2-2025-12-11`    | Advanced model       |
| `GPT54()`             | `LanguageModel` | `gpt-5.4-2026-03-05`    | Latest model         |
| `GPT4oAudioPreview()` | `LanguageModel` | `gpt-4o-audio-preview`  | Audio input/output   |
| `O1()`                | `Generator`     | `o1`                    | Reasoning model      |
| `O1Mini()`            | `Generator`     | `o1-mini`               | Fast reasoning       |
| `O1Preview()`         | `Generator`     | `o1-preview`            | Preview reasoning    |
| `Model(id)`           | `Generator`     | custom                  | Any model ID         |
| `StreamingModel(id)`  | `LanguageModel` | custom                  | With streaming       |

### Embedding Models

| Method                  | Returns          | Model ID                 | Dimensions |
| ----------------------- | ---------------- | ------------------------ | ---------- |
| `TextEmbedding3Small()` | `EmbeddingModel` | `text-embedding-3-small` | 1536       |
| `TextEmbedding3Large()` | `EmbeddingModel` | `text-embedding-3-large` | 3072       |
| `TextEmbeddingAda002()` | `EmbeddingModel` | `text-embedding-ada-002` | 1536       |
| `Embedding(id)`         | `EmbeddingModel` | custom                   | Varies     |

### Image Models

| Method        | Returns      | Model ID      | Notes           |
| ------------- | ------------ | ------------- | --------------- |
| `DALLE2()`    | `ImageModel` | `dall-e-2`    | Multiple images |
| `DALLE3()`    | `ImageModel` | `dall-e-3`    | Higher quality  |
| `GPTImage1()` | `ImageModel` | `gpt-image-1` | Latest model    |
| `Image(id)`   | `ImageModel` | custom        | Any model ID    |

### Transcription Models

| Method              | Returns              | Model ID    |
| ------------------- | -------------------- | ----------- |
| `Whisper1()`        | `TranscriptionModel` | `whisper-1` |
| `Transcription(id)` | `TranscriptionModel` | custom      |

### Speech Models

| Method           | Returns       | Model ID          | Notes             |
| ---------------- | ------------- | ----------------- | ----------------- |
| `TTS1()`         | `SpeechModel` | `tts-1`           | Standard quality  |
| `TTS1HD()`       | `SpeechModel` | `tts-1-hd`        | Higher quality    |
| `GPT4oMiniTTS()` | `SpeechModel` | `gpt-4o-mini-tts` | With instructions |
| `Speech(id)`     | `SpeechModel` | custom            |

### Provider-Specific Options

```go
// Image generation options
model := client.DALLE3(
    openai.WithImageSize("1792x1024"),
    openai.WithImageQuality("hd"),
    openai.WithImageStyle("vivid"),
)

// Speech options
model := client.TTS1(
    openai.WithVoice("nova"),
    openai.WithResponseFormat("mp3"),
)

// Transcription options
model := client.Whisper1(
    openai.WithLanguage("en"),
    openai.WithTimestampGranularity("word"),
)
```

---

## Type System Reference

### Request Types

#### GenerateRequest

```go
type GenerateRequest struct {
    Prompt   string    // Simple prompt
    Messages []Message // Conversation history
    System   string    // System prompt
    Config   Config    // Model configuration
}

type Config struct {
    System         string
    MaxTokens      int
    Temperature    float64
    TopP           float64
    TopK           int
    StopSequences  []string
    Tools          []ToolDefinition
    ToolChoice     ToolChoice
    Seed           *int
    ResponseFormat *ResponseFormat
}
```

#### EmbedRequest

```go
type EmbedRequest struct {
    Texts []string
}
```

#### ImageRequest

```go
type ImageRequest struct {
    Prompt          string
    Files           []ImageFile          // Input images for editing
    Mask            *ImageFile           // Mask for editing
    N               int                  // Number of images
    Size            string               // "256x256", "512x512", etc.
    Quality         string               // "standard", "hd"
    Format          string               // "png", "jpeg", "webp"
    ProviderOptions map[string]any       // Provider-specific options
}
```

#### TranscriptionRequest

```go
type TranscriptionRequest struct {
    Audio          []byte
    MediaType      string
    Language       string
    Prompt         string
    ProviderOptions map[string]any
}
```

#### SpeechRequest

```go
type SpeechRequest struct {
    Text           string
    Voice          string
    Format         string
    Speed          float64
    Instructions   string
    ProviderOptions map[string]any
}
```

### Result Types

#### GenerateResult

```go
type GenerateResult struct {
    // Access via methods:
    Text() string
    Content() []Content
    ToolCalls() []ToolCall
    FinishReason() FinishReason
    Usage() Usage
    Message() Message  // Assistant message representation
}
```

#### StreamResult

```go
type StreamResult struct {
    Stream <-chan StreamPart  // Real-time parts
    Done  <-chan struct{}     // Completion signal
    Err   error               // Stream error

    // Blocking methods (wait for completion):
    Text() (string, error)
    Usage() (Usage, error)
    FinishReason() (FinishReason, error)
}
```

#### EmbedResult

```go
type Result struct {
    Embedding []float64
    Usage     Usage
}

type BatchResult struct {
    Embeddings [][]float64
    Usage      Usage
}
```

#### ImageResult

```go
type ImageResult struct {
    Images         [][]byte
    RevisedPrompts []string
    Usage          ImageUsage
}
```

#### TranscriptionResult

```go
type TranscriptionResult struct {
    Text     string
    Segments []TranscriptSegment
    Language string
    Duration float64
}

type TranscriptSegment struct {
    ID    int
    Start float64
    End   float64
    Text  string
}
```

#### SpeechResult

```go
type SpeechResult struct {
    Audio     []byte
    MediaType string
}
```

### Utility Types

#### Usage

```go
type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}

func (u Usage) Add(other Usage) Usage
```

#### FinishReason

```go
type FinishReason string

const (
    FinishReasonStop          FinishReason = "stop"
    FinishReasonLength        FinishReason = "length"
    FinishReasonToolCalls     FinishReason = "tool_calls"
    FinishReasonContentFilter FinishReason = "content_filter"
    FinishReasonError         FinishReason = "error"
)
```

#### ToolDefinition

```go
type ToolDefinition struct {
    Name        string
    Description string
    InputSchema json.RawMessage
}
```

#### ToolCall

```go
type ToolCall struct {
    ID    string
    Name  string
    Input json.RawMessage
}
```

#### ToolChoice

```go
type ToolChoice struct {
    Type     string // "auto", "none", "required", "function"
    ToolName string // Required when Type == "function"
}

// Constructors
func ToolChoiceAuto() ToolChoice
func ToolChoiceNone() ToolChoice
func ToolChoiceRequired() ToolChoice
func ToolChoiceNamed(toolName string) ToolChoice
```

#### ResponseFormat

```go
type ResponseFormat struct {
    Type       string          // "text", "json_object", "json_schema"
    JSONSchema json.RawMessage // Required when Type == "json_schema"
}

// Constructors
func ResponseFormatText() ResponseFormat
func ResponseFormatJSON() ResponseFormat
func ResponseFormatJSONSchema(schema json.RawMessage) ResponseFormat
```

---

## Best Practices

### Context Management

Always use context with timeouts for production code:

```go
// With timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result, err := generate.Generate(ctx, model, "prompt")

// With cancellation
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// In a goroutine, you can cancel based on user input:
go func() {
    <-userCancelSignal
    cancel()
}()

result := stream.Stream(ctx, model, "prompt")
```

### Error Handling

Handle errors gracefully with proper retry logic:

```go
func generateWithRetry(ctx context.Context, model provider.Generator, prompt string, maxRetries int) (*generate.Result, error) {
    var lastErr error

    for attempt := 0; attempt < maxRetries; attempt++ {
        result, err := generate.Generate(ctx, model, prompt)
        if err == nil {
            return result, nil
        }

        lastErr = err

        // Don't retry on certain errors
        if provider.IsAuthenticationError(err) ||
           provider.IsNotFoundError(err) ||
           provider.IsInvalidInput(err) ||
           provider.IsContentFiltered(err) {
            return nil, err
        }

        // Handle rate limiting
        if provider.IsRateLimited(err) {
            delay := provider.RetryAfter(err)
            if delay == 0 {
                delay = time.Second * time.Duration(1<<attempt)
            }
            select {
            case <-time.After(delay):
                continue
            case <-ctx.Done():
                return nil, ctx.Err()
            }
        }

        // Handle timeouts with exponential backoff
        if provider.IsTimeout(err) && attempt < maxRetries-1 {
            time.Sleep(time.Second * time.Duration(1<<attempt))
            continue
        }
    }

    return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}
```

### Streaming Cleanup

Always handle stream completion properly:

```go
result := stream.Stream(ctx, model, "prompt")

// Option 1: Consume and wait
for part := range result.Stream() {
    // Handle parts
}
// Stream is now complete

// Option 2: Use blocking methods
text, err := result.Text()
if err != nil {
    // Handle error
}

// Option 3: Goroutine pattern
go func() {
    for part := range result.Stream() {
        // Handle parts
    }
}()

<-result.Done()
text, _ := result.Text()
```

### Resource Management

Use middleware for cross-cutting concerns:

```go
// Production setup
client := openai.NewProvider(
    openai.APIKey(os.Getenv("OPENAI_API_KEY")),
    openai.WithMiddleware(
        middleware.Chain(
            middleware.Recover(),                          // Panic recovery
            middleware.Logging(logger),                    // Logging
            middleware.Metrics(metricsCollector),          // Metrics
            middleware.TimeoutWithDefault(30*time.Second), // Timeout
            middleware.Retry(middleware.RetryConfig{       // Retry
                MaxAttempts:  3,
                InitialDelay: time.Second,
                RetryOn: func(err error) bool {
                    return provider.IsRateLimited(err) || provider.IsTimeout(err)
                },
            }),
        ),
    ),
)
```

### Type Safety with Structured Output

Use struct tags for schema validation:

```go
type User struct {
    Name  string `json:"name" jsonschema:"required,minLength=1,maxLength=100,description=User's full name"`
    Email string `json:"email" jsonschema:"required,format=email,description=User's email address"`
    Age   int    `json:"age" jsonschema:"minimum=0,maximum=150,description=User's age in years"`
    Role  string `json:"role" jsonschema:"enum=admin,enum=user,enum=guest,description=User's role"`
}

result, err := generate.GenerateObject[User](ctx, model, "Generate a random user")
if err != nil {
    // Check if it's a parse error
    var parseErr *provider.ParseError
    if errors.As(err, &parseErr) {
        fmt.Printf("Failed to parse field %s: %v\n", parseErr.Field, parseErr.Err)
    }
    return err
}

user := result.Object
fmt.Printf("Name: %s, Email: %s, Age: %d\n", user.Name, user.Email, user.Age)
```

---

## Examples

The SDK includes comprehensive examples in the `examples/` directory:

| Example                                                     | Description              |
| ----------------------------------------------------------- | ------------------------ |
| [chat](./examples/openai/chat/)                             | Basic text generation    |
| [stream](./examples/openai/stream/)                         | Streaming generation     |
| [structured](./examples/openai/structured/)                 | JSON structured output   |
| [tools](./examples/openai/tools/)                           | Tool/function calling    |
| [embed](./examples/openai/embed/)                           | Text embeddings          |
| [middleware](./examples/openai/middleware/)                 | Logging and metrics      |
| [middleware_timeout](./examples/openai/middleware_timeout/) | Timeout configuration    |
| [errors](./examples/openai/errors/)                         | Error handling patterns  |
| [image](./examples/openai/image/)                           | DALL-E image generation  |
| [speech](./examples/openai/speech/)                         | Text-to-speech synthesis |
| [transcription](./examples/openai/transcription/)           | Audio transcription      |
| [audio_chat](./examples/openai/audio_chat/)                 | Multimodal audio chat    |

---

## API Stability

### Version Compatibility

The SDK follows semantic versioning:

- **Major versions (1.x.x)**: May contain breaking changes
- **Minor versions (x.1.x)**: New features, backward compatible
- **Patch versions (x.x.1)**: Bug fixes, backward compatible

### Breaking Change Policy

Breaking changes will only occur in major version releases and will be documented with migration guides.

### Deprecation Process

1. Feature marked as deprecated in release notes and godoc
2. Feature remains functional for at least one minor version
3. Feature removed in next major version

### Current Status

**Alpha/Early Development** - Expect changes between versions.  
The API may evolve significantly as the SDK matures.

---

## Comparison with Vercel AI SDK

Lamar SDK is a Go alternative to Vercel's TypeScript AI SDK with similar goals but adapted for Go idioms.

### Feature Comparison

| Feature           | Lamar SDK (Go) | Vercel AI SDK (TypeScript)   |
| ----------------- | -------------- | ---------------------------- |
| Text Generation   | ✅             | ✅                           |
| Streaming         | ✅             | ✅                           |
| Structured Output | ✅             | ✅                           |
| Tool Calling      | ✅             | ✅                           |
| Embeddings        | ✅             | ✅                           |
| Image Generation  | ✅             | ✅                           |
| Audio             | ✅             | ✅                           |
| Multi-provider    | ✅ (OpenAI)    | ✅ (OpenAI, Anthropic, etc.) |
| Middleware        | ✅             | ✅                           |
| Type Safety       | ✅ Go generics | ✅ TypeScript types          |

### Key Differences

1. **Language**: Go vs TypeScript
2. **Async Model**: Goroutines/channels vs async/await
3. **Error Handling**: Structured errors with codes vs try-catch
4. **Type System**: Struct tags for schemas vs TypeScript types
5. **Provider Ecosystem**: Currently OpenAI only vs multiple providers

### For TypeScript Developers

If you're familiar with Vercel AI SDK, here's a quick mapping:

```typescript
// TypeScript (Vercel AI SDK)
import { generateText } from "ai";
const { text } = await generateText({
  model: openai("gpt-5-mini-2025-08-07"),
  prompt: "Hello",
});
```

```go
// Go (Lamar SDK)
import "github.com/seifalmotaz/lamar-sdk/generate"
client := openai.NewProvider()
result, _ := generate.Generate(ctx, client.GPT5Mini(), "Hello")
text := result.Text()
```

---

## License

MIT License - see [LICENSE](LICENSE) for details.
