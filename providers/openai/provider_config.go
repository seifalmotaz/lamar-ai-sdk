package openai

import (
	"net/http"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/httpx"
)

// Config holds provider configuration.
type Config struct {
	APIKey  string
	BaseURL string
}

// NewProviderWithConfig creates a new provider with explicit configuration.
func NewProviderWithConfig(cfg Config) *Provider {
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	p := &Provider{
		client:  httpx.NewClient(baseURL, http.DefaultClient),
		apiKey:  cfg.APIKey,
		baseURL: baseURL,
	}
	p.client.SetHeader("Authorization", "Bearer "+cfg.APIKey)
	return p
}
