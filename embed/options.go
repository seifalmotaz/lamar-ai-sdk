package embed

import (
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Config holds the configuration for embedding operations.
type Config struct {
	// Timeout is the timeout for the request (0 uses default).
	Timeout time.Duration
	// Logger provides logging for the request.
	Logger provider.Logger
	// Metrics collects metrics for the request.
	Metrics provider.MetricsCollector
	// ProviderConfigs allows provider-specific configuration.
	ProviderConfigs map[string]any
}

// Option is a functional option for configuring embedding.
type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		ProviderConfigs: make(map[string]any),
	}
}

// WithTimeout sets a custom timeout for the request.
func WithTimeout(d time.Duration) Option {
	return func(c *Config) {
		c.Timeout = d
	}
}

// WithNoTimeout disables the default timeout.
func WithNoTimeout() Option {
	return func(c *Config) {
		c.Timeout = -1
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
