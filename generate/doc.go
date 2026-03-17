package generate

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

/*
Package generate provides a high-level API for text generation with AI models.

The generate package wraps provider.Generator implementations with:

  - Functional options pattern for configuration
  - Input validation with fail-fast semantics
  - Default timeouts (30 seconds)
  - Clean Result type with accessor methods

# Basic Usage

	pkg main

	import (
	    "context"
	    "fmt"
	    "time"

	    "github.com/seifalmotaz/lamar-ai-sdk/generate"
	    "github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	)

	func main() {
	    ctx := context.Background()
	    model := openai.NewChatModel("gpt-4", "api-key")

	    result, err := generate.Generate(ctx, model, "What is Go?",
	        generate.MaxTokens(100),
	        generate.Temperature(0.7),
	    )
	    if err != nil {
	        panic(err)
	    }

	    fmt.Println(result.Text())
	}

# With System Prompt

	result, err := generate.Generate(ctx, model, "Explain quantum computing",
	    generate.System("You are a helpful physics tutor."),
	    generate.MaxTokens(500),
	)

# With Conversation History

	result, err := generate.Generate(ctx, model, "",
	    generate.Messages(
	        provider.UserMessage("Hello!"),
	        provider.AssistantMessage("Hi! How can I help you?"),
	        provider.UserMessage("Tell me about Go."),
	    ),
	)

# With Tools

	tool := provider.ToolDefinition{
	    Name:        "get_weather",
	    Description: "Get current weather",
	    InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string"}}}`),
	}

	result, err := generate.Generate(ctx, model, "What's the weather in Tokyo?",
	    generate.Tools(tool),
	    generate.ToolChoice(provider.ToolChoiceAuto()),
	)

# Timeouts

	// Use default 30 second timeout
	result, err := generate.Generate(ctx, model, "Hello")

	// Override with custom timeout
	result, err = generate.Generate(ctx, model, "Write a novel",
	    generate.WithTimeout(5*time.Minute),
	)

	// No timeout (context deadline only)
	result, err = generate.Generate(ctx, model, "Hello",
	    generate.WithTimeout(0),
	)
*/
type _ = provider.Model
