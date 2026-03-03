// Package main demonstrates middleware usage with OpenAI.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-sdk/generate"
	"github.com/seifalmotaz/lamar-sdk/providers/openai"
)

// consoleLogger implements provider.Logger
type consoleLogger struct{}

func (l *consoleLogger) Debug(msg string, args ...any) {
	fmt.Printf("[DEBUG] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *consoleLogger) Info(msg string, args ...any) {
	fmt.Printf("[INFO] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *consoleLogger) Warn(msg string, args ...any) {
	fmt.Printf("[WARN] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

func (l *consoleLogger) Error(msg string, args ...any) {
	fmt.Printf("[ERROR] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

// consoleMetrics implements provider.MetricsCollector
type consoleMetrics struct{}

func (m *consoleMetrics) RecordRequest(ctx context.Context, provider, model string, duration time.Duration, err error) {
	if err != nil {
		fmt.Printf("[METRIC] request: provider=%s model=%s duration=%v error=%v\n", provider, model, duration, err)
	} else {
		fmt.Printf("[METRIC] request: provider=%s model=%s duration=%v\n", provider, model, duration)
	}
}

func (m *consoleMetrics) RecordTokens(ctx context.Context, provider, model string, prompt, completion int) {
	fmt.Printf("[METRIC] tokens: provider=%s model=%s prompt=%d completion=%d\n", provider, model, prompt, completion)
}

func (m *consoleMetrics) RecordStreamStart(ctx context.Context, provider, model string) {
	fmt.Printf("[METRIC] stream_start: provider=%s model=%s\n", provider, model)
}

func (m *consoleMetrics) RecordStreamEvent(ctx context.Context, provider, model, eventType string) {
	fmt.Printf("[METRIC] stream_event: provider=%s model=%s type=%s\n", provider, model, eventType)
}

func (m *consoleMetrics) RecordStreamEnd(ctx context.Context, provider, model string, duration time.Duration) {
	fmt.Printf("[METRIC] stream_end: provider=%s model=%s duration=%v\n", provider, model, duration)
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		fmt.Println("OPENAI_API_KEY environment variable not set")
		os.Exit(1)
	}

	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	fmt.Println("Middleware example:")
	fmt.Println("===================")
	fmt.Println()

	// Generate with logging and metrics
	result, err := generate.Generate(context.Background(), model,
		"What is 2 + 2? Answer with just the number.",
		generate.WithLogger(&consoleLogger{}),
		generate.WithMetrics(&consoleMetrics{}),
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Printf("Result: %s\n", result.Text())
	fmt.Printf("Tokens: %d\n", result.Usage().TotalTokens)

	// Demonstrate retry middleware concept
	fmt.Println()
	fmt.Println("Retry middleware example:")
	fmt.Println("-------------------------")
	fmt.Println("The retry middleware automatically retries failed requests with exponential backoff.")
	fmt.Println("Configuration:")
	fmt.Printf("  - MaxAttempts: 3\n")
	fmt.Printf("  - InitialDelay: 1s\n")
	fmt.Printf("  - MaxDelay: 30s\n")
	fmt.Printf("  - Multiplier: 2.0\n")
	fmt.Println()
	fmt.Println("Retry is triggered on:")
	fmt.Println("  - Rate limits (HTTP 429)")
	fmt.Println("  - Timeouts")
	fmt.Println("  - Server errors (HTTP 5xx)")
	fmt.Println()
	fmt.Println("Usage in middleware chain:")
	fmt.Println("  handler := middleware.Chain(")
	fmt.Println("      middleware.Logging(logger),")
	fmt.Println("      middleware.Metrics(collector),")
	fmt.Println("      middleware.Retry(middleware.DefaultRetryConfig()),")
	fmt.Println("  )(baseHandler)")
}
