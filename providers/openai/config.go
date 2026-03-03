package openai

import (
	"github.com/seifalmotaz/lamar-sdk/generate"
)

// ChatConfig holds OpenAI-specific configuration for chat completions.
type ChatConfig struct {
	// LogitBias modifies the likelihood of specified tokens appearing.
	LogitBias map[int]float64
	// ReasoningEffort controls the reasoning effort for O1 models ("low", "medium", "high").
	ReasoningEffort string
	// User is a unique identifier for the end-user.
	User string
}

// ChatOption is a functional option for configuring OpenAI chat completions.
type ChatOption func(*ChatConfig)

// WithLogitBias sets the logit bias for specific tokens.
func WithLogitBias(bias map[int]float64) ChatOption {
	return func(c *ChatConfig) {
		c.LogitBias = bias
	}
}

// WithReasoningEffort sets the reasoning effort level for O1 models.
func WithReasoningEffort(effort string) ChatOption {
	return func(c *ChatConfig) {
		c.ReasoningEffort = effort
	}
}

// WithUser sets the end-user identifier for the request.
func WithUser(user string) ChatOption {
	return func(c *ChatConfig) {
		c.User = user
	}
}

// NewChatConfig creates a new ChatConfig with the given options.
func NewChatConfig(opts ...ChatOption) *ChatConfig {
	c := &ChatConfig{}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Apply applies the configuration to a ChatModel.
func (c *ChatConfig) Apply(model *ChatModel) {
	model.config = *c
}

// WithOpenAIConfig creates a generate.Option that applies OpenAI-specific configuration.
//
// Example:
//
//	result, err := generate.Generate(ctx, model, "Hello",
//	    openai.WithOpenAIConfig(
//	        openai.WithReasoningEffort("high"),
//	    ),
//	)
func WithOpenAIConfig(opts ...ChatOption) generate.Option {
	c := NewChatConfig(opts...)
	return func(gc *generate.Config) {
		if gc.ProviderConfigs == nil {
			gc.ProviderConfigs = make(map[string]any)
		}
		gc.ProviderConfigs["openai"] = c
	}
}
