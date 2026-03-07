package image

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type mockImageModel struct {
	err            error
	images         [][]byte
	revisedPrompts []string
}

func (m *mockImageModel) Provider() string { return "mock" }
func (m *mockImageModel) ModelID() string  { return "mock-dalle-3" }
func (m *mockImageModel) GenerateImage(ctx context.Context, req *provider.ImageRequest) (*provider.ImageResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.ImageResult{
		Images:         m.images,
		RevisedPrompts: m.revisedPrompts,
	}, nil
}
func (m *mockImageModel) MaxImagesPerCall() int { return 1 }

func TestGenerateNilModel(t *testing.T) {
	_, err := Generate(context.Background(), nil, "a sunset")
	if err != provider.ErrInvalidModel {
		t.Errorf("expected ErrInvalidModel, got %v", err)
	}
}

func TestGenerateEmptyPrompt(t *testing.T) {
	model := &mockImageModel{}
	_, err := Generate(context.Background(), model, "")
	if err != provider.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGenerateSuccess(t *testing.T) {
	model := &mockImageModel{
		images:         [][]byte{[]byte("test image data")},
		revisedPrompts: []string{"A beautiful sunset"},
	}
	ctx := context.Background()
	result, err := Generate(ctx, model, "A sunset over mountains")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Images) != 1 {
		t.Errorf("expected 1 image, got %d", len(result.Images))
	}
	if string(result.Images[0]) != "test image data" {
		t.Errorf("expected 'test image data', got %s", result.Images[0])
	}
	if len(result.RevisedPrompts) != 1 {
		t.Errorf("expected 1 revised prompt, got %d", len(result.RevisedPrompts))
	}
}

func TestGenerateError(t *testing.T) {
	model := &mockImageModel{
		err: errors.New("generation failed"),
	}
	ctx := context.Background()
	_, err := Generate(ctx, model, "A landscape")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "generation failed" {
		t.Errorf("expected 'generation failed', got %v", err)
	}
}

func TestGenerateContextCancellation(t *testing.T) {
	model := &mockImageModel{
		images: [][]byte{[]byte("test")},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Generate(ctx, model, "A landscape")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestGenerateDefaultTimeout(t *testing.T) {
	model := &mockImageModel{
		images: [][]byte{[]byte("test")},
	}
	ctx := context.Background()
	_, err := Generate(ctx, model, "A landscape")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateCustomTimeout(t *testing.T) {
	model := &mockImageModel{
		images: [][]byte{[]byte("test")},
	}
	ctx := context.Background()
	_, err := Generate(ctx, model, "A landscape", WithTimeout(60*time.Second))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestGenerateNoTimeout(t *testing.T) {
	model := &mockImageModel{
		images: [][]byte{[]byte("test")},
	}
	ctx := context.Background()
	_, err := Generate(ctx, model, "A landscape", WithNoTimeout())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptionsWithSize(t *testing.T) {
	cfg := defaultConfig()
	WithSize("1024x1024")(cfg)
	if cfg.Size != "1024x1024" {
		t.Errorf("expected '1024x1024', got %s", cfg.Size)
	}
}

func TestOptionsWithQuality(t *testing.T) {
	cfg := defaultConfig()
	WithQuality("hd")(cfg)
	if cfg.Quality != "hd" {
		t.Errorf("expected 'hd', got %s", cfg.Quality)
	}
}

func TestOptionsWithFormat(t *testing.T) {
	cfg := defaultConfig()
	WithFormat("png")(cfg)
	if cfg.Format != "png" {
		t.Errorf("expected 'png', got %s", cfg.Format)
	}
}

func TestOptionsWithBackground(t *testing.T) {
	cfg := defaultConfig()
	WithBackground("transparent")(cfg)
	if cfg.Background != "transparent" {
		t.Errorf("expected 'transparent', got %s", cfg.Background)
	}
}

func TestOptionsWithN(t *testing.T) {
	cfg := defaultConfig()
	WithN(3)(cfg)
	if cfg.N != 3 {
		t.Errorf("expected 3, got %d", cfg.N)
	}
}

func TestDetermineMediaType(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"png", "image/png"},
		{"jpeg", "image/jpeg"},
		{"jpg", "image/jpeg"},
		{"webp", "image/webp"},
		{"", "image/png"},
		{"unknown", "image/png"},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := determineMediaType(tt.format)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg == nil {
		t.Error("expected non-nil config")
	}
}
