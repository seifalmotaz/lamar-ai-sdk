package stream

import (
	"context"
	"sync"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// Result contains a streaming text generation result.
// It is thread-safe for concurrent access.
type Result struct {
	stream chan provider.StreamPart
	data   *streamData
	mu     sync.RWMutex
	done   chan struct{}
}

type streamData struct {
	text         string
	usage        provider.Usage
	finishReason provider.FinishReason
	err          error
}

// Stream returns the channel for consuming stream parts.
// The channel is closed when streaming completes.
func (r *Result) Stream() <-chan provider.StreamPart {
	return r.stream
}

// Text blocks until streaming completes and returns the full text.
func (r *Result) Text() (string, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return "", nil
	}
	return r.data.text, r.data.err
}

// Usage blocks until streaming completes and returns usage statistics.
func (r *Result) Usage() (provider.Usage, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return provider.Usage{}, nil
	}
	return r.data.usage, r.data.err
}

// FinishReason blocks until streaming completes and returns the finish reason.
func (r *Result) FinishReason() (provider.FinishReason, error) {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return provider.FinishReasonStop, nil
	}
	return r.data.finishReason, r.data.err
}

// Err blocks until streaming completes and returns any error.
func (r *Result) Err() error {
	<-r.done
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.data == nil {
		return nil
	}
	return r.data.err
}

// Wait blocks until streaming completes.
func (r *Result) Wait() <-chan struct{} {
	return r.done
}

// newResult creates a new streaming result.
func newResult() *Result {
	return &Result{
		stream: make(chan provider.StreamPart, 100),
		done:   make(chan struct{}),
	}
}

// setText stores the accumulated text.
func (r *Result) setText(text string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &streamData{}
	}
	r.data.text = text
}

// setUsage stores the final usage.
func (r *Result) setUsage(usage provider.Usage) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &streamData{}
	}
	r.data.usage = usage
}

// setFinishReason stores the finish reason.
func (r *Result) setFinishReason(reason provider.FinishReason) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &streamData{}
	}
	r.data.finishReason = reason
}

// setError stores an error.
func (r *Result) setError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.data == nil {
		r.data = &streamData{}
	}
	r.data.err = err
}

// close closes the stream channels.
func (r *Result) close() {
	close(r.stream)
	close(r.done)
}

func (c *Config) logDebug(msg string, args ...any) {
	if c.Logger == nil {
		return
	}
	c.Logger.Debug(msg, args...)
}

func (c *Config) logError(msg string, args ...any) {
	if c.Logger == nil {
		return
	}
	c.Logger.Error(msg, args...)
}

func (c *Config) recordMetricsWithErr(ctx context.Context, model provider.Model, duration time.Duration, err error) {
	if c.Metrics == nil {
		return
	}
	c.Metrics.RecordRequest(ctx, model.Provider(), model.ModelID(), duration, err)
}

func (c *Config) recordStreamStart(ctx context.Context, model provider.Model) {
	if c.Metrics == nil {
		return
	}
	c.Metrics.RecordStreamStart(ctx, model.Provider(), model.ModelID())
}

func (c *Config) recordStreamEnd(ctx context.Context, model provider.Model, duration time.Duration) {
	if c.Metrics == nil {
		return
	}
	c.Metrics.RecordStreamEnd(ctx, model.Provider(), model.ModelID(), duration)
}
