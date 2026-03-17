// Package main demonstrates vision/image analysis with Anthropic Claude.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/anthropic"
)

func main() {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		fmt.Println("ANTHROPIC_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Sonnet()

	fmt.Println("Vision Example with Claude")
	fmt.Println("==========================")

	fmt.Println("\nExample 1: Image from file")
	fmt.Println("---------------------------")

	imageData, err := os.ReadFile("example.jpg")
	if err != nil {
		fmt.Printf("Note: Could not read example.jpg: %v\n", err)
		fmt.Println("To test with an image, place a file named 'example.jpg' in this directory.")
		fmt.Println("\nShowing URL example instead...")
	}

	if len(imageData) > 0 {
		msg := provider.UserMessageWithContent(
			provider.Text("What do you see in this image? Describe it in detail."),
			provider.Image(imageData, "image/jpeg"),
		)

		result, err := generate.Generate(context.Background(), model, "",
			generate.Messages(msg),
			generate.MaxTokens(500),
		)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(result.Text())
		fmt.Printf("\nTokens: %d\n", result.Usage().TotalTokens)
	}

	fmt.Println("\nExample 2: Image from URL")
	fmt.Println("--------------------------")

	msg := provider.UserMessageWithContent(
		provider.Text("Describe what you see in this image."),
		provider.ImageFromURL("https://upload.wikimedia.org/wikipedia/commons/thumb/d/dd/Gfp-wisconsin-madison-the-nature-boardwalk.jpg/1280px-Gfp-wisconsin-madison-the-nature-boardwalk.jpg"),
	)

	result, err := generate.Generate(context.Background(), model, "",
		generate.Messages(msg),
		generate.MaxTokens(500),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(result.Text())
	fmt.Printf("\nTokens: %d\n", result.Usage().TotalTokens)
}
