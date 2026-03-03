package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestMapStatusCodeToCode(t *testing.T) {
	tests := []struct {
		statusCode int
		want       provider.ErrorCode
	}{
		{400, provider.CodeInvalidRequest},
		{401, provider.CodeAuthenticationFailed},
		{403, provider.CodeAuthenticationFailed},
		{404, provider.CodeModelNotFound},
		{429, provider.CodeRateLimited},
		{500, provider.CodeAPITimeout},
		{502, provider.CodeAPITimeout},
		{503, provider.CodeAPITimeout},
		{504, provider.CodeAPITimeout},
		{200, provider.CodeUnknown},
		{418, provider.CodeUnknown},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.statusCode), func(t *testing.T) {
			got := mapStatusCodeToCode(tt.statusCode)
			if got != tt.want {
				t.Errorf("mapStatusCodeToCode(%d) = %v, want %v", tt.statusCode, got, tt.want)
			}
		})
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantCode   provider.ErrorCode
		wantMsg    string
	}{
		{
			name:       "OpenAI error format",
			statusCode: 401,
			body:       `{"error":{"message":"Incorrect API key","type":"invalid_request_error","code":"invalid_api_key"}}`,
			wantCode:   provider.CodeAuthenticationFailed,
			wantMsg:    "Incorrect API key",
		},
		{
			name:       "Rate limit with message",
			statusCode: 429,
			body:       `{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`,
			wantCode:   provider.CodeRateLimited,
			wantMsg:    "Rate limit exceeded",
		},
		{
			name:       "Model not found",
			statusCode: 404,
			body:       `{"error":{"message":"The model 'gpt-5' does not exist","type":"invalid_request_error"}}`,
			wantCode:   provider.CodeModelNotFound,
			wantMsg:    "The model 'gpt-5' does not exist",
		},
		{
			name:       "Unknown error format",
			statusCode: 500,
			body:       `internal server error`,
			wantCode:   provider.CodeAPITimeout,
			wantMsg:    "HTTP 500: internal server error",
		},
		{
			name:       "Invalid JSON",
			statusCode: 400,
			body:       `{invalid json}`,
			wantCode:   provider.CodeInvalidRequest,
			wantMsg:    "HTTP 400: {invalid json}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				StatusCode: tt.statusCode,
				Header:     make(http.Header),
			}
			body := []byte(tt.body)

			got := mapError(resp, body)

			if got.Code != tt.wantCode {
				t.Errorf("Code = %v, want %v", got.Code, tt.wantCode)
			}
			if got.Message != tt.wantMsg {
				t.Errorf("Message = %q, want %q", got.Message, tt.wantMsg)
			}
			if got.Provider != "openai" {
				t.Errorf("Provider = %q, want %q", got.Provider, "openai")
			}
			if got.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %d, want %d", got.StatusCode, tt.statusCode)
			}
		})
	}
}

func TestParseRetryAfter(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    time.Duration
	}{
		{
			name:    "no header",
			headers: nil,
			want:    0,
		},
		{
			name:    "seconds",
			headers: map[string]string{"Retry-After": "30"},
			want:    30 * time.Second,
		},
		{
			name:    "invalid format",
			headers: map[string]string{"Retry-After": "invalid"},
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := make(http.Header)
			for k, v := range tt.headers {
				h.Set(k, v)
			}
			got := parseRetryAfter(h)
			if got != tt.want {
				t.Errorf("parseRetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewClient(t *testing.T) {
	t.Run("default HTTP client", func(t *testing.T) {
		c := NewClient("https://api.example.com", nil)
		if c.HTTPClient != http.DefaultClient {
			t.Error("expected default HTTP client")
		}
		if c.BaseURL != "https://api.example.com" {
			t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.example.com")
		}
	})

	t.Run("custom HTTP client", func(t *testing.T) {
		custom := &http.Client{Timeout: 10 * time.Second}
		c := NewClient("https://api.example.com/v1/", custom)
		if c.HTTPClient != custom {
			t.Error("expected custom HTTP client")
		}
		if c.BaseURL != "https://api.example.com/v1" {
			t.Errorf("BaseURL = %q, want %q", c.BaseURL, "https://api.example.com/v1")
		}
	})
}

func TestClientSetHeader(t *testing.T) {
	c := NewClient("https://api.example.com", nil)
	c.SetHeader("Authorization", "Bearer test-key")
	c.SetHeader("X-Custom", "value")

	if c.Headers["Authorization"] != "Bearer test-key" {
		t.Error("Authorization header not set")
	}
	if c.Headers["X-Custom"] != "value" {
		t.Error("X-Custom header not set")
	}
}

func TestClientDoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[{"message":{"content":"Hello"}}]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)
	c.SetHeader("Authorization", "Bearer test-key")

	var result map[string]any
	err := c.Get(context.Background(), "/chat/completions", &result)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)
	c.SetHeader("Authorization", "Bearer invalid")

	var result map[string]any
	err := c.Get(context.Background(), "/chat/completions", &result)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("expected provider.Error, got %T", err)
	}
	if providerErr.Code != provider.CodeAuthenticationFailed {
		t.Errorf("Code = %v, want %v", providerErr.Code, provider.CodeAuthenticationFailed)
	}
}

func TestClientDoPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test","choices":[]}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)

	body := map[string]string{"model": "gpt-4"}
	var result map[string]any
	err := c.Post(context.Background(), "/chat/completions", body, &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoContextCanceled(t *testing.T) {
	c := NewClient("https://httpbin.org/delay/10", &http.Client{Timeout: 30 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var result map[string]any
	err := c.Get(ctx, "/get", &result)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != provider.ErrContextCanceled {
		t.Errorf("expected ErrContextCanceled, got %v", err)
	}
}

func TestClientDoWithNilBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)

	var result map[string]any
	err := c.Get(context.Background(), "/test", &result)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClientDoWithNilResult(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id":"test"}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)

	err := c.Get(context.Background(), "/test", nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMapErrorRetryAfter(t *testing.T) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     http.Header{"Retry-After": []string{"60"}},
	}
	body := []byte(`{"error":{"message":"Rate limit exceeded"}}`)

	got := mapError(resp, body)

	if got.RetryAfter != 60*time.Second {
		t.Errorf("RetryAfter = %v, want %v", got.RetryAfter, 60*time.Second)
	}
}

func BenchmarkMapError(b *testing.B) {
	resp := &http.Response{
		StatusCode: 429,
		Header:     make(http.Header),
	}
	body := []byte(`{"error":{"message":"Rate limit exceeded","type":"rate_limit_error"}}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mapError(resp, body)
	}
}

func BenchmarkParseRetryAfter(b *testing.B) {
	h := http.Header{"Retry-After": []string{"60"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = parseRetryAfter(h)
	}
}

func TestClientDoStreamSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("Accept header = %q, want %q", r.Header.Get("Accept"), "text/event-stream")
		}
		if r.Header.Get("Cache-Control") != "no-cache" {
			t.Errorf("Cache-Control header = %q, want %q", r.Header.Get("Cache-Control"), "no-cache")
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Content-Type header = %q, want %q", r.Header.Get("Content-Type"), "application/json")
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("data: {\"content\":\"hello\"}\n\n"))
		w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)
	c.SetHeader("Authorization", "Bearer test-key")

	body := map[string]string{"model": "gpt-4", "stream": "true"}
	rc, err := c.DoStream(context.Background(), http.MethodPost, "/chat/completions", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("failed to read stream: %v", err)
	}

	expected := "data: {\"content\":\"hello\"}\n\ndata: [DONE]\n\n"
	if string(data) != expected {
		t.Errorf("got %q, want %q", string(data), expected)
	}
}

func TestClientDoStreamError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error":{"message":"Invalid API key","type":"invalid_request_error"}}`))
	}))
	defer server.Close()

	c := NewClient(server.URL, nil)

	body := map[string]string{"model": "gpt-4"}
	_, err := c.DoStream(context.Background(), http.MethodPost, "/chat/completions", body)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var providerErr *provider.Error
	if !errors.As(err, &providerErr) {
		t.Errorf("expected provider.Error, got %T", err)
	}
	if providerErr.Code != provider.CodeAuthenticationFailed {
		t.Errorf("Code = %v, want %v", providerErr.Code, provider.CodeAuthenticationFailed)
	}
}

func TestClientDoStreamContextCanceled(t *testing.T) {
	c := NewClient("https://httpbin.org/delay/10", &http.Client{Timeout: 30 * time.Second})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	body := map[string]string{"model": "gpt-4"}
	_, err := c.DoStream(ctx, http.MethodPost, "/chat/completions", body)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != provider.ErrContextCanceled {
		t.Errorf("expected ErrContextCanceled, got %v", err)
	}
}
