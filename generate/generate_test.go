package generate

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type mockGenerator struct {
	provider     string
	modelID      string
	generateFunc func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error)
}

func (m *mockGenerator) Provider() string { return m.provider }
func (m *mockGenerator) ModelID() string  { return m.modelID }
func (m *mockGenerator) Generate(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
	if m.generateFunc != nil {
		return m.generateFunc(ctx, req)
	}
	return &provider.GenerateResult{
		Text:         "mock response",
		FinishReason: provider.FinishReasonStop,
		Usage:        provider.Usage{PromptTokens: 10, CompletionTokens: 5, TotalTokens: 15},
	}, nil
}

func TestGenerateNilModel(t *testing.T) {
	ctx := context.Background()
	_, err := Generate(ctx, nil, "test")
	if !errors.Is(err, provider.ErrInvalidModel) {
		t.Errorf("Generate() error = %v, want ErrInvalidModel", err)
	}
}

func TestGenerateEmptyPrompt(t *testing.T) {
	ctx := context.Background()
	model := &mockGenerator{provider: "test", modelID: "test-model"}
	_, err := Generate(ctx, model, "")
	if !errors.Is(err, provider.ErrInvalidPrompt) {
		t.Errorf("Generate() error = %v, want ErrInvalidPrompt", err)
	}
}

func TestGenerateEmptyPromptWithMessages(t *testing.T) {
	ctx := context.Background()
	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			return &provider.GenerateResult{Text: "response"}, nil
		},
	}
	result, err := Generate(ctx, model, "", Messages(provider.UserMessage("hello")))
	if err != nil {
		t.Errorf("Generate() error = %v, want nil", err)
	}
	if result.Text() != "response" {
		t.Errorf("Generate() text = %q, want %q", result.Text(), "response")
	}
}

func TestGenerateSuccess(t *testing.T) {
	ctx := context.Background()
	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			if req.Prompt != "hello" {
				t.Errorf("prompt = %q, want %q", req.Prompt, "hello")
			}
			return &provider.GenerateResult{
				Text:         "hi there",
				FinishReason: provider.FinishReasonStop,
				Usage:        provider.Usage{PromptTokens: 5, CompletionTokens: 10, TotalTokens: 15},
			}, nil
		},
	}

	result, err := Generate(ctx, model, "hello")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if result.Text() != "hi there" {
		t.Errorf("Text() = %q, want %q", result.Text(), "hi there")
	}
	if result.FinishReason() != provider.FinishReasonStop {
		t.Errorf("FinishReason() = %q, want %q", result.FinishReason(), provider.FinishReasonStop)
	}
	if result.Usage().TotalTokens != 15 {
		t.Errorf("Usage().TotalTokens = %d, want 15", result.Usage().TotalTokens)
	}
}

func TestGenerateWithOptions(t *testing.T) {
	ctx := context.Background()
	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			if req.System != "Be helpful" {
				t.Errorf("System = %q, want %q", req.System, "Be helpful")
			}
			if req.Config.MaxTokens != 100 {
				t.Errorf("MaxTokens = %d, want 100", req.Config.MaxTokens)
			}
			if req.Config.Temperature != 0.7 {
				t.Errorf("Temperature = %f, want 0.7", req.Config.Temperature)
			}
			return &provider.GenerateResult{Text: "ok"}, nil
		},
	}

	_, err := Generate(ctx, model, "test",
		System("Be helpful"),
		MaxTokens(100),
		Temperature(0.7),
	)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateError(t *testing.T) {
	ctx := context.Background()
	expectedErr := &provider.Error{Code: provider.CodeAPITimeout, Message: "timeout"}
	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			return nil, expectedErr
		},
	}

	_, err := Generate(ctx, model, "test")
	if err == nil {
		t.Fatal("Generate() error = nil, want error")
	}
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("Generate() error = %v, want *provider.Error", err)
	}
}

func TestGenerateContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &provider.GenerateResult{Text: "ok"}, nil
			}
		},
	}

	_, err := Generate(ctx, model, "test")
	if err == nil {
		t.Error("Generate() error = nil, want context canceled error")
	}
}

