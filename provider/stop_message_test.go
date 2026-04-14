package provider

import (
	"testing"
)

func TestDefaultIsStopMessage(t *testing.T) {
	tests := []struct {
		name     string
		result   *GenerateResult
		expected bool
	}{
		{
			name:     "nil result is terminal",
			result:   nil,
			expected: true,
		},
		{
			name: "result with tool calls is not terminal",
			result: &GenerateResult{
				ToolCalls: []ToolCall{{ID: "1", Name: "test"}},
			},
			expected: false,
		},
		{
			name: "result with finish_reason stop is terminal",
			result: &GenerateResult{
				FinishReason: FinishReasonStop,
				Text:         "Hello",
			},
			expected: true,
		},
		{
			name: "result with finish_reason length is terminal",
			result: &GenerateResult{
				FinishReason: FinishReasonLength,
			},
			expected: true,
		},
		{
			name: "result with finish_reason content_filter is terminal",
			result: &GenerateResult{
				FinishReason: FinishReasonContentFilter,
			},
			expected: true,
		},
		{
			name: "result with finish_reason tool_calls and tool calls is not terminal",
			result: &GenerateResult{
				FinishReason: FinishReasonToolCalls,
				ToolCalls:    []ToolCall{{ID: "1", Name: "test"}},
			},
			expected: false,
		},
		{
			name: "result with text content and no finish reason is not terminal",
			result: &GenerateResult{
				Text: "Hello world",
			},
			expected: false,
		},
		{
			name: "empty result with error finish reason is not terminal",
			result: &GenerateResult{
				FinishReason: FinishReasonError,
			},
			expected: true,
		},
		{
			name:     "empty result with no finish reason is terminal",
			result:   &GenerateResult{},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DefaultIsStopMessage(tt.result)
			if got != tt.expected {
				t.Errorf("DefaultIsStopMessage() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsStopMessage(t *testing.T) {
	t.Run("uses default when model does not implement StopMessageChecker", func(t *testing.T) {
		model := NewModelBuilder("test", "test-model").Build()
		result := &GenerateResult{
			FinishReason: FinishReasonStop,
		}
		if !IsStopMessage(model, result) {
			t.Error("IsStopMessage() = false, want true for stop finish reason")
		}
	})

	t.Run("uses custom logic when model implements StopMessageChecker", func(t *testing.T) {
		model := &customStopCheckerModel{Model: NewModelBuilder("test", "test-model").Build()}
		result := &GenerateResult{
			FinishReason: FinishReasonStop,
			ToolCalls:    []ToolCall{{ID: "1", Name: "test"}},
		}
		// Custom checker always returns true
		if !IsStopMessage(model, result) {
			t.Error("IsStopMessage() = false, want true for custom checker")
		}
	})
}

// customStopCheckerModel implements StopMessageChecker with custom logic
type customStopCheckerModel struct {
	Model
}

func (m *customStopCheckerModel) IsStopMessage(result *GenerateResult) bool {
	// Custom: always consider terminal
	return true
}
