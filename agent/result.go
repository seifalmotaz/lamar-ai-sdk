package agent

import (
	"strings"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

// Result is the final result of agent execution.
type Result struct {
	// Steps contains all executed steps in order.
	Steps []StepResult

	// FinalMessages is the complete message history.
	FinalMessages []provider.Message

	// FinalContent is the last content from the model (for convenience).
	FinalContent []provider.Content

	// FinalText extracts text from FinalContent (for convenience).
	// If there are multiple text parts, they are joined with newlines.
	FinalText string

	// TotalUsage aggregates usage across all steps.
	TotalUsage provider.Usage

	// TotalDuration is the sum of all step durations.
	TotalDuration time.Duration

	// FinishReason from the final step.
	FinishReason provider.FinishReason
}

// ExtractText extracts and joins all text content from the given content parts.
// Non-text content (images, audio, tool calls) is ignored.
// If multiple text parts exist, they are joined with newlines.
func ExtractText(content []provider.Content) string {
	var texts []string
	for _, c := range content {
		if tc, ok := c.(provider.TextContent); ok {
			texts = append(texts, tc.Text)
		}
	}
	return strings.Join(texts, "\n")
}
