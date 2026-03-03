package openai

import (
	"net/http"
	"os"

	"github.com/seifalmotaz/lamar-sdk/internal/httpx"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

const (
	DefaultBaseURL = "https://api.openai.com/v1"
)

type Provider struct {
	client    *httpx.Client
	apiKey    string
	baseURL   string
	orgID     string
	projectID string
}

type Option func(*Provider)

func APIKey(key string) Option {
	return func(p *Provider) {
		p.apiKey = key
	}
}

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

func OrgID(orgID string) Option {
	return func(p *Provider) {
		p.orgID = orgID
	}
}

func ProjectID(projectID string) Option {
	return func(p *Provider) {
		p.projectID = projectID
	}
}

func NewProvider(opts ...Option) *Provider {
	p := &Provider{
		baseURL: DefaultBaseURL,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.apiKey == "" {
		p.apiKey = os.Getenv("OPENAI_API_KEY")
	}

	if p.projectID == "" {
		p.projectID = os.Getenv("OPENAI_PROJECT_ID")
	}

	if p.client == nil {
		p.client = httpx.NewClient(p.baseURL, http.DefaultClient)
	}

	p.client.SetHeader("Authorization", "Bearer "+p.apiKey)
	if p.orgID != "" {
		p.client.SetHeader("OpenAI-Organization", p.orgID)
	}
	if p.projectID != "" {
		p.client.SetHeader("OpenAI-Project", p.projectID)
	}

	return p
}

func (p *Provider) Model(id string) provider.Generator {
	return &ChatModel{
		id:       id,
		provider: p,
	}
}

// StreamingModel returns a model that supports both generation and streaming.
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

func (p *Provider) GPT4() provider.Generator {
	return p.Model("gpt-4")
}

func (p *Provider) GPT4o() provider.LanguageModel {
	return p.StreamingModel("gpt-4o")
}

func (p *Provider) GPT4oMini() provider.LanguageModel {
	return p.StreamingModel("gpt-4o-mini")
}

func (p *Provider) GPT4Turbo() provider.Generator {
	return p.Model("gpt-4-turbo")
}

func (p *Provider) O1() provider.Generator {
	return p.Model("o1")
}

func (p *Provider) O1Mini() provider.Generator {
	return p.Model("o1-mini")
}

func (p *Provider) O1Preview() provider.Generator {
	return p.Model("o1-preview")
}

func (p *Provider) TextEmbedding3Small() provider.EmbeddingModel {
	return p.Embedding("text-embedding-3-small")
}

func (p *Provider) TextEmbedding3Large() provider.EmbeddingModel {
	return p.Embedding("text-embedding-3-large")
}

func (p *Provider) TextEmbeddingAda002() provider.EmbeddingModel {
	return p.Embedding("text-embedding-ada-002")
}

func GPT4() provider.Generator {
	return NewProvider().GPT4()
}

func GPT4o() provider.Generator {
	return NewProvider().GPT4o()
}

func GPT4oMini() provider.Generator {
	return NewProvider().GPT4oMini()
}

func O1() provider.Generator {
	return NewProvider().O1()
}

func O1Mini() provider.Generator {
	return NewProvider().O1Mini()
}

func TextEmbedding3Small() provider.EmbeddingModel {
	return NewProvider().TextEmbedding3Small()
}

func TextEmbedding3Large() provider.EmbeddingModel {
	return NewProvider().TextEmbedding3Large()
}
