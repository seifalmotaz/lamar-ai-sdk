package generate

import "github.com/seifalmotaz/lamar-ai-sdk/provider"

// Result contains the result of a text generation request.
// Use the accessor methods to safely retrieve values.
type Result struct {
	inner *provider.GenerateResult
}

func newResult(inner *provider.GenerateResult) *Result {
	return &Result{inner: inner}
}

// Text returns the generated text content.
func (r *Result) Text() string {
	if r == nil || r.inner == nil {
		return ""
	}
	return r.inner.Text
}

// Content returns all content parts in the response.
func (r *Result) Content() []provider.Content {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.Content
}

// ToolCalls returns the tool calls made by the model.
func (r *Result) ToolCalls() []provider.ToolCall {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.ToolCalls
}

// FinishReason returns the reason why generation stopped.
func (r *Result) FinishReason() provider.FinishReason {
	if r == nil || r.inner == nil {
		return ""
	}
	return r.inner.FinishReason
}

// Usage returns the token usage statistics.
func (r *Result) Usage() provider.Usage {
	if r == nil || r.inner == nil {
		return provider.Usage{}
	}
	return r.inner.Usage
}

// String returns the generated text. Implements fmt.Stringer.
func (r *Result) String() string {
	return r.Text()
}
