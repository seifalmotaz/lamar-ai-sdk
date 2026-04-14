# Lamar SDK Native Layer Improvements Plan

## Executive Summary

This plan outlines improvements to Lamar SDK's native/provider layer based on analysis of the AIGO architecture. The focus is on the core provider interfaces and types, not higher-level orchestration (memory, agents, etc.).

**Key Goals:**
- Add critical missing features for production readiness
- Maintain backward compatibility where possible
- Keep the clean segregated interface design

---

## Phase 1: Critical Gaps

### 1.1 Add `IsStopMessage` Helper Function

**Problem:** Agent loops need reliable terminal response detection. Manual `FinishReason` checks don't handle edge cases (e.g., providers that incorrectly set finish reason with tool calls).

**Decision: Use helper function (no interface change)**

Adding to `Generator` interface would break all existing implementations. Instead, use a helper function with default behavior that can be overridden via interface assertion.

**Files to modify:**
- `provider/provider.go` - Add `StopMessageChecker` interface and `IsStopMessage()` helper

**Implementation:**
```go
// provider/provider.go

// StopMessageChecker is an optional interface for providers with custom
// stop message detection logic. If not implemented, DefaultIsStopMessage is used.
type StopMessageChecker interface {
    IsStopMessage(result *GenerateResult) bool
}

// DefaultIsStopMessage provides the default implementation.
// Returns false if tool calls are present, true if finish reason is "stop".
func DefaultIsStopMessage(result *GenerateResult) bool {
    if len(result.ToolCalls) > 0 {
        return false
    }
    return result.FinishReason == FinishReasonStop
}

// IsStopMessage checks if a response represents a terminal state.
// If the model implements StopMessageChecker, uses its logic.
// Otherwise, falls back to DefaultIsStopMessage.
func IsStopMessage(m Model, result *GenerateResult) bool {
    if checker, ok := m.(StopMessageChecker); ok {
        return checker.IsStopMessage(result)
    }
    return DefaultIsStopMessage(result)
}
```

**Usage:**
```go
// In agent package
if provider.IsStopMessage(model, result) {
    // No more tool calls, stop the loop
}
```

---

### 1.2 Standardized Tool Result Structure

**Problem:** `json.RawMessage` provides no structure for LLMs to understand tool outcomes. AIGO's `ToolResult{Success, Error, Message, Data}` helps LLMs consistently interpret results.

**Decision: Add structured ToolResult with constructors, keep existing ToolResultContent**

**Files to create/modify:**
- `provider/types.go` - Add `ToolResult` struct and constructors

**Implementation:**
```go
// provider/types.go

// ToolResult represents a standardized tool execution result.
// This structure helps LLMs understand tool outcomes consistently.
type ToolResult struct {
    Success bool          `json:"success"`
    Error   string        `json:"error,omitempty"`   // Error type (e.g., "not_found", "timeout")
    Message string        `json:"message,omitempty"` // Human-readable message
    Data    interface{}   `json:"data,omitempty"`   // Result data on success
}

// NewToolResultSuccess creates a successful tool result.
func NewToolResultSuccess(data interface{}) ToolResult {
    return ToolResult{Success: true, Data: data}
}

// NewToolResultError creates a failed tool result.
func NewToolResultError(errorType, message string) ToolResult {
    return ToolResult{Success: false, Error: errorType, Message: message}
}

// ToJSON marshals the result to JSON for use as ToolResultContent.Result.
func (tr ToolResult) ToJSON() (json.RawMessage, error) {
    data, err := json.Marshal(tr)
    if err != nil {
        return nil, err
    }
    return json.RawMessage(data), nil
}

// ToToolResultContent creates a ToolResultContent from this ToolResult.
func (tr ToolResult) ToToolResultContent(id, name string) ToolResultContent {
    result, _ := tr.ToJSON()
    return ToolResultContent{
        ID:      id,
        Name:    name,
        Result:  result,
        IsError: !tr.Success,
    }
}
```

**Usage:**
```go
// Tool execution
result := provider.NewToolResultSuccess(map[string]any{"temperature": 72})
content := result.ToToolResultContent(toolCallID, "get_weather")
message := provider.ToolResultMessage(content)
```

---

### 1.3 Add Grounding/Citation Metadata

**Problem:** No way to represent RAG citations, web search grounding, or source attribution in responses.

**Files to modify:**
- `provider/types.go` - Add `GroundingMetadata`, `GroundingSource`, `Citation` structs
- `provider/types.go` - Add `Grounding *GroundingMetadata` to `GenerateResult`

