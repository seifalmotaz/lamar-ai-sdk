package generate

import "github.com/seifalmotaz/lamar-sdk/provider"

type Result struct {
	inner *provider.GenerateResult
}

func newResult(inner *provider.GenerateResult) *Result {
	return &Result{inner: inner}
}

func (r *Result) Text() string {
	if r == nil || r.inner == nil {
		return ""
	}
	return r.inner.Text
}

func (r *Result) Content() []provider.Content {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.Content
}

func (r *Result) ToolCalls() []provider.ToolCall {
	if r == nil || r.inner == nil {
		return nil
	}
	return r.inner.ToolCalls
}

func (r *Result) FinishReason() provider.FinishReason {
	if r == nil || r.inner == nil {
		return ""
	}
	return r.inner.FinishReason
}

func (r *Result) Usage() provider.Usage {
	if r == nil || r.inner == nil {
		return provider.Usage{}
	}
	return r.inner.Usage
}

func (r *Result) String() string {
	return r.Text()
}
