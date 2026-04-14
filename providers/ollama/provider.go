package ollama

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/httpx"
	"github.com/seifalmotaz/lamar-ai-sdk/middleware"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

const DefaultBaseURL = "http://127.0.0.1:11434"

type Provider struct {
	client      *httpx.Client
	baseURL     string
	middlewares []middleware.Middleware
}

type Option func(*Provider)

func BaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

func HTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.client = httpx.NewClient(p.baseURL, client)
	}
}

func WithMiddleware(middlewares ...middleware.Middleware) Option {
	return func(p *Provider) {
		p.middlewares = append(p.middlewares, middlewares...)
	}
}

func Timeout(d time.Duration) Option {
	return func(p *Provider) {
		if p.client == nil {
			p.client = httpx.NewClient(p.baseURL, &http.Client{Timeout: d})
		} else {
			p.client.HTTPClient.Timeout = d
		}
	}
}

func NewProvider(opts ...Option) *Provider {
	p := &Provider{
		baseURL: DefaultBaseURL,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.client == nil {
		p.client = httpx.NewClient(p.baseURL, http.DefaultClient)
	}

	return p
}

func (p *Provider) Model(id string) provider.Generator {
	return &ChatModel{
		id:       id,
		provider: p,
	}
}

func (p *Provider) StreamingModel(id string) provider.LanguageModel {
	return &ChatModel{
		id:       id,
		provider: p,
	}
}

func (p *Provider) Embedding(id string) provider.EmbeddingModel {
	return &EmbeddingModel{
		id:       id,
		provider: p,
	}
}

func (p *Provider) ModelWithConfig(id string, config *ChatConfig) provider.LanguageModel {
	return &ChatModel{
		id:       id,
		provider: p,
		config:   *config,
	}
}

func (p *Provider) hasMiddleware() bool {
	return len(p.middlewares) > 0
}

func (p *Provider) wrapGenerate(
	ctx context.Context,
	modelID string,
	req *provider.GenerateRequest,
	core func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error),
) (*provider.GenerateResult, error) {
	if !p.hasMiddleware() {
		return core(ctx, req)
	}

	handler := middleware.Chain(p.middlewares...)(middleware.HandlerFunc(func(ctx context.Context, r middleware.Request) (middleware.Response, error) {
		result, err := core(ctx, req)
		if err != nil {
			return nil, err
		}
		return &middleware.GenerateResponse{
			Text:             result.Text,
			Content:          result.Content,
			ToolCalls:        result.ToolCalls,
			FinishReasonData: result.FinishReason,
			UsageData:        result.Usage,
		}, nil
	}))

	mwReq := &middleware.GenerateRequest{
		ProviderName: "ollama",
		Model:        modelID,
		Prompt:       req.Prompt,
		Messages:     req.Messages,
		Config:       req.Config,
	}

	resp, err := handler.Handle(ctx, mwReq)
	if err != nil {
		return nil, err
	}

	genResp, ok := resp.(*middleware.GenerateResponse)
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

func (p *Provider) wrapEmbed(
	ctx context.Context,
	modelID string,
	texts []string,
	core func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error),
) ([][]float64, provider.Usage, error) {
	if !p.hasMiddleware() {
		return core(ctx, texts)
	}

	handler := middleware.Chain(p.middlewares...)(middleware.HandlerFunc(func(ctx context.Context, r middleware.Request) (middleware.Response, error) {
		embeddings, usage, err := core(ctx, texts)
		if err != nil {
			return nil, err
		}
		return &middleware.EmbedResponse{
			Embeddings: embeddings,
			UsageData:  usage,
		}, nil
	}))

	mwReq := &middleware.EmbedRequest{
		ProviderName: "ollama",
		Model:        modelID,
		Texts:        texts,
	}

	resp, err := handler.Handle(ctx, mwReq)
	if err != nil {
		return nil, provider.Usage{}, err
	}

	embedResp, ok := resp.(*middleware.EmbedResponse)
	if !ok {
		return nil, provider.Usage{}, &provider.Error{
			Code:    provider.CodeUnknown,
			Message: fmt.Sprintf("unexpected response type: %T", resp),
		}
	}
	return embedResp.Embeddings, embedResp.UsageData, nil
}

func (p *Provider) wrapStream(
	ctx context.Context,
	modelID string,
	req *provider.GenerateRequest,
	core func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error),
) (*provider.StreamResult, error) {
	if !p.hasMiddleware() {
		return core(ctx, req)
	}

	handler := middleware.Chain(p.middlewares...)(middleware.HandlerFunc(func(ctx context.Context, r middleware.Request) (middleware.Response, error) {
		result, err := core(ctx, req)
		if err != nil {
			return nil, err
		}
		return &middleware.StreamResponse{
			StreamChan:       result.Stream,
			DoneChan:         result.Done,
			TextFunc:         result.Text,
			UsageFunc:        result.Usage,
			FinishReasonFunc: result.FinishReason,
		}, nil
	}))

	mwReq := &middleware.StreamRequest{
		ProviderName: "ollama",
		Model:        modelID,
		Prompt:       req.Prompt,
		Messages:     req.Messages,
		Config:       req.Config,
	}

	resp, err := handler.Handle(ctx, mwReq)
	if err != nil {
		return nil, err
	}

	streamResp, ok := resp.(*middleware.StreamResponse)
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

func getEnvOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
