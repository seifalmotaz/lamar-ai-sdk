package transcription

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type mockTranscriptionModel struct {
	err      error
	text     string
	language string
	duration float64
	segments []provider.TranscriptSegment
}

func (m *mockTranscriptionModel) Provider() string { return "mock" }
func (m *mockTranscriptionModel) ModelID() string  { return "mock-whisper-1" }
func (m *mockTranscriptionModel) Transcribe(ctx context.Context, req *provider.TranscriptionRequest) (*provider.TranscriptionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &provider.TranscriptionResult{
		Text:     m.text,
		Language: m.language,
		Duration: m.duration,
		Segments: m.segments,
	}, nil
}

func TestTranscribeNilModel(t *testing.T) {
	_, err := Transcribe(context.Background(), nil, []byte("audio"), "audio/mp3")
	if err != provider.ErrInvalidModel {
		t.Errorf("expected ErrInvalidModel, got %v", err)
	}
}

func TestTranscribeEmptyAudio(t *testing.T) {
	model := &mockTranscriptionModel{}
	_, err := Transcribe(context.Background(), model, []byte{}, "audio/mp3")
	if err != provider.ErrInvalidInput {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTranscribeEmptyMediaType(t *testing.T) {
	model := &mockTranscriptionModel{}
	_, err := Transcribe(context.Background(), model, []byte("audio"), "")
	if err != provider.ErrInvalidMediaType {
		t.Errorf("expected ErrInvalidMediaType, got %v", err)
	}
}

func TestTranscribeSuccess(t *testing.T) {
	model := &mockTranscriptionModel{
		text:     "Hello world",
		language: "en",
		duration: 5.2,
	}
	ctx := context.Background()
	result, err := Transcribe(ctx, model, []byte("audio data"), "audio/mp3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "Hello world" {
		t.Errorf("expected 'Hello world', got %s", result.Text)
	}
	if result.Language != "en" {
		t.Errorf("expected 'en', got %s", result.Language)
	}
	if result.Duration != 5.2 {
		t.Errorf("expected 5.2, got %f", result.Duration)
	}
}

func TestTranscribeError(t *testing.T) {
	model := &mockTranscriptionModel{
		err: errors.New("transcription failed"),
	}
	ctx := context.Background()
	_, err := Transcribe(ctx, model, []byte("audio"), "audio/mp3")
	if err == nil {
		t.Error("expected error, got nil")
	}
	if err.Error() != "transcription failed" {
		t.Errorf("expected 'transcription failed', got %v", err)
	}
}

func TestTranscribeContextCancellation(t *testing.T) {
	model := &mockTranscriptionModel{
		text: "test",
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := Transcribe(ctx, model, []byte("audio"), "audio/mp3")
	if err == nil {
		t.Error("expected error for canceled context")
	}
}

func TestTranscribeDefaultTimeout(t *testing.T) {
	model := &mockTranscriptionModel{
		text: "test",
	}
	ctx := context.Background()
	_, err := Transcribe(ctx, model, []byte("audio"), "audio/mp3")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTranscribeCustomTimeout(t *testing.T) {
	model := &mockTranscriptionModel{
		text: "test",
	}
	ctx := context.Background()
	_, err := Transcribe(ctx, model, []byte("audio"), "audio/mp3", WithTimeout(30*time.Second))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTranscribeNoTimeout(t *testing.T) {
	model := &mockTranscriptionModel{
		text: "test",
	}
	ctx := context.Background()
	_, err := Transcribe(ctx, model, []byte("audio"), "audio/mp3", WithNoTimeout())
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestOptionsWithLanguage(t *testing.T) {
	cfg := defaultConfig()
	WithLanguage("en")(cfg)
	if cfg.Language != "en" {
		t.Errorf("expected 'en', got %s", cfg.Language)
	}
}

func TestOptionsWithPrompt(t *testing.T) {
	cfg := defaultConfig()
	WithPrompt("transcription hint")(cfg)
	if cfg.Prompt != "transcription hint" {
		t.Errorf("expected 'transcription hint', got %s", cfg.Prompt)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := defaultConfig()
	if cfg == nil {
		t.Error("expected non-nil config")
	}
}
