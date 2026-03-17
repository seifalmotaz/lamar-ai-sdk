package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestRetry_Success(t *testing.T) {
	attempts := 0
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		attempts++
		if attempts < 3 {
			return nil, &provider.Error{Code: provider.CodeRateLimited, Message: "rate limited"}
		}
		return &mockResponse{usage: provider.Usage{TotalTokens: 100}}, nil
	})

	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	handler := Retry(cfg)(base)
	resp, err := handler.Handle(context.Background(), &mockRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_MaxAttempts(t *testing.T) {
	attempts := 0
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		attempts++
		return nil, &provider.Error{Code: provider.CodeRateLimited, Message: "rate limited"}
	})

	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
	}

	handler := Retry(cfg)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{})

	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetry_NonRetriable(t *testing.T) {
	attempts := 0
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		attempts++
		return nil, &provider.Error{Code: provider.CodeInvalidRequest, Message: "invalid request"}
	})

	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
	}

	handler := Retry(cfg)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{})

	if err == nil {
		t.Fatal("expected error")
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt (no retry), got %d", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	attempts := 0
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		attempts++
		return nil, &provider.Error{Code: provider.CodeRateLimited, Message: "rate limited"}
	})

	cfg := RetryConfig{
		MaxAttempts:  10,
		InitialDelay: 1 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	handler := Retry(cfg)(base)
	_, err := handler.Handle(ctx, &mockRequest{})

	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
	if attempts > 1 {
		t.Errorf("expected at most 1 attempt, got %d", attempts)
	}
}

func TestRetry_RetryAfter(t *testing.T) {
	attempts := 0
	retryAfterUsed := false

	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		attempts++
		if attempts == 1 {
			return nil, &provider.Error{
				Code:       provider.CodeRateLimited,
				Message:    "rate limited",
				RetryAfter: 50 * time.Millisecond,
			}
		}
		retryAfterUsed = true
		return &mockResponse{usage: provider.Usage{TotalTokens: 100}}, nil
	})

	cfg := RetryConfig{
		MaxAttempts:  2,
		InitialDelay: 1 * time.Millisecond,
	}

	start := time.Now()
	handler := Retry(cfg)(base)
	resp, err := handler.Handle(context.Background(), &mockRequest{})
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}
	if !retryAfterUsed {
		t.Error("expected retry-after to be used")
	}
	// Should have waited at least 50ms from retry-after
	if elapsed < 40*time.Millisecond {
		t.Errorf("expected at least 40ms delay (retry-after), got %v", elapsed)
	}
}

func TestRetry_DefaultConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts 3, got %d", cfg.MaxAttempts)
	}
	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("expected InitialDelay 1s, got %v", cfg.InitialDelay)
	}
	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("expected MaxDelay 30s, got %v", cfg.MaxDelay)
	}
	if cfg.Multiplier != 2.0 {
		t.Errorf("expected Multiplier 2.0, got %v", cfg.Multiplier)
	}
	if cfg.RetryOn == nil {
		t.Error("expected RetryOn predicate to be set")
	}
}

func TestDefaultRetryPredicate(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "rate limited",
			err:      &provider.Error{Code: provider.CodeRateLimited},
			expected: true,
		},
		{
			name:     "timeout",
			err:      &provider.Error{Code: provider.CodeAPITimeout},
			expected: true,
		},
		{
			name:     "server error 500",
			err:      &provider.Error{StatusCode: 500},
			expected: true,
		},
		{
			name:     "server error 502",
			err:      &provider.Error{StatusCode: 502},
			expected: true,
		},
		{
			name:     "server error 503",
			err:      &provider.Error{StatusCode: 503},
			expected: true,
		},
		{
			name:     "client error 400",
			err:      &provider.Error{Code: provider.CodeInvalidRequest, StatusCode: 400},
			expected: false,
		},
		{
			name:     "client error 401",
			err:      &provider.Error{Code: provider.CodeAuthenticationFailed, StatusCode: 401},
			expected: false,
		},
		{
			name:     "generic error",
			err:      errors.New("generic error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DefaultRetryPredicate(tt.err)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestRetry_ZeroAttempts(t *testing.T) {
	// Should default to at least 1 attempt
	attempts := 0
	base := HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		attempts++
		return &mockResponse{usage: provider.Usage{TotalTokens: 100}}, nil
	})

	cfg := RetryConfig{
		MaxAttempts: 0,
	}

	handler := Retry(cfg)(base)
	_, err := handler.Handle(context.Background(), &mockRequest{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}
