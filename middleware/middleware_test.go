package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type mockRequest struct {
	provider string
	modelID  string
}

func (r *mockRequest) Provider() string { return r.provider }
func (r *mockRequest) ModelID() string  { return r.modelID }
func (r *mockRequest) InputCount() int  { return 1 }

type mockResponse struct {
	usage provider.Usage
}

func (r *mockResponse) Usage() provider.Usage               { return r.usage }
func (r *mockResponse) FinishReason() provider.FinishReason { return provider.FinishReasonStop }

type mockHandler struct {
	handleFunc func(ctx context.Context, req Request) (Response, error)
}

func (h *mockHandler) Handle(ctx context.Context, req Request) (Response, error) {
	if h.handleFunc != nil {
		return h.handleFunc(ctx, req)
	}
	return &mockResponse{usage: provider.Usage{TotalTokens: 100}}, nil
}

func TestChainOrder(t *testing.T) {
	var order []string

	m1 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			order = append(order, "m1-before")
			resp, err := next.Handle(ctx, req)
			order = append(order, "m1-after")
			return resp, err
		})
	}

	m2 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			order = append(order, "m2-before")
			resp, err := next.Handle(ctx, req)
			order = append(order, "m2-after")
			return resp, err
		})
	}

	m3 := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			order = append(order, "m3-before")
			resp, err := next.Handle(ctx, req)
			order = append(order, "m3-after")
			return resp, err
		})
	}

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		order = append(order, "base")
		return &mockResponse{}, nil
	})

	handler := Chain(m1, m2, m3)(base)
	_, _ = handler.Handle(context.Background(), &mockRequest{provider: "test", modelID: "model"})

	expected := []string{
		"m1-before", "m2-before", "m3-before",
		"base",
		"m3-after", "m2-after", "m1-after",
	}
	if len(order) != len(expected) {
		t.Errorf("order length = %d, want %d", len(order), len(expected))
	}
	for i, v := range expected {
		if i >= len(order) || order[i] != v {
			t.Errorf("order[%d] = %q, want %q", i, order[i], v)
		}
	}
}

func TestChainSingleMiddleware(t *testing.T) {
	called := false
	m := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			called = true
			return next.Handle(ctx, req)
		})
	}

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		return &mockResponse{}, nil
	})

	handler := Chain(m)(base)
	_, _ = handler.Handle(context.Background(), &mockRequest{})

	if !called {
		t.Error("middleware was not called")
	}
}

func TestChainEmpty(t *testing.T) {
	called := false
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		called = true
		return &mockResponse{}, nil
	})

	handler := Chain()(base)
	_, _ = handler.Handle(context.Background(), &mockRequest{})

	if !called {
		t.Error("base handler was not called")
	}
}

func TestHandlerFunc(t *testing.T) {
	called := false
	h := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		called = true
		return &mockResponse{usage: provider.Usage{TotalTokens: 50}}, nil
	})

	resp, err := h.Handle(context.Background(), &mockRequest{})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
	if resp.Usage().TotalTokens != 50 {
		t.Errorf("usage = %d, want 50", resp.Usage().TotalTokens)
	}
}

func TestLoggingSuccess(t *testing.T) {
	logs := &testLogCollector{}
	logger := &testLogger{logs: logs}

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		return &mockResponse{usage: provider.Usage{TotalTokens: 100}}, nil
	})

	handler := Logging(logger)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{provider: "openai", modelID: "gpt-4"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(logs.debugs) != 2 {
		t.Errorf("expected 2 debug logs, got %d", len(logs.debugs))
	}
}

func TestLoggingError(t *testing.T) {
	logs := &testLogCollector{}
	logger := &testLogger{logs: logs}
	expectedErr := errors.New("test error")

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		return nil, expectedErr
	})

	handler := Logging(logger)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{provider: "openai", modelID: "gpt-4"})

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
	if len(logs.errors) != 1 {
		t.Errorf("expected 1 error log, got %d", len(logs.errors))
	}
}

func TestMetricsSuccess(t *testing.T) {
	collector := &testMetricsCollector{}

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		return &mockResponse{usage: provider.Usage{PromptTokens: 50, CompletionTokens: 100, TotalTokens: 150}}, nil
	})

	handler := Metrics(collector)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{provider: "openai", modelID: "gpt-4"})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(collector.requests) != 1 {
		t.Errorf("expected 1 request recorded, got %d", len(collector.requests))
	}
	if len(collector.tokens) != 1 {
		t.Errorf("expected 1 token record, got %d", len(collector.tokens))
	}
	if collector.tokens[0].prompt != 50 || collector.tokens[0].completion != 100 {
		t.Errorf("tokens = prompt:%d,completion:%d, want prompt:50,completion:100", collector.tokens[0].prompt, collector.tokens[0].completion)
	}
}

