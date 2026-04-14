package ollama

import (
	"net/http"
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
	wrapper     *middleware.Wrapper
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

	p.wrapper = middleware.NewWrapper("ollama", p.middlewares)

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
