package anthropic

type ChatConfig struct {
	Thinking               *ThinkingConfig
	SendReasoning          *bool
	StructuredOutputMode   string
	DisableParallelToolUse bool
	ToolStreaming          *bool
	CacheControl           *CacheControlConfig
	Speed                  string
	Effort                 string
	Container              *ContainerConfig
	MCPServers             []MCPServerConfig
}

type ThinkingConfig struct {
	Type         string
	BudgetTokens int
}

type CacheControlConfig struct {
	Type string
	TTL  string
}

type ContainerConfig struct {
	ID     string
	Skills []ContainerSkill
}

type ContainerSkill struct {
	Type    string
	SkillID string
	Version string
}

type MCPServerConfig struct {
	Type               string
	Name               string
	URL                string
	AuthorizationToken string
	ToolConfiguration  *MCPToolConfiguration
}

type MCPToolConfiguration struct {
	Enabled      bool
	AllowedTools []string
}

type ChatOption func(*ChatConfig)

func ThinkingAdaptive() ChatOption {
	return func(c *ChatConfig) {
		c.Thinking = &ThinkingConfig{Type: "adaptive"}
	}
}

func ThinkingEnabled(budgetTokens int) ChatOption {
	return func(c *ChatConfig) {
		if budgetTokens < 1024 {
			budgetTokens = 1024
		}
		c.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: budgetTokens,
		}
	}
}

func ThinkingDisabled() ChatOption {
	return func(c *ChatConfig) {
		c.Thinking = &ThinkingConfig{Type: "disabled"}
	}
}

func SendReasoning(send bool) ChatOption {
	return func(c *ChatConfig) {
		c.SendReasoning = &send
	}
}

func StructuredOutputMode(mode string) ChatOption {
	return func(c *ChatConfig) {
		c.StructuredOutputMode = mode
	}
}

func DisableParallelToolUse() ChatOption {
	return func(c *ChatConfig) {
		c.DisableParallelToolUse = true
	}
}

func ToolStreaming(enabled bool) ChatOption {
	return func(c *ChatConfig) {
		c.ToolStreaming = &enabled
	}
}

func CacheControl(ttl string) ChatOption {
	return func(c *ChatConfig) {
		c.CacheControl = &CacheControlConfig{
			Type: "ephemeral",
			TTL:  ttl,
		}
	}
}

func Speed(mode string) ChatOption {
	return func(c *ChatConfig) {
		c.Speed = mode
	}
}

func Effort(level string) ChatOption {
	return func(c *ChatConfig) {
		c.Effort = level
	}
}

func WithContainer(id string, skills []ContainerSkill) ChatOption {
	return func(c *ChatConfig) {
		c.Container = &ContainerConfig{
			ID:     id,
			Skills: skills,
		}
	}
}

func WithMCPServers(servers []MCPServerConfig) ChatOption {
	return func(c *ChatConfig) {
		c.MCPServers = servers
	}
}

func mergeChatConfig(opts ...ChatOption) *ChatConfig {
	config := &ChatConfig{}
	for _, opt := range opts {
		opt(config)
	}
	return config
}
