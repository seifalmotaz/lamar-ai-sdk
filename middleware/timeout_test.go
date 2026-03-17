package middleware

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestTimeout(t *testing.T) {
	t.Run("applies default timeout", func(t *testing.T) {
		timeout := 100 * time.Millisecond
		mw := Timeout(TimeoutConfig{Default: timeout})

		start := time.Now()
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
				return &GenerateResponse{}, nil
			}
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{})
		if err == nil {
			t.Error("expected timeout error")
		}

		elapsed := time.Since(start)
		if elapsed > 500*time.Millisecond {
			t.Errorf("timeout took too long: %v", elapsed)
		}
	})

	t.Run("respects existing deadline", func(t *testing.T) {
		mw := Timeout(TimeoutConfig{Default: time.Second})

		called := false
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			called = true
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("expected deadline to be set")
			}
			if deadline.Sub(time.Now()) > 200*time.Millisecond {
				t.Error("expected shorter deadline from context")
			}
			return &GenerateResponse{}, nil
		}))

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		handler.Handle(ctx, &GenerateRequest{})
		if !called {
			t.Error("handler not called")
		}
	})

	t.Run("per-provider timeout override", func(t *testing.T) {
		mw := Timeout(TimeoutConfig{
			Default: time.Second,
			PerProvider: map[string]time.Duration{
				"openai": 50 * time.Millisecond,
			},
		})

		start := time.Now()
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
				return &GenerateResponse{}, nil
			}
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{ProviderName: "openai"})
		if err == nil {
			t.Error("expected timeout error")
		}

		elapsed := time.Since(start)
		if elapsed > 200*time.Millisecond {
			t.Errorf("provider timeout override not applied: %v", elapsed)
		}
	})

	t.Run("per-model timeout override", func(t *testing.T) {
		mw := Timeout(TimeoutConfig{
			Default: time.Second,
			PerProvider: map[string]time.Duration{
				"openai": 200 * time.Millisecond,
			},
			PerModel: map[string]time.Duration{
				"gpt-4o": 30 * time.Millisecond,
			},
		})

		start := time.Now()
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(time.Second):
				return &GenerateResponse{}, nil
			}
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{
			ProviderName: "openai",
			Model:        "gpt-4o",
		})
		if err == nil {
			t.Error("expected timeout error")
		}

		elapsed := time.Since(start)
		if elapsed > 150*time.Millisecond {
			t.Errorf("model timeout override not applied: %v", elapsed)
		}
	})

	t.Run("zero timeout passthrough", func(t *testing.T) {
		mw := Timeout(TimeoutConfig{Default: 0})

		called := false
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			called = true
			return &GenerateResponse{}, nil
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("handler not called")
		}
	})

	t.Run("wraps deadline exceeded with provider error", func(t *testing.T) {
		mw := Timeout(TimeoutConfig{Default: 10 * time.Millisecond})

		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{
			ProviderName: "openai",
			Model:        "gpt-4o",
		})

		var providerErr *provider.Error
		if !errors.As(err, &providerErr) {
			t.Fatalf("expected provider.Error, got: %v", err)
		}
		if providerErr.Code != provider.CodeAPITimeout {
			t.Errorf("expected CodeAPITimeout, got: %v", providerErr.Code)
		}
		if providerErr.Provider != "openai" {
			t.Errorf("expected provider 'openai', got: %v", providerErr.Provider)
		}
		if providerErr.ModelID != "gpt-4o" {
			t.Errorf("expected model 'gpt-4o', got: %v", providerErr.ModelID)
		}
	})
}

func TestTimeoutWithDefault(t *testing.T) {
	mw := TimeoutWithDefault(50 * time.Millisecond)

	start := time.Now()
	handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}))

	_, err := handler.Handle(context.Background(), &GenerateRequest{})
	if err == nil {
		t.Error("expected timeout error")
	}

	elapsed := time.Since(start)
	if elapsed > 200*time.Millisecond {
		t.Errorf("timeout took too long: %v", elapsed)
	}
}

func TestTimeoutPerProvider(t *testing.T) {
	timeouts := map[string]time.Duration{
		"openai":    30 * time.Millisecond,
		"anthropic": 100 * time.Millisecond,
	}
	mw := TimeoutPerProvider(timeouts)

	t.Run("openai timeout", func(t *testing.T) {
		start := time.Now()
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{ProviderName: "openai"})
		if err == nil {
			t.Error("expected timeout error")
		}

		elapsed := time.Since(start)
		if elapsed > 200*time.Millisecond {
			t.Errorf("timeout took too long: %v", elapsed)
		}
	})

	t.Run("unknown provider no timeout", func(t *testing.T) {
		called := false
		handler := mw(HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			called = true
			return &GenerateResponse{}, nil
		}))

		_, err := handler.Handle(context.Background(), &GenerateRequest{ProviderName: "unknown"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if !called {
			t.Error("handler not called")
		}
	})
}
