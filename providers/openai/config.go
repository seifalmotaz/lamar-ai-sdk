package openai

import (
	"github.com/seifalmotaz/lamar-sdk/generate"
)

type ChatConfig struct {
	LogitBias       map[int]float64
	ReasoningEffort string
	User            string
}

type ChatOption func(*ChatConfig)

func WithLogitBias(bias map[int]float64) ChatOption {
	return func(c *ChatConfig) {
		c.LogitBias = bias
	}
}

func WithReasoningEffort(effort string) ChatOption {
	return func(c *ChatConfig) {
		c.ReasoningEffort = effort
	}
}

func WithUser(user string) ChatOption {
	return func(c *ChatConfig) {
		c.User = user
	}
}

func NewChatConfig(opts ...ChatOption) *ChatConfig {
	c := &ChatConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *ChatConfig) Apply(model *ChatModel) {
	model.config = *c
}

func WithOpenAIConfig(opts ...ChatOption) generate.Option {
	c := NewChatConfig(opts...)
	return func(gc *generate.Config) {
		if gc.ProviderConfigs == nil {
			gc.ProviderConfigs = make(map[string]any)
		}
		gc.ProviderConfigs["openai"] = c
	}
}
