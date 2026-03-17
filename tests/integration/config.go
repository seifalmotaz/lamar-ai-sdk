package integration

import (
	"os"
	"time"
)

type TestConfig struct {
	OpenAIAPIKey    string
	AnthropicAPIKey string
	Timeout         time.Duration
	BaseURL         string
}

func LoadTestConfig() *TestConfig {
	timeout := 60 * time.Second
	if timeoutStr := os.Getenv("TEST_TIMEOUT"); timeoutStr != "" {
		if d, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = d
		}
	}

	return &TestConfig{
		OpenAIAPIKey:    os.Getenv("OPENAI_API_KEY"),
		AnthropicAPIKey: os.Getenv("ANTHROPIC_API_KEY"),
		Timeout:         timeout,
		BaseURL:         os.Getenv("TEST_BASE_URL"),
	}
}

func (c *TestConfig) HasOpenAI() bool {
	return c.OpenAIAPIKey != ""
}

func (c *TestConfig) HasAnthropic() bool {
	return c.AnthropicAPIKey != ""
}

func (c *TestConfig) WithTimeout(fn func()) {
	if c.Timeout > 0 {
		done := make(chan struct{})
		go func() {
			defer close(done)
			fn()
		}()
		select {
		case <-done:
		case <-time.After(c.Timeout):
			panic("test timed out")
		}
	} else {
		fn()
	}
}

var defaultConfig *TestConfig
var configOnce bool

func GetTestConfig() *TestConfig {
	if !configOnce {
		defaultConfig = LoadTestConfig()
		configOnce = true
	}
	return defaultConfig
}
