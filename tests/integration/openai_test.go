package integration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/embed"
	"github.com/seifalmotaz/lamar-ai-sdk/generate"
	"github.com/seifalmotaz/lamar-ai-sdk/middleware"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
	"github.com/seifalmotaz/lamar-ai-sdk/providers/openai"
	"github.com/seifalmotaz/lamar-ai-sdk/stream"
	"github.com/seifalmotaz/lamar-ai-sdk/tool"
)

const testModel = "gpt-5-mini-2025-08-07"
const embeddingModel = "text-embedding-3-small"

func getAPIKey(t *testing.T) string {
	key := os.Getenv("OPENAI_API_KEY")
	if key == "" {
		t.Skip("OPENAI_API_KEY not set")
	}
	return key
}

func getTimeout() time.Duration {
	if timeout := os.Getenv("TEST_TIMEOUT"); timeout != "" {
		if d, err := time.ParseDuration(timeout); err == nil {
			return d
		}
	}
	return 60 * time.Second
}

func TestOpenAI_Generate(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'hello' and nothing else.",
		generate.MaxTokens(10),
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

func TestOpenAI_GenerateWithSystem(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "What is 2+2?",
		generate.System("You are a helpful math teacher. Always answer briefly."),
		generate.MaxTokens(20),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result.Text() == "" {
		t.Error("Expected non-empty text")
	}

	t.Logf("Response: %s", result.Text())
}

func TestOpenAI_Stream(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result := stream.Stream(ctx, model, "Count from 1 to 5",
		stream.MaxTokens(20),
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

func TestOpenAI_Embed(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := embed.Embed(ctx, model, "Hello, world!")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(result.Embedding) == 0 {
		t.Error("Expected non-empty embedding")
	}
	if result.Usage.TotalTokens == 0 {
		t.Error("Expected non-zero token usage")
	}

	t.Logf("Embedding dimension: %d", len(result.Embedding))
	t.Logf("Tokens: %d", result.Usage.TotalTokens)
}

func TestOpenAI_EmbedBatch(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	texts := []string{
		"The quick brown fox",
		"jumps over the lazy dog",
		"Hello world",
	}

	result, err := embed.EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(result.Embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}

	for i, emb := range result.Embeddings {
		if len(emb) == 0 {
			t.Errorf("Embedding %d is empty", i)
		}
	}

	t.Logf("Generated %d embeddings", len(result.Embeddings))
	t.Logf("Tokens: %d", result.Usage.TotalTokens)
}

func TestOpenAI_GenerateObject(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type Person struct {
		Name string `json:"name" jsonschema:"required,description=The person's name"`
		Age  int    `json:"age" jsonschema:"required,minimum=0,description=Age in years"`
	}

	result, err := generate.GenerateObject[Person](ctx, model,
		"Generate a random fictional person. Respond with just a name and age.",
		generate.MaxTokens(50),
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

func TestOpenAI_StreamObject(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type Color struct {
		Name    string `json:"name" jsonschema:"required,description=Color name"`
		Hex     string `json:"hex" jsonschema:"required,description=Hex color code"`
		Popular bool   `json:"popular" jsonschema:"description=Is this color popular"`
	}

	result := stream.StreamObject[Color](ctx, model,
		"Generate a random color. Respond with name, hex code, and whether it's popular.",
		stream.MaxTokens(50),
		stream.WithTimeout(30*time.Second),
	)

	// Drain stream
	for part := range result.Stream() {
		if part.Type == "error" {
			t.Fatalf("Stream error: %v", part.Error)
		}
	}

	obj, err := result.Object()
	if err != nil {
		t.Fatalf("Object() failed: %v", err)
	}

	if obj.Name == "" {
		t.Error("Expected non-empty name")
	}

	t.Logf("Generated color: %+v", obj)

	usage, err := result.Usage()
	if err != nil {
		t.Fatalf("Usage() failed: %v", err)
	}
	t.Logf("Tokens: %d", usage.TotalTokens)
}

func TestOpenAI_GenerateWithContextCancellation(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

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

func TestOpenAI_StreamWithTimeout(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx := context.Background()

	result := stream.Stream(ctx, model, "Hello",
		stream.WithTimeout(30*time.Second),
	)

	// Drain stream
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

func TestOpenAI_GenerateTemperature(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := generate.Generate(ctx, model, "Say 'test'",
		generate.MaxTokens(5),
		generate.Temperature(0.1), // Low temperature for deterministic output
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	t.Logf("Response: %s", result.Text())
}

func TestOpenAI_EmbedBatchUneven(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TextEmbedding3Small()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	// Test with 10 items - will create uneven batches
	texts := []string{
		"one", "two", "three", "four", "five",
		"six", "seven", "eight", "nine", "ten",
	}

	result, err := embed.EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch failed: %v", err)
	}

	if len(result.Embeddings) != len(texts) {
		t.Errorf("Expected %d embeddings, got %d", len(texts), len(result.Embeddings))
	}

	// Verify all embeddings are present
	for i, emb := range result.Embeddings {
		if len(emb) == 0 {
			t.Errorf("Embedding %d is empty", i)
		}
	}

	t.Logf("Generated %d embeddings", len(result.Embeddings))
}

type calculatorInput struct {
	A int `json:"a" jsonschema:"required,description=First number"`
	B int `json:"b" jsonschema:"required,description=Second number"`
}

type calculatorOutput struct {
	Result int `json:"result"`
}

func TestOpenAI_GenerateWithToolExecution(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	calculatorTool := tool.NewTool("calculator", "Perform basic arithmetic",
		func(ctx context.Context, input calculatorInput) (calculatorOutput, error) {
			return calculatorOutput{Result: input.A + input.B}, nil
		},
	)

	result, err := generate.Generate(ctx, model,
		"What is 15 + 27? Use the calculator tool.",
		generate.Tools(tool.ToDefinition(calculatorTool)),
		generate.MaxTokens(100),
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

	var input calculatorInput
	if err := json.Unmarshal(tc.Input, &input); err != nil {
		t.Fatalf("Failed to unmarshal tool input: %v", err)
	}

	if input.A != 15 && input.B != 27 {
		if input.A != 27 && input.B != 15 {
			t.Logf("Warning: input values are %d and %d (expected 15 and 27)", input.A, input.B)
		}
	}

	output, err := calculatorTool.Execute(ctx, tc.Input)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	var result2 calculatorOutput
	if err := json.Unmarshal(output, &result2); err != nil {
		t.Fatalf("Failed to unmarshal tool output: %v", err)
	}

	expectedResult := input.A + input.B
	if result2.Result != expectedResult {
		t.Errorf("Expected result %d, got %d", expectedResult, result2.Result)
	}

	t.Logf("Tool call: %s(%s) = %d", tc.Name, string(tc.Input), result2.Result)
	t.Logf("Finish reason: %s", result.FinishReason())
	t.Logf("Tokens: %d", result.Usage().TotalTokens)
}

func TestOpenAI_GenerateWithToolLoop(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type weatherInput struct {
		Location string `json:"location" jsonschema:"required,description=City name"`
	}

	type weatherOutput struct {
		Temperature float64 `json:"temperature"`
		Condition   string  `json:"condition"`
	}

	weatherTool := tool.NewTool("get_weather", "Get current weather for a location",
		func(ctx context.Context, input weatherInput) (weatherOutput, error) {
			return weatherOutput{
				Temperature: 22.5,
				Condition:   "sunny",
			}, nil
		},
	)

	result, err := generate.Generate(ctx, model,
		"What's the weather like in Tokyo?",
		generate.Tools(tool.ToDefinition(weatherTool)),
		generate.MaxTokens(100),
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
		generate.MaxTokens(100),
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
	t.Logf("Total tokens across both calls: %d", result.Usage().TotalTokens+result2.Usage().TotalTokens)
}

func TestOpenAI_StreamWithToolExecution(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	calculatorTool := tool.NewTool("calculator", "Perform basic arithmetic",
		func(ctx context.Context, input calculatorInput) (calculatorOutput, error) {
			return calculatorOutput{Result: input.A * input.B}, nil
		},
	)

	result := stream.Stream(ctx, model,
		"Calculate 7 times 8 using the calculator tool.",
		stream.Tools(tool.ToDefinition(calculatorTool)),
		stream.MaxTokens(100),
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

	if len(toolCallParts) == 0 {
		t.Fatalf("Expected tool call parts, got none. Text: %s", strings.Join(textParts, ""))
	}

	tc := toolCallParts[0]
	if tc.ToolCall.Name != "calculator" {
		t.Errorf("Expected tool name 'calculator', got %s", tc.ToolCall.Name)
	}

	var input calculatorInput
	if err := json.Unmarshal(tc.ToolCall.Input, &input); err != nil {
		t.Fatalf("Failed to unmarshal tool input: %v", err)
	}

	output, err := calculatorTool.Execute(ctx, tc.ToolCall.Input)
	if err != nil {
		t.Fatalf("Tool execution failed: %v", err)
	}

	var calcResult calculatorOutput
	if err := json.Unmarshal(output, &calcResult); err != nil {
		t.Fatalf("Failed to unmarshal tool output: %v", err)
	}

	expectedResult := input.A * input.B
	if calcResult.Result != expectedResult {
		t.Errorf("Expected result %d, got %d", expectedResult, calcResult.Result)
	}

	t.Logf("Tool call streamed: %s(%s) = %d", tc.ToolCall.Name, string(tc.ToolCall.Input), calcResult.Result)
	t.Logf("Text parts received: %d", len(textParts))
}

func TestOpenAI_StreamWithToolLoop(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type weatherInput struct {
		Location string `json:"location" jsonschema:"required,description=City name"`
	}

	type weatherOutput struct {
		Temperature float64 `json:"temperature"`
		Condition   string  `json:"condition"`
	}

	weatherTool := tool.NewTool("get_weather", "Get current weather",
		func(ctx context.Context, input weatherInput) (weatherOutput, error) {
			temps := map[string]float64{"Tokyo": 25.0, "Paris": 18.0}
			conds := map[string]string{"Tokyo": "cloudy", "Paris": "rainy"}
			return weatherOutput{
				Temperature: temps[input.Location],
				Condition:   conds[input.Location],
			}, nil
		},
	)

	result := stream.Stream(ctx, model,
		"What's the weather like in Tokyo and Paris?",
		stream.Tools(tool.ToDefinition(weatherTool)),
		stream.MaxTokens(150),
	)

	var toolCallParts []provider.StreamToolCallPart
	var assistantContent []provider.Content

	for part := range result.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			assistantContent = append(assistantContent, provider.TextContent{Text: p.Delta})
		case provider.StreamToolCallPart:
			toolCallParts = append(toolCallParts, p)
			assistantContent = append(assistantContent, provider.ToolCallContent{
				ID:    p.ToolCall.ID,
				Name:  p.ToolCall.Name,
				Input: p.ToolCall.Input,
			})
		case provider.StreamErrorPart:
			t.Fatalf("Stream error: %v", p.Error)
		}
	}

	if len(toolCallParts) == 0 {
		t.Fatal("Expected tool call parts in first stream")
	}

	t.Logf("Received %d tool calls", len(toolCallParts))

	var toolResults []provider.Content
	for _, tc := range toolCallParts {
		output, err := weatherTool.Execute(ctx, tc.ToolCall.Input)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		var input weatherInput
		json.Unmarshal(tc.ToolCall.Input, &input)
		t.Logf("Tool: %s for %s -> %s", tc.ToolCall.Name, input.Location, string(output))

		toolResults = append(toolResults,
			provider.NewToolResultContent(tc.ToolCall.ID, tc.ToolCall.Name, output, false),
		)
	}

	messages := []provider.Message{
		provider.UserMessage("What's the weather like in Tokyo and Paris?"),
		{Role: provider.RoleAssistant, Content: assistantContent},
	}

	for _, tr := range toolResults {
		messages = append(messages, provider.ToolResultMessage(tr.(provider.ToolResultContent)))
	}

	result2 := stream.Stream(ctx, model, "",
		stream.Messages(messages...),
		stream.Tools(tool.ToDefinition(weatherTool)),
		stream.MaxTokens(150),
	)

	var finalTextParts []string
	for part := range result2.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			finalTextParts = append(finalTextParts, p.Delta)
		case provider.StreamErrorPart:
			t.Fatalf("Second stream error: %v", p.Error)
		}
	}

	finalText := strings.Join(finalTextParts, "")
	if finalText == "" {
		t.Error("Expected non-empty final text")
	}

	t.Logf("Final response: %s", finalText)
}

func TestOpenAI_MultiTurnWithToolsStreaming(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), 2*getTimeout())
	defer cancel()

	calculatorTool := tool.NewTool("calculator", "Perform arithmetic",
		func(ctx context.Context, input calculatorInput) (calculatorOutput, error) {
			return calculatorOutput{Result: input.A + input.B}, nil
		},
	)

	var conversation []provider.Message

	result1 := stream.Stream(ctx, model, "My name is Alice.",
		stream.MaxTokens(50),
	)

	var text1Parts []string
	for part := range result1.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			text1Parts = append(text1Parts, p.Delta)
		case provider.StreamErrorPart:
			t.Fatalf("Turn 1 stream error: %v", p.Error)
		}
	}

	text1 := strings.Join(text1Parts, "")
	t.Logf("Turn 1 response: %s", text1)

	conversation = append(conversation,
		provider.UserMessage("My name is Alice."),
		provider.AssistantMessage(text1),
	)

	conversation = append(conversation, provider.UserMessage("What is 10 + 20?"))

	result2 := stream.Stream(ctx, model, "",
		stream.Messages(conversation...),
		stream.Tools(tool.ToDefinition(calculatorTool)),
		stream.MaxTokens(100),
	)

	var turn2Content []provider.Content
	var toolCalls []provider.ToolCall

	for part := range result2.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			turn2Content = append(turn2Content, provider.TextContent{Text: p.Delta})
		case provider.StreamToolCallPart:
			toolCalls = append(toolCalls, p.ToolCall)
			turn2Content = append(turn2Content, provider.ToolCallContent{
				ID:    p.ToolCall.ID,
				Name:  p.ToolCall.Name,
				Input: p.ToolCall.Input,
			})
		case provider.StreamErrorPart:
			t.Fatalf("Turn 2 stream error: %v", p.Error)
		}
	}

	if len(toolCalls) > 0 {
		t.Logf("Turn 2: Tool call detected")

		var toolResults []provider.Content
		for _, tc := range toolCalls {
			output, err := calculatorTool.Execute(ctx, tc.Input)
			if err != nil {
				t.Fatalf("Tool execution failed: %v", err)
			}

			t.Logf("Calculator result: %s", string(output))
			toolResults = append(toolResults,
				provider.NewToolResultContent(tc.ID, tc.Name, output, false),
			)
		}

		conversation = append(conversation,
			provider.Message{Role: provider.RoleAssistant, Content: turn2Content},
		)

		for _, tr := range toolResults {
			conversation = append(conversation, provider.ToolResultMessage(tr.(provider.ToolResultContent)))
		}

		result3 := stream.Stream(ctx, model, "",
			stream.Messages(conversation...),
			stream.Tools(tool.ToDefinition(calculatorTool)),
			stream.MaxTokens(100),
		)

		var text3Parts []string
		for part := range result3.Stream() {
			switch p := part.(type) {
			case provider.StreamTextPart:
				text3Parts = append(text3Parts, p.Delta)
			case provider.StreamErrorPart:
				t.Fatalf("Turn 3 stream error: %v", p.Error)
			}
		}

		turn2FinalText := strings.Join(text3Parts, "")
		t.Logf("Turn 2 (after tool): %s", turn2FinalText)

		conversation = append(conversation, provider.AssistantMessage(turn2FinalText))
	} else {
		var turn2Text string
		for _, c := range turn2Content {
			if tc, ok := c.(provider.TextContent); ok {
				turn2Text += tc.Text
			}
		}
		t.Logf("Turn 2: No tool call, direct response: %s", turn2Text)
		conversation = append(conversation, provider.Message{Role: provider.RoleAssistant, Content: turn2Content})
	}

	conversation = append(conversation,
		provider.UserMessage("What was my name and what was the calculation result?"),
	)

	result4 := stream.Stream(ctx, model, "",
		stream.Messages(conversation...),
		stream.MaxTokens(100),
	)

	var text4Parts []string
	for part := range result4.Stream() {
		switch p := part.(type) {
		case provider.StreamTextPart:
			text4Parts = append(text4Parts, p.Delta)
		case provider.StreamErrorPart:
			t.Fatalf("Turn 4 stream error: %v", p.Error)
		}
	}

	finalText := strings.Join(text4Parts, "")
	if finalText == "" {
		t.Error("Expected non-empty final response")
	}

	finalLower := strings.ToLower(finalText)
	hasName := strings.Contains(finalLower, "alice")
	hasResult := strings.Contains(finalLower, "30") || strings.Contains(finalLower, "thirty")

	t.Logf("Final response: %s", finalText)
	t.Logf("Contains name 'Alice': %v, Contains result '30': %v", hasName, hasResult)
}

