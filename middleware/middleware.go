package middleware

import (
	"context"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type Handler interface {
	Handle(ctx context.Context, req Request) (Response, error)
}

type Request interface {
	Provider() string
	ModelID() string
	InputCount() int
}

type Response interface {
	Usage() provider.Usage
	FinishReason() provider.FinishReason
}

type HandlerFunc func(ctx context.Context, req Request) (Response, error)

func (f HandlerFunc) Handle(ctx context.Context, req Request) (Response, error) {
	return f(ctx, req)
}

type Middleware func(Handler) Handler

func Chain(middlewares ...Middleware) Middleware {
	return func(final Handler) Handler {
		for i := len(middlewares) - 1; i >= 0; i-- {
			final = middlewares[i](final)
		}
		return final
	}
}

type GenerateRequest struct {
	ProviderName string
	Model        string
	Prompt       string
	Messages     []provider.Message
	Config       provider.Config
}

func (r *GenerateRequest) Provider() string { return r.ProviderName }
func (r *GenerateRequest) ModelID() string  { return r.Model }
func (r *GenerateRequest) InputCount() int  { return 1 } // Single prompt or messages

type GenerateResponse struct {
	Text             string
	Content          []provider.Content
	ToolCalls        []provider.ToolCall
	FinishReasonData provider.FinishReason
	UsageData        provider.Usage
}

func (r *GenerateResponse) Usage() provider.Usage               { return r.UsageData }
func (r *GenerateResponse) FinishReason() provider.FinishReason { return r.FinishReasonData }

type EmbedRequest struct {
	ProviderName string
	Model        string
	Texts        []string
}

func (r *EmbedRequest) Provider() string { return r.ProviderName }
func (r *EmbedRequest) ModelID() string  { return r.Model }
func (r *EmbedRequest) InputCount() int  { return len(r.Texts) }

type EmbedResponse struct {
	Embeddings [][]float64
	UsageData  provider.Usage
}

func (r *EmbedResponse) Usage() provider.Usage { return r.UsageData }
