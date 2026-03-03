// Package sse provides Server-Sent Events (SSE) parsing for streaming responses.
//
// SSE is a standard format for streaming real-time data from server to client.
// Each event consists of fields in the format "field: value" separated by newlines,
// with events separated by blank lines.
//
// OpenAI uses SSE for streaming chat completions with JSON data in the "data:" field.
// The stream terminates with "data: [DONE]".
//
// Example SSE stream:
//
//	data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":"Hello"}}]}
//
//	data: {"id":"chatcmpl-xxx","choices":[{"delta":{"content":" world"}}]}
//
//	data: [DONE]
package sse