**Implementation:**
```go
// provider/types.go

// GroundingMetadata contains citation and source attribution from grounded responses.
// Present when models use web search, RAG, or other grounding mechanisms.
type GroundingMetadata struct {
    Sources  []GroundingSource  // Source documents/URLs
    Citations []Citation        // Text segments linked to sources
}

// GroundingSource represents a source document or URL.
type GroundingSource struct {
    Index int    // 0-based index for reference from Citations
    URI   string // Source URL or file path
    Title string // Optional title
}

// Citation links a text segment to its supporting sources.
type Citation struct {
    Text          string    // The cited text (optional)
    StartIndex    int       // Character start position (0-indexed)
    EndIndex      int       // Character end position (exclusive)
    SourceIndices []int     // References to Sources array
    Confidence    []float64 // Confidence scores (optional)
}

// Update GenerateResult:
type GenerateResult struct {
    // ... existing fields
    Grounding *GroundingMetadata // NEW: Citation/source attribution
}
```

**Provider support:** Initially nil for OpenAI. Will be populated for Gemini with `_google_search`.

---

## Phase 2: Important Enhancements

### 2.1 Add Code Execution Support

**Problem:** No support for server-side code execution results (Gemini `code_execution` tool).

**Files to modify:**
- `provider/types.go` - Add `CodeExecution` struct
- `provider/types.go` - Add `CodeExecutions []CodeExecution` to `GenerateResult`

**Implementation:**
```go
// provider/types.go

// CodeExecution represents a server-side code execution result.
// Generated by models like Gemini with code_execution tool enabled.
type CodeExecution struct {
    Language string  // Programming language (e.g., "PYTHON")
    Code     string  // The code that was executed
    Outcome  string  // "OUTCOME_OK", "OUTCOME_FAILED", "OUTCOME_DEADLINE_EXCEEDED"
    Output   string  // stdout on success, stderr on failure
}

// Update GenerateResult:
type GenerateResult struct {
    // ... existing fields
    CodeExecutions []CodeExecution // NEW: Server-side code execution results
}
```

---

### 2.2 Extend Generation Config

**Problem:** Missing support for reasoning models, safety settings, and multi-modal output specification.

**Files to modify:**
- `provider/types.go` - Extend `Config` struct
- `providers/openai/*.go` - Handle new config fields

**Implementation:**
```go
// provider/types.go

// SafetySetting configures content safety thresholds.
type SafetySetting struct {
    Category  string // Provider-specific category identifier
    Threshold string // Provider-specific threshold level
}

// Config extensions
type Config struct {
    // ... existing fields
    
    // Reasoning models (o1, o3, etc.)
    ThinkingBudget   *int  // Token budget for reasoning; nil = default, -1 = dynamic
    IncludeThoughts bool   // Include reasoning in response text
    
    // Safety settings (provider-specific)
    SafetySettings []SafetySetting
    
    // Multi-modal output specification
    ResponseModalities []string // ["TEXT"], ["TEXT", "IMAGE"], etc.
}
```

**Provider mapping:**
- OpenAI: Map `ThinkingBudget` to `reasoning_effort`, ignore others
- Gemini: Map to `thinkingConfig`, `safetySettings`, `responseModalities`
- Anthropic: Map to `thinking` block, ignore safety settings

---

### 2.3 Add Model Pricing and Modalities

**Problem:** No cost information in `ModelInfo` - essential for production cost tracking and optimization.

**Files to modify:**
- `provider/types.go` - Add `ModelCost`, `ContextTier`, `Modality` types
- `provider/types.go` - Extend `ModelInfo`

**Implementation:**
```go
// provider/types.go

// Modality represents an input or output modality supported by a model.
type Modality string

const (
    ModalityText     Modality = "text"
    ModalityImage    Modality = "image"
    ModalityAudio    Modality = "audio"
    ModalityVideo    Modality = "video"
    ModalityDocument Modality = "document" // PDF, etc.
)

// ContextTier defines tiered pricing for large context windows.
type ContextTier struct {
    InputTokenThreshold  int     // Threshold for this tier
    InputCostPerMillion  float64 // Per-million input tokens
    OutputTokenThreshold int     // Threshold for this tier
    OutputCostPerMillion float64 // Per-million output tokens
}

// ModelCost represents pricing information for a model.
type ModelCost struct {
    InputCostPerMillion       float64
    OutputCostPerMillion      float64
    CachedInputCostPerMillion float64 // Discounted cached tokens
    ReasoningCostPerMillion   float64 // Reasoning tokens (o1/o3)
    ContextTiers              []ContextTier // Tiered pricing
    ImageCostPerUnit          float64 // Per generated image
    AudioCostPerUnit          float64 // Per generated audio
}

// ModelInfo extensions
type ModelInfo struct {
    Provider        string
    ModelID         string
    Capabilities    []Capability
    MaxTokens       int
    ContextSize     int
    
    // NEW
    Pricing         *ModelCost    // Cost structure
    InputModalities []Modality    // ["text", "image", "audio"]
    OutputModalities []Modality   // ["text", "image"]
    Deprecated      bool          // Whether model is deprecated
}
```