func TestMetricsError(t *testing.T) {
	collector := &testMetricsCollector{}
	expectedErr := errors.New("test error")

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		return nil, expectedErr
	})

	handler := Metrics(collector)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{provider: "openai", modelID: "gpt-4"})

	if err != expectedErr {
		t.Errorf("error = %v, want %v", err, expectedErr)
	}
	if len(collector.requests) != 1 {
		t.Errorf("expected 1 request recorded, got %d", len(collector.requests))
	}
	if collector.requests[0].err == nil {
		t.Error("expected error to be recorded")
	}
	if len(collector.tokens) != 0 {
		t.Errorf("expected 0 token records on error, got %d", len(collector.tokens))
	}
}

func TestRecoverNoPanic(t *testing.T) {
	called := false
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		called = true
		return &mockResponse{}, nil
	})

	handler := Recover()(base)
	_, err := handler.Handle(context.Background(), &mockRequest{})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("handler was not called")
	}
}

func TestRecoverWithPanic(t *testing.T) {
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		panic("something went wrong")
	})

	handler := Recover()(base)
	resp, err := handler.Handle(context.Background(), &mockRequest{})

	if resp != nil {
		t.Error("expected nil response on panic")
	}
	if err == nil {
		t.Fatal("expected error on panic")
	}
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("error type = %T, want *provider.Error", err)
	}
	if providerErr.Code != provider.CodeUnknown {
		t.Errorf("error code = %v, want CodeUnknown", providerErr.Code)
	}
}

func TestGenerateRequest(t *testing.T) {
	req := &GenerateRequest{
		ProviderName: "openai",
		Model:        "gpt-4",
		Prompt:       "Hello",
		Config:       provider.Config{MaxTokens: 100},
	}

	if req.Provider() != "openai" {
		t.Errorf("Provider() = %q, want %q", req.Provider(), "openai")
	}
	if req.ModelID() != "gpt-4" {
		t.Errorf("ModelID() = %q, want %q", req.ModelID(), "gpt-4")
	}
}

func TestGenerateResponse(t *testing.T) {
	resp := &GenerateResponse{
		Text:             "Hello",
		FinishReasonData: provider.FinishReasonStop,
		UsageData:        provider.Usage{TotalTokens: 50},
	}

	if resp.Usage().TotalTokens != 50 {
		t.Errorf("Usage().TotalTokens = %d, want 50", resp.Usage().TotalTokens)
	}
}

func TestEmbedRequest(t *testing.T) {
	req := &EmbedRequest{
		ProviderName: "openai",
		Model:        "text-embedding-3-small",
		Texts:        []string{"Hello", "World"},
	}

	if req.Provider() != "openai" {
		t.Errorf("Provider() = %q, want %q", req.Provider(), "openai")
	}
	if req.ModelID() != "text-embedding-3-small" {
		t.Errorf("ModelID() = %q, want %q", req.ModelID(), "text-embedding-3-small")
	}
}

func TestEmbedResponse(t *testing.T) {
	resp := &EmbedResponse{
		Embeddings: [][]float64{{0.1, 0.2}, {0.3, 0.4}},
		UsageData:  provider.Usage{TotalTokens: 20},
	}

	if resp.Usage().TotalTokens != 20 {
		t.Errorf("Usage().TotalTokens = %d, want 20", resp.Usage().TotalTokens)
	}
}

type testLogCollector struct {
	debugs []string
	infos  []string
	warns  []string
	errors []string
}

type testLogger struct {
	logs *testLogCollector
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.logs.debugs = append(l.logs.debugs, msg)
}
func (l *testLogger) Info(msg string, args ...any) {
	l.logs.infos = append(l.logs.infos, msg)
}
func (l *testLogger) Warn(msg string, args ...any) {
	l.logs.warns = append(l.logs.warns, msg)
}
func (l *testLogger) Error(msg string, args ...any) {
	l.logs.errors = append(l.logs.errors, msg)
}

type testMetricsCollector struct {
	requests []struct {
		provider string
		model    string
		duration time.Duration
		err      error
	}
	tokens []struct {
		provider   string
		model      string
		prompt     int
		completion int
	}
}

func (m *testMetricsCollector) RecordRequest(ctx context.Context, provider, model string, duration time.Duration, err error) {
	m.requests = append(m.requests, struct {
		provider string
		model    string
		duration time.Duration
		err      error
	}{provider, model, duration, err})
}

func (m *testMetricsCollector) RecordTokens(ctx context.Context, provider, model string, prompt, completion int) {
	m.tokens = append(m.tokens, struct {
		provider   string
		model      string
		prompt     int
		completion int
	}{provider, model, prompt, completion})
}

func (m *testMetricsCollector) RecordStreamStart(ctx context.Context, provider, model string) {}
func (m *testMetricsCollector) RecordStreamEvent(ctx context.Context, provider, model, eventType string) {
}
func (m *testMetricsCollector) RecordStreamEnd(ctx context.Context, provider, model string, duration time.Duration) {
}
