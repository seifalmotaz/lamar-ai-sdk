package speech

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type mockSpeechModel struct {
	err    error
	audio  []byte
	format string
}

func (m *mockSpeechModel) Provider() string { return "mock" }
func (m *mockSpeechModel) ModelID() string  { return "mock-speech-1" }
func (m *mockSpeechModel) Synthesize(ctx context.Context, req *provider.SpeechRequest) (*provider.SpeechResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.SpeechResult{
		Audio:     m.audio,
		MediaType: m.format,
	}, nil
}

func TestSynthesizeNilModel(t *testing.T) {
	_, err := Synthesize(context.Background(), nil, "hello")
	if err != provider.ErrInvalidModel {
		t.Errorf("expected ErrInvalidModel, got %v", err)
	}
}

func TestSynthesizeEmptyText(t *testing.T) {
	model := &mockSpeechModel{}
	_, err := Synthesize(context.Background(), model, "")
	if err != provider.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSynthesizeSuccess(t *testing.T) {
	model := &mockSpeechModel{
		audio:  []byte("test audio data"),
		format: "audio/mp3",
	}
	ctx := context.Background()
	result, err := Synthesize(ctx, model, "Hello, world!")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(result.Audio) != "test audio data" {
		t.Errorf("expected 'test audio data', got %s", result.Audio)
	}
	if result.MediaType != "audio/mp3" {
		t.Errorf("expected 'audio/mp3', got %s", result.MediaType)
	}
}

func TestSynthesizeError(t *testing.T) {
	model := &mockSpeechModel{
		err: errors.New("synthesis failed"),
	}
	ctx := context.Background()
	_, err := Synthesize(ctx, model, "Hello")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "synthesis failed" {
		t.Errorf("expected 'synthesis failed', got %v", err)
	}
}

func TestSynthesizeContextCancellation(t *testing.T) {
	model := &mockSpeechModel{
		audio: []byte("test"),
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Synthesize(ctx, model, "Hello")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestSynthesizeDefaultTimeout(t *testing.T) {
	model := &mockSpeechModel{
		audio: []byte("test"),
	}
	ctx := context.Background()
	_, err := Synthesize(ctx, model, "Hello")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSynthesizeCustomTimeout(t *testing.T) {
	model := &mockSpeechModel{
		audio: []byte("test"),
	}
	ctx := context.Background()
	_, err := Synthesize(ctx, model, "Hello", WithTimeout(5*time.Second))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSynthesizeNoTimeout(t *testing.T) {
	model := &mockSpeechModel{
		audio: []byte("test"),
	}
	ctx := context.Background()
	_, err := Synthesize(ctx, model, "Hello", WithNoTimeout())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptionsWithVoice(t *testing.T) {
	cfg := defaultConfig()
	WithVoice("alloy")(cfg)
	if cfg.Voice != "alloy" {
		t.Errorf("expected 'alloy', got %s", cfg.Voice)
	}
}

func TestOptionsWithFormat(t *testing.T) {
	cfg := defaultConfig()
	WithFormat("mp3")(cfg)
	if cfg.Format != "mp3" {
		t.Errorf("expected 'mp3', got %s", cfg.Format)
	}
}

func TestOptionsWithSpeed(t *testing.T) {
	cfg := defaultConfig()
	WithSpeed(1.5)(cfg)
	if cfg.Speed != 1.5 {
		t.Errorf("expected 1.5, got %f", cfg.Speed)
	}
}

func TestOptionsWithInstructions(t *testing.T) {
	cfg := defaultConfig()
	WithInstructions("speak slowly")(cfg)
	if cfg.Instructions != "speak slowly" {
		t.Errorf("expected 'speak slowly', got %s", cfg.Instructions)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg == nil {
		t.Error("expected non-nil config")
	}
}
