// Package main demonstrates timeout middleware usage with OpenAI.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/middleware"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	fmt.Println("Timeout Middleware Example")
	fmt.Println("==========================")
	fmt.Println()

	// Example 1: Simple timeout with default
	fmt.Println("1. Simple timeout with default duration:")
	fmt.Println()

	client := openai.NewProvider(
		openai.APIKey(apiKey),
		openai.WithMiddleware(
			middleware.TimeoutWithDefault(30*time.Second),
		),
	)
	model := client.GPT5Mini()

	result, err := generate.Generate(context.Background(), model,
		"What is 2 + 2? Answer with just the number.",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %s\n\n", result.Text())

	// Example 2: Per-provider timeouts
	fmt.Println("2. Per-provider timeout configuration:")
	fmt.Println()

	multiProviderTimeout := middleware.Timeout(middleware.TimeoutConfig{
		Default: 60 * time.Second,
		PerProvider: map[string]time.Duration{
			"openai":    45 * time.Second,
			"anthropic": 90 * time.Second,
		},
	})

	client2 := openai.NewProvider(
		openai.APIKey(apiKey),
		openai.WithMiddleware(
			multiProviderTimeout,
		),
	)

	result2, err := generate.Generate(context.Background(), client2.GPT5Mini(),
		"Say hello in French.",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %s\n\n", result2.Text())

	// Example 3: Per-model timeouts
	fmt.Println("3. Per-model timeout configuration:")
	fmt.Println()

	modelTimeout := middleware.Timeout(middleware.TimeoutConfig{
		Default: 30 * time.Second,
		PerModel: map[string]time.Duration{
			"o1":         120 * time.Second,
			"o1-mini":    90 * time.Second,
			"o1-preview": 120 * time.Second,
		},
	})

	client3 := openai.NewProvider(
		openai.APIKey(apiKey),
		openai.WithMiddleware(
			modelTimeout,
		),
	)

	result3, err := generate.Generate(context.Background(), client3.GPT5Mini(),
		"What is 3 + 3? Answer with just the number.",
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Result: %s\n\n", result3.Text())

	// Example 4: Context deadline takes precedence
	fmt.Println("4. Context deadline takes precedence:")
	fmt.Println()

	client4 := openai.NewProvider(
		openai.APIKey(apiKey),
		openai.WithMiddleware(
			middleware.TimeoutWithDefault(30*time.Second),
		),
	)

	// Create a context with a shorter deadline
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result4, err := generate.Generate(ctx, client4.GPT5Mini(),
		"Say goodbye.",
	)
	if err != nil {
		fmt.Printf("Error (expected): %v\n", err)
	} else {
		fmt.Printf("Result: %s\n", result4.Text())
	}

	fmt.Println()
	fmt.Println("Key Points:")
	fmt.Println("- Timeout middleware enforces request timeouts")
	fmt.Println("- Context deadlines take precedence over middleware timeout")
	fmt.Println("- Per-model and per-provider overrides are supported")
	fmt.Println("- Zero timeout means no timeout (passthrough)")
}
