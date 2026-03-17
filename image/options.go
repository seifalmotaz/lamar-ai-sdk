package image

import (
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type Config struct {
	Size       string
	Quality    string
	Format     string
	Background string
	N          int
	Timeout    time.Duration
	Logger     provider.Logger
	Metrics    provider.MetricsCollector
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}

func WithSize(size string) Option {
	return func(c *Config) {
		c.Size = size
	}
}

func WithQuality(quality string) Option {
	return func(c *Config) {
		c.Quality = quality
	}
}

func WithFormat(format string) Option {
	return func(c *Config) {
		c.Format = format
	}
}

func WithBackground(background string) Option {
	return func(c *Config) {
		c.Background = background
	}
}

func WithN(n int) Option {
	return func(c *Config) {
		c.N = n
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
