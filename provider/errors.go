package provider

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ErrorCode int

const (
	CodeUnknown ErrorCode = iota
	CodeInvalidRequest
	CodeInvalidModel
	CodeInvalidPrompt
	CodeInvalidInput
	CodeAuthenticationFailed
	CodeRateLimited
	CodeModelNotFound
	CodeContentFiltered
	CodeContextCanceled
	CodeAPITimeout
	CodeParseError
	CodeUnsupportedModel
	CodeUnsupportedOperation
)

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
	Code       ErrorCode
	Message    string
	Cause      error
	Provider   string
	ModelID    string
	RetryAfter time.Duration
	StatusCode int
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

func NewError(code ErrorCode, message string, cause error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

func NewErrorWithMeta(code ErrorCode, message string, cause error, provider, modelID string) *Error {
	return &Error{
		Code:     code,
		Message:  message,
		Cause:    cause,
		Provider: provider,
		ModelID:  modelID,
	}
}

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

func IsRateLimited(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Code == CodeRateLimited
}

func RetryAfter(err error) time.Duration {
	var e *Error
	if errors.As(err, &e) {
		return e.RetryAfter
	}
	return 0
}

func ErrorCodeOf(err error) ErrorCode {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeUnknown
}

func IsContextCanceled(err error) bool {
	if errors.Is(err, context.Canceled) {
		return true
	}
	return ErrorCodeOf(err) == CodeContextCanceled
}

func IsTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	return ErrorCodeOf(err) == CodeAPITimeout
}

func IsAuthenticationError(err error) bool {
	return ErrorCodeOf(err) == CodeAuthenticationFailed
}

func IsNotFoundError(err error) bool {
	code := ErrorCodeOf(err)
	return code == CodeModelNotFound
}

func IsInvalidInput(err error) bool {
	code := ErrorCodeOf(err)
	return code == CodeInvalidInput || code == CodeInvalidPrompt || code == CodeInvalidModel
}

func IsContentFiltered(err error) bool {
	return ErrorCodeOf(err) == CodeContentFiltered
}

// IsError checks if err is a provider.Error and assigns it to target.
func IsError(err error, target **Error) bool {
	return errors.As(err, target)
}

type ParseError struct {
	Field string
	Err   error
}

func (e *ParseError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("parse error in %s: %v", e.Field, e.Err)
	}
	return fmt.Sprintf("parse error: %v", e.Err)
}

func (e *ParseError) Unwrap() error { return e.Err }

func NewParseError(field string, err error) *ParseError {
	return &ParseError{Field: field, Err: err}
}
