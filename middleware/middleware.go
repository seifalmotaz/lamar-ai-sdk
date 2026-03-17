package middleware

import (
	"context"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Handler processes a request and returns a response.
// Implementations wrap the core generation or embedding logic.
type Handler interface {
	Handle(ctx context.Context, req Request) (Response, error)
}

// Request represents a request to be processed by middleware.
type Request interface {
	// Provider returns the provider name.
	Provider() string
	// ModelID returns the model identifier.
	ModelID() string
	// InputCount returns the number of inputs in the request.
	InputCount() int
}

// Response represents a response from processing a request.
type Response interface {
	// Usage returns token usage statistics.
	Usage() provider.Usage
	// FinishReason returns why generation stopped.
	FinishReason() provider.FinishReason
}

// HandlerFunc is an adapter to allow using functions as Handlers.
type HandlerFunc func(ctx context.Context, req Request) (Response, error)

// Handle implements the Handler interface for HandlerFunc.
func (f HandlerFunc) Handle(ctx context.Context, req Request) (Response, error) {
	return f(ctx, req)
}

// Middleware wraps a Handler to add behavior before/after processing.
// Middlewares are chained together to form a processing pipeline.
type Middleware func(Handler) Handler

// Chain combines multiple middlewares into a single middleware.
// Middlewares are applied in the order they are provided.
// The first middleware in the list will be the outermost wrapper.
//
// Example:
//
//	handler := middleware.Chain(
//	    middleware.Logging(logger),
//	    middleware.Metrics(collector),
//	    middleware.Retry(config),
//	)(baseHandler)
func Chain(middlewares ...Middleware) Middleware {
	return func(final Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

// GenerateRequest represents a generation request for middleware processing.
type GenerateRequest struct {
	ProviderName string
	Model        string
	Prompt       string
	Messages     []provider.Message
	Config       provider.Config
}

func (r *GenerateRequest) Provider() string { return r.ProviderName }
func (r *GenerateRequest) ModelID() string  { return r.Model }
func (r *GenerateRequest) InputCount() int  { return 1 }

// GenerateResponse represents a generation response for middleware processing.
type GenerateResponse struct {
	Text             string
	Content          []provider.Content
	ToolCalls        []provider.ToolCall
	FinishReasonData provider.FinishReason
	UsageData        provider.Usage
}

func (r *GenerateResponse) Usage() provider.Usage               { return r.UsageData }
func (r *GenerateResponse) FinishReason() provider.FinishReason { return r.FinishReasonData }

// EmbedRequest represents an embedding request for middleware processing.
type EmbedRequest struct {
	ProviderName string
	Model        string
	Texts        []string
}

func (r *EmbedRequest) Provider() string { return r.ProviderName }
func (r *EmbedRequest) ModelID() string  { return r.Model }
func (r *EmbedRequest) InputCount() int  { return len(r.Texts) }

// EmbedResponse represents an embedding response for middleware processing.
type EmbedResponse struct {
	Embeddings [][]float64
	UsageData  provider.Usage
}

func (r *EmbedResponse) Usage() provider.Usage               { return r.UsageData }
func (r *EmbedResponse) FinishReason() provider.FinishReason { return provider.FinishReasonStop }

// StreamRequest represents a streaming request for middleware processing.
type StreamRequest struct {
	ProviderName string
	Model        string
	Prompt       string
	Messages     []provider.Message
	Config       provider.Config
}

func (r *StreamRequest) Provider() string { return r.ProviderName }
func (r *StreamRequest) ModelID() string  { return r.Model }
func (r *StreamRequest) InputCount() int  { return 1 }

// StreamResponse represents a streaming response for middleware processing.
// Note: Usage and FinishReason are not available until the stream completes,
// so middleware should rely on the Stream channel for observing stream behavior.
type StreamResponse struct {
	StreamChan       <-chan provider.StreamPart
	DoneChan         <-chan struct{}
	TextFunc         func() (string, error)
	UsageFunc        func() (provider.Usage, error)
	FinishReasonFunc func() (provider.FinishReason, error)
}

func (r *StreamResponse) Usage() provider.Usage {
	u, _ := r.UsageFunc()
	return u
}
func (r *StreamResponse) FinishReason() provider.FinishReason {
	fr, _ := r.FinishReasonFunc()
	return fr
}
