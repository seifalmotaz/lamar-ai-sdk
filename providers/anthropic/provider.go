// Package anthropic provides an implementation of the Lamar SDK provider interface
// for Anthropic's Claude models via the Anthropic Messages API.
//
// The provider supports text generation, streaming, tool calling, vision (images),
// extended thinking, prompt caching, and other Anthropic-specific features.
//
// Basic usage:
//
//	client := anthropic.NewProvider(anthropic.APIKey("your-api-key"))
//	model := client.Claude45Sonnet()
//
//	result, err := generate.Generate(ctx, model, "Hello, world!")
//
// For streaming:
//
//	result := stream.Stream(ctx, model, "Tell me a story")
//	for part := range result.Stream() {
//	    fmt.Print(part.(provider.StreamTextPart).Delta)
//	}
//
// Anthropic-specific features like extended thinking can be enabled via options:
//
//	model := client.Claude46Sonnet(
//	    anthropic.ThinkingEnabled(2048),
//	    anthropic.Effort("high"),
//	)
package anthropic

import (
	"net/http"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/httpx"
	"github.com/seifalmotaz/lamar-ai-sdk/middleware"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// DefaultBaseURL is the default Anthropic API endpoint.
const DefaultBaseURL = "https://api.anthropic.com/v1"

// Provider implements the Lamar SDK provider interface for Anthropic's Claude models.
// It supports both API key and OAuth token authentication.
type Provider struct {
	client     *httpx.Client
	apiKey     string
	authToken  string
	baseURL    string
	customName string

	middlewares []middleware.Middleware
	wrapper     *middleware.Wrapper
}

// Option configures the Provider.
type Option func(*Provider)

// APIKey sets the API key for authentication.
// If not provided, uses the ANTHROPIC_API_KEY environment variable.
func APIKey(key string) Option {
	return func(p *Provider) {
		p.apiKey = key
	}
}

// AuthToken sets the OAuth token for authentication.
// If not provided, uses the ANTHROPIC_AUTH_TOKEN environment variable.
func AuthToken(token string) Option {
	return func(p *Provider) {
		p.authToken = token
	}
}

// BaseURL sets a custom API endpoint.
// Useful for proxies or custom deployments.
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

// WithHeader adds a custom header to all requests.
func WithHeader(key, value string) Option {
	return func(p *Provider) {
		p.client.SetHeader(key, value)
	}
}

// WithMiddleware adds middleware to the provider's processing chain.
// Middleware is applied in order for all generate and stream operations.
func WithMiddleware(middlewares ...middleware.Middleware) Option {
	return func(p *Provider) {
		p.middlewares = append(p.middlewares, middlewares...)
	}
}

// Name sets a custom provider name for identification purposes.
func Name(name string) Option {
	return func(p *Provider) {
		p.customName = name
	}
}

// NewProvider creates a new Anthropic provider with the given options.
//
// Example:
//
//	client := anthropic.NewProvider(
//	    anthropic.APIKey("sk-ant-..."),
//	    anthropic.WithMiddleware(middleware.TimeoutWithDefault(30*time.Second)),
//	)
func NewProvider(opts ...Option) *Provider {
	p := &Provider{
		baseURL: DefaultBaseURL,
	}

	for _, opt := range opts {
		opt(p)
	}

	if p.apiKey == "" {
		p.apiKey = os.Getenv("ANTHROPIC_API_KEY")
	}
	if p.authToken == "" {
		p.authToken = os.Getenv("ANTHROPIC_AUTH_TOKEN")
	}

	if p.client == nil {
		p.client = httpx.NewClient(p.baseURL, http.DefaultClient)
	}

	if p.apiKey != "" {
		p.client.SetHeader("x-api-key", p.apiKey)
	}
	if p.authToken != "" {
		p.client.SetHeader("Authorization", "Bearer "+p.authToken)
	}

	p.client.SetHeader("anthropic-version", "2023-06-01")
	p.client.SetHeader("Content-Type", "application/json")

	p.wrapper = middleware.NewWrapper("anthropic", p.middlewares)

	return p
}

func (p *Provider) name() string {
	if p.customName != "" {
		return p.customName
	}
	return "anthropic"
}

func (p *Provider) Model(id string, opts ...ChatOption) provider.Generator {
	return NewChatModel(id, p, opts...)
}

func (p *Provider) StreamingModel(id string, opts ...ChatOption) provider.LanguageModel {
	return NewChatModel(id, p, opts...)
}

func (p *Provider) Claude3Haiku(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-3-haiku-20240307", opts...)
}

func (p *Provider) Claude3Opus(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-3-opus-20240229", opts...)
}

func (p *Provider) Claude35Sonnet(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-3-5-sonnet-20241022", opts...)
}

func (p *Provider) Claude35Haiku(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-3-5-haiku-20241022", opts...)
}

func (p *Provider) Claude4Sonnet(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-sonnet-4-20250514", opts...)
}

func (p *Provider) Claude4Opus(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-opus-4-20250514", opts...)
}

func (p *Provider) Claude45Sonnet(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-sonnet-4-5-20250929", opts...)
}

func (p *Provider) Claude45Opus(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-opus-4-5-20251101", opts...)
}

func (p *Provider) Claude46Sonnet(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-sonnet-4-6", opts...)
}

func (p *Provider) Claude46Opus(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-opus-4-6", opts...)
}

func (p *Provider) ClaudeHaiku45(opts ...ChatOption) provider.LanguageModel {
	return p.StreamingModel("claude-haiku-4-5-20251001", opts...)
}

func Claude3Haiku(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude3Haiku(opts...)
}

func Claude3Opus(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude3Opus(opts...)
}

func Claude35Sonnet(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude35Sonnet(opts...)
}

func Claude35Haiku(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude35Haiku(opts...)
}

func Claude4Sonnet(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude4Sonnet(opts...)
}

func Claude4Opus(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude4Opus(opts...)
}

func Claude45Sonnet(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude45Sonnet(opts...)
}

func Claude45Opus(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude45Opus(opts...)
}

func Claude46Sonnet(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude46Sonnet(opts...)
}

func Claude46Opus(opts ...ChatOption) provider.LanguageModel {
	return NewProvider().Claude46Opus(opts...)
}
