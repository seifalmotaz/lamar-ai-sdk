package ollama

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func mapOllamaError(err error, modelID string) *provider.Error {
	var errResp ErrorResponse
	if jsonErr, ok := err.(*json.SyntaxError); ok {
		return &provider.Error{
			Code:     provider.CodeParseError,
			Message:  fmt.Sprintf("failed to parse error response: %v", jsonErr),
			Provider: "ollama",
			ModelID:  modelID,
			Cause:    err,
		}
	}

	if parseErr := json.Unmarshal([]byte(err.Error()), &errResp); parseErr == nil && errResp.Error != "" {
		errMsg := strings.ToLower(errResp.Error)

		if strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found") {
			return &provider.Error{
				Code:     provider.CodeModelNotFound,
				Message:  errResp.Error,
				Provider: "ollama",
				ModelID:  modelID,
			}
		}

		if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") {
			return &provider.Error{
				Code:     provider.CodeAPITimeout,
				Message:  fmt.Sprintf("failed to connect to Ollama server: %s", errResp.Error),
				Provider: "ollama",
				ModelID:  modelID,
				Cause:    err,
			}
		}

		return &provider.Error{
			Code:     provider.CodeUnknown,
			Message:  errResp.Error,
			Provider: "ollama",
			ModelID:  modelID,
		}
	}

	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "model") && strings.Contains(errMsg, "not found") {
		return &provider.Error{
			Code:     provider.CodeModelNotFound,
			Message:  err.Error(),
			Provider: "ollama",
			ModelID:  modelID,
			Cause:    err,
		}
	}

	if strings.Contains(errMsg, "connection refused") || strings.Contains(errMsg, "no such host") {
		return &provider.Error{
			Code:     provider.CodeAPITimeout,
			Message:  fmt.Sprintf("failed to connect to Ollama server: %v", err),
			Provider: "ollama",
			ModelID:  modelID,
			Cause:    err,
		}
	}

	return &provider.Error{
		Code:     provider.CodeUnknown,
		Message:  err.Error(),
		Provider: "ollama",
		ModelID:  modelID,
		Cause:    err,
	}
}
