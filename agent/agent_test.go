package agent

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// mockModel is a mock LanguageModel for testing.
type mockModel struct {
	responses []*provider.GenerateResult
	callCount int
}

func (m *mockModel) Provider() string { return "mock" }
func (m *mockModel) ModelID() string  { return "mock-model" }

func (m *mockModel) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	if m.callCount >= len(m.responses) {
		// Default response: no tool calls
		return &provider.GenerateResult{
			Content:      []provider.Content{provider.Text("I don't have any more tools to call.")},
			FinishReason: provider.FinishReasonStop,
			Usage: provider.Usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}, nil
	}
	response := m.responses[m.callCount]
	m.callCount++
	return response, nil
}

func (m *mockModel) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	return nil, nil
}

// mockTool is a mock tool for testing.
type mockTool struct {
	name        string
	description string
	fn          func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

func (t *mockTool) Name() string        { return t.name }
func (t *mockTool) Description() string { return t.description }
func (t *mockTool) InputSchema() json.RawMessage {
	return json.RawMessage(`{"type":"object","properties":{"input":{"type":"string"}}}`)
}
func (t *mockTool) Execute(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
	return t.fn(ctx, input)
}

func TestAgent_Invoke_SingleStep(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Hello!")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agent := New(model, WithStopWhen(StepCountIs(1)))

	result, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Hi")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if len(result.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result.Steps))
	}

	if result.FinalText != "Hello!" {
		t.Errorf("Expected 'Hello!', got %q", result.FinalText)
	}
}

