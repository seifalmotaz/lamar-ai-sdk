package embed

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Default timeout constants for embedding operations.
const (
	// DefaultTimeout is the default timeout for single embedding requests.
	DefaultTimeout = 10 * time.Second
	// DefaultBatchTimeout is the default timeout for batch embedding requests.
	DefaultBatchTimeout = 5 * time.Minute
)

// Embed generates an embedding for a single text using the specified model.
// It returns a Result containing the embedding vector and usage statistics,
// or an error if the request fails.
//
// Example:
//
//	result, err := embed.Embed(ctx, model, "Hello, world!")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Embedding dimension: %d\n", len(result.Embedding))
func Embed(ctx context.Context, model provider.EmbeddingModel, text string, opts ...Option) (*Result, error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
	}
	if text == "" {
		return nil, provider.ErrInvalidInput
	}

	// Check if context is already canceled
	select {
	case <-ctx.Done():
		return nil, provider.NewError(provider.CodeContextCanceled, "context already canceled", ctx.Err())
	default:
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	} else if timeout < 0 {
		timeout = 0
	}
	ctx, cancel := applyTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	cfg.logDebug("embed: starting", "provider", model.Provider(), "model", model.ModelID())

	result, err := model.Embed(ctx, &provider.EmbedRequest{
		Texts: []string{text},
	})
	if err != nil {
		cfg.logError("embed: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("embed: completed", "provider", model.Provider(), "model", model.ModelID(), "tokens", result.Usage.TotalTokens)
	cfg.recordMetrics(ctx, model, time.Since(start), nil)
	cfg.recordTokens(ctx, model, result.Usage)

	if len(result.Embeddings) == 0 {
		return &Result{Usage: result.Usage}, nil
	}

	return &Result{
		Embedding: result.Embeddings[0],
		Usage:     result.Usage,
	}, nil
}

// EmbedBatch generates embeddings for multiple texts using the specified model.
// It automatically handles batching based on the model's MaxEmbeddingsPerCall limit.
// Batches are processed in parallel for improved performance.
//
// Example:
//
//	texts := []string{"Hello", "World", "Go"}
//	result, err := embed.EmbedBatch(ctx, model, texts)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Generated %d embeddings\n", len(result.Embeddings))
func EmbedBatch(ctx context.Context, model provider.EmbeddingModel, texts []string, opts ...Option) (*BatchResult, error) {
	if model == nil {
		return nil, provider.ErrInvalidModel
	}
	if len(texts) == 0 {
		return nil, provider.ErrInvalidInput
	}

	// Check if context is already canceled
	select {
	case <-ctx.Done():
		return nil, provider.NewError(provider.CodeContextCanceled, "context already canceled", ctx.Err())
	default:
	}

	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultBatchTimeout
	} else if timeout < 0 {
		timeout = 0
	}
	ctx, cancel := applyTimeout(ctx, timeout)
	defer cancel()

	maxPerCall := model.MaxEmbeddingsPerCall()
	if maxPerCall <= 0 || maxPerCall >= len(texts) {
		return singleBatchCall(ctx, model, texts, cfg)
	}

	return processBatches(ctx, model, texts, maxPerCall, cfg)
}

func singleBatchCall(ctx context.Context, model provider.EmbeddingModel, texts []string, cfg *Config) (*BatchResult, error) {
	start := time.Now()
	cfg.logDebug("embed: batch starting", "provider", model.Provider(), "model", model.ModelID(), "count", len(texts))

	result, err := model.Embed(ctx, &provider.EmbedRequest{Texts: texts})
	if err != nil {
		cfg.logError("embed: batch failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
		cfg.recordMetrics(ctx, model, time.Since(start), err)
		return nil, err
	}

	cfg.logDebug("embed: batch completed", "provider", model.Provider(), "model", model.ModelID(), "tokens", result.Usage.TotalTokens)
	cfg.recordMetrics(ctx, model, time.Since(start), nil)
	cfg.recordTokens(ctx, model, result.Usage)

	return &BatchResult{
		Embeddings: result.Embeddings,
		Usage:      result.Usage,
	}, nil
}

// BatchError represents errors that occurred during batch embedding processing.
// It contains all errors from individual batch calls.
type BatchError struct {
	Errors []error
}

// Error implements the error interface for BatchError.
func (e *BatchError) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("batch processing failed with %d errors", len(e.Errors))
}

// Unwrap returns all errors in the batch for use with errors.As/Is.
func (e *BatchError) Unwrap() []error {
	return e.Errors
}

func processBatches(ctx context.Context, model provider.EmbeddingModel, texts []string, batchSize int, cfg *Config) (*BatchResult, error) {
	batches := splitIntoBatches(texts, batchSize)
	results := make([][]float64, len(texts))
	var totalUsage provider.Usage
	var mu sync.Mutex
	var wg sync.WaitGroup
	errChan := make(chan error, len(batches))

	startIdx := 0
	for i, batch := range batches {
		wg.Add(1)
		go func(batchIdx int, batchTexts []string, idx int) {
			defer wg.Done()

			start := time.Now()
			cfg.logDebug("embed: batch chunk starting", "batch", batchIdx, "count", len(batchTexts))

			result, err := model.Embed(ctx, &provider.EmbedRequest{Texts: batchTexts})
			if err != nil {
				cfg.logError("embed: batch chunk failed", "batch", batchIdx, "error", err)
				cfg.recordMetrics(ctx, model, time.Since(start), err)
				errChan <- err
				return
			}

			cfg.recordMetrics(ctx, model, time.Since(start), nil)
			cfg.recordTokens(ctx, model, result.Usage)

			mu.Lock()
			for j, emb := range result.Embeddings {
				results[idx+j] = emb
			}
			totalUsage = totalUsage.Add(result.Usage)
			mu.Unlock()
		}(i, batch, startIdx)
		startIdx += len(batch)
	}

	wg.Wait()
	close(errChan)

	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return nil, &BatchError{Errors: errors}
	}

	return &BatchResult{
		Embeddings: results,
		Usage:      totalUsage,
	}, nil
}

func splitIntoBatches(texts []string, batchSize int) [][]string {
	var batches [][]string
	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}
		batches = append(batches, texts[i:end])
	}
	return batches
}

func applyTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout == 0 {
		return ctx, func() {}
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining < timeout {
			return ctx, func() {}
		}
	}

	return context.WithTimeout(ctx, timeout)
}

func (c *Config) logDebug(msg string, args ...any) {
	if c.Logger != nil {
		c.Logger.Debug(msg, args...)
	}
}

func (c *Config) logError(msg string, args ...any) {
	if c.Logger != nil {
		c.Logger.Error(msg, args...)
	}
}

func (c *Config) recordMetrics(ctx context.Context, model provider.Model, duration time.Duration, err error) {
	if c.Metrics != nil {
		c.Metrics.RecordRequest(ctx, model.Provider(), model.ModelID(), duration, err)
	}
}

func (c *Config) recordTokens(ctx context.Context, model provider.Model, usage provider.Usage) {
	if c.Metrics != nil {
		c.Metrics.RecordTokens(ctx, model.Provider(), model.ModelID(), usage.PromptTokens, usage.CompletionTokens)
	}
}
