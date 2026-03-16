package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/generate"
	"github.com/seifalmotaz/lamar-sdk/middleware"
	"github.com/seifalmotaz/lamar-sdk/provider"
	"github.com/seifalmotaz/lamar-sdk/providers/anthropic"
	"github.com/seifalmotaz/lamar-sdk/stream"
	"github.com/seifalmotaz/lamar-sdk/tool"
)

const anthropicTestModel = "claude-3-5-haiku-20241022"

func getAnthropicAPIKey(t *testing.T) string {
	key := os.Getenv("ANTHROPIC_API_KEY")
	if key == "" {
		t.Skip("ANTHROPIC_API_KEY not set")
	}
	return key
}

func getAnthropicTimeout() time.Duration {
	if timeout := os.Getenv("TEST_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			return d
		}
	}
	return 60 * time.Second
}

func TestAnthropic_Generate(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'hello' and nothing else.",
		generate.MaxTokens(50),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}
	if result.Usage().TotalTokens == 0 {
		t.Error("Expected non-zero token usage")
	}

	t.Logf("Response: %s", result.Text())
	t.Logf("Tokens: %d", result.Usage().TotalTokens)
}

func TestAnthropic_GenerateWithSystem(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "What is 2+2?",
		generate.System("You are a helpful math teacher. Always answer briefly."),
		generate.MaxTokens(50),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", result.Text())
}

func TestAnthropic_Stream(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	result := stream.Stream(ctx, model, "Count from 1 to 5",
		stream.MaxTokens(50),
	)

	var textParts []string
	var finishFound bool

	for part := range result.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			textParts = append(textParts, p.Delta)
		case provider.StreamFinishPart:
			finishFound = true
		case provider.StreamErrorPart:
			t.Fatalf("Stream error: %v", p.Error)
		}
	}

	if !finishFound {
		t.Error("Expected finish part in stream")
	}

	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}
	if text == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", text)

	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() failed: %v", err)
	}
	t.Logf("Tokens: %d", usage.TotalTokens)
}

func TestAnthropic_GenerateObject(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	type Person struct {
		Name string `json:"name" jsonschema:"required,description=The person's name"`
		Age  int    `json:"age" jsonschema:"required,minimum=0,description=Age in years"`
	}

	result, err := generate.GenerateObject[Person](ctx, model,
		"Generate a random fictional person. Respond with just name and age.",
		generate.MaxTokens(100),
	)
	if err != nil {
		t.Fatalf("GenerateObject failed: %v", err)
	}

	if result.Object.Name == "" {
		t.Error("Expected non-empty name")
	}
	if result.Object.Age < 0 {
		t.Error("Age should not be negative")
	}

	t.Logf("Generated person: %+v", result.Object)
	t.Logf("Tokens: %d", result.Usage.TotalTokens)
}

func TestAnthropic_GenerateWithToolExecution(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	calculatorTool := tool.NewTool("calculator", "Perform basic arithmetic",
		func(ctx context.Context, input struct {
			A int `json:"a"`
			B int `json:"b"`
		}) (struct {
			Result int `json:"result"`
		}, error) {
			return struct {
				Result int `json:"result"`
			}{Result: input.A + input.B}, nil
		},
	)

	result, err := generate.Generate(ctx, model,
		"What is 15 + 27? Use the calculator tool.",
		generate.Tools(tool.ToDefinition(calculatorTool)),
		generate.MaxTokens(200),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	toolCalls := result.ToolCalls()
	if len(toolCalls) == 0 {
		t.Fatalf("Expected tool calls, got none. Finish reason: %s, Text: %s",
			result.FinishReason(), result.Text())
	}

	tc := toolCalls[0]
	if tc.Name != "calculator" {
		t.Errorf("Expected tool name 'calculator', got %s", tc.Name)
	}

	var input struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	if err := json.Unmarshal(tc.Input, &input); err != nil {
		t.Fatalf("Failed to unmarshal tool input: %v", err)
	}

	output, err := calculatorTool.Execute(ctx, tc.Input)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	t.Logf("Tool call: %s(%s) = %s", tc.Name, string(tc.Input), string(output))
	t.Logf("Finish reason: %s", result.FinishReason())
	t.Logf("Tokens: %d", result.Usage().TotalTokens)
}

func TestAnthropic_StreamWithToolExecution(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	weatherTool := tool.NewTool("get_weather", "Get current weather",
		func(ctx context.Context, input struct {
			Location string `json:"location"`
		}) (struct {
			Temperature float64 `json:"temperature"`
			Condition   string  `json:"condition"`
		}, error) {
			return struct {
				Temperature float64 `json:"temperature"`
				Condition   string  `json:"condition"`
			}{Temperature: 22.5, Condition: "sunny"}, nil
		},
	)

	result := stream.Stream(ctx, model,
		"Calculate 7 times 8 using the calculator tool.",
		stream.Tools(tool.ToDefinition(weatherTool)),
		stream.MaxTokens(200),
	)

	var toolCallParts []provider.StreamToolCallPart
	var textParts []string
	var finishFound bool

	for part := range result.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			textParts = append(textParts, p.Delta)
		case provider.StreamToolCallPart:
			toolCallParts = append(toolCallParts, p)
		case provider.StreamFinishPart:
			finishFound = true
		case provider.StreamErrorPart:
			t.Fatalf("Stream error: %v", p.Error)
		}
	}

	if !finishFound {
		t.Error("Expected finish part in stream")
	}

	_ = toolCallParts
	_ = textParts
}

