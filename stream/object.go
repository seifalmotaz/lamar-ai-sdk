package stream

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/seifalmotaz/lamar-sdk/internal/schema"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

// ObjectPart represents a part of a structured object stream.
type ObjectPart[T any] struct {
	Type   string // "object", "text-delta", "error", "finish"
	Object T      // Partial or complete object (for "object" type)
	Delta  string // Text delta (for "text-delta" type)
	Error  error  // Error (for "error" type)
}

// StreamObjectResult contains the result of streaming structured object generation.
type StreamObjectResult[T any] struct {
	stream chan ObjectPart[T]
	data   *objectData[T]
	mu     sync.RWMutex
	done   chan struct{}
}

type objectData[T any] struct {
	object       T
	text         string
	finishReason provider.FinishReason
	usage        provider.Usage
	err          error
}

// Stream returns the channel for consuming object stream parts.
func (r *StreamObjectResult[T]) Stream() <-chan ObjectPart[T] {
	return r.stream
}

// Object blocks until streaming completes and returns the final object.
func (r *StreamObjectResult[T]) Object() (T, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	var zero T
	if r.data == nil {
		return zero, nil
	}
	return r.data.object, r.data.err
}

// Text blocks until streaming completes and returns the raw JSON text.
func (r *StreamObjectResult[T]) Text() (string, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return "", nil
	}
	return r.data.text, r.data.err
}

// Usage blocks until streaming completes and returns usage statistics.
func (r *StreamObjectResult[T]) Usage() (provider.Usage, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return provider.Usage{}, nil
	}
	return r.data.usage, r.data.err
}

// FinishReason blocks until streaming completes.
func (r *StreamObjectResult[T]) FinishReason() (provider.FinishReason, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return provider.FinishReasonStop, nil
	}
	return r.data.finishReason, r.data.err
}

// Err blocks until streaming completes and returns any error.
func (r *StreamObjectResult[T]) Err() error {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return nil
	}
	return r.data.err
}

// Wait blocks until streaming completes.
func (r *StreamObjectResult[T]) Wait() <-chan struct{} {
	return r.done
}

func newObjectResult[T any]() *StreamObjectResult[T] {
	return &StreamObjectResult[T]{
		stream: make(chan ObjectPart[T], 100),
		done:   make(chan struct{}),
	}
}

func (r *StreamObjectResult[T]) setObject(obj T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &objectData[T]{}
	}
	r.data.object = obj
}

func (r *StreamObjectResult[T]) setText(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &objectData[T]{}
	}
	r.data.text = text
}

func (r *StreamObjectResult[T]) setUsage(usage provider.Usage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &objectData[T]{}
	}
	r.data.usage = usage
}

func (r *StreamObjectResult[T]) setFinishReason(reason provider.FinishReason) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &objectData[T]{}
	}
	r.data.finishReason = reason
}

func (r *StreamObjectResult[T]) setError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &objectData[T]{}
	}
	r.data.err = err
}

func (r *StreamObjectResult[T]) close() {
	close(r.stream)
	close(r.done)
}

