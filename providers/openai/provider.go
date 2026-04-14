package openai

import (
	"net/http"
	"os"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/httpx"
	"github.com/seifalmotaz/lamar-ai-sdk/middleware"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
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

	// Middleware support
	middlewares []middleware.Middleware
	wrapper     *middleware.Wrapper
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

// WithMiddleware adds middleware to the provider's processing chain.
// Middleware is applied in order for all generate, stream, and embed operations.
//
// Example:
//
//	client := openai.NewProvider(
//	    openai.WithMiddleware(
//	        middleware.TimeoutWithDefault(30 * time.Second),
//	        middleware.Logging(logger),
//	    ),
//	)
func WithMiddleware(middlewares ...middleware.Middleware) Option {
	return func(p *Provider) {
		p.middlewares = append(p.middlewares, middlewares...)
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

	p.wrapper = middleware.NewWrapper("openai", p.middlewares)

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

// Image returns an image generation model with the given ID.
func (p *Provider) Image(id string, opts ...ImageOption) provider.ImageModel {
	return NewImageModel(id, p, opts...)
}

// Transcription returns a transcription model with the given ID.
func (p *Provider) Transcription(id string, opts ...TranscriptionOption) provider.TranscriptionModel {
	return NewTranscriptionModel(id, p, opts...)
}

// Speech returns a speech synthesis model with the given ID.
func (p *Provider) Speech(id string, opts ...SpeechOption) provider.SpeechModel {
	return NewSpeechModel(id, p, opts...)
}

// GPT5Mini returns a GPT-5-mini model with streaming support.
func (p *Provider) GPT5Mini() provider.LanguageModel {
	return p.StreamingModel("gpt-5-mini-2025-08-07")
}

// GPT51 returns a GPT-5.1 model with streaming support.
func (p *Provider) GPT51() provider.LanguageModel {
	return p.StreamingModel("gpt-5.1-2025-11-13")
}

// GPT52 returns a GPT-5.2 model with streaming support.
func (p *Provider) GPT52() provider.LanguageModel {
	return p.StreamingModel("gpt-5.2-2025-12-11")
}

// GPT54 returns a GPT-5.4 model with streaming support.
func (p *Provider) GPT54() provider.LanguageModel {
	return p.StreamingModel("gpt-5.4-2026-03-05")
}

// GPT4oAudioPreview returns a GPT-4o model with audio input support.
// This model can process audio content in messages.
func (p *Provider) GPT4oAudioPreview() provider.LanguageModel {
	return p.StreamingModel("gpt-4o-audio-preview")
}

// GPT4o returns a GPT-4o model with streaming support.
func (p *Provider) GPT4o() provider.LanguageModel {
	return p.StreamingModel("gpt-4o")
}

// GPT4oMini returns a GPT-4o-mini model with streaming support.
func (p *Provider) GPT4oMini() provider.LanguageModel {
	return p.StreamingModel("gpt-4o-mini")
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

// DALLE2 returns a DALL-E 2 image generation model.
func (p *Provider) DALLE2(opts ...ImageOption) provider.ImageModel {
	return p.Image("dall-e-2", opts...)
}

// DALLE3 returns a DALL-E 3 image generation model.
func (p *Provider) DALLE3(opts ...ImageOption) provider.ImageModel {
	return p.Image("dall-e-3", opts...)
}

// GPTImage1 returns a GPT Image 1 model.
func (p *Provider) GPTImage1(opts ...ImageOption) provider.ImageModel {
	return p.Image("gpt-image-1", opts...)
}

// Whisper1 returns a Whisper transcription model.
func (p *Provider) Whisper1(opts ...TranscriptionOption) provider.TranscriptionModel {
	return p.Transcription("whisper-1", opts...)
}

// TTS1 returns a TTS-1 speech synthesis model.
func (p *Provider) TTS1(opts ...SpeechOption) provider.SpeechModel {
	return p.Speech("tts-1", opts...)
}

// TTS1HD returns a TTS-1-HD speech synthesis model.
func (p *Provider) TTS1HD(opts ...SpeechOption) provider.SpeechModel {
	return p.Speech("tts-1-hd", opts...)
}

// GPT4oMiniTTS returns a GPT-4o-mini TTS model.
func (p *Provider) GPT4oMiniTTS(opts ...SpeechOption) provider.SpeechModel {
	return p.Speech("gpt-4o-mini-tts", opts...)
}

// GPT5Mini creates a GPT-5-mini model using the default provider.
func GPT5Mini() provider.LanguageModel {
	return NewProvider().GPT5Mini()
}

// GPT51 creates a GPT-5.1 model using the default provider.
func GPT51() provider.LanguageModel {
	return NewProvider().GPT51()
}

// GPT52 creates a GPT-5.2 model using the default provider.
func GPT52() provider.LanguageModel {
	return NewProvider().GPT52()
}

// GPT54 creates a GPT-5.4 model using the default provider.
func GPT54() provider.LanguageModel {
	return NewProvider().GPT54()
}

// GPT4oAudioPreview creates a GPT-4o-audio-preview model using the default provider.
func GPT4oAudioPreview() provider.LanguageModel {
	return NewProvider().GPT4oAudioPreview()
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