func TestAnthropic_ContextCancellation(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := generate.Generate(ctx, model, "Hello")
	if err == nil {
		t.Error("Expected error for canceled context")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("Expected provider.Error, got %T", err)
		return
	}
	if providerErr.Code != provider.CodeContextCanceled {
		t.Errorf("Expected CodeContextCanceled, got %v", providerErr.Code)
	}
}

func TestAnthropic_StreamWithTimeout(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx := context.Background()

	result := stream.Stream(ctx, model, "Hello",
		stream.WithTimeout(60*time.Second),
	)

	for range result.Stream() {
	}

	text, err := result.Text()
	if err != nil {
		t.Fatalf("Text() failed: %v", err)
	}

	if text == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", text)
}

func TestAnthropic_GenerateTemperature(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'test'",
		generate.MaxTokens(20),
		generate.Temperature(0.1),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	t.Logf("Response: %s", result.Text())
}

func TestAnthropic_ToolLoop(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), 2*getAnthropicTimeout())
	defer cancel()

	weatherTool := tool.NewTool("get_weather", "Get current weather for a location",
		func(ctx context.Context, input struct {
			Location string `json:"location" jsonschema:"required,description=City name"`
		}) (struct {
			Temperature float64 `json:"temperature"`
			Condition   string  `json:"condition"`
		}, error) {
			return struct {
				Temperature float64 `json:"temperature"`
				Condition   string  `json:"condition"`
			}{Temperature: 22.5, Condition: "sunny"}, nil
		},
	)

	result, err := generate.Generate(ctx, model,
		"What's the weather like in Tokyo?",
		generate.Tools(tool.ToDefinition(weatherTool)),
		generate.MaxTokens(200),
	)
	if err != nil {
		t.Fatalf("First generate failed: %v", err)
	}

	toolCalls := result.ToolCalls()
	if len(toolCalls) == 0 {
		t.Fatalf("Expected tool calls on first request, got none. Finish reason: %s",
			result.FinishReason())
	}

	if result.FinishReason() != provider.FinishReasonToolCalls {
		t.Errorf("Expected FinishReasonToolCalls, got %s", result.FinishReason())
	}

	var toolResults []provider.Content
	for _, tc := range toolCalls {
		output, err := weatherTool.Execute(ctx, tc.Input)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		toolResults = append(toolResults,
			provider.NewToolResultContent(tc.ID, tc.Name, output, false),
		)
		t.Logf("Tool executed: %s -> %s", tc.Name, string(output))
	}

	messages := []provider.Message{
		provider.UserMessage("What's the weather like in Tokyo?"),
		{Role: provider.RoleAssistant, Content: result.Content()},
	}

	for _, tr := range toolResults {
		messages = append(messages, provider.ToolResultMessage(tr.(provider.ToolResultContent)))
	}

	result2, err := generate.Generate(ctx, model, "",
		generate.Messages(messages...),
		generate.Tools(tool.ToDefinition(weatherTool)),
		generate.MaxTokens(200),
	)
	if err != nil {
		t.Fatalf("Second generate failed: %v", err)
	}

	text := result2.Text()
	if text == "" {
		t.Error("Expected non-empty text in second response")
	}

	textLower := strings.ToLower(text)
	if !strings.Contains(textLower, "22") && !strings.Contains(textLower, "sunny") {
		t.Logf("Warning: response may not reference tool result. Got: %s", text)
	}

	t.Logf("Final response: %s", text)
}

func TestAnthropic_Thinking(t *testing.T) {
	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(anthropic.APIKey(apiKey))

	model := client.Claude35Sonnet(
		anthropic.ThinkingEnabled(1024),
	)

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "What is 15 * 7?",
		generate.MaxTokens(500),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", result.Text())
	t.Logf("Tokens: %d", result.Usage().TotalTokens)
}

func TestAnthropic_Middleware(t *testing.T) {
	var requestCount int
	loggingMiddleware := middleware.Middleware(func(next middleware.Handler) middleware.Handler {
		return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
			requestCount++
			t.Logf("[Middleware] Request #%d: provider=%s, model=%s", requestCount, req.Provider(), req.ModelID())
			return next.Handle(ctx, req)
		})
	})

	apiKey := getAnthropicAPIKey(t)
	client := anthropic.NewProvider(
		anthropic.APIKey(apiKey),
		anthropic.WithMiddleware(loggingMiddleware),
	)
	model := client.Claude35Haiku()

	ctx, cancel := context.WithTimeout(context.Background(), getAnthropicTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'test'",
		generate.MaxTokens(20),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", result.Text())

	if requestCount < 1 {
		t.Errorf("Expected at least 1 middleware invocation, got %d", requestCount)
	}
}
