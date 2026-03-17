package transcription

import (
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type Config struct {
	Language string
	Prompt   string
	Timeout  time.Duration
	Logger   provider.Logger
	Metrics  provider.MetricsCollector
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithLanguage(language string) Option {
	return func(c *Config) {
		c.Language = language
	}
}

func WithPrompt(prompt string) Option {
	return func(c *Config) {
		c.Prompt = prompt
	}
}

func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

func WithNoTimeout() Option {
	return func(c *Config) {
		c.Timeout = -1
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