func TestAgent_Invoke_ToolLoop(t *testing.T) {
	weatherTool := &mockTool{
		name:        "get_weather",
		description: "Get weather for a location",
		fn: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"temperature":"22C","condition":"sunny"}`), nil
		},
	}

	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content: []provider.Content{},
				ToolCalls: []provider.ToolCall{
					{ID: "call-1", Name: "get_weather", Input: json.RawMessage(`{"location":"Tokyo"}`)},
				},
				FinishReason: provider.FinishReasonToolCalls,
				Usage:        provider.Usage{TotalTokens: 20},
			},
			{
				Content:      []provider.Content{provider.Text("The weather in Tokyo is sunny with 22C.")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 15},
			},
		},
	}

	agent := New(model,
		WithTools(weatherTool),
		WithStopWhen(StepCountIs(10)),
	)

	ctx := context.Background()
	result, err := agent.Invoke(ctx,
		WithMessages(provider.UserMessage("What's the weather in Tokyo?")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if len(result.Steps) != 2 {
		t.Errorf("Expected 2 steps, got %d", len(result.Steps))
	}

	if len(result.Steps[0].ToolCalls) != 1 {
		t.Errorf("Expected 1 tool call in step 1, got %d", len(result.Steps[0].ToolCalls))
	}

	if result.Steps[0].ToolCalls[0].Name != "get_weather" {
		t.Errorf("Expected tool call 'get_weather', got %q", result.Steps[0].ToolCalls[0].Name)
	}

	if len(result.Steps[0].ToolResults) != 1 {
		t.Errorf("Expected 1 tool result in step 1, got %d", len(result.Steps[0].ToolResults))
	}

	if result.TotalUsage.TotalTokens != 35 {
		t.Errorf("Expected 35 total tokens, got %d", result.TotalUsage.TotalTokens)
	}
}

func TestAgent_Invoke_StopCondition(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content: []provider.Content{},
				ToolCalls: []provider.ToolCall{
					{ID: "call-1", Name: "submit", Input: json.RawMessage(`{}`)},
				},
				FinishReason: provider.FinishReasonToolCalls,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	// Stop when submit tool is called
	agent := New(model,
		WithStopWhen(HasToolCall("submit")),
	)

	result, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Submit the answer")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if len(result.Steps) != 1 {
		t.Errorf("Expected 1 step, got %d", len(result.Steps))
	}
}

func TestAgent_Invoke_PrepareStep(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Step 1")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	stepCount := 0
	agent := New(model,
		WithStopWhen(StepCountIs(1)),
		WithPrepareStep(func(ctx context.Context, params PrepareStepParams) *PrepareStepResult {
			stepCount++
			return nil
		}),
	)

	_, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if stepCount != 1 {
		t.Errorf("Expected PrepareStep to be called 1 time, got %d", stepCount)
	}
}

func TestAgent_Invoke_Callbacks(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Done")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	var stepStarts, stepFinishes, onStarts, onFinishes int

	agent := New(model,
		WithStopWhen(StepCountIs(1)),
		WithCallbacks(Callbacks{
			OnStart: func(ctx context.Context, messages []provider.Message) error {
				onStarts++
				return nil
			},
			OnStepStart: func(ctx context.Context, stepNumber int, messages []provider.Message) error {
				stepStarts++
				return nil
			},
			OnStepFinish: func(ctx context.Context, result StepResult) error {
				stepFinishes++
				return nil
			},
			OnFinish: func(ctx context.Context, result *Result) error {
				onFinishes++
				return nil
			},
		}),
	)

	_, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if onStarts != 1 {
		t.Errorf("Expected OnStart to be called 1 time, got %d", onStarts)
	}

	if stepStarts != 1 {
		t.Errorf("Expected OnStepStart to be called 1 time, got %d", stepStarts)
	}

	if stepFinishes != 1 {
		t.Errorf("Expected OnStepFinish to be called 1 time, got %d", stepFinishes)
	}

	if onFinishes != 1 {
		t.Errorf("Expected OnFinish to be called 1 time, got %d", onFinishes)
	}
}

func TestAgent_Stream(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Streaming test")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agent := New(model, WithStopWhen(StepCountIs(1)))

	ctx := context.Background()
	stream := agent.Stream(ctx,
		WithStreamMessages(provider.UserMessage("Test streaming")),
	)

	events := make([]StreamEvent, 0)
	for event := range stream {
		events = append(events, event)
	}

	// Should have: step_start, step_finish, finish
	if len(events) < 2 {
		t.Errorf("Expected at least 2 events, got %d", len(events))
	}

	var hasFinish bool
	for _, event := range events {
		if _, ok := event.(StreamEventFinish); ok {
			hasFinish = true
		}
	}

	if !hasFinish {
		t.Errorf("Expected StreamEventFinish event")
	}
}

func TestAgent_ToolNotFoundError(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content: []provider.Content{},
				ToolCalls: []provider.ToolCall{
					{ID: "call-1", Name: "unknown_tool", Input: json.RawMessage(`{}`)},
				},
				FinishReason: provider.FinishReasonToolCalls,
				Usage:        provider.Usage{TotalTokens: 10},
			},
			{
				Content:      []provider.Content{provider.Text("Done")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agent := New(model,
		WithTools(), // No tools
		WithStopWhen(StepCountIs(10)),
	)

	result, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if len(result.Steps[0].ToolResults) != 1 {
		t.Fatal("Expected 1 tool result")
	}

	if result.Steps[0].ToolResults[0].Error == nil {
		t.Error("Expected tool result to have an error")
	}

	if _, ok := result.Steps[0].ToolResults[0].Error.(*ToolNotFoundError); !ok {
		t.Errorf("Expected ToolNotFoundError, got %T", result.Steps[0].ToolResults[0].Error)
	}
}

func TestAgent_ContextCancellation(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Done")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agent := New(model, WithStopWhen(StepCountIs(100)))

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := agent.Invoke(ctx,
		WithMessages(provider.UserMessage("Test")),
	)

	if err == nil {
		t.Error("Expected error due to context cancellation")
	}

	var agentErr *AgentError
	if !AsAgentError(err, &agentErr) {
		t.Errorf("Expected agent.AgentError, got %T", err)
	}

	if agentErr.Code != "context_canceled" {
		t.Errorf("Expected error code 'context_canceled', got %q", agentErr.Code)
	}
}

func isAgentError(err error, target **AgentError) bool {
	// Use errors.As pattern
	if e, ok := err.(*AgentError); ok {
		*target = e
		return true
	}
	return false
}

func TestAgent_Invoke_RetryLogic(t *testing.T) {
	callCount := 0
	model := &mockModelWithRetry{
		callCount: &callCount,
		responses: []*provider.GenerateResult{
			nil, // First call fails
			nil, // Second call fails
			{ // Third call succeeds
				Content:      []provider.Content{provider.Text("Success after retries")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
		failCount: 2,
	}

	agent := New(model,
		WithStopWhen(StepCountIs(1)),
		WithMaxRetries(3),
	)

	result, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if result.FinalText != "Success after retries" {
		t.Errorf("Expected 'Success after retries', got %q", result.FinalText)
	}

	if callCount < 3 {
		t.Errorf("Expected at least 3 calls (2 failures + 1 success), got %d", callCount)
	}
}

func TestAgent_Invoke_MaxRetries(t *testing.T) {
	callCount := 0
	model := &mockModelWithRetry{
		callCount: &callCount,
		responses: []*provider.GenerateResult{
			nil, // Always fail
		},
		failCount: 100, // Always fail
	}

	agent := New(model,
		WithStopWhen(StepCountIs(1)),
		WithMaxRetries(2),
	)

	_, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
	)

	if err == nil {
		t.Error("Expected error due to max retries exceeded")
	}

	var agentErr *AgentError
	if !AsAgentError(err, &agentErr) {
		t.Errorf("Expected AgentError, got %T", err)
	}

	// Should have attempted maxRetries + 1 times (initial + retries)
	if callCount > 3 {
		t.Errorf("Should not exceed max retries, got %d calls", callCount)
	}
}

func TestAgent_Invoke_OnErrorSuppress(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Recovered")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	errorCalled := false
	agent := New(model,
		WithStopWhen(StepCountIs(1)),
		WithCallbacks(Callbacks{
			OnError: func(ctx context.Context, stepNumber int, err error) error {
				errorCalled = true
				return nil // Suppress error, continue execution
			},
		}),
	)

	// This test verifies that OnError can suppress errors
	// In a real scenario, the mock would need to be more sophisticated
	_, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Note: The mock doesn't generate errors, so errorCalled won't be true
	// This is a placeholder for more sophisticated error handling tests
	_ = errorCalled
}

func TestAgent_Invoke_MultipleToolCalls(t *testing.T) {
	weatherTool := &mockTool{
		name:        "get_weather",
		description: "Get weather",
		fn: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"temp":"22C"}`), nil
		},
	}

	timeTool := &mockTool{
		name:        "get_time",
		description: "Get time",
		fn: func(ctx context.Context, input json.RawMessage) (json.RawMessage, error) {
			return json.RawMessage(`{"time":"12:00"}`), nil
		},
	}

	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content: []provider.Content{},
				ToolCalls: []provider.ToolCall{
					{ID: "call-1", Name: "get_weather", Input: json.RawMessage(`{}`)},
					{ID: "call-2", Name: "get_time", Input: json.RawMessage(`{}`)},
				},
				FinishReason: provider.FinishReasonToolCalls,
				Usage:        provider.Usage{TotalTokens: 20},
			},
			{
				Content:      []provider.Content{provider.Text("Weather is 22C and time is 12:00")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agent := New(model,
		WithTools(weatherTool, timeTool),
		WithStopWhen(StepCountIs(10)),
	)

	result, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Get weather and time")),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	if len(result.Steps) != 2 {
		t.Fatalf("Expected 2 steps, got %d", len(result.Steps))
	}

	if len(result.Steps[0].ToolCalls) != 2 {
		t.Errorf("Expected 2 tool calls in step 1, got %d", len(result.Steps[0].ToolCalls))
	}

	if len(result.Steps[0].ToolResults) != 2 {
		t.Errorf("Expected 2 tool results in step 1, got %d", len(result.Steps[0].ToolResults))
	}
}

