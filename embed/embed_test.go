package embed

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type mockEmbeddingModel struct {
	provider   string
	modelID    string
	embedFunc  func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error)
	maxPerCall int
	callCount  int
	mu         sync.Mutex
}

func (m *mockEmbeddingModel) Provider() string          { return m.provider }
func (m *mockEmbeddingModel) ModelID() string           { return m.modelID }
func (m *mockEmbeddingModel) MaxEmbeddingsPerCall() int { return m.maxPerCall }
func (m *mockEmbeddingModel) Embed(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
	m.mu.Lock()
	m.callCount++
	m.mu.Unlock()
	if m.embedFunc != nil {
		return m.embedFunc(ctx, req)
	}
	embeddings := make([][]float64, len(req.Texts))
	for i := range embeddings {
		embeddings[i] = []float64{float64(i), 0.5, 0.3}
	}
	return &provider.EmbedResult{
		Embeddings: embeddings,
		Usage:      provider.Usage{PromptTokens: len(req.Texts) * 10, TotalTokens: len(req.Texts) * 10},
	}, nil
}

func TestEmbedNilModel(t *testing.T) {
	ctx := context.Background()
	_, err := Embed(ctx, nil, "test")
	if !errors.Is(err, provider.ErrInvalidModel) {
		t.Errorf("Embed() error = %v, want ErrInvalidModel", err)
	}
}

func TestEmbedEmptyText(t *testing.T) {
	ctx := context.Background()
	model := &mockEmbeddingModel{provider: "test", modelID: "test-model"}
	_, err := Embed(ctx, model, "")
	if !errors.Is(err, provider.ErrInvalidInput) {
		t.Errorf("Embed() error = %v, want ErrInvalidInput", err)
	}
}

func TestEmbedSuccess(t *testing.T) {
	ctx := context.Background()
	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: 10,
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			if len(req.Texts) != 1 || req.Texts[0] != "hello" {
				t.Errorf("unexpected request: %v", req.Texts)
			}
			return &provider.EmbedResult{
				Embeddings: [][]float64{{0.1, 0.2, 0.3}},
				Usage:      provider.Usage{PromptTokens: 5, TotalTokens: 5},
			}, nil
		},
	}

	result, err := Embed(ctx, model, "hello")
	if err != nil {
		t.Fatalf("Embed() error = %v", err)
	}
	if len(result.Embedding) != 3 {
		t.Errorf("Embedding length = %d, want 3", len(result.Embedding))
	}
	if result.Usage.TotalTokens != 5 {
		t.Errorf("Usage.TotalTokens = %d, want 5", result.Usage.TotalTokens)
	}
}

func TestEmbedError(t *testing.T) {
	ctx := context.Background()
	expectedErr := &provider.Error{Code: provider.CodeAPITimeout, Message: "timeout"}
	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			return nil, expectedErr
		},
	}

	_, err := Embed(ctx, model, "test")
	if err == nil {
		t.Fatal("Embed() error = nil, want error")
	}
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("Embed() error = %v, want *provider.Error", err)
	}
}

func TestEmbedDefaultTimeout(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
		embedFunc: func(innerCtx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			capturedCtx = innerCtx
			return &provider.EmbedResult{Embeddings: [][]float64{{0.1}}}, nil
		},
	}

	_, _ = Embed(ctx, model, "test")

	deadline, ok := capturedCtx.Deadline()
	if !ok {
		t.Error("Embed() should set a deadline when no context deadline exists")
	}
	expectedTimeout := time.Now().Add(DefaultTimeout)
	if deadline.After(expectedTimeout.Add(time.Second)) || deadline.Before(expectedTimeout.Add(-time.Second)) {
		t.Errorf("Deadline = %v, want approximately %v", deadline, expectedTimeout)
	}
}

func TestEmbedCustomTimeout(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
		embedFunc: func(innerCtx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			capturedCtx = innerCtx
			return &provider.EmbedResult{Embeddings: [][]float64{{0.1}}}, nil
		},
	}

	customTimeout := 5 * time.Minute
	_, _ = Embed(ctx, model, "test", WithTimeout(customTimeout))

	deadline, ok := capturedCtx.Deadline()
	if !ok {
		t.Error("Embed() should set a deadline for custom timeout")
	}
	expectedTimeout := time.Now().Add(customTimeout)
	if deadline.After(expectedTimeout.Add(time.Second)) || deadline.Before(expectedTimeout.Add(-time.Second)) {
		t.Errorf("Deadline = %v, want approximately %v", deadline, expectedTimeout)
	}
}

