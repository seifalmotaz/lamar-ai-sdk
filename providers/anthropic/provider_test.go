package anthropic

import (
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestNewProvider(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	if p == nil {
		t.Fatal("expected provider to be created")
	}
	if p.apiKey != "test-key" {
		t.Errorf("expected apiKey to be 'test-key', got %s", p.apiKey)
	}
}

func TestProviderModel(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.Model("claude-3-haiku-20240307")

	if model == nil {
		t.Fatal("expected model to be created")
	}

	chatModel := model.(*ChatModel)
	if chatModel.id != "claude-3-haiku-20240307" {
		t.Errorf("expected model id 'claude-3-haiku-20240307', got %s", chatModel.id)
	}
	if chatModel.Provider() != "anthropic" {
		t.Errorf("expected provider 'anthropic', got %s", chatModel.Provider())
	}
}

func TestProviderStreamingModel(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.StreamingModel("claude-3-5-sonnet-20241022")

	if model == nil {
		t.Fatal("expected model to be created")
	}

	chatModel := model.(*ChatModel)
	if chatModel.id != "claude-3-5-sonnet-20241022" {
		t.Errorf("expected model id 'claude-3-5-sonnet-20241022', got %s", chatModel.id)
	}

	var _ provider.LanguageModel = chatModel
}

func TestConvenienceModels(t *testing.T) {
	p := NewProvider(APIKey("test-key"))

	tests := []struct {
		name     string
		fn       func() provider.LanguageModel
		expected string
	}{
		{"Claude3Haiku", func() provider.LanguageModel { return p.Claude3Haiku() }, "claude-3-haiku-20240307"},
		{"Claude3Opus", func() provider.LanguageModel { return p.Claude3Opus() }, "claude-3-opus-20240229"},
		{"Claude35Sonnet", func() provider.LanguageModel { return p.Claude35Sonnet() }, "claude-3-5-sonnet-20241022"},
		{"Claude35Haiku", func() provider.LanguageModel { return p.Claude35Haiku() }, "claude-3-5-haiku-20241022"},
		{"Claude4Sonnet", func() provider.LanguageModel { return p.Claude4Sonnet() }, "claude-sonnet-4-20250514"},
		{"Claude4Opus", func() provider.LanguageModel { return p.Claude4Opus() }, "claude-opus-4-20250514"},
		{"Claude45Sonnet", func() provider.LanguageModel { return p.Claude45Sonnet() }, "claude-sonnet-4-5-20250929"},
		{"Claude45Opus", func() provider.LanguageModel { return p.Claude45Opus() }, "claude-opus-4-5-20251101"},
		{"Claude46Sonnet", func() provider.LanguageModel { return p.Claude46Sonnet() }, "claude-sonnet-4-6"},
		{"Claude46Opus", func() provider.LanguageModel { return p.Claude46Opus() }, "claude-opus-4-6"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := tt.fn()
			chatModel := model.(*ChatModel)
			if chatModel.id != tt.expected {
				t.Errorf("expected model id %q, got %q", tt.expected, chatModel.id)
			}
		})
	}
}

func TestChatConfig(t *testing.T) {
	tests := []struct {
		name  string
		opts  []ChatOption
		check func(*testing.T, *ChatConfig)
	}{
		{
			name: "ThinkingAdaptive",
			opts: []ChatOption{ThinkingAdaptive()},
			check: func(t *testing.T, c *ChatConfig) {
				if c.Thinking == nil || c.Thinking.Type != "adaptive" {
					t.Error("expected thinking type 'adaptive'")
				}
			},
		},
		{
			name: "ThinkingEnabled",
			opts: []ChatOption{ThinkingEnabled(2048)},
			check: func(t *testing.T, c *ChatConfig) {
				if c.Thinking == nil || c.Thinking.Type != "enabled" {
					t.Error("expected thinking type 'enabled'")
				}
				if c.Thinking.BudgetTokens != 2048 {
					t.Errorf("expected budget tokens 2048, got %d", c.Thinking.BudgetTokens)
				}
			},
		},
		{
			name: "ThinkingEnabledMinimumBudget",
			opts: []ChatOption{ThinkingEnabled(500)},
			check: func(t *testing.T, c *ChatConfig) {
				if c.Thinking.BudgetTokens < 1024 {
					t.Errorf("expected budget tokens >= 1024, got %d", c.Thinking.BudgetTokens)
				}
			},
		},
		{
			name: "DisableParallelToolUse",
			opts: []ChatOption{DisableParallelToolUse()},
			check: func(t *testing.T, c *ChatConfig) {
				if !c.DisableParallelToolUse {
					t.Error("expected DisableParallelToolUse to be true")
				}
			},
		},
		{
			name: "StructuredOutputMode",
			opts: []ChatOption{StructuredOutputMode("outputFormat")},
			check: func(t *testing.T, c *ChatConfig) {
				if c.StructuredOutputMode != "outputFormat" {
					t.Errorf("expected structured output mode 'outputFormat', got %s", c.StructuredOutputMode)
				}
			},
		},
		{
			name: "Speed",
			opts: []ChatOption{Speed("fast")},
			check: func(t *testing.T, c *ChatConfig) {
				if c.Speed != "fast" {
					t.Errorf("expected speed 'fast', got %s", c.Speed)
				}
			},
		},
		{
			name: "Effort",
			opts: []ChatOption{Effort("high")},
			check: func(t *testing.T, c *ChatConfig) {
				if c.Effort != "high" {
					t.Errorf("expected effort 'high', got %s", c.Effort)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := mergeChatConfig(tt.opts...)
			tt.check(t, config)
		})
	}
}

func TestConvertToolChoice(t *testing.T) {
	tests := []struct {
		name     string
		choice   provider.ToolChoice
		expected ToolChoice
	}{
		{
			name:     "auto",
			choice:   provider.ToolChoiceAuto(),
			expected: ToolChoice{Type: "auto"},
		},
		{
			name:     "required",
			choice:   provider.ToolChoiceRequired(),
			expected: ToolChoice{Type: "any"},
		},
		{
			name:     "tool",
			choice:   provider.ToolChoiceNamed("get_weather"),
			expected: ToolChoice{Type: "tool", Name: "get_weather"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertToolChoice(tt.choice, false)
			if result.Type != tt.expected.Type {
				t.Errorf("expected type %q, got %q", tt.expected.Type, result.Type)
			}
			if result.Name != tt.expected.Name {
				t.Errorf("expected name %q, got %q", tt.expected.Name, result.Name)
			}
		})
	}
}

func TestMapStopReason(t *testing.T) {
	tests := []struct {
		reason   string
		expected provider.FinishReason
	}{
		{"end_turn", provider.FinishReasonStop},
		{"max_tokens", provider.FinishReasonLength},
		{"stop_sequence", provider.FinishReasonStop},
		{"tool_use", provider.FinishReasonToolCalls},
		{"unknown", provider.FinishReasonStop},
	}

	for _, tt := range tests {
		t.Run(tt.reason, func(t *testing.T) {
			result := mapStopReason(tt.reason)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}
