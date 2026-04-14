package middleware

import (
	"context"
	"errors"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestWrapper_Generate_NoMiddleware(t *testing.T) {
	wrapper := NewWrapper("test", nil)

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	coreCalled := false
	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
		coreCalled = true
		return &provider.GenerateResult{
			Text:         "world",
			FinishReason: provider.FinishReasonStop,
		}, nil
	}

	result, err := wrapper.Generate(context.Background(), "gpt-4", req, core)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !coreCalled {
		t.Fatal("core function was not called")
	}
	if result.Text != "world" {
		t.Errorf("expected text 'world', got '%s'", result.Text)
	}
}

func TestWrapper_Generate_WithMiddleware(t *testing.T) {
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			resp, err := next.Handle(ctx, req)
			if err != nil {
				return nil, err
			}
			genResp := resp.(*GenerateResponse)
			genResp.Text = "intercepted: " + genResp.Text
			return genResp, nil
		})
	}

	wrapper := NewWrapper("test", []Middleware{mw})

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
		return &provider.GenerateResult{
			Text:         "world",
			FinishReason: provider.FinishReasonStop,
		}, nil
	}

	result, err := wrapper.Generate(context.Background(), "gpt-4", req, core)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "intercepted: world" {
		t.Errorf("expected 'intercepted: world', got '%s'", result.Text)
	}
}

func TestWrapper_Generate_ErrorPropagation(t *testing.T) {
	wrapper := NewWrapper("test", nil)

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	expectedErr := errors.New("core error")
	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
		return nil, expectedErr
	}

	_, err := wrapper.Generate(context.Background(), "gpt-4", req, core)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected core error, got %v", err)
	}
}

func TestWrapper_Generate_WithMiddlewareError(t *testing.T) {
	expectedErr := errors.New("middleware error")
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			return nil, expectedErr
		})
	}

	wrapper := NewWrapper("test", []Middleware{mw})

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
		t.Fatal("core should not be called when middleware fails")
		return nil, nil
	}

	_, err := wrapper.Generate(context.Background(), "gpt-4", req, core)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected middleware error, got %v", err)
	}
}

func TestWrapper_Stream_NoMiddleware(t *testing.T) {
	wrapper := NewWrapper("test", nil)

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	coreCalled := false
	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
		coreCalled = true
		stream := make(chan provider.StreamPart, 10)
		done := make(chan struct{})
		close(done)
		return &provider.StreamResult{
			Stream: stream,
			Done:   done,
			Text:   func() (string, error) { return "test", nil },
		}, nil
	}

	result, err := wrapper.Stream(context.Background(), "gpt-4", req, core)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !coreCalled {
		t.Fatal("core function was not called")
	}
	if result == nil {
		t.Fatal("result is nil")
	}
}

func TestWrapper_Stream_WithMiddleware(t *testing.T) {
	streamText := ""
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			resp, err := next.Handle(ctx, req)
			if err != nil {
				return nil, err
			}
			streamResp := resp.(*StreamResponse)
			// Capture the text func result
			streamText, _ = streamResp.TextFunc()
			return streamResp, nil
		})
	}

	wrapper := NewWrapper("test", []Middleware{mw})

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
		stream := make(chan provider.StreamPart, 10)
		done := make(chan struct{})
		close(done)
		return &provider.StreamResult{
			Stream: stream,
			Done:   done,
			Text:   func() (string, error) { return "world", nil },
		}, nil
	}

	result, err := wrapper.Stream(context.Background(), "gpt-4", req, core)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if streamText != "world" {
		t.Errorf("expected 'world', got '%s'", streamText)
	}
}

func TestWrapper_Stream_ErrorPropagation(t *testing.T) {
	wrapper := NewWrapper("test", nil)

	req := &provider.GenerateRequest{
		Prompt: "Hello",
	}

	expectedErr := errors.New("stream error")
	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
		return nil, expectedErr
	}

	_, err := wrapper.Stream(context.Background(), "gpt-4", req, core)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected stream error, got %v", err)
	}
}

func TestWrapper_Embed_NoMiddleware(t *testing.T) {
	wrapper := NewWrapper("test", nil)

	coreCalled := false
	core := func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error) {
		coreCalled = true
		return [][]float64{{1.0, 2.0}}, provider.Usage{PromptTokens: 10}, nil
	}

	embeddings, usage, err := wrapper.Embed(context.Background(), "embed-model", []string{"hello"}, core)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !coreCalled {
		t.Fatal("core function was not called")
	}
	if len(embeddings) != 1 || len(embeddings[0]) != 2 {
		t.Errorf("unexpected embeddings: %v", embeddings)
	}
	if usage.PromptTokens != 10 {
		t.Errorf("expected 10 prompt tokens, got %d", usage.PromptTokens)
	}
}

func TestWrapper_Embed_WithMiddleware(t *testing.T) {
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			resp, err := next.Handle(ctx, req)
			if err != nil {
				return nil, err
			}
			embedResp := resp.(*EmbedResponse)
			// Multiply embeddings by 2 as intercept
			for i := range embedResp.Embeddings {
				for j := range embedResp.Embeddings[i] {
					embedResp.Embeddings[i][j] *= 2
				}
			}
			return embedResp, nil
		})
	}

	wrapper := NewWrapper("test", []Middleware{mw})

	core := func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error) {
		return [][]float64{{1.0, 2.0}}, provider.Usage{PromptTokens: 10}, nil
	}

	embeddings, _, err := wrapper.Embed(context.Background(), "embed-model", []string{"hello"}, core)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if embeddings[0][0] != 2.0 || embeddings[0][1] != 4.0 {
		t.Errorf("expected [2.0, 4.0], got %v", embeddings[0])
	}
}

