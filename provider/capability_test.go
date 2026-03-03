package provider

import (
	"testing"
)

func TestCapabilityConstants(t *testing.T) {
	capabilities := []struct {
		cap  Capability
		want string
	}{
		{CapStreaming, "streaming"},
		{CapTools, "tools"},
		{CapVision, "vision"},
		{CapAudio, "audio"},
		{CapJSON, "json"},
		{CapReasoning, "reasoning"},
	}

	for _, tt := range capabilities {
		t.Run(tt.want, func(t *testing.T) {
			if string(tt.cap) != tt.want {
				t.Errorf("Capability = %q, want %q", tt.cap, tt.want)
			}
		})
	}
}

func TestModelInfoHasCapability(t *testing.T) {
	info := ModelInfo{
		Provider:     "openai",
		ModelID:      "gpt-4o",
		Capabilities: []Capability{CapStreaming, CapTools, CapVision, CapAudio},
		MaxTokens:    16384,
		ContextSize:  128000,
	}

	tests := []struct {
		name     string
		cap      Capability
		expected bool
	}{
		{"has streaming", CapStreaming, true},
		{"has tools", CapTools, true},
		{"has vision", CapVision, true},
		{"has audio", CapAudio, true},
		{"missing json", CapJSON, false},
		{"missing reasoning", CapReasoning, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := info.HasCapability(tt.cap); got != tt.expected {
				t.Errorf("HasCapability(%q) = %v, want %v", tt.cap, got, tt.expected)
			}
		})
	}
}

func TestModelInfoEmptyCapabilities(t *testing.T) {
	info := ModelInfo{
		Provider:     "test",
		ModelID:      "test-model",
		Capabilities: []Capability{},
	}

	if info.HasCapability(CapStreaming) {
		t.Error("HasCapability should return false for empty capabilities")
	}
}

type mockModelWithInfo struct {
	provider string
	modelID  string
	info     ModelInfo
}

func (m *mockModelWithInfo) Provider() string { return m.provider }
func (m *mockModelWithInfo) ModelID() string  { return m.modelID }
func (m *mockModelWithInfo) Info() ModelInfo  { return m.info }
func (m *mockModelWithInfo) HasCapability(cap Capability) bool {
	return m.info.HasCapability(cap)
}

func TestModelWithInfoInterface(t *testing.T) {
	info := ModelInfo{
		Provider:     "openai",
		ModelID:      "gpt-4o",
		Capabilities: []Capability{CapStreaming, CapTools, CapVision},
	}

	model := &mockModelWithInfo{
		provider: "openai",
		modelID:  "gpt-4o",
		info:     info,
	}

	var _ Model = model
	var _ ModelWithInfo = model

	if model.Provider() != "openai" {
		t.Errorf("Provider() = %q, want %q", model.Provider(), "openai")
	}
	if model.ModelID() != "gpt-4o" {
		t.Errorf("ModelID() = %q, want %q", model.ModelID(), "gpt-4o")
	}
	if !model.HasCapability(CapVision) {
		t.Error("HasCapability(CapVision) should be true")
	}
}

func TestHasCapability(t *testing.T) {
	t.Run("ModelWithInfo", func(t *testing.T) {
		info := ModelInfo{
			Provider:     "openai",
			ModelID:      "gpt-4o",
			Capabilities: []Capability{CapStreaming, CapTools},
		}
		model := &mockModelWithInfo{info: info}

		if !HasCapability(model, CapStreaming) {
			t.Error("HasCapability(CapStreaming) should be true")
		}
		if !HasCapability(model, CapTools) {
			t.Error("HasCapability(CapTools) should be true")
		}
		if HasCapability(model, CapVision) {
			t.Error("HasCapability(CapVision) should be false")
		}
	})

	t.Run("Streamer", func(t *testing.T) {
		streamer := &mockStreamer{mockModel: &mockModel{}}

		if !HasCapability(streamer, CapStreaming) {
			t.Error("Streamer should have CapStreaming")
		}
		if HasCapability(streamer, CapTools) {
			t.Error("Streamer should not have CapTools (fallback returns false)")
		}
	})

	t.Run("Plain Model", func(t *testing.T) {
		model := &mockModel{}

		if HasCapability(model, CapStreaming) {
			t.Error("Plain model should not have any capabilities")
		}
	})
}

