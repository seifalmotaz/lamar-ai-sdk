package speech

import (
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type Config struct {
	Voice        string
	Format       string
	Speed        float64
	Instructions string
	Timeout      time.Duration
	Logger       provider.Logger
	Metrics      provider.MetricsCollector
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithVoice(voice string) Option {
	return func(c *Config) {
		c.Voice = voice
	}
}

func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

func WithSpeed(speed float64) Option {
	return func(c *Config) {
		c.Speed = speed
	}
}

func WithInstructions(instructions string) Option {
	return func(c *Config) {
		c.Instructions = instructions
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