func TestWrapper_Embed_ErrorPropagation(t *testing.T) {
	wrapper := NewWrapper("test", nil)

	expectedErr := errors.New("embed error")
	core := func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error) {
		return nil, provider.Usage{}, expectedErr
	}

	_, _, err := wrapper.Embed(context.Background(), "embed-model", []string{"hello"}, core)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected embed error, got %v", err)
	}
}

func TestWrapper_Generate_RequestConversion(t *testing.T) {
	var capturedReq *GenerateRequest
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			capturedReq = req.(*GenerateRequest)
			return next.Handle(ctx, req)
		})
	}

	wrapper := NewWrapper("myprovider", []Middleware{mw})

	providerReq := &provider.GenerateRequest{
		Prompt: "test prompt",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.Content{provider.Text("message")}},
		},
		Config: provider.Config{
			Temperature: 0.7,
			MaxTokens:   100,
		},
	}

	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.GenerateResult, error) {
		return &provider.GenerateResult{Text: "response"}, nil
	}

	_, _ = wrapper.Generate(context.Background(), "my-model", providerReq, core)

	if capturedReq == nil {
		t.Fatal("request was not captured")
	}
	if capturedReq.ProviderName != "myprovider" {
		t.Errorf("expected provider 'myprovider', got '%s'", capturedReq.ProviderName)
	}
	if capturedReq.Model != "my-model" {
		t.Errorf("expected model 'my-model', got '%s'", capturedReq.Model)
	}
	if capturedReq.Prompt != "test prompt" {
		t.Errorf("expected prompt 'test prompt', got '%s'", capturedReq.Prompt)
	}
	if len(capturedReq.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(capturedReq.Messages))
	}
	if capturedReq.Config.Temperature != 0.7 {
		t.Errorf("expected temperature 0.7, got %f", capturedReq.Config.Temperature)
	}
}

func TestWrapper_Embed_RequestConversion(t *testing.T) {
	var capturedReq *EmbedRequest
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			capturedReq = req.(*EmbedRequest)
			return next.Handle(ctx, req)
		})
	}

	wrapper := NewWrapper("test", []Middleware{mw})

	core := func(ctx context.Context, texts []string) ([][]float64, provider.Usage, error) {
		return nil, provider.Usage{}, nil
	}

	_, _, _ = wrapper.Embed(context.Background(), "embed-model", []string{"text1", "text2"}, core)

	if capturedReq == nil {
		t.Fatal("request was not captured")
	}
	if capturedReq.ProviderName != "test" {
		t.Errorf("expected provider 'test', got '%s'", capturedReq.ProviderName)
	}
	if capturedReq.Model != "embed-model" {
		t.Errorf("expected model 'embed-model', got '%s'", capturedReq.Model)
	}
	if len(capturedReq.Texts) != 2 {
		t.Errorf("expected 2 texts, got %d", len(capturedReq.Texts))
	}
	if capturedReq.Texts[0] != "text1" || capturedReq.Texts[1] != "text2" {
		t.Errorf("unexpected texts: %v", capturedReq.Texts)
	}
}

func TestWrapper_Stream_RequestConversion(t *testing.T) {
	var capturedReq *StreamRequest
	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, req Request) (Response, error) {
			capturedReq = req.(*StreamRequest)
			return next.Handle(ctx, req)
		})
	}

	wrapper := NewWrapper("streamtest", []Middleware{mw})

	providerReq := &provider.GenerateRequest{
		Prompt: "stream prompt",
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: []provider.Content{provider.Text("msg")}},
		},
		Config: provider.Config{
			Temperature: 0.5,
		},
	}

	stream := make(chan provider.StreamPart)
	done := make(chan struct{})
	close(done)

	core := func(ctx context.Context, req *provider.GenerateRequest) (*provider.StreamResult, error) {
		return &provider.StreamResult{
			Stream: stream,
			Done:   done,
		}, nil
	}

	_, _ = wrapper.Stream(context.Background(), "stream-model", providerReq, core)

	if capturedReq == nil {
		t.Fatal("request was not captured")
	}
	if capturedReq.ProviderName != "streamtest" {
		t.Errorf("expected provider 'streamtest', got '%s'", capturedReq.ProviderName)
	}
	if capturedReq.Model != "stream-model" {
		t.Errorf("expected model 'stream-model', got '%s'", capturedReq.Model)
	}
	if capturedReq.Prompt != "stream prompt" {
		t.Errorf("expected prompt 'stream prompt', got '%s'", capturedReq.Prompt)
	}
}

func TestWrapper_NewWrapper(t *testing.T) {
	w := NewWrapper("test", nil)
	if w.name != "test" {
		t.Errorf("expected name 'test', got '%s'", w.name)
	}
	if w.middlewares != nil {
		t.Errorf("expected nil middlewares, got %v", w.middlewares)
	}
}
