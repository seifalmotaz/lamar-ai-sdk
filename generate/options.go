package generate

import (
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type Config struct {
	System         string
	Messages       []provider.Message
	MaxTokens      int
	Temperature    float64
	TopP           float64
	TopK           int
	StopSequences  []string
	Tools          []provider.ToolDefinition
	ToolChoice     provider.ToolChoice
	Seed           *int
	ResponseFormat *provider.ResponseFormat

	Timeout *time.Duration

	Logger  provider.Logger
	Metrics provider.MetricsCollector

	ProviderConfigs map[string]any
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		ProviderConfigs: make(map[string]any),
	}
}

func System(prompt string) Option {
	return func(c *Config) {
		c.System = prompt
	}
}

func Messages(msgs ...provider.Message) Option {
	return func(c *Config) {
		c.Messages = msgs
	}
}

func MaxTokens(n int) Option {
	return func(c *Config) {
		c.MaxTokens = n
	}
}

func Temperature(t float64) Option {
	return func(c *Config) {
		c.Temperature = t
	}
}

func TopP(p float64) Option {
	return func(c *Config) {
		c.TopP = p
	}
}

func TopK(k int) Option {
	return func(c *Config) {
		c.TopK = k
	}
}

func StopSequences(seqs ...string) Option {
	return func(c *Config) {
		c.StopSequences = seqs
	}
}

func Tools(tools ...provider.ToolDefinition) Option {
	return func(c *Config) {
		c.Tools = tools
	}
}

func ToolChoice(tc provider.ToolChoice) Option {
	return func(c *Config) {
		c.ToolChoice = tc
	}
}

func Seed(seed int) Option {
	return func(c *Config) {
		c.Seed = &seed
	}
}

func ResponseFormat(rf provider.ResponseFormat) Option {
	return func(c *Config) {
		c.ResponseFormat = &rf
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = &d
	}
}

func WithNoTimeout() Option {
	return func(c *Config) {
		c.Timeout = new(time.Duration)
	}
}

func WithLogger(l provider.Logger) Option {
	return func(c *Config) {
		c.Logger = l
	}
}

func WithMetrics(m provider.MetricsCollector) Option {
	return func(c *Config) {
		c.Metrics = m
	}
}
