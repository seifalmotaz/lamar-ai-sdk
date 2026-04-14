package middleware

import (
	"context"
	"fmt"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Wrapper wraps core operations through the middleware chain.
// It provides zero-allocation fast path when no middlewares are configured.
type Wrapper struct {
	name        string
	middlewares []Middleware
}

// NewWrapper creates a new wrapper with the given provider name and middlewares.
func NewWrapper(name string, middlewares []Middleware) *Wrapper {
	return &Wrapper{
		name:        name,
		middlewares: middlewares,
	}
}

// GenerateFunc is the core function type for non-streaming generation.
type GenerateFunc func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error)

// StreamFunc is the core function type for streaming generation.
type StreamFunc func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error)

// EmbedFunc is the core function type for embeddings.
type EmbedFunc func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error)

// Generate wraps a generate call through the middleware chain.
// If no middleware is configured, calls the core function directly.
func (w *Wrapper) Generate(ctx context.Context, modelID string, req *provider.GenerateRequest, core GenerateFunc) (*provider.GenerateResult, error) {
	if len(w.middlewares) == 0 {
		return core(ctx, req)
	}

	handler := Chain(w.middlewares...)(HandlerFunc(func(ctx context.Context, r Request) (Response, error) {
		result, err := core(ctx, req)
		if err != nil {
			return nil, err
		}
		return &GenerateResponse{
			Text:             result.Text,
			Content:          result.Content,
			ToolCalls:        result.ToolCalls,
			FinishReasonData: result.FinishReason,
			UsageData:        result.Usage,
		}, nil
	}))

	mwReq := &GenerateRequest{
		ProviderName: w.name,
		Model:        modelID,
		Prompt:       req.Prompt,
		Messages:     req.Messages,
		Config:       req.Config,
	}

	resp, err := handler.Handle(ctx, mwReq)
	if err != nil {
		return nil, err
	}

	genResp, ok := resp.(*GenerateResponse)
	if !ok {
		return nil, &provider.Error{
			Code:    provider.CodeUnknown,
			Message: fmt.Sprintf("unexpected response type: %T", resp),
		}
	}

	return &provider.GenerateResult{
		Text:         genResp.Text,
		Content:      genResp.Content,
		ToolCalls:    genResp.ToolCalls,
		FinishReason: genResp.FinishReasonData,
		Usage:        genResp.UsageData,
	}, nil
}

// Stream wraps a stream call through the middleware chain.
// If no middleware is configured, calls the core function directly.
func (w *Wrapper) Stream(ctx context.Context, modelID string, req *provider.GenerateRequest, core StreamFunc) (*provider.StreamResult, error) {
	if len(w.middlewares) == 0 {
		return core(ctx, req)
	}

	handler := Chain(w.middlewares...)(HandlerFunc(func(ctx context.Context, r Request) (Response, error) {
		result, err := core(ctx, req)
		if err != nil {
			return nil, err
		}
		return &StreamResponse{
			StreamChan:       result.Stream,
			DoneChan:         result.Done,
			TextFunc:         result.Text,
			UsageFunc:        result.Usage,
			FinishReasonFunc: result.FinishReason,
		}, nil
	}))

	mwReq := &StreamRequest{
		ProviderName: w.name,
		Model:        modelID,
		Prompt:       req.Prompt,
		Messages:     req.Messages,
		Config:       req.Config,
	}

	resp, err := handler.Handle(ctx, mwReq)
	if err != nil {
		return nil, err
	}

	streamResp, ok := resp.(*StreamResponse)
	if !ok {
		return nil, &provider.Error{
			Code:    provider.CodeUnknown,
			Message: fmt.Sprintf("unexpected response type: %T", resp),
		}
	}

	return &provider.StreamResult{
		Stream:       streamResp.StreamChan,
		Done:         streamResp.DoneChan,
		Text:         streamResp.TextFunc,
		Usage:        streamResp.UsageFunc,
		FinishReason: streamResp.FinishReasonFunc,
	}, nil
}

// Embed wraps an embed call through the middleware chain.
// If no middleware is configured, calls the core function directly.
func (w *Wrapper) Embed(ctx context.Context, modelID string, texts []string, core EmbedFunc) ([][]float64, provider.Usage, error) {
	if len(w.middlewares) == 0 {
		return core(ctx, texts)
	}

	handler := Chain(w.middlewares...)(HandlerFunc(func(ctx context.Context, r Request) (Response, error) {
		embeddings, usage, err := core(ctx, texts)
		if err != nil {
			return nil, err
		}
		return &EmbedResponse{
			Embeddings: embeddings,
			UsageData:  usage,
		}, nil
	}))

	mwReq := &EmbedRequest{
		ProviderName: w.name,
		Model:        modelID,
		Texts:        texts,
	}

	resp, err := handler.Handle(ctx, mwReq)
	if err != nil {
		return nil, provider.Usage{}, err
	}

	embedResp, ok := resp.(*EmbedResponse)
	if !ok {
		return nil, provider.Usage{}, &provider.Error{
			Code:    provider.CodeUnknown,
			Message: fmt.Sprintf("unexpected response type: %T", resp),
		}
	}

	return embedResp.Embeddings, embedResp.UsageData, nil
}