func TestGetModelInfo(t *testing.T) {
	t.Run("ModelWithInfo", func(t *testing.T) {
		expectedInfo := ModelInfo{
			Provider:     "openai",
			ModelID:      "gpt-4o",
			Capabilities: []Capability{CapStreaming},
			MaxTokens:    16384,
		}
		model := &mockModelWithInfo{info: expectedInfo}

		info, ok := GetModelInfo(model)
		if !ok {
			t.Fatal("GetModelInfo should return true for ModelWithInfo")
		}
		if info.Provider != "openai" {
			t.Errorf("Provider = %q, want %q", info.Provider, "openai")
		}
		if info.MaxTokens != 16384 {
			t.Errorf("MaxTokens = %d, want %d", info.MaxTokens, 16384)
		}
	})

	t.Run("Plain Model", func(t *testing.T) {
		model := &mockModel{}

		_, ok := GetModelInfo(model)
		if ok {
			t.Error("GetModelInfo should return false for plain Model")
		}
	})
}

func TestBaseModel(t *testing.T) {
	model := NewBaseModel("openai", "gpt-4o", CapStreaming, CapTools, CapVision)

	var _ Model = model
	var _ ModelWithInfo = model

	t.Run("Provider", func(t *testing.T) {
		if model.Provider() != "openai" {
			t.Errorf("Provider() = %q, want %q", model.Provider(), "openai")
		}
	})

	t.Run("ModelID", func(t *testing.T) {
		if model.ModelID() != "gpt-4o" {
			t.Errorf("ModelID() = %q, want %q", model.ModelID(), "gpt-4o")
		}
	})

	t.Run("Info", func(t *testing.T) {
		info := model.Info()
		if info.Provider != "openai" {
			t.Errorf("Info().Provider = %q, want %q", info.Provider, "openai")
		}
		if info.ModelID != "gpt-4o" {
			t.Errorf("Info().ModelID = %q, want %q", info.ModelID, "gpt-4o")
		}
		if len(info.Capabilities) != 3 {
			t.Errorf("Info().Capabilities length = %d, want 3", len(info.Capabilities))
		}
	})

	t.Run("HasCapability", func(t *testing.T) {
		if !model.HasCapability(CapStreaming) {
			t.Error("HasCapability(CapStreaming) should be true")
		}
		if !model.HasCapability(CapTools) {
			t.Error("HasCapability(CapTools) should be true")
		}
		if !model.HasCapability(CapVision) {
			t.Error("HasCapability(CapVision) should be true")
		}
		if model.HasCapability(CapAudio) {
			t.Error("HasCapability(CapAudio) should be false")
		}
	})
}

func TestBaseModelSetters(t *testing.T) {
	model := NewBaseModel("openai", "gpt-4o", CapStreaming)

	t.Run("SetMaxTokens", func(t *testing.T) {
		model.SetMaxTokens(4096)
		info := model.Info()
		if info.MaxTokens != 4096 {
			t.Errorf("MaxTokens = %d, want 4096", info.MaxTokens)
		}
	})

	t.Run("SetContextSize", func(t *testing.T) {
		model.SetContextSize(8192)
		info := model.Info()
		if info.ContextSize != 8192 {
			t.Errorf("ContextSize = %d, want 8192", info.ContextSize)
		}
	})
}

func TestBaseModelNoCapabilities(t *testing.T) {
	model := NewBaseModel("test", "test-model")

	if model.HasCapability(CapStreaming) {
		t.Error("Model with no capabilities should not have CapStreaming")
	}
}
