package anthropic

import (
	"net/http"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestMapError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCode   provider.ErrorCode
	}{
		{
			name:       "invalid request",
			statusCode: 400,
			body:       `{"type":"error","error":{"type":"invalid_request_error","message":"Invalid request"}}`,
			wantCode:   provider.CodeInvalidRequest,
		},
		{
			name:       "authentication error",
			statusCode: 401,
			body:       `{"type":"error","error":{"type":"authentication_error","message":"Invalid API key"}}`,
			wantCode:   provider.CodeAuthenticationFailed,
		},
		{
			name:       "permission denied",
			statusCode: 403,
			body:       `{"type":"error","error":{"type":"permission_error","message":"Permission denied"}}`,
			wantCode:   provider.CodeAuthenticationFailed,
		},
		{
			name:       "not found",
			statusCode: 404,
			body:       `{"type":"error","error":{"type":"not_found_error","message":"Model not found"}}`,
			wantCode:   provider.CodeModelNotFound,
		},
		{
			name:       "rate limit",
			statusCode: 429,
			body:       `{"type":"error","error":{"type":"rate_limit_error","message":"Rate limit exceeded"}}`,
			wantCode:   provider.CodeRateLimited,
		},
		{
			name:       "overloaded",
			statusCode: 529,
			body:       `{"type":"error","error":{"type":"overloaded_error","message":"Service overloaded"}}`,
			wantCode:   provider.CodeRateLimited,
		},
		{
			name:       "api error",
			statusCode: 500,
			body:       `{"type":"error","error":{"type":"api_error","message":"Internal error"}}`,
			wantCode:   provider.CodeAPITimeout,
		},
		{
			name:       "unknown error",
			statusCode: 418,
			body:       `{"type":"error","error":{"type":"unknown_error","message":"I'm a teapot"}}`,
			wantCode:   provider.CodeUnknown,
		},
		{
			name:       "malformed response",
			statusCode: 500,
			body:       `invalid json`,
			wantCode:   provider.CodeAPITimeout,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}

			err := mapError(tt.statusCode, []byte(tt.body), headers)

			if err.Code != tt.wantCode {
				t.Errorf("mapError().Code = %v, want %v", err.Code, tt.wantCode)
			}

			if err.StatusCode != tt.statusCode {
				t.Errorf("mapError().StatusCode = %v, want %v", err.StatusCode, tt.statusCode)
			}

			if err.Provider != "anthropic" {
				t.Errorf("mapError().Provider = %v, want anthropic", err.Provider)
			}
		})
	}
}

func TestMapErrorTypeToCode(t *testing.T) {
	tests := []struct {
		errorType string
		wantCode  provider.ErrorCode
	}{
		{"invalid_request_error", provider.CodeInvalidRequest},
		{"authentication_error", provider.CodeAuthenticationFailed},
		{"permission_error", provider.CodeAuthenticationFailed},
		{"not_found_error", provider.CodeModelNotFound},
		{"rate_limit_error", provider.CodeRateLimited},
		{"overloaded_error", provider.CodeRateLimited},
		{"api_error", provider.CodeAPITimeout},
		{"unknown_error", provider.CodeUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.errorType, func(t *testing.T) {
			result := mapErrorTypeToCode(tt.errorType, 0)
			if result != tt.wantCode {
				t.Errorf("mapErrorTypeToCode(%q) = %v, want %v", tt.errorType, result, tt.wantCode)
			}
		})
	}
}

func TestMapStatusCodeToCode(t *testing.T) {
	tests := []struct {
		statusCode int
		wantCode   provider.ErrorCode
	}{
		{400, provider.CodeInvalidRequest},
		{401, provider.CodeAuthenticationFailed},
		{403, provider.CodeAuthenticationFailed},
		{404, provider.CodeModelNotFound},
		{413, provider.CodeInvalidRequest},
		{429, provider.CodeRateLimited},
		{500, provider.CodeAPITimeout},
		{502, provider.CodeAPITimeout},
		{503, provider.CodeAPITimeout},
		{504, provider.CodeAPITimeout},
		{529, provider.CodeRateLimited},
		{200, provider.CodeUnknown},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := mapStatusCodeToCode(tt.statusCode)
			if result != tt.wantCode {
				t.Errorf("mapStatusCodeToCode(%d) = %v, want %v", tt.statusCode, result, tt.wantCode)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		want     int
		wantZero bool
	}{
		{
			name:   "seconds format",
			header: "30",
			want:   30,
		},
		{
			name:   "large seconds",
			header: "120",
			want:   120,
		},
		{
			name:     "empty header",
			header:   "",
			wantZero: true,
		},
		{
			name:     "invalid format",
			header:   "invalid",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers := http.Header{}
			if tt.header != "" {
				headers.Set("Retry-After", tt.header)
			}

			result := parseRetryAfter(headers)

			if tt.wantZero {
				if result != 0 {
					t.Errorf("parseRetryAfter() = %v, want 0", result)
				}
			} else {
				if int(result.Seconds()) != tt.want {
					t.Errorf("parseRetryAfter() = %v seconds, want %v", int(result.Seconds()), tt.want)
				}
			}
		})
	}
}

func TestAnthropicErrorTypes(t *testing.T) {
	if AnthropicErrorInvalidRequest != "invalid_request_error" {
		t.Errorf("AnthropicErrorInvalidRequest = %q, want %q", AnthropicErrorInvalidRequest, "invalid_request_error")
	}
	if AnthropicErrorAuthentication != "authentication_error" {
		t.Errorf("AnthropicErrorAuthentication = %q, want %q", AnthropicErrorAuthentication, "authentication_error")
	}
	if AnthropicErrorPermissionDenied != "permission_error" {
		t.Errorf("AnthropicErrorPermissionDenied = %q, want %q", AnthropicErrorPermissionDenied, "permission_error")
	}
	if AnthropicErrorNotFound != "not_found_error" {
		t.Errorf("AnthropicErrorNotFound = %q, want %q", AnthropicErrorNotFound, "not_found_error")
	}
	if AnthropicErrorRateLimit != "rate_limit_error" {
		t.Errorf("AnthropicErrorRateLimit = %q, want %q", AnthropicErrorRateLimit, "rate_limit_error")
	}
	if AnthropicErrorAPI != "api_error" {
		t.Errorf("AnthropicErrorAPI = %q, want %q", AnthropicErrorAPI, "api_error")
	}
	if AnthropicErrorOverloaded != "overloaded_error" {
		t.Errorf("AnthropicErrorOverloaded = %q, want %q", AnthropicErrorOverloaded, "overloaded_error")
	}
}
