package provider

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestErrorCodeString(t *testing.T) {
	tests := []struct {
		code     ErrorCode
		expected string
	}{
		{CodeUnknown, "UNKNOWN"},
		{CodeInvalidRequest, "INVALID_REQUEST"},
		{CodeInvalidModel, "INVALID_MODEL"},
		{CodeInvalidPrompt, "INVALID_PROMPT"},
		{CodeInvalidInput, "INVALID_INPUT"},
		{CodeAuthenticationFailed, "AUTHENTICATION_FAILED"},
		{CodeRateLimited, "RATE_LIMITED"},
		{CodeModelNotFound, "MODEL_NOT_FOUND"},
		{CodeContentFiltered, "CONTENT_FILTERED"},
		{CodeContextCanceled, "CONTEXT_CANCELED"},
		{CodeAPITimeout, "API_TIMEOUT"},
		{CodeParseError, "PARSE_ERROR"},
		{CodeUnsupportedModel, "UNSUPPORTED_MODEL"},
		{CodeUnsupportedOperation, "UNSUPPORTED_OPERATION"},
		{ErrorCode(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("ErrorCode.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      *Error
		contains string
	}{
		{
			name:     "basic error",
			err:      &Error{Code: CodeInvalidModel, Message: "model is nil"},
			contains: "INVALID_MODEL: model is nil",
		},
		{
			name:     "error with provider",
			err:      &Error{Code: CodeRateLimited, Message: "rate limit exceeded", Provider: "openai"},
			contains: "RATE_LIMITED: rate limit exceeded (provider=openai)",
		},
		{
			name:     "error with provider and model",
			err:      &Error{Code: CodeAPITimeout, Message: "timeout", Provider: "openai", ModelID: "gpt-4"},
			contains: "API_TIMEOUT: timeout (provider=openai, model=gpt-4)",
		},
		{
			name:     "error with cause",
			err:      &Error{Code: CodeParseError, Message: "failed to parse", Cause: errors.New("invalid JSON")},
			contains: "PARSE_ERROR: failed to parse: invalid JSON",
		},
		{
			name:     "error with all fields",
			err:      &Error{Code: CodeRateLimited, Message: "rate limited", Provider: "openai", ModelID: "gpt-4", Cause: errors.New("HTTP 429")},
			contains: "RATE_LIMITED: rate limited (provider=openai, model=gpt-4): HTTP 429",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if !containsAll(got, tt.contains) {
				t.Errorf("Error.Error() = %q, should contain %q", got, tt.contains)
			}
		})
	}
}

func containsAll(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr || len(s) > 0 && containsAll(s[1:], substr)
}

func TestErrorUnwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &Error{Code: CodeParseError, Message: "parse failed", Cause: cause}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Error.Unwrap() = %v, want %v", unwrapped, cause)
	}

	if !errors.Is(err, cause) {
		t.Error("errors.Is should match the cause")
	}
}

