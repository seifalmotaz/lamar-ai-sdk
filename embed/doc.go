package embed

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

/*
Package embed provides a high-level API for generating text embeddings with AI models.

The embed package wraps provider.EmbeddingModel implementations with:

  - Functional options pattern for configuration
  - Input validation with fail-fast semantics
  - Default timeouts (10s for single, 5min for batch)
  - Automatic batching for EmbedBatch

# Basic Usage

	pkg main

	import (
	    "context"
	    "fmt"

	    "github.com/seifalmotaz/lamar-ai-sdk/embed"
	    "github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	)

	func main() {
	    ctx := context.Background()
	    model := openai.NewEmbeddingModel("text-embedding-3-small", "api-key")

	    result, err := embed.Embed(ctx, model, "Hello, world!")
	    if err != nil {
	        panic(err)
	    }

	    fmt.Printf("Embedding dimension: %d\n", len(result.Embedding))
	}

# Batch Embeddings

	texts := []string{"Hello", "World", "Goodbye"}
	result, err := embed.EmbedBatch(ctx, model, texts)
	if err != nil {
	    panic(err)
	}

	for i, emb := range result.Embeddings {
	    fmt.Printf("Text %d: %d dimensions\n", i, len(emb))
	}

# Timeouts

	// Use default 10 second timeout
	result, err := embed.Embed(ctx, model, "text")

	// Override with custom timeout
	result, err = embed.Embed(ctx, model, "text",
	    embed.WithTimeout(30*time.Second),
	)

	// No timeout (context deadline only)
	result, err = embed.Embed(ctx, model, "text",
	    embed.WithNoTimeout(),
	)
*/
type _ = provider.Model