func TestOpenAI_ToolsWithStructuredOutputAndMiddleware(t *testing.T) {
	apiKey := getAPIKey(t)

	var requestCount int
	loggingMiddleware := middleware.Middleware(func(next middleware.Handler) middleware.Handler {
		return middleware.HandlerFunc(func(ctx context.Context, req middleware.Request) (middleware.Response, error) {
			requestCount++
			t.Logf("[Middleware] Request #%d: provider=%s, model=%s", requestCount, req.Provider(), req.ModelID())
			return next.Handle(ctx, req)
		})
	})

	timeoutMiddleware := middleware.TimeoutWithDefault(60 * time.Second)

	client := openai.NewProvider(
		openai.APIKey(apiKey),
		openai.WithMiddleware(loggingMiddleware, timeoutMiddleware),
	)
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	calculatorTool := tool.NewTool("calculator", "Perform arithmetic",
		func(ctx context.Context, input calculatorInput) (calculatorOutput, error) {
			return calculatorOutput{Result: input.A + input.B}, nil
		},
	)

	result, err := generate.Generate(ctx, model,
		"Calculate 25 + 17 using the calculator tool.",
		generate.Tools(tool.ToDefinition(calculatorTool)),
		generate.MaxTokens(100),
	)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	toolCalls := result.ToolCalls()
	if len(toolCalls) > 0 {
		t.Logf("Tool call detected, executing...")
		output, err := calculatorTool.Execute(ctx, toolCalls[0].Input)
		if err != nil {
			t.Fatalf("Tool execution failed: %v", err)
		}

		var calcResult calculatorOutput
		json.Unmarshal(output, &calcResult)
		t.Logf("Calculator result: %d", calcResult.Result)
	}

	type CalculationResult struct {
		Operation string `json:"operation" jsonschema:"required,description=The operation performed"`
		Result    int    `json:"result" jsonschema:"required,description=The result of the calculation"`
	}

	structuredResult, err := generate.GenerateObject[CalculationResult](ctx, model,
		"Summarize the calculation 25 + 17 = 42 as a structured result.",
		generate.MaxTokens(50),
	)
	if err != nil {
		t.Fatalf("GenerateObject failed: %v", err)
	}

	if structuredResult.Object.Operation == "" {
		t.Error("Expected non-empty operation")
	}
	if structuredResult.Object.Result != 42 {
		t.Logf("Warning: expected result 42, got %d", structuredResult.Object.Result)
	}

	t.Logf("Structured result: %+v", structuredResult.Object)
	t.Logf("Middleware was invoked %d times", requestCount)

	if requestCount < 2 {
		t.Errorf("Expected at least 2 middleware invocations, got %d", requestCount)
	}
}