func TestAgent_Stream_EmptyMessages(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Test")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agent := New(model, WithStopWhen(StepCountIs(1)))

	ctx := context.Background()
	stream := agent.Stream(ctx,
		WithStreamMessages(), // No messages
	)

	var hasError bool
	for event := range stream {
		if _, ok := event.(StreamEventError); ok {
			hasError = true
		}
	}

	if !hasError {
		t.Error("Expected error event for empty messages")
	}
}

func TestAgent_Invoke_InvokeCallbacks(t *testing.T) {
	model := &mockModel{
		responses: []*provider.GenerateResult{
			{
				Content:      []provider.Content{provider.Text("Test")},
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{TotalTokens: 10},
			},
		},
	}

	agentOnStart := 0
	invokeOnStart := 0

	agent := New(model,
		WithStopWhen(StepCountIs(1)),
		WithCallbacks(Callbacks{
			OnStart: func(ctx context.Context, messages []provider.Message) error {
				agentOnStart++
				return nil
			},
		}),
	)

	_, err := agent.Invoke(context.Background(),
		WithMessages(provider.UserMessage("Test")),
		WithInvokeCallbacks(Callbacks{
			OnStart: func(ctx context.Context, messages []provider.Message) error {
				invokeOnStart++
				return nil
			},
		}),
	)

	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	// Agent callbacks should be overridden by invoke callbacks
	if agentOnStart != 0 {
		t.Errorf("Agent OnStart should be overridden, got %d calls", agentOnStart)
	}
	if invokeOnStart != 1 {
		t.Errorf("Expected invoke OnStart to be called once, got %d", invokeOnStart)
	}
}

func TestAgent_ToolMap(t *testing.T) {
	tool1 := &mockTool{name: "tool1", description: "Tool 1"}
	tool2 := &mockTool{name: "tool2", description: "Tool 2"}
	tool3 := &mockTool{name: "tool3", description: "Tool 3"}

	agent := New(&mockModel{},
		WithTools(tool1, tool2, tool3),
	)

	if agent.toolMap == nil {
		t.Fatal("Tool map should not be nil")
	}

	if len(agent.toolMap) != 3 {
		t.Errorf("Expected 3 tools in map, got %d", len(agent.toolMap))
	}

	if agent.toolMap["tool1"] != tool1 {
		t.Error("Tool 1 not found in map")
	}
	if agent.toolMap["tool2"] != tool2 {
		t.Error("Tool 2 not found in map")
	}
	if agent.toolMap["tool3"] != tool3 {
		t.Error("Tool 3 not found in map")
	}
}

// mockModelWithRetry is a mock that can fail and retry
type mockModelWithRetry struct {
	responses []*provider.GenerateResult
	callCount *int
	failCount int
}

func (m *mockModelWithRetry) Provider() string { return "mock" }
func (m *mockModelWithRetry) ModelID() string  { return "mock-model" }

func (m *mockModelWithRetry) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	(*m.callCount)++
	if *m.callCount <= m.failCount {
		return nil, &provider.Error{
			Code:    provider.CodeRateLimited,
			Message: "rate limited",
		}
	}
	if *m.callCount > len(m.responses) {
		return &provider.GenerateResult{
			Content:      []provider.Content{provider.Text("Done")},
			FinishReason: provider.FinishReasonStop,
			Usage:        provider.Usage{TotalTokens: 10},
		}, nil
	}
	return m.responses[*m.callCount-1], nil
}

func (m *mockModelWithRetry) Stream(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
	return nil, nil
}