func TestGenerateDefaultTimeout(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(innerCtx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			capturedCtx = innerCtx
			return &provider.GenerateResult{Text: "ok"}, nil
		},
	}

	_, _ = Generate(ctx, model, "test")

	deadline, ok := capturedCtx.Deadline()
	if !ok {
		t.Error("Generate() should set a deadline when no context deadline exists")
	}
	expectedTimeout := time.Now().Add(DefaultTimeout)
	if deadline.After(expectedTimeout.Add(time.Second)) || deadline.Before(expectedTimeout.Add(-time.Second)) {
		t.Errorf("Deadline = %v, want approximately %v", deadline, expectedTimeout)
	}
}

func TestGenerateCustomTimeout(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(innerCtx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			capturedCtx = innerCtx
			return &provider.GenerateResult{Text: "ok"}, nil
		},
	}

	customTimeout := 5 * time.Minute
	_, _ = Generate(ctx, model, "test", WithTimeout(customTimeout))

	deadline, ok := capturedCtx.Deadline()
	if !ok {
		t.Error("Generate() should set a deadline for custom timeout")
	}
	expectedTimeout := time.Now().Add(customTimeout)
	if deadline.After(expectedTimeout.Add(time.Second)) || deadline.Before(expectedTimeout.Add(-time.Second)) {
		t.Errorf("Deadline = %v, want approximately %v", deadline, expectedTimeout)
	}
}

func TestGenerateNoTimeout(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(innerCtx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			capturedCtx = innerCtx
			return &provider.GenerateResult{Text: "ok"}, nil
		},
	}

	_, _ = Generate(ctx, model, "test", WithNoTimeout())

	_, ok := capturedCtx.Deadline()
	if ok {
		t.Error("Generate() should not set deadline when WithNoTimeout()")
	}
}

func TestGenerateExistingContextDeadline(t *testing.T) {
	existingDeadline := time.Now().Add(10 * time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), existingDeadline)
	defer cancel()

	var capturedCtx context.Context

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(innerCtx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			capturedCtx = innerCtx
			return &provider.GenerateResult{Text: "ok"}, nil
		},
	}

	_, _ = Generate(ctx, model, "test", WithTimeout(30*time.Second))

	capturedDeadline, ok := capturedCtx.Deadline()
	if !ok {
		t.Error("Context should have a deadline")
	}
	if capturedDeadline.After(existingDeadline.Add(time.Second)) {
		t.Errorf("Deadline should respect existing context deadline, got %v, want <= %v", capturedDeadline, existingDeadline)
	}
}

func TestResultNilSafety(t *testing.T) {
	var r *Result
	if r.Text() != "" {
		t.Error("Result.Text() on nil should return empty string")
	}
	if r.Content() != nil {
		t.Error("Result.Content() on nil should return nil")
	}
	if r.ToolCalls() != nil {
		t.Error("Result.ToolCalls() on nil should return nil")
	}
	if r.FinishReason() != "" {
		t.Error("Result.FinishReason() on nil should return empty string")
	}
	if r.Usage() != (provider.Usage{}) {
		t.Error("Result.Usage() on nil should return zero Usage")
	}
}

func TestGenerateWithMessages(t *testing.T) {
	ctx := context.Background()
	var capturedReq *provider.GenerateRequest

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(innerCtx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			capturedReq = req
			return &provider.GenerateResult{Text: "response"}, nil
		},
	}

	msgs := []provider.Message{
		provider.UserMessage("hello"),
		provider.AssistantMessage("hi!"),
		provider.UserMessage("how are you?"),
	}

	_, err := Generate(ctx, model, "", Messages(msgs...))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if len(capturedReq.Messages) != 3 {
		t.Errorf("Messages count = %d, want 3", len(capturedReq.Messages))
	}
}

func TestGenerateWithBothPromptAndMessages(t *testing.T) {
	ctx := context.Background()
	var capturedReq *provider.GenerateRequest

	model := &mockGenerator{
		provider: "test",
		modelID:  "test-model",
		generateFunc: func(innerCtx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
			capturedReq = req
			return &provider.GenerateResult{Text: "response"}, nil
		},
	}

	msgs := []provider.Message{provider.UserMessage("previous")}
	_, err := Generate(ctx, model, "current prompt", Messages(msgs...))
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	if capturedReq.Prompt != "current prompt" {
		t.Errorf("Prompt = %q, want %q", capturedReq.Prompt, "current prompt")
	}
	if len(capturedReq.Messages) != 1 {
		t.Errorf("Messages count = %d, want 1", len(capturedReq.Messages))
	}
}