func TestOpenAI_MultiToolExecution(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.GPT4o()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	type weatherInput struct {
		Location string `json:"location" jsonschema:"required,description=City name"`
	}
	type weatherOutput struct {
		Temperature float64 `json:"temperature"`
		Condition   string  `json:"condition"`
	}

	weatherTool := tool.NewTool("get_weather", "Get weather for a location",
		func(ctx context.Context, input weatherInput) (weatherOutput, error) {
			return weatherOutput{Temperature: 23.0, Condition: "sunny"}, nil
		},
	)

	type timeInput struct {
		Timezone string `json:"timezone" jsonschema:"required,description=Timezone like JST or EST"`
	}
	type timeOutput struct {
		Time string `json:"time"`
	}

	timeTool := tool.NewTool("get_time", "Get current time in a timezone",
		func(ctx context.Context, input timeInput) (timeOutput, error) {
			return timeOutput{Time: "15:30"}, nil
		},
	)

	calcTool := tool.NewTool("calculator", "Perform arithmetic",
		func(ctx context.Context, input calculatorInput) (calculatorOutput, error) {
			return calculatorOutput{Result: input.A / input.B}, nil
		},
	)

	result, err := generate.Generate(ctx, model,
		"Get the weather in Tokyo, the time in JST, and calculate 100 / 4.",
		generate.Tools(
			tool.ToDefinition(weatherTool),
			tool.ToDefinition(timeTool),
			tool.ToDefinition(calcTool),
		),
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

	t.Logf("Received %d tool calls", len(toolCalls))

	toolNames := make(map[string]bool)
	for _, tc := range toolCalls {
		toolNames[tc.Name] = true
	}

	var toolResults []provider.Content
	for _, tc := range toolCalls {
		var output json.RawMessage
		var err error

		switch tc.Name {
		case "get_weather":
			output, err = weatherTool.Execute(ctx, tc.Input)
		case "get_time":
			output, err = timeTool.Execute(ctx, tc.Input)
		case "calculator":
			output, err = calcTool.Execute(ctx, tc.Input)
		default:
			t.Logf("Unknown tool: %s", tc.Name)
			continue
		}

		if err != nil {
			t.Fatalf("Tool %s execution failed: %v", tc.Name, err)
		}

		t.Logf("Tool %s executed: %s", tc.Name, string(output))
		toolResults = append(toolResults,
			provider.NewToolResultContent(tc.ID, tc.Name, output, false),
		)
	}

	if len(toolResults) < 2 {
		t.Logf("Warning: expected at least 2 different tool calls, got %d", len(toolResults))
	}

	messages := []provider.Message{
		provider.UserMessage("Get the weather in Tokyo, the time in JST, and calculate 100 / 4."),
		{Role: provider.RoleAssistant, Content: result.Content()},
	}

	for _, tr := range toolResults {
		messages = append(messages, provider.ToolResultMessage(tr.(provider.ToolResultContent)))
	}

	result2, err := generate.Generate(ctx, model, "",
		generate.Messages(messages...),
		generate.Tools(
			tool.ToDefinition(weatherTool),
			tool.ToDefinition(timeTool),
			tool.ToDefinition(calcTool),
		),
		generate.MaxTokens(200),
	)
	if err != nil {
		t.Fatalf("Second generate failed: %v", err)
	}

	finalText := result2.Text()
	if finalText == "" {
		t.Error("Expected non-empty final text")
	}

	t.Logf("Final response: %s", finalText)
	t.Logf("Tools called: %v", toolNames)
}

func TestOpenAI_Transcription(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.Whisper1()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	audioData := []byte("fake audio data - in real test, this would be actual audio bytes")
	result, err := model.Transcribe(ctx, &provider.TranscriptionRequest{
		Audio:     audioData,
		MediaType: "audio/mp3",
		Language:  "en",
	})
	if err != nil {
		var providerErr *provider.Error
		if errors.As(err, &providerErr) {
			if providerErr.Code == provider.CodeInvalidRequest || providerErr.Code == provider.CodeParseError {
				t.Skipf("Skipping: %v", err)
			}
		}
		t.Fatalf("Transcribe failed: %v", err)
	}

	if result.Text == "" {
		t.Error("Expected non-empty transcription text")
	}

	t.Logf("Transcription: %s", result.Text)
	if result.Language != "" {
		t.Logf("Language: %s", result.Language)
	}
	if result.Duration > 0 {
		t.Logf("Duration: %.2f seconds", result.Duration)
	}
	if len(result.Segments) > 0 {
		t.Logf("Segments: %d", len(result.Segments))
	}
}

func TestOpenAI_Speech(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TTS1()

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := model.Synthesize(ctx, &provider.SpeechRequest{
		Text:  "Hello, this is a test of the text to speech API.",
		Voice: "alloy",
	})
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}

	if len(result.Audio) == 0 {
		t.Error("Expected non-empty audio data")
	}
	if result.MediaType == "" {
		t.Error("Expected media type to be set")
	}

	t.Logf("Audio size: %d bytes", len(result.Audio))
	t.Logf("Media type: %s", result.MediaType)
}

func TestOpenAI_SpeechWithFormat(t *testing.T) {
	apiKey := getAPIKey(t)
	client := openai.NewProvider(openai.APIKey(apiKey))
	model := client.TTS1HD(openai.SpeechFormat("mp3"))

	ctx, cancel := context.WithTimeout(context.Background(), getTimeout())
	defer cancel()

	result, err := model.Synthesize(ctx, &provider.SpeechRequest{
		Text:   "Testing different audio formats.",
		Voice:  "nova",
		Format: "opus",
	})
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}

	if len(result.Audio) == 0 {
		t.Error("Expected non-empty audio data")
	}
	if result.MediaType != "audio/opus" {
		t.Errorf("Media type = %q, want audio/opus", result.MediaType)
	}

	t.Logf("Audio size: %d bytes", len(result.Audio))
	t.Logf("Media type: %s", result.MediaType)
}
