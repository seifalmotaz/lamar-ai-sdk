package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ErrorCode represents a structured error type for provider operations.
// Each code corresponds to a specific category of error.
type ErrorCode int

// Error code constants define the type of error that occurred.
const (
	// CodeUnknown indicates an unspecified error type.
	CodeUnknown ErrorCode = iota
	// CodeInvalidRequest indicates the request was malformed.
	CodeInvalidRequest
	// CodeInvalidModel indicates the model parameter is invalid (e.g., nil).
	CodeInvalidModel
	// CodeInvalidPrompt indicates the prompt is empty or invalid.
	CodeInvalidPrompt
	// CodeInvalidInput indicates the input data is invalid.
	CodeInvalidInput
	// CodeAuthenticationFailed indicates authentication with the provider failed.
	CodeAuthenticationFailed
	// CodeRateLimited indicates the request was rate limited by the provider.
	CodeRateLimited
	// CodeModelNotFound indicates the requested model does not exist.
	CodeModelNotFound
	// CodeContentFiltered indicates the content was filtered by the provider's safety systems.
	CodeContentFiltered
	// CodeContextCanceled indicates the context was canceled before completion.
	CodeContextCanceled
	// CodeAPITimeout indicates the API request timed out.
	CodeAPITimeout
	// CodeParseError indicates a failure to parse the response.
	CodeParseError
	// CodeUnsupportedModel indicates the model does not support the requested operation.
	CodeUnsupportedModel
	// CodeUnsupportedOperation indicates the operation is not supported.
	CodeUnsupportedOperation
)

// String returns the string representation of the error code.
func (e ErrorCode) String() string {
	switch e {
	case CodeInvalidRequest:
		return "INVALID_REQUEST"
	case CodeInvalidModel:
		return "INVALID_MODEL"
	case CodeInvalidPrompt:
		return "INVALID_PROMPT"
	case CodeInvalidInput:
		return "INVALID_INPUT"
	case CodeAuthenticationFailed:
		return "AUTHENTICATION_FAILED"
	case CodeRateLimited:
		return "RATE_LIMITED"
	case CodeModelNotFound:
		return "MODEL_NOT_FOUND"
	case CodeContentFiltered:
		return "CONTENT_FILTERED"
	case CodeContextCanceled:
		return "CONTEXT_CANCELED"
	case CodeAPITimeout:
		return "API_TIMEOUT"
	case CodeParseError:
		return "PARSE_ERROR"
	case CodeUnsupportedModel:
		return "UNSUPPORTED_MODEL"
	case CodeUnsupportedOperation:
		return "UNSUPPORTED_OPERATION"
	default:
		return "UNKNOWN"
	}
}

type Error struct {
	Code       ErrorCode     // The structured error code
	Message    string        // Human-readable error message
	Cause      error         // Underlying error, if any
	Provider   string        // Provider identifier (e.g., "openai")
	ModelID    string        // Model identifier (e.g., "gpt-4o")
	RetryAfter time.Duration // Suggested retry delay for rate limits
	StatusCode int           // HTTP status code, if applicable
}

func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString(e.Code.String())
	b.WriteString(": ")
	b.WriteString(e.Message)

	if e.Provider != "" {
		b.WriteString(" (provider=")
		b.WriteString(e.Provider)
		if e.ModelID != "" {
			b.WriteString(", model=")
			b.WriteString(e.ModelID)
		}
		b.WriteString(")")
	}

	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}

	return b.String()
}

func (e *Error) Unwrap() error { return e.Cause }

// NewError creates a new Error with the given code, message, and cause.
func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// NewErrorWithMeta creates a new Error with provider and model metadata.
func NewErrorWithMeta(code ErrorCode, message string, cause error, provider, modelID string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Cause:    cause,
		Provider: provider,
		ModelID:  modelID,
	}
}

// Sentinel errors for common error conditions.
var (
	ErrInvalidModel         = &Error{Code: CodeInvalidModel, Message: "model is nil"}
	ErrInvalidPrompt        = &Error{Code: CodeInvalidPrompt, Message: "prompt cannot be empty"}
	ErrInvalidInput         = &Error{Code: CodeInvalidInput, Message: "input cannot be empty"}
	ErrRateLimited          = &Error{Code: CodeRateLimited, Message: "rate limit exceeded"}
	ErrContextCanceled      = &Error{Code: CodeContextCanceled, Message: "context canceled"}
	ErrAPITimeout           = &Error{Code: CodeAPITimeout, Message: "API request timed out"}
	ErrAuthenticationFailed = &Error{Code: CodeAuthenticationFailed, Message: "authentication failed"}
	ErrModelNotFound        = &Error{Code: CodeModelNotFound, Message: "model not found"}
	ErrContentFiltered      = &Error{Code: CodeContentFiltered, Message: "content filtered by provider"}
	ErrUnsupportedModel     = &Error{Code: CodeUnsupportedModel, Message: "unsupported model"}
	ErrUnsupportedOperation = &Error{Code: CodeUnsupportedOperation, Message: "unsupported operation"}
)

// IsRateLimited returns true if the error indicates a rate limit was exceeded.
func IsRateLimited(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == CodeRateLimited
}

// RetryAfter returns the suggested retry duration for rate-limited errors.
// Returns 0 if the error is not a rate limit error or has no RetryAfter value.
func RetryAfter(err error) time.Duration {
	var e *Error
	if errors.As(err, &e) {
		return e.RetryAfter
	}
	return 0
}

// ErrorCodeOf extracts the error code from an error.
// Returns CodeUnknown if the error is not a provider.Error.
func ErrorCodeOf(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeUnknown
}

// IsContextCanceled returns true if the error indicates context cancellation.
func IsContextCanceled(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	return ErrorCodeOf(err) == CodeContextCanceled
}

// IsTimeout returns true if the error indicates a timeout.
func IsTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return ErrorCodeOf(err) == CodeAPITimeout
}

// IsAuthenticationError returns true if the error indicates an authentication failure.
func IsAuthenticationError(err error) bool {
	return ErrorCodeOf(err) == CodeAuthenticationFailed
}

// IsNotFoundError returns true if the error indicates a resource was not found.
func IsNotFoundError(err error) bool {
	code := ErrorCodeOf(err)
	return code == CodeModelNotFound
}

// IsInvalidInput returns true if the error indicates invalid input.
func IsInvalidInput(err error) bool {
	code := ErrorCodeOf(err)
	return code == CodeInvalidInput || code == CodeInvalidPrompt || code == CodeInvalidModel
}

// IsContentFiltered returns true if the error indicates content was filtered.
func IsContentFiltered(err error) bool {
	return ErrorCodeOf(err) == CodeContentFiltered
}

// IsError checks if err is a provider.Error and assigns it to target.
// This is a convenience wrapper around errors.As for provider.Error.
func IsError(err error, target **Error) bool {
	return errors.As(err, target)
}

// ParseError represents a JSON parsing failure.
type ParseError struct {
	Field string // The field that failed to parse
	Err   error  // The underlying error
}

// Error implements the error interface for ParseError.
func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("parse error in %s: %v", e.Field, e.Err)
	}
	return fmt.Sprintf("parse error: %v", e.Err)
}

// Unwrap returns the underlying error.
func (e *ParseError) Unwrap() error { return e.Err }

// NewParseError creates a new ParseError for the given field and error.
func NewParseError(field string, err error) *ParseError {
	return &ParseError{Field: field, Err: err}
}
