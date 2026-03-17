package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/image"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	prompt := "A serene mountain landscape at sunset with a lake in the foreground"
	if len(os.Args) > 1 {
		prompt = os.Args[1]
	}

	outputFile := "output.png"
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.DALLE3()

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Println("Image Generation")
	fmt.Println("----------------")
	fmt.Printf("Prompt: %s\n", prompt)
	fmt.Printf("Model: dall-e-3\n")
	fmt.Println()

	result, err := image.Generate(ctx, model, prompt,
		image.WithSize("1024x1024"),
		image.WithQuality("standard"),
		image.WithFormat("png"),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	if len(result.Images) == 0 {
		fmt.Println("No images generated")
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, result.Images[0], 0644); err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Image saved to: %s\n", outputFile)
	fmt.Printf("Size: %d bytes\n", len(result.Images[0]))
	fmt.Printf("Media type: %s\n", result.MediaType)
	if len(result.RevisedPrompts) > 0 {
		fmt.Printf("Revised prompt: %s\n", result.RevisedPrompts[0])
	}
	fmt.Println()
	fmt.Printf("You can view it with: open %s (macOS) or xdg-open %s (Linux)\n", outputFile, outputFile)
}
