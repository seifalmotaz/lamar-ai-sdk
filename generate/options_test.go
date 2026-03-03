package generate

import (
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestOptionsSystem(t *testing.T) {
	cfg := defaultConfig()
	System("You are helpful")(cfg)
	if cfg.System != "You are helpful" {
		t.Errorf("System() = %q, want %q", cfg.System, "You are helpful")
	}
}

func TestOptionsMessages(t *testing.T) {
	msgs := []provider.Message{
		provider.UserMessage("hello"),
		provider.AssistantMessage("hi"),
	}
	cfg := defaultConfig()
	Messages(msgs...)(cfg)
	if len(cfg.Messages) != 2 {
		t.Errorf("Messages() = %d messages, want 2", len(cfg.Messages))
	}
}

func TestOptionsMaxTokens(t *testing.T) {
	cfg := defaultConfig()
	MaxTokens(100)(cfg)
	if cfg.MaxTokens != 100 {
		t.Errorf("MaxTokens() = %d, want 100", cfg.MaxTokens)
	}
}

func TestOptionsTemperature(t *testing.T) {
	cfg := defaultConfig()
	Temperature(0.7)(cfg)
	if cfg.Temperature != 0.7 {
		t.Errorf("Temperature() = %f, want 0.7", cfg.Temperature)
	}
}

func TestOptionsTopP(t *testing.T) {
	cfg := defaultConfig()
	TopP(0.9)(cfg)
	if cfg.TopP != 0.9 {
		t.Errorf("TopP() = %f, want 0.9", cfg.TopP)
	}
}

func TestOptionsTopK(t *testing.T) {
	cfg := defaultConfig()
	TopK(50)(cfg)
	if cfg.TopK != 50 {
		t.Errorf("TopK() = %d, want 50", cfg.TopK)
	}
}

func TestOptionsStopSequences(t *testing.T) {
	cfg := defaultConfig()
	StopSequences("STOP", "END")(cfg)
	if len(cfg.StopSequences) != 2 {
		t.Errorf("StopSequences() = %d sequences, want 2", len(cfg.StopSequences))
	}
	if cfg.StopSequences[0] != "STOP" || cfg.StopSequences[1] != "END" {
		t.Errorf("StopSequences() = %v, want [STOP, END]", cfg.StopSequences)
	}
}

func TestOptionsTools(t *testing.T) {
	tools := []provider.ToolDefinition{
		{Name: "tool1", Description: "desc1"},
		{Name: "tool2", Description: "desc2"},
	}
	cfg := defaultConfig()
	Tools(tools...)(cfg)
	if len(cfg.Tools) != 2 {
		t.Errorf("Tools() = %d tools, want 2", len(cfg.Tools))
	}
}

func TestOptionsToolChoice(t *testing.T) {
	cfg := defaultConfig()
	ToolChoice(provider.ToolChoiceAuto())(cfg)
	if cfg.ToolChoice.Type != "auto" {
		t.Errorf("ToolChoice() = %q, want auto", cfg.ToolChoice.Type)
	}
}

func TestOptionsSeed(t *testing.T) {
	cfg := defaultConfig()
	Seed(42)(cfg)
	if cfg.Seed == nil || *cfg.Seed != 42 {
		t.Errorf("Seed() = %v, want 42", cfg.Seed)
	}
}

func TestOptionsResponseFormat(t *testing.T) {
	cfg := defaultConfig()
	rf := provider.ResponseFormatJSON()
	ResponseFormat(rf)(cfg)
	if cfg.ResponseFormat == nil || cfg.ResponseFormat.Type != "json_object" {
		t.Errorf("ResponseFormat() = %v, want json_object", cfg.ResponseFormat)
	}
}

func TestOptionsWithTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithTimeout(60 * time.Second)(cfg)
	if cfg.Timeout == nil || *cfg.Timeout != 60*time.Second {
		t.Errorf("WithTimeout() = %v, want 60s", cfg.Timeout)
	}
}

func TestOptionsWithNoTimeout(t *testing.T) {
	cfg := defaultConfig()
	WithNoTimeout()(cfg)
	if cfg.Timeout == nil {
		t.Error("WithNoTimeout() should set Timeout to non-nil pointer to 0")
	}
	if *cfg.Timeout != 0 {
		t.Errorf("WithNoTimeout() = %v, want 0", *cfg.Timeout)
	}
}

func TestOptionsWithLogger(t *testing.T) {
	cfg := defaultConfig()
	logger := &noopLogger{}
	WithLogger(logger)(cfg)
	if cfg.Logger != logger {
		t.Error("WithLogger() did not set logger")
	}
}

func TestOptionsWithMetrics(t *testing.T) {
	cfg := defaultConfig()
	metrics := provider.NoopMetrics{}
	WithMetrics(metrics)(cfg)
	if cfg.Metrics != metrics {
		t.Error("WithMetrics() did not set metrics")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg.ProviderConfigs == nil {
		t.Error("defaultConfig() ProviderConfigs should not be nil")
	}
	if len(cfg.ProviderConfigs) != 0 {
		t.Error("defaultConfig() ProviderConfigs should be empty")
	}
}

func TestMultipleOptions(t *testing.T) {
	cfg := defaultConfig()
	MaxTokens(100)(
		cfg,
	)
	Temperature(0.5)(cfg)
	TopP(0.9)(cfg)

	if cfg.MaxTokens != 100 {
		t.Errorf("MaxTokens = %d, want 100", cfg.MaxTokens)
	}
	if cfg.Temperature != 0.5 {
		t.Errorf("Temperature = %f, want 0.5", cfg.Temperature)
	}
	if cfg.TopP != 0.9 {
		t.Errorf("TopP = %f, want 0.9", cfg.TopP)
	}
}

type noopLogger struct{}

func (l *noopLogger) Debug(msg string, args ...any) {}
func (l *noopLogger) Info(msg string, args ...any)  {}
func (l *noopLogger) Warn(msg string, args ...any)  {}
func (l *noopLogger) Error(msg string, args ...any) {}
