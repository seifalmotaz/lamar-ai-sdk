package tool

import (
	"encoding/json"
	"testing"
)

func TestSuccessResult(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		expected map[string]interface{}
	}{
		{
			name: "simple string",
			data: "hello",
			expected: map[string]interface{}{
				"success": true,
				"data":    "hello",
			},
		},
		{
			name: "map data",
			data: map[string]interface{}{
				"temperature": 72,
				"condition":   "sunny",
			},
			expected: map[string]interface{}{
				"success": true,
				"data": map[string]interface{}{
					"temperature": float64(72),
					"condition":   "sunny",
				},
			},
		},
		{
			name: "nil data",
			data: nil,
			expected: map[string]interface{}{
				"success": true,
				"data":    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SuccessResult(tt.data)

			var got map[string]interface{}
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			// Check success field
			if got["success"] != true {
				t.Errorf("success = %v, want true", got["success"])
			}

			// Check data field exists
			if _, ok := got["data"]; !ok {
				t.Error("missing 'data' field in result")
			}
		})
	}
}

func TestErrorResult(t *testing.T) {
	tests := []struct {
		name      string
		errorType string
		message   string
		expected  map[string]interface{}
	}{
		{
			name:      "not_found error",
			errorType: "not_found",
			message:   "Resource not found",
			expected: map[string]interface{}{
				"success": false,
				"error":   "not_found",
				"message": "Resource not found",
			},
		},
		{
			name:      "timeout error",
			errorType: "timeout",
			message:   "Request timed out after 30s",
			expected: map[string]interface{}{
				"success": false,
				"error":   "timeout",
				"message": "Request timed out after 30s",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ErrorResult(tt.errorType, tt.message)

			var got map[string]interface{}
			if err := json.Unmarshal(result, &got); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}

			if got["success"] != false {
				t.Errorf("success = %v, want false", got["success"])
			}

			if got["error"] != tt.errorType {
				t.Errorf("error = %v, want %v", got["error"], tt.errorType)
			}

			if got["message"] != tt.message {
				t.Errorf("message = %v, want %v", got["message"], tt.message)
			}
		})
	}
}

func TestSuccessResultJSONValid(t *testing.T) {
	result := SuccessResult(map[string]int{"count": 42})

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("SuccessResult produced invalid JSON: %v", err)
	}
}

func TestErrorResultJSONValid(t *testing.T) {
	result := ErrorResult("test_error", "Test message")

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(result, &parsed); err != nil {
		t.Fatalf("ErrorResult produced invalid JSON: %v", err)
	}
}
