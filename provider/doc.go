package provider

import "context"

const doc = `
Package provider defines the core interfaces and types for the Lamar AI SDK.

The provider package is the abstraction layer that all AI providers implement.
It defines interfaces for different model capabilities (generation, streaming,
embeddings) and common types used throughout the SDK.

# Interface Segregation

Models implement only what they support:

	- Model: Base interface (Provider, ModelID)
	- Generator: Non-streaming text generation
	- Streamer: Streaming text generation
	- LanguageModel: Full-featured (Generator + Streamer)
	- EmbeddingModel: Text embeddings

Check capabilities with helper functions:

	if provider.CanStream(model) {
	    // Model supports streaming
	}

# Content Types

Content is a polymorphic type representing different message parts:

	content := provider.Text("Hello, world!")

Use type assertion to determine the specific type:

	switch c := content.(type) {
	case provider.TextContent:
	    fmt.Println(c.Text)
	case provider.ImageContent:
	    // Handle image
	}

# Error Handling

All errors are structured with codes for programmatic handling:

	var err *provider.Error
	if errors.As(err, &providerErr) {
	    switch providerErr.Code {
	    case provider.CodeRateLimited:
	        retryAfter := providerErr.RetryAfter
	    }
	}

Helper functions are available:

	if provider.IsRateLimited(err) {
	    // Handle rate limit
	}

# Example

	package main

	import (
	    "context"

	    "github.com/seifalmotaz/lamar/provider"
	)

	func main() {
	    var model provider.Generator = getMyModel()

	    result, err := model.Generate(context.Background(), &provider.GenerateRequest{
	        Prompt: "Hello, world!",
	    })
	    if err != nil {
	        panic(err)
	    }

	    fmt.Println(result.Text)
	}
`

var _ = context.Background