// StreamObject streams structured object generation from a model.
// It uses JSON schema response format to ensure structured output.
//
// For MVP, this accumulates all text deltas and parses once at completion.
// Future versions may support incremental JSON parsing.
//
// Example:
//
//	type Person struct {
//	    Name string `json:"name" jsonschema:"required"`
//	    Age  int    `json:"age" jsonschema:"required,minimum=0"`
//	}
//
//	result := stream.StreamObject[Person](ctx, model, "Generate a person")
//	for part := range result.Stream() {
//	    if part.Type == "object" {
//	        fmt.Printf("Partial: %+v\n", part.Object)
//	    }
//	}
//	final, err := result.Object()
func StreamObject[T any](ctx context.Context, model provider.Streamer, prompt string, opts ...Option) *StreamObjectResult[T] {
	cfg := defaultConfig()
	for _, opt := range opts {
		opt(cfg)
	}

	result := newObjectResult[T]()

	go func() {
		defer result.close()

		// Validate model
		if model == nil {
			result.stream <- ObjectPart[T]{Type: "error", Error: provider.ErrInvalidModel}
			result.setError(provider.ErrInvalidModel)
			return
		}

		if prompt == "" && len(cfg.Messages) == 0 {
			result.stream <- ObjectPart[T]{Type: "error", Error: provider.ErrInvalidPrompt}
			result.setError(provider.ErrInvalidPrompt)
			return
		}

		// Extract schema from type T
		sch := schema.FromStruct[T]()
		schemaBytes, err := json.Marshal(sch)
		if err != nil {
			result.stream <- ObjectPart[T]{Type: "error", Error: err}
			result.setError(err)
			return
		}

		start := time.Now()
		cfg.logDebug("stream_object: starting", "provider", model.Provider(), "model", model.ModelID())
		cfg.recordStreamStart(ctx, model)

		// Build request with JSON schema response format
		req := &provider.GenerateRequest{
			Prompt:   prompt,
			Messages: cfg.Messages,
			System:   cfg.System,
			Config: provider.Config{
				MaxTokens:     cfg.MaxTokens,
				Temperature:   cfg.Temperature,
				TopP:          cfg.TopP,
				TopK:          cfg.TopK,
				StopSequences: cfg.StopSequences,
				Seed:          cfg.Seed,
				ResponseFormat: &provider.ResponseFormat{
					Type:       "json_schema",
					JSONSchema: json.RawMessage(schemaBytes),
				},
			},
		}

		// Apply timeout
		streamCtx := ctx
		var cancel context.CancelFunc
		if cfg.Timeout != nil {
			if *cfg.Timeout > 0 {
				streamCtx, cancel = context.WithTimeout(ctx, *cfg.Timeout)
			}
		}

		// Call model
		streamResult, err := model.Stream(streamCtx, req)
		if err != nil {
			result.stream <- ObjectPart[T]{Type: "error", Error: err}
			result.setError(err)
			cfg.logError("stream_object: failed", "error", err, "provider", model.Provider(), "model", model.ModelID())
			cfg.recordMetricsWithErr(ctx, model, time.Since(start), err)
			cfg.recordStreamEnd(ctx, model, time.Since(start))
			if cancel != nil {
				cancel()
			}
			return
		}

		if cancel != nil {
			defer cancel()
		}

		// Accumulate text
		var textBuilder strings.Builder
		var usage provider.Usage
		var finishReason provider.FinishReason
		var streamErr error

		for part := range streamResult.Stream {
			switch p := part.(type) {
			case provider.StreamTextPart:
				textBuilder.WriteString(p.Delta)
				result.stream <- ObjectPart[T]{Type: "text-delta", Delta: p.Delta}
			case provider.StreamFinishPart:
				usage = p.Usage
				finishReason = p.FinishReason
			case provider.StreamErrorPart:
				streamErr = p.Error
				result.stream <- ObjectPart[T]{Type: "error", Error: p.Error}
			}
		}

		// Parse final object
		text := textBuilder.String()
		result.setText(text)

		if streamErr == nil && text != "" {
			var obj T
			if err := json.Unmarshal([]byte(text), &obj); err != nil {
				streamErr = &provider.ParseError{Field: "object", Err: err}
				result.stream <- ObjectPart[T]{Type: "error", Error: streamErr}
			} else {
				result.setObject(obj)
				result.stream <- ObjectPart[T]{Type: "object", Object: obj}
			}
		}

		result.setUsage(usage)
		result.setFinishReason(finishReason)
		if streamErr != nil {
			result.setError(streamErr)
		}

		result.stream <- ObjectPart[T]{Type: "finish"}

		cfg.logDebug("stream_object: completed", "provider", model.Provider(), "model", model.ModelID(), "tokens", usage.TotalTokens)
		cfg.recordMetricsWithErr(ctx, model, time.Since(start), streamErr)
		cfg.recordStreamEnd(ctx, model, time.Since(start))
	}()

	return result
}
