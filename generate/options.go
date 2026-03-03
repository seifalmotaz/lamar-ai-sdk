package generate

import (
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// Config holds the configuration for text generation.
type Config struct {
	// System is the system prompt to prepend to the conversation.
	System string
	// Messages is the conversation history.
	Messages []provider.Message
	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int
	// Temperature controls randomness in generation (0.0 to 2.0).
	Temperature float64
	// TopP controls nucleus sampling (0.0 to 1.0).
	TopP float64
	// TopK controls top-k sampling.
	TopK int
	// StopSequences stops generation when encountered.
	StopSequences []string
	// Tools are the tools available for the model to call.
	Tools []provider.ToolDefinition
	// ToolChoice controls how the model uses tools.
	ToolChoice provider.ToolChoice
	// Seed enables deterministic sampling.
	Seed *int
	// ResponseFormat controls the format of the response.
	ResponseFormat *provider.ResponseFormat
	// Timeout overrides the default timeout.
	Timeout *time.Duration
	// Logger provides logging for the request.
	Logger provider.Logger
	// Metrics collects metrics for the request.
	Metrics provider.MetricsCollector
	// ProviderConfigs allows provider-specific configuration.
	ProviderConfigs map[string]any
}

// Option is a functional option for configuring generation.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		ProviderConfigs: make(map[string]any),
	}
}

// System sets the system prompt for the generation.
func System(prompt string) Option {
	return func(c *Config) {
		c.System = prompt
	}
}

// Messages sets the conversation history.
func Messages(msgs ...provider.Message) Option {
	return func(c *Config) {
		c.Messages = msgs
	}
}

// MaxTokens sets the maximum number of tokens to generate.
func MaxTokens(n int) Option {
	return func(c *Config) {
		c.MaxTokens = n
	}
}

// Temperature sets the sampling temperature (0.0 to 2.0).
// Higher values make output more random, lower values make it more deterministic.
func Temperature(t float64) Option {
	return func(c *Config) {
		c.Temperature = t
	}
}

// TopP sets the nucleus sampling parameter (0.0 to 1.0).
func TopP(p float64) Option {
	return func(c *Config) {
		c.TopP = p
	}
}

// TopK sets the top-k sampling parameter.
func TopK(k int) Option {
	return func(c *Config) {
		c.TopK = k
	}
}

// StopSequences sets the stop sequences for generation.
func StopSequences(seqs ...string) Option {
	return func(c *Config) {
		c.StopSequences = seqs
	}
}

// Tools sets the tools available for the model to call.
func Tools(tools ...provider.ToolDefinition) Option {
	return func(c *Config) {
		c.Tools = tools
	}
}

// ToolChoice sets how the model should use tools.
func ToolChoice(tc provider.ToolChoice) Option {
	return func(c *Config) {
		c.ToolChoice = tc
	}
}

// Seed sets a seed for deterministic sampling.
func Seed(seed int) Option {
	return func(c *Config) {
		c.Seed = &seed
	}
}

// ResponseFormat sets the response format for the generation.
func ResponseFormat(rf provider.ResponseFormat) Option {
	return func(c *Config) {
		c.ResponseFormat = &rf
	}
}

// WithTimeout sets a custom timeout for the request.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = &d
	}
}

// WithNoTimeout disables the default timeout.
func WithNoTimeout() Option {
	return func(c *Config) {
		c.Timeout = new(time.Duration)
	}
}

// WithLogger sets a logger for the request.
func WithLogger(l provider.Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}

// WithMetrics sets a metrics collector for the request.
func WithMetrics(m provider.MetricsCollector) Option {
	return func(c *Config) {
		c.Metrics = m
	}
}
