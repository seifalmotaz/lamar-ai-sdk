package openai

import (
	"net/http"
	"os"

	"github.com/seifalmotaz/lamar-sdk/internal/httpx"
	"github.com/seifalmotaz/lamar-sdk/provider"
)

// DefaultBaseURL is the default OpenAI API endpoint.
const DefaultBaseURL = "https://api.openai.com/v1"

// Provider is the OpenAI API provider.
// It implements the provider interfaces for chat completions and embeddings.
type Provider struct {
	client    *httpx.Client
	apiKey    string
	baseURL   string
	orgID     string
	projectID string
}

// Option configures the Provider.
type Option func(*Provider)

// APIKey sets the API key for authentication.
// If not provided, uses the OPENAI_API_KEY environment variable.
func APIKey(key string) Option {
	return func(p *Provider) {
		p.apiKey = key
	}
}

// BaseURL sets the base URL for the API.
// Useful for Azure OpenAI or custom endpoints.
func BaseURL(url string) Option {
	return func(p *Provider) {
		p.baseURL = url
	}
}

// HTTPClient sets a custom HTTP client for requests.
func HTTPClient(client *http.Client) Option {
	return func(p *Provider) {
		p.client = httpx.NewClient(p.baseURL, client)
	}
}

// OrgID sets the OpenAI organization ID for requests.
func OrgID(orgID string) Option {
	return func(p *Provider) {
		p.orgID = orgID
	}
}

// ProjectID sets the OpenAI project ID for requests.
// If not provided, uses the OPENAI_PROJECT_ID environment variable.
func ProjectID(projectID string) Option {
	return func(p *Provider) {
		p.projectID = projectID
	}
}

// NewProvider creates a new OpenAI provider with the given options.
//
// Example:
//
//	client := openai.NewProvider(
//	    openai.APIKey("sk-..."),
//	    openai.OrgID("org-..."),
//	)
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

// Model returns a non-streaming chat model with the given ID.
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

// Embedding returns an embedding model with the given ID.
func (p *Provider) Embedding(id string) provider.EmbeddingModel {
	return &EmbeddingModel{
		id:       id,
		provider: p,
	}
}

// GPT4 returns a GPT-4 model.
func (p *Provider) GPT4() provider.Generator {
	return p.Model("gpt-4")
}

// GPT4o returns a GPT-4o model with streaming support.
func (p *Provider) GPT4o() provider.LanguageModel {
	return p.StreamingModel("gpt-4o")
}

// GPT4oMini returns a GPT-4o-mini model with streaming support.
func (p *Provider) GPT4oMini() provider.LanguageModel {
	return p.StreamingModel("gpt-4o-mini")
}

// GPT4Turbo returns a GPT-4-turbo model.
func (p *Provider) GPT4Turbo() provider.Generator {
	return p.Model("gpt-4-turbo")
}

// O1 returns an O1 model.
func (p *Provider) O1() provider.Generator {
	return p.Model("o1")
}

// O1Mini returns an O1-mini model.
func (p *Provider) O1Mini() provider.Generator {
	return p.Model("o1-mini")
}

// O1Preview returns an O1-preview model.
func (p *Provider) O1Preview() provider.Generator {
	return p.Model("o1-preview")
}

// TextEmbedding3Small returns a text-embedding-3-small embedding model.
func (p *Provider) TextEmbedding3Small() provider.EmbeddingModel {
	return p.Embedding("text-embedding-3-small")
}

// TextEmbedding3Large returns a text-embedding-3-large embedding model.
func (p *Provider) TextEmbedding3Large() provider.EmbeddingModel {
	return p.Embedding("text-embedding-3-large")
}

// TextEmbeddingAda002 returns a text-embedding-ada-002 embedding model.
func (p *Provider) TextEmbeddingAda002() provider.EmbeddingModel {
	return p.Embedding("text-embedding-ada-002")
}

// GPT4 creates a GPT-4 model using the default provider.
func GPT4() provider.Generator {
	return NewProvider().GPT4()
}

// GPT4o creates a GPT-4o model using the default provider.
func GPT4o() provider.Generator {
	return NewProvider().GPT4o()
}

// GPT4oMini creates a GPT-4o-mini model using the default provider.
func GPT4oMini() provider.Generator {
	return NewProvider().GPT4oMini()
}

// O1 creates an O1 model using the default provider.
func O1() provider.Generator {
	return NewProvider().O1()
}

// O1Mini creates an O1-mini model using the default provider.
func O1Mini() provider.Generator {
	return NewProvider().O1Mini()
}

// TextEmbedding3Small creates a text-embedding-3-small model using the default provider.
func TextEmbedding3Small() provider.EmbeddingModel {
	return NewProvider().TextEmbedding3Small()
}

// TextEmbedding3Large creates a text-embedding-3-large model using the default provider.
func TextEmbedding3Large() provider.EmbeddingModel {
	return NewProvider().TextEmbedding3Large()
}
