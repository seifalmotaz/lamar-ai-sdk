package anthropic

import (
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestChatModelImplementsGenerator(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.Model("claude-3-haiku-20240307")

	var _ provider.Generator = model
}

func TestChatModelImplementsStreamer(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.StreamingModel("claude-3-haiku-20240307")

	var _ provider.Streamer = model
}

func TestChatModelImplementsLanguageModel(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.StreamingModel("claude-3-haiku-20240307")

	var _ provider.LanguageModel = model
}

func TestChatModelImplementsModel(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.Model("claude-3-haiku-20240307")

	var _ provider.Model = model
}

func TestProviderImplementsCapabilityChecks(t *testing.T) {
	p := NewProvider(APIKey("test-key"))
	model := p.StreamingModel("claude-3-haiku-20240307")

	if !provider.CanGenerate(model) {
		t.Error("expected model to implement Generator")
	}
	if !provider.CanStream(model) {
		t.Error("expected model to implement Streamer")
	}
	if !provider.IsLanguageModel(model) {
		t.Error("expected model to implement LanguageModel")
	}
}
