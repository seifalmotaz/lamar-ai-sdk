package provider

// Capability represents a specific ability or feature that a model supports.
// Capabilities are used to query model features like streaming, tools, vision, etc.
type Capability string

// Model capability constants.
// These define the features that a model may support.
const (
	// CapStreaming indicates the model supports streaming responses
	CapStreaming Capability = "streaming"
	// CapTools indicates the model supports function/tool calling
	CapTools Capability = "tools"
	// CapVision indicates the model supports image understanding
	CapVision Capability = "vision"
	// CapAudio indicates the model supports audio input/output
	CapAudio Capability = "audio"
	// CapJSON indicates the model supports structured JSON output
	CapJSON Capability = "json"
	// CapReasoning indicates the model supports reasoning (e.g., O1 models)
	CapReasoning Capability = "reasoning"
)

// ModelWithInfo is a Model that also provides capability information.
// Models can implement this interface to declare their capabilities
// without requiring type assertions.
type ModelWithInfo interface {
	Model
	Info() ModelInfo
	HasCapability(cap Capability) bool
}

// HasCapability checks if a model has a specific capability.
// If the model implements ModelWithInfo, it uses that interface.
// Otherwise, it falls back to interface checks (e.g., CapStreaming checks for Streamer).
func HasCapability(m Model, cap Capability) bool {
	if mi, ok := m.(ModelWithInfo); ok {
		return mi.HasCapability(cap)
	}
	switch cap {
	case CapStreaming:
		return CanStream(m)
	default:
		return false
	}
}

// GetModelInfo returns the ModelInfo for a model if it implements ModelWithInfo.
// If the model does not implement ModelWithInfo, it returns false.
func GetModelInfo(m Model) (ModelInfo, bool) {
	if mi, ok := m.(ModelWithInfo); ok {
		return mi.Info(), true
	}
	return ModelInfo{}, false
}

// BaseModel provides a base implementation of Model and ModelWithInfo.
// It can be embedded in provider-specific model implementations.
type BaseModel struct {
	providerName string
	modelID      string
	info         ModelInfo
}

// NewBaseModel creates a new BaseModel with the given provider name, model ID, and capabilities.
func NewBaseModel(providerName, modelID string, capabilities ...Capability) *BaseModel {
	return &BaseModel{
		providerName: providerName,
		modelID:      modelID,
		info: ModelInfo{
			Provider:     providerName,
			ModelID:      modelID,
			Capabilities: capabilities,
		},
	}
}

func (m *BaseModel) Provider() string                  { return m.providerName }
func (m *BaseModel) ModelID() string                   { return m.modelID }
func (m *BaseModel) Info() ModelInfo                   { return m.info }
func (m *BaseModel) HasCapability(cap Capability) bool { return m.info.HasCapability(cap) }

func (m *BaseModel) SetMaxTokens(maxTokens int) {
	m.info.MaxTokens = maxTokens
}

func (m *BaseModel) SetContextSize(contextSize int) {
	m.info.ContextSize = contextSize
}

var (
	_ Model         = (*BaseModel)(nil)
	_ ModelWithInfo = (*BaseModel)(nil)
)