---

## Phase 3: Optional Improvements

### 3.1 Unified Observability Interface (DEFERRED)

**Current state:** `Logger` and `MetricsCollector` are separate interfaces in `provider/types.go`.

**Recommendation:** DEFER - Current middleware approach is sufficient. If needed later, create separate `observability/` package.

---

### 3.2 Pseudo-Tools / Built-in Capabilities

**Problem:** Provider-specific features like Gemini's `_google_search` are not discoverable.

**Files to create/modify:**
- `provider/builtin_tools.go` - Add constants and helpers

**Implementation:**
```go
// provider/builtin_tools.go

package provider

// Built-in pseudo-tool names for provider-specific capabilities.
// These are not user-defined tools but provider features.
const (
    BuiltinGoogleSearch    = "_google_search"     // Gemini web grounding
    BuiltinURLContext      = "_url_context"       // Gemini URL context  
    BuiltinCodeExecution   = "_code_execution"    // Server-side code execution
)

// IsBuiltinTool returns true if the tool name is a built-in pseudo-tool.
func IsBuiltinTool(name string) bool {
    return len(name) > 0 && name[0] == '_'
}
```

**Usage:** Document how to enable these in provider-specific options.

---

### 3.3 Streaming Event Type Alias (DEFERRED)

**Current state:** Stream parts require type assertions.

**Recommendation:** DEFER - Breaking change. Consider in future major version or when Go 1.23+ adoption is widespread. Current design works well.

---

## Implementation Order

### Priority 1 (Phase 1.1-1.3) - Critical
1. `IsStopMessage` helper function
2. `ToolResult` struct with constructors
3. `GroundingMetadata` and citations

**Estimated effort:** 1-2 days
**Breaking changes:** None

### Priority 2 (Phase 2.1-2.3) - Important
4. `CodeExecution` struct
5. Extended `Config` with reasoning/safety settings
6. `ModelCost` and model modalities

**Estimated effort:** 2-3 days
**Breaking changes:** None (additive only)

### Priority 3 (Phase 3) - Optional
7. Built-in pseudo-tool constants
8. Observability interface (deferred)
9. Streaming event types (deferred)

---

## Testing Strategy

### Unit Tests
- `provider/provider_test.go` - Test `IsStopMessage` with various results
- `provider/types_test.go` - Test `ToolResult` marshaling/unmarshaling
- `provider/types_test.go` - Test `GroundingMetadata` JSON roundtrip

### Integration Tests
- Update OpenAI provider tests to verify new fields are populated
- Add Gemini-specific tests for grounding and code execution

### Contract Tests
- Update `contract/` tests to expect new optional fields

---

## Documentation Updates

1. Update `README.md` with new types and helper functions
2. Update `PRD.md` with grounding and code execution requirements
3. Add code examples for `ToolResult` usage
4. Document `IsStopMessage` in agent framework docs

---

## Backward Compatibility

All changes in this plan are **additive**:
- New struct types (`ToolResult`, `GroundingMetadata`, `CodeExecution`, `ModelCost`)
- New helper functions (`IsStopMessage`, `NewToolResultSuccess`, `NewToolResultError`)
- New optional fields with `nil` as default

No existing interfaces are modified. No existing method signatures change.

---

## Open Questions

1. Should `SafetySetting` be provider-agnostic with constants, or use raw strings?
2. Should `ModelCost` calculation methods be added (e.g., `CalculateInputCost(tokens int) float64`)?
3. Should we add `IsBuiltinTool` check in the `agent` package to skip tool catalog lookup?

---

## Files Changed Summary

| File | Changes |
|------|---------|
| `provider/provider.go` | Add `StopMessageChecker` interface, `IsStopMessage()` helper |
| `provider/types.go` | Add `ToolResult`, `GroundingMetadata`, `CodeExecution`, `ModelCost`, extend `Config`, extend `ModelInfo` |
| `provider/builtin_tools.go` | NEW - Built-in pseudo-tool constants |
| `providers/openai/provider.go` | Implement `StopMessageChecker` (optional) |
| `README.md` | Document new types and helpers |

---

## Success Criteria

1. Agent loops can reliably detect terminal responses with `provider.IsStopMessage()`
2. Tool results have consistent structure for LLM interpretation
3. RAG/web-grounded responses can carry citation metadata
4. Cost tracking is possible via `ModelInfo.Pricing`
5. No breaking changes to existing code