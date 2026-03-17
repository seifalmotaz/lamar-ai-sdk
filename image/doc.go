package image

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

/*
Package image provides a high-level API for image generation with AI models.

The image package wraps provider.ImageModel implementations with:

  - Functional options pattern for configuration
  - Input validation with fail-fast semantics
  - Default timeouts (120 seconds)
  - Logging and metrics support

# Basic Usage

	package main

	import (
	    "context"
	    "fmt"
	    "os"

	    "github.com/seifalmotaz/lamar-ai-sdk/image"
	    "github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	)

	func main() {
	    ctx := context.Background()
	    client := openai.NewProvider(openai.APIKey(os.Getenv("OPENAI_API_KEY")))
	    model := client.DALLE3()

	    result, err := image.Generate(ctx, model, "A serene mountain landscape at sunset",
	        image.WithSize("1024x1024"),
	        image.WithQuality("standard"),
	    )
	    if err != nil {
	        panic(err)
	    }

	    fmt.Printf("Generated %d image(s)\n", len(result.Images))
	    fmt.Printf("Media type: %s\n", result.MediaType)
	}

# With Custom Timeout

	result, err := image.Generate(ctx, model, "A beautiful sunset",
	    image.WithTimeout(180*time.Second),
	)

# No Timeout (context deadline only)

	result, err := image.Generate(ctx, model, "Generate art",
	    image.WithNoTimeout(),
	)

# Multiple Images

	model := client.DALLE2() // DALL-E 2 supports up to 10 images per request
	result, err := image.Generate(ctx, model, "A cat wearing a hat",
	    image.WithN(3), // Generate 3 images
	)

# Quality and Format Options

	result, err := image.Generate(ctx, model, "A futuristic city",
	    image.WithSize("1792x1024"),
	    image.WithQuality("hd"),
	    image.WithFormat("png"),
	)
*/
type _ = provider.ImageModel