func TestNewError(t *testing.T) {
	cause := errors.New("cause")
	err := NewError(CodeInvalidRequest, "invalid request", cause)

	if err.Code != CodeInvalidRequest {
		t.Errorf("Code = %v, want %v", err.Code, CodeInvalidRequest)
	}
	if err.Message != "invalid request" {
		t.Errorf("Message = %v, want %v", err.Message, "invalid request")
	}
	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestNewErrorWithMeta(t *testing.T) {
	cause := errors.New("cause")
	err := NewErrorWithMeta(CodeRateLimited, "rate limited", cause, "openai", "gpt-4")

	if err.Provider != "openai" {
		t.Errorf("Provider = %v, want %v", err.Provider, "openai")
	}
	if err.ModelID != "gpt-4" {
		t.Errorf("ModelID = %v, want %v", err.ModelID, "gpt-4")
	}
}

func TestErrorIs(t *testing.T) {
	err := &Error{Code: CodeRateLimited, Message: "rate limited"}

	var providerErr *Error
	if !errors.As(err, &providerErr) {
		t.Error("error should be convertible to *Error via errors.As")
	}

	if providerErr.Code != CodeRateLimited {
		t.Errorf("Code = %v, want %v", providerErr.Code, CodeRateLimited)
	}
}

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"rate limited error", &Error{Code: CodeRateLimited}, true},
		{"other error", &Error{Code: CodeInvalidRequest}, false},
		{"nil error", nil, false},
		{"standard error", errors.New("some error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRateLimited(tt.err); got != tt.expected {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRetryAfter(t *testing.T) {
	duration := 30 * time.Second
	err := &Error{Code: CodeRateLimited, RetryAfter: duration}

	if got := RetryAfter(err); got != duration {
		t.Errorf("RetryAfter() = %v, want %v", got, duration)
	}

	if got := RetryAfter(errors.New("other error")); got != 0 {
		t.Errorf("RetryAfter() = %v, want 0", got)
	}
}

func TestErrorCodeOf(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{"provider error", &Error{Code: CodeAPITimeout}, CodeAPITimeout},
		{"standard error", errors.New("error"), CodeUnknown},
		{"nil error", nil, CodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorCodeOf(tt.err); got != tt.expected {
				t.Errorf("ErrorCodeOf() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsContextCanceled(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"context canceled", context.Canceled, true},
		{"provider context canceled", &Error{Code: CodeContextCanceled}, true},
		{"other error", &Error{Code: CodeInvalidRequest}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContextCanceled(tt.err); got != tt.expected {
				t.Errorf("IsContextCanceled() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsTimeout(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"deadline exceeded", context.DeadlineExceeded, true},
		{"provider timeout", &Error{Code: CodeAPITimeout}, true},
		{"other error", &Error{Code: CodeInvalidRequest}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTimeout(tt.err); got != tt.expected {
				t.Errorf("IsTimeout() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthenticationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"auth error", &Error{Code: CodeAuthenticationFailed}, true},
		{"other error", &Error{Code: CodeInvalidRequest}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthenticationError(tt.err); got != tt.expected {
				t.Errorf("IsAuthenticationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"model not found", &Error{Code: CodeModelNotFound}, true},
		{"other error", &Error{Code: CodeInvalidRequest}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"invalid model", &Error{Code: CodeInvalidModel}, true},
		{"invalid prompt", &Error{Code: CodeInvalidPrompt}, true},
		{"invalid input", &Error{Code: CodeInvalidInput}, true},
		{"other error", &Error{Code: CodeRateLimited}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInvalidInput(tt.err); got != tt.expected {
				t.Errorf("IsInvalidInput() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsContentFiltered(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"content filtered", &Error{Code: CodeContentFiltered}, true},
		{"other error", &Error{Code: CodeInvalidRequest}, false},
		{"nil", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsContentFiltered(tt.err); got != tt.expected {
				t.Errorf("IsContentFiltered() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestParseError(t *testing.T) {
	cause := errors.New("unexpected EOF")

	t.Run("with field", func(t *testing.T) {
		err := &ParseError{Field: "input", Err: cause}
		expected := "parse error in input: unexpected EOF"
		if got := err.Error(); got != expected {
			t.Errorf("ParseError.Error() = %q, want %q", got, expected)
		}
	})

	t.Run("without field", func(t *testing.T) {
		err := &ParseError{Err: cause}
		expected := "parse error: unexpected EOF"
		if got := err.Error(); got != expected {
			t.Errorf("ParseError.Error() = %q, want %q", got, expected)
		}
	})

	t.Run("unwrap", func(t *testing.T) {
		err := &ParseError{Field: "input", Err: cause}
		if !errors.Is(err.Unwrap(), cause) {
			t.Error("ParseError.Unwrap() should return the cause")
		}
	})
}

func TestNewParseError(t *testing.T) {
	cause := errors.New("parse error")
	err := NewParseError("field", cause)

	if err.Field != "field" {
		t.Errorf("Field = %v, want %v", err.Field, "field")
	}
	if err.Err != cause {
		t.Errorf("Err = %v, want %v", err.Err, cause)
	}
}

func TestSentinelErrors(t *testing.T) {
	sentinels := []struct {
		name string
		err  *Error
		code ErrorCode
		msg  string
	}{
		{"ErrInvalidModel", ErrInvalidModel, CodeInvalidModel, "model is nil"},
		{"ErrInvalidPrompt", ErrInvalidPrompt, CodeInvalidPrompt, "prompt cannot be empty"},
		{"ErrInvalidInput", ErrInvalidInput, CodeInvalidInput, "input cannot be empty"},
		{"ErrInvalidMediaType", ErrInvalidMediaType, CodeInvalidRequest, "media type is required"},
		{"ErrRateLimited", ErrRateLimited, CodeRateLimited, "rate limit exceeded"},
		{"ErrContextCanceled", ErrContextCanceled, CodeContextCanceled, "context canceled"},
		{"ErrAPITimeout", ErrAPITimeout, CodeAPITimeout, "API request timed out"},
		{"ErrAuthenticationFailed", ErrAuthenticationFailed, CodeAuthenticationFailed, "authentication failed"},
		{"ErrModelNotFound", ErrModelNotFound, CodeModelNotFound, "model not found"},
		{"ErrContentFiltered", ErrContentFiltered, CodeContentFiltered, "content filtered by provider"},
		{"ErrUnsupportedModel", ErrUnsupportedModel, CodeUnsupportedModel, "unsupported model"},
		{"ErrUnsupportedOperation", ErrUnsupportedOperation, CodeUnsupportedOperation, "unsupported operation"},
	}

	for _, tt := range sentinels {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
			if tt.err.Message != tt.msg {
				t.Errorf("Message = %v, want %v", tt.err.Message, tt.msg)
			}
		})
	}
}

func TestErrorWithRetryAfter(t *testing.T) {
	retryAfter := 60 * time.Second
	err := &Error{
		Code:       CodeRateLimited,
		Message:    "rate limited",
		RetryAfter: retryAfter,
	}

	if err.RetryAfter != retryAfter {
		t.Errorf("RetryAfter = %v, want %v", err.RetryAfter, retryAfter)
	}
}

func TestErrorWithStatusCode(t *testing.T) {
	err := &Error{
		Code:       CodeInvalidRequest,
		Message:    "bad request",
		StatusCode: 400,
	}

	if err.StatusCode != 400 {
		t.Errorf("StatusCode = %v, want %v", err.StatusCode, 400)
	}
}

func TestJSONMarshalUnmarshalError(t *testing.T) {
	original := &Error{
		Code:       CodeRateLimited,
		Message:    "rate limited",
		Provider:   "openai",
		ModelID:    "gpt-4",
		StatusCode: 429,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Failed to marshal error: %v", err)
	}

	var unmarshaled Error
	if err := json.Unmarshal(data, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal error: %v", err)
	}

	if unmarshaled.Code != original.Code {
		t.Errorf("Code = %v, want %v", unmarshaled.Code, original.Code)
	}
	if unmarshaled.Message != original.Message {
		t.Errorf("Message = %v, want %v", unmarshaled.Message, original.Message)
	}
	if unmarshaled.Provider != original.Provider {
		t.Errorf("Provider = %v, want %v", unmarshaled.Provider, original.Provider)
	}
}
