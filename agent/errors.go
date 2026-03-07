package agent

import (
	"fmt"
)

// AgentError represents an agent error with context.
type AgentError struct {
	// Code is a machine-readable error code.
	Code string

	// Message is a human-readable error message.
	Message string

	// Step is the step number where the error occurred (0 if before first step).
	Step int

	// Cause is the underlying error (may be nil).
	Cause error
}

func (e *AgentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("agent error (step %d): %s: %v", e.Step, e.Message, e.Cause)
	}
	return fmt.Sprintf("agent error (step %d): %s", e.Step, e.Message)
}

func (e *AgentError) Unwrap() error {
	return e.Cause
}

// NewAgentError creates a new agent error.
func NewAgentError(code, message string, step int, cause error) *AgentError {
	return &AgentError{
		Code:    code,
		Message: message,
		Step:    step,
		Cause:   cause,
	}
}

// ToolNotFoundError indicates a tool was requested that doesn't exist.
type ToolNotFoundError struct {
	// ToolName is the name of the tool that was not found.
	ToolName string
}

func (e *ToolNotFoundError) Error() string {
	return fmt.Sprintf("tool not found: %s", e.ToolName)
}

// MaxStepsExceededError indicates the agent exceeded the maximum number of steps.
type MaxStepsExceededError struct {
	// Steps is the number of steps that were executed.
	Steps int

	// MaxSteps is the maximum allowed steps.
	MaxSteps int
}

func (e *MaxStepsExceededError) Error() string {
	return fmt.Sprintf("max steps exceeded: %d steps (max: %d)", e.Steps, e.MaxSteps)
}

// NewError creates a new agent error.
// Deprecated: Use NewAgentError instead.
func NewError(code, message string, step int, cause error) *AgentError {
	return NewAgentError(code, message, step, cause)
}

// Error is an alias for AgentError for backward compatibility.
// Deprecated: Use AgentError instead.
type Error = AgentError

// IsAgentError checks if err is an AgentError and assigns it to target.
func IsAgentError(err error, target **AgentError) bool {
	return AsAgentError(err, target)
}

// AsAgentError checks if err is an AgentError and assigns it to target.
// This is a convenience wrapper around errors.As for AgentError.
func AsAgentError(err error, target **AgentError) bool {
	if err == nil {
		return false
	}
	if ae, ok := err.(*AgentError); ok {
		*target = ae
		return true
	}
	// Check wrapped errors
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return AsAgentError(unwrapper.Unwrap(), target)
	}
	return false
}
