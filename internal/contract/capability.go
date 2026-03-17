package contract

import (
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type TestCapability string

const (
	CapTextGeneration   TestCapability = "textGeneration"
	CapObjectGeneration TestCapability = "objectGeneration"
	CapToolCalls        TestCapability = "toolCalls"
	CapEmbedding        TestCapability = "embedding"
	CapImageGeneration  TestCapability = "imageGeneration"
	CapVision           TestCapability = "vision"
	CapAudioInput       TestCapability = "audioInput"
	CapTranscription    TestCapability = "transcription"
	CapSpeech           TestCapability = "speech"
	CapReasoning        TestCapability = "reasoning"
)

type ModelWithCapabilities struct {
	Model        provider.Model
	Capabilities []TestCapability
}

func HasTestCapability(m ModelWithCapabilities, cap TestCapability) bool {
	for _, c := range m.Capabilities {
		if c == cap {
			return true
		}
	}
	return false
}

func HasAllTestCapabilities(m ModelWithCapabilities, caps []TestCapability) bool {
	for _, cap := range caps {
		if !HasTestCapability(m, cap) {
			return false
		}
	}
	return true
}

func RequireCapability(t *testing.T, m ModelWithCapabilities, cap TestCapability) {
	t.Helper()
	if !HasTestCapability(m, cap) {
		t.Skipf("model %s lacks capability: %s", m.Model.ModelID(), cap)
	}
}

func RequireAllCapabilities(t *testing.T, m ModelWithCapabilities, caps []TestCapability) {
	t.Helper()
	if !HasAllTestCapabilities(m, caps) {
		t.Skipf("model %s lacks capabilities: %v", m.Model.ModelID(), caps)
	}
}

func ProviderCapability(cap TestCapability) provider.Capability {
	switch cap {
	case CapVision:
		return provider.CapVision
	case CapImageGeneration:
		return provider.CapImageGeneration
	case CapTranscription:
		return provider.CapTranscription
	case CapSpeech:
		return provider.CapSpeech
	case CapReasoning:
		return provider.CapReasoning
	default:
		return ""
	}
}

func ModelWithCaps(model provider.Model, caps ...TestCapability) ModelWithCapabilities {
	return ModelWithCapabilities{
		Model:        model,
		Capabilities: caps,
	}
}