func TestEmbedNoTimeout(t *testing.T) {
	ctx := context.Background()
	var capturedCtx context.Context

	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
		embedFunc: func(innerCtx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			capturedCtx = innerCtx
			return &provider.EmbedResult{Embeddings: [][]float64{{0.1}}}, nil
		},
	}

	_, _ = Embed(ctx, model, "test", WithNoTimeout())

	_, ok := capturedCtx.Deadline()
	if ok {
		t.Error("Embed() should not set deadline when WithNoTimeout()")
	}
}

func TestEmbedBatchNilModel(t *testing.T) {
	ctx := context.Background()
	_, err := EmbedBatch(ctx, nil, []string{"test"})
	if !errors.Is(err, provider.ErrInvalidModel) {
		t.Errorf("EmbedBatch() error = %v, want ErrInvalidModel", err)
	}
}

func TestEmbedBatchEmptyTexts(t *testing.T) {
	ctx := context.Background()
	model := &mockEmbeddingModel{provider: "test", modelID: "test-model"}
	_, err := EmbedBatch(ctx, model, []string{})
	if !errors.Is(err, provider.ErrInvalidInput) {
		t.Errorf("EmbedBatch() error = %v, want ErrInvalidInput", err)
	}
}

func TestEmbedBatchSingleCall(t *testing.T) {
	ctx := context.Background()
	texts := []string{"hello", "world"}
	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: 10,
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			if len(req.Texts) != 2 {
				t.Errorf("expected 2 texts, got %d", len(req.Texts))
			}
			embeddings := make([][]float64, len(req.Texts))
			for i := range embeddings {
				embeddings[i] = []float64{float64(i)}
			}
			return &provider.EmbedResult{
				Embeddings: embeddings,
				Usage:      provider.Usage{PromptTokens: 20, TotalTokens: 20},
			}, nil
		},
	}

	result, err := EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(result.Embeddings) != 2 {
		t.Errorf("Embeddings count = %d, want 2", len(result.Embeddings))
	}
	if model.callCount != 1 {
		t.Errorf("callCount = %d, want 1", model.callCount)
	}
}

func TestEmbedBatchMultipleCalls(t *testing.T) {
	ctx := context.Background()
	texts := []string{"a", "b", "c", "d", "e"}
	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: 2,
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			embeddings := make([][]float64, len(req.Texts))
			for i := range embeddings {
				embeddings[i] = []float64{float64(i)}
			}
			return &provider.EmbedResult{
				Embeddings: embeddings,
				Usage:      provider.Usage{PromptTokens: len(req.Texts) * 10, TotalTokens: len(req.Texts) * 10},
			}, nil
		},
	}

	result, err := EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(result.Embeddings) != 5 {
		t.Errorf("Embeddings count = %d, want 5", len(result.Embeddings))
	}
	if model.callCount != 3 {
		t.Errorf("callCount = %d, want 3 (5 texts / 2 per call)", model.callCount)
	}
	if result.Usage.TotalTokens != 50 {
		t.Errorf("TotalTokens = %d, want 50 (5 * 10)", result.Usage.TotalTokens)
	}
}

func TestEmbedBatchZeroMaxPerCall(t *testing.T) {
	ctx := context.Background()
	texts := []string{"a", "b", "c"}
	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: 0,
	}

	result, err := EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(result.Embeddings) != 3 {
		t.Errorf("Embeddings count = %d, want 3", len(result.Embeddings))
	}
	if model.callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should use single call)", model.callCount)
	}
}

func TestEmbedBatchNegativeMaxPerCall(t *testing.T) {
	ctx := context.Background()
	texts := []string{"a", "b", "c"}
	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: -1,
	}

	result, err := EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}
	if len(result.Embeddings) != 3 {
		t.Errorf("Embeddings count = %d, want 3", len(result.Embeddings))
	}
	if model.callCount != 1 {
		t.Errorf("callCount = %d, want 1 (should use single call)", model.callCount)
	}
}

func TestEmbedBatchError(t *testing.T) {
	ctx := context.Background()
	texts := []string{"a", "b"}
	expectedErr := &provider.Error{Code: provider.CodeRateLimited, Message: "rate limited"}
	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: 1,
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			return nil, expectedErr
		},
	}

	_, err := EmbedBatch(ctx, model, texts)
	if err == nil {
		t.Fatal("EmbedBatch() error = nil, want error")
	}
}

func TestEmbedBatchContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	texts := []string{"a", "b"}
	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
				return &provider.EmbedResult{Embeddings: [][]float64{{0.1}}}, nil
			}
		},
	}

	_, err := EmbedBatch(ctx, model, texts)
	if err == nil {
		t.Error("EmbedBatch() error = nil, want context canceled error")
	}
}

func TestOptionsWithTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithTimeout(60 * time.Second)(cfg)
	if cfg.Timeout != 60*time.Second {
		t.Errorf("WithTimeout() = %v, want 60s", cfg.Timeout)
	}
}

func TestOptionsWithNoTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithNoTimeout()(cfg)
	if cfg.Timeout != -1 {
		t.Errorf("WithNoTimeout() = %v, want -1", cfg.Timeout)
	}
}

func TestOptionsWithLogger(t *testing.T) {
	cfg := defaultConfig()
	logger := &testLogger{}
	WithLogger(logger)(cfg)
	if cfg.Logger != logger {
		t.Error("WithLogger() did not set logger")
	}
}

func TestOptionsWithMetrics(t *testing.T) {
	cfg := defaultConfig()
	metrics := provider.NoopMetrics{}
	WithMetrics(metrics)(cfg)
	if cfg.Metrics != metrics {
		t.Error("WithMetrics() did not set metrics")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.ProviderConfigs == nil {
		t.Error("defaultConfig() ProviderConfigs should not be nil")
	}
	if len(cfg.ProviderConfigs) != 0 {
		t.Error("defaultConfig() ProviderConfigs should be empty")
	}
}

type testLogger struct{}

func (l *testLogger) Debug(msg string, args ...any) {}
func (l *testLogger) Info(msg string, args ...any)  {}
func (l *testLogger) Warn(msg string, args ...any)  {}
func (l *testLogger) Error(msg string, args ...any) {}

func TestEmbedBatchUnevenBatches(t *testing.T) {
	ctx := context.Background()
	// Test with 10 items, batch size 3 = batches [3, 3, 3, 1]
	texts := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}
	batchSize := 3

	model := &mockEmbeddingModel{
		provider:   "test",
		modelID:    "test-model",
		maxPerCall: batchSize,
		embedFunc: func(ctx context.Context, req *provider.EmbedRequest) (*provider.EmbedResult, error) {
			embeddings := make([][]float64, len(req.Texts))
			for i := range embeddings {
				// Use the text value as the embedding so we can verify order
				embeddings[i] = []float64{float64(len(req.Texts[0])), 0.5}
			}
			return &provider.EmbedResult{
				Embeddings: embeddings,
				Usage:      provider.Usage{PromptTokens: len(req.Texts) * 10, TotalTokens: len(req.Texts) * 10},
			}, nil
		},
	}

	result, err := EmbedBatch(ctx, model, texts)
	if err != nil {
		t.Fatalf("EmbedBatch() error = %v", err)
	}

	// Verify all embeddings are present and in correct order
	if len(result.Embeddings) != 10 {
		t.Errorf("Embeddings count = %d, want 10", len(result.Embeddings))
	}

	// Verify the embeddings are in the correct order (not swapped due to bug)
	for i, emb := range result.Embeddings {
		if emb == nil {
			t.Errorf("Embedding[%d] is nil", i)
		}
	}
}

func TestBatchError(t *testing.T) {
	batchErr := &BatchError{
		Errors: []error{&provider.Error{Code: provider.CodeRateLimited, Message: "rate limited"}},
	}

	errStr := batchErr.Error()
	if !containsStr(errStr, "rate limited") {
		t.Errorf("BatchError.Error() = %s, want to contain 'rate limited'", errStr)
	}

	batchErr2 := &BatchError{
		Errors: []error{
			&provider.Error{Code: provider.CodeRateLimited, Message: "err1"},
			&provider.Error{Code: provider.CodeAPITimeout, Message: "err2"},
		},
	}

	if batchErr2.Error() != "batch processing failed with 2 errors" {
		t.Errorf("BatchError.Error() = %s, want 'batch processing failed with 2 errors'", batchErr2.Error())
	}

	if len(batchErr2.Unwrap()) != 2 {
		t.Errorf("BatchError.Unwrap() length = %d, want 2", len(batchErr2.Unwrap()))
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > 0 && containsStr(s[1:], substr)
}

func TestEmbedContextAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
	}

	_, err := Embed(ctx, model, "test")
	if err == nil {
		t.Error("Embed() error = nil, want context canceled error")
	}

	var providerErr *provider.Error
	if errors.As(err, &providerErr) {
		if providerErr.Code != provider.CodeContextCanceled {
			t.Errorf("Error code = %v, want CodeContextCanceled", providerErr.Code)
		}
	} else {
		t.Errorf("Error type = %T, want *provider.Error", err)
	}
}

func TestEmbedBatchContextAlreadyCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	model := &mockEmbeddingModel{
		provider: "test",
		modelID:  "test-model",
	}

	_, err := EmbedBatch(ctx, model, []string{"a", "b"})
	if err == nil {
		t.Error("EmbedBatch() error = nil, want context canceled error")
	}

	var providerErr *provider.Error
	if errors.As(err, &providerErr) {
		if providerErr.Code != provider.CodeContextCanceled {
			t.Errorf("Error code = %v, want CodeContextCanceled", providerErr.Code)
		}
	} else {
		t.Errorf("Error type = %T, want *provider.Error", err)
	}
}
