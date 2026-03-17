package anthropic

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

const (
	AnthropicErrorInvalidRequest   = "invalid_request_error"
	AnthropicErrorAuthentication   = "authentication_error"
	AnthropicErrorPermissionDenied = "permission_error"
	AnthropicErrorNotFound         = "not_found_error"
	AnthropicErrorRateLimit        = "rate_limit_error"
	AnthropicErrorAPI              = "api_error"
	AnthropicErrorOverloaded       = "overloaded_error"
)

func mapError(statusCode int, body []byte, respHeader http.Header) *provider.Error {
	var errResp ErrorResponse
	if err := parseJSON(body, &errResp); err == nil && errResp.Error.Message != "" {
		code := mapErrorTypeToCode(errResp.Error.Type, statusCode)
		return &provider.Error{
			Code:       code,
			Message:    errResp.Error.Message,
			Provider:   "anthropic",
			StatusCode: statusCode,
			RetryAfter: parseRetryAfter(respHeader),
		}
	}

	return &provider.Error{
		Code:       mapStatusCodeToCode(statusCode),
		Message:    fmt.Sprintf("HTTP %d: %s", statusCode, string(body)),
		Provider:   "anthropic",
		StatusCode: statusCode,
		RetryAfter: parseRetryAfter(respHeader),
	}
}

func mapErrorTypeToCode(errorType string, statusCode int) provider.ErrorCode {
	switch errorType {
	case AnthropicErrorInvalidRequest:
		return provider.CodeInvalidRequest
	case AnthropicErrorAuthentication:
		return provider.CodeAuthenticationFailed
	case AnthropicErrorPermissionDenied:
		return provider.CodeAuthenticationFailed
	case AnthropicErrorNotFound:
		return provider.CodeModelNotFound
	case AnthropicErrorRateLimit:
		return provider.CodeRateLimited
	case AnthropicErrorOverloaded:
		return provider.CodeRateLimited
	case AnthropicErrorAPI:
		return provider.CodeAPITimeout
	default:
		return mapStatusCodeToCode(statusCode)
	}
}

func mapStatusCodeToCode(statusCode int) provider.ErrorCode {
	switch statusCode {
	case 400:
		return provider.CodeInvalidRequest
	case 401:
		return provider.CodeAuthenticationFailed
	case 403:
		return provider.CodeAuthenticationFailed
	case 404:
		return provider.CodeModelNotFound
	case 413:
		return provider.CodeInvalidRequest
	case 429:
		return provider.CodeRateLimited
	case 500, 502, 503, 504:
		return provider.CodeAPITimeout
	case 529:
		return provider.CodeRateLimited
	default:
		return provider.CodeUnknown
	}
}

func parseRetryAfter(headers http.Header) time.Duration {
	retryAfter := headers.Get("retry-after")
	if retryAfter == "" {
		return 0
	}

	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}

	return 0
}

func parseJSON(data []byte, v any) error {
	return json.Unmarshal(data, v)
}
