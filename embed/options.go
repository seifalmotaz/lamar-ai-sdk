package embed

import (
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type Config struct {
	Timeout time.Duration
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
