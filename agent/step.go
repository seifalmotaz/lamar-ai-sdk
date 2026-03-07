package agent

import (
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

// StepResult contains the outcome of a single agent step.
// Each step represents one call to the language model and the subsequent
// execution of any tool calls it requested.
type StepResult struct {
	// StepNumber is the 1-indexed step number (first step = 1).
	StepNumber int

	// Content contains the content parts from the model response.
	Content []provider.Content

	// ToolCalls contains the tool calls requested by the model in this step.
	ToolCalls []provider.ToolCall

	// ToolResults contains the results from executing the tool calls.
	ToolResults []ToolExecutionResult

	// Usage contains the token usage for this step.
	Usage provider.Usage

	// FinishReason indicates why the model finished generating.
	FinishReason provider.FinishReason

	// Model identifies which model was used for this step.
	Model ModelInfo

	// Duration is the total time for this step (model call + tool execution).
	Duration time.Duration
}

// ToolExecutionResult is the result of executing a single tool.
type ToolExecutionResult struct {
	// ToolCallID is the unique identifier for the tool call.
	ToolCallID string

	// ToolName is the name of the tool that was executed.
	ToolName string

	// Input is the input that was passed to the tool.
	Input any

	// Output is the result from the tool (nil if error).
	Output any

	// Error is the error from tool execution (nil if success).
	Error error

	// Duration is the time it took to execute the tool.
	Duration time.Duration
}

// ModelInfo identifies which model was used for a step.
type ModelInfo struct {
	// Provider is the name of the provider (e.g., "openai", "anthropic").
	Provider string

	// ModelID is the model identifier (e.g., "gpt-4o", "claude-3-opus").
	ModelID string
}
