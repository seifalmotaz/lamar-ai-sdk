package contract_test

import (
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/contract"
	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type mockModel struct {
	providerName string
	modelID      string
}

func (m *mockModel) Provider() string { return m.providerName }
func (m *mockModel) ModelID() string  { return m.modelID }

func TestHasTestCapability(t *testing.T) {
	model := &mockModel{providerName: "test", modelID: "test-model"}

	tests := []struct {
		name     string
		caps     []contract.TestCapability
		check    contract.TestCapability
		expected bool
	}{
		{
			name:     "has capability",
			caps:     []contract.TestCapability{contract.CapTextGeneration, contract.CapToolCalls},
			check:    contract.CapTextGeneration,
			expected: true,
		},
		{
			name:     "does not have capability",
			caps:     []contract.TestCapability{contract.CapTextGeneration},
			check:    contract.CapEmbedding,
			expected: false,
		},
		{
			name:     "empty capabilities",
			caps:     []contract.TestCapability{},
			check:    contract.CapTextGeneration,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := contract.ModelWithCapabilities{
				Model:        model,
				Capabilities: tt.caps,
			}
			result := contract.HasTestCapability(m, tt.check)
			if result != tt.expected {
				t.Errorf("HasTestCapability() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHasAllTestCapabilities(t *testing.T) {
	model := &mockModel{providerName: "test", modelID: "test-model"}

	tests := []struct {
		name     string
		caps     []contract.TestCapability
		check    []contract.TestCapability
		expected bool
	}{
		{
			name:     "has all capabilities",
			caps:     []contract.TestCapability{contract.CapTextGeneration, contract.CapToolCalls, contract.CapVision},
			check:    []contract.TestCapability{contract.CapTextGeneration, contract.CapToolCalls},
			expected: true,
		},
		{
			name:     "missing one capability",
			caps:     []contract.TestCapability{contract.CapTextGeneration},
			check:    []contract.TestCapability{contract.CapTextGeneration, contract.CapToolCalls},
			expected: false,
		},
		{
			name:     "empty check list",
			caps:     []contract.TestCapability{contract.CapTextGeneration},
			check:    []contract.TestCapability{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := contract.ModelWithCapabilities{
				Model:        model,
				Capabilities: tt.caps,
			}
			result := contract.HasAllTestCapabilities(m, tt.check)
			if result != tt.expected {
				t.Errorf("HasAllTestCapabilities() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestModelWithCaps(t *testing.T) {
	model := &mockModel{providerName: "test", modelID: "test-model"}

	m := contract.ModelWithCaps(model, contract.CapTextGeneration, contract.CapToolCalls)

	if m.Model != model {
		t.Error("ModelWithCaps did not set model correctly")
	}
	if len(m.Capabilities) != 2 {
		t.Errorf("expected 2 capabilities, got %d", len(m.Capabilities))
	}
	if !contract.HasTestCapability(m, contract.CapTextGeneration) {
		t.Error("expected CapTextGeneration capability")
	}
	if !contract.HasTestCapability(m, contract.CapToolCalls) {
		t.Error("expected CapToolCalls capability")
	}
}

func TestRequireCapability_Skips(t *testing.T) {
	model := &mockModel{providerName: "test", modelID: "test-model"}

	t.Run("skips when capability missing", func(t *testing.T) {
		m := contract.ModelWithCapabilities{
			Model:        model,
			Capabilities: []contract.TestCapability{contract.CapTextGeneration},
		}

		skipped := false
		t.Run("subtest", func(t *testing.T) {
			contract.RequireCapability(t, m, contract.CapEmbedding)
			skipped = true
		})

		if skipped {
			t.Error("test should have been skipped")
		}
	})

	t.Run("continues when capability present", func(t *testing.T) {
		m := contract.ModelWithCapabilities{
			Model:        model,
			Capabilities: []contract.TestCapability{contract.CapTextGeneration},
		}

		ran := false
		t.Run("subtest", func(t *testing.T) {
			contract.RequireCapability(t, m, contract.CapTextGeneration)
			ran = true
		})

		if !ran {
			t.Error("test should not have been skipped")
		}
	})
}

func TestProviderCapabilityMapping(t *testing.T) {
	tests := []struct {
		testCap contract.TestCapability
		want    provider.Capability
	}{
		{contract.CapVision, provider.CapVision},
		{contract.CapImageGeneration, provider.CapImageGeneration},
		{contract.CapTranscription, provider.CapTranscription},
		{contract.CapSpeech, provider.CapSpeech},
		{contract.CapReasoning, provider.CapReasoning},
		{contract.CapTextGeneration, ""},
		{contract.CapEmbedding, ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.testCap), func(t *testing.T) {
			got := contract.ProviderCapability(tt.testCap)
			if got != tt.want {
				t.Errorf("ProviderCapability(%s) = %v, want %v", tt.testCap, got, tt.want)
			}
		})
	}
}
