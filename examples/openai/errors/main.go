// Package main demonstrates error handling with the Lamar SDK.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
)

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	fmt.Println("Lamar SDK Error Handling Examples")
	fmt.Println("==================================")
	fmt.Println()

	exampleInvalidModel()
	exampleContextCancellation()
	exampleStructuredErrors()
	exampleErrorHelpers()
	exampleRateLimitHandling()
}

func exampleInvalidModel() {
	fmt.Println("1. Invalid Model Error")
	fmt.Println("----------------------")

	_, err := generate.Generate(context.Background(), nil, "Hello")
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		var providerErr *provider.Error
		if errors.As(err, &providerErr) {
			fmt.Printf("  Code: %s\n", providerErr.Code)
			fmt.Printf("  Message: %s\n", providerErr.Message)
		}

		if errors.Is(err, provider.ErrInvalidModel) {
			fmt.Println("  Detected: Invalid model (using errors.Is)")
		}
	}
	fmt.Println()
}

func exampleContextCancellation() {
	fmt.Println("2. Context Cancellation")
	fmt.Println("-----------------------")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	client := openai.NewProvider(openai.APIKey("test-key"))
	_, err := generate.Generate(ctx, client.GPT5Mini(), "Hello")

	if err != nil {
		fmt.Printf("Error: %v\n", err)

		if provider.IsContextCanceled(err) {
			fmt.Println("  Detected: Context was canceled")
		}
	}
	fmt.Println()
}

func exampleStructuredErrors() {
	fmt.Println("3. Structured Error Inspection")
	fmt.Println("------------------------------")

	// Create a sample error with metadata
	err := provider.NewErrorWithMeta(
		provider.CodeModelNotFound,
		"The specified model does not exist",
		nil,
		"openai",
		"gpt-5-turbo",
	)

	fmt.Printf("Error: %v\n", err)

	var providerErr *provider.Error
	if errors.As(err, &providerErr) {
		fmt.Printf("  Code: %s\n", providerErr.Code)
		fmt.Printf("  Provider: %s\n", providerErr.Provider)
		fmt.Printf("  ModelID: %s\n", providerErr.ModelID)
		fmt.Printf("  RetryAfter: %v\n", providerErr.RetryAfter)
	}
	fmt.Println()
}

func exampleErrorHelpers() {
	fmt.Println("4. Error Helper Functions")
	fmt.Println("-------------------------")

	err := provider.NewError(provider.CodeRateLimited, "Too many requests", nil)

	code := provider.ErrorCodeOf(err)
	fmt.Printf("Error code: %s\n", code)

	if provider.IsRateLimited(err) {
		fmt.Println("  IsRateLimited: true")
	}

	if !provider.IsTimeout(err) {
		fmt.Println("  IsTimeout: false")
	}

	if !provider.IsAuthenticationError(err) {
		fmt.Println("  IsAuthenticationError: false")
	}
	fmt.Println()
}

func exampleRateLimitHandling() {
	fmt.Println("5. Rate Limit with Retry-After")
	fmt.Println("------------------------------")

	err := &provider.Error{
		Code:       provider.CodeRateLimited,
		Message:    "Rate limit exceeded",
		Provider:   "openai",
		ModelID:    "gpt-5-mini-2025-08-07",
		RetryAfter: 30 * time.Second,
	}

	fmt.Printf("Error: %v\n", err)

	if provider.IsRateLimited(err) {
		retryAfter := provider.RetryAfter(err)
		fmt.Printf("  Rate limited! Retry after: %v\n", retryAfter)
		fmt.Printf("  Suggested action: Wait %v before retrying\n", retryAfter)
	}
	fmt.Println()
}
