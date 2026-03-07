package openai

import (
	"testing"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestImageModel_ModelInfo(t *testing.T) {
	p := NewProvider()
	model := p.Image("dall-e-3")

	if model.Provider() != "openai" {
		t.Errorf("expected provider 'openai', got %s", model.Provider())
	}
	if model.ModelID() != "dall-e-3" {
		t.Errorf("expected model ID 'dall-e-3', got %s", model.ModelID())
	}
}

func TestImageModel_MaxImagesPerCall(t *testing.T) {
	p := NewProvider()

	tests := []struct {
		modelID     string
		expectedMax int
	}{
		{"dall-e-2", 10},
		{"dall-e-3", 1},
		{"gpt-image-1", 10},
		{"gpt-image-1-mini", 10},
		{"unknown-model", 1},
	}

	for _, tt := range tests {
		t.Run(tt.modelID, func(t *testing.T) {
			model := p.Image(tt.modelID)
			if got := model.MaxImagesPerCall(); got != tt.expectedMax {
				t.Errorf("MaxImagesPerCall() = %d, want %d", got, tt.expectedMax)
			}
		})
	}
}

func TestImageModel_Interface(t *testing.T) {
	p := NewProvider()
	model := p.Image("dall-e-3")

	var _ provider.ImageModel = model
	var _ provider.Model = model

	if !provider.CanGenerateImage(model) {
		t.Error("expected CanGenerateImage to return true")
	}
}

func TestTranscriptionModel_ModelInfo(t *testing.T) {
	p := NewProvider()
	model := p.Transcription("whisper-1")

	if model.Provider() != "openai" {
		t.Errorf("expected provider 'openai', got %s", model.Provider())
	}
	if model.ModelID() != "whisper-1" {
		t.Errorf("expected model ID 'whisper-1', got %s", model.ModelID())
	}
}

func TestTranscriptionModel_Interface(t *testing.T) {
	p := NewProvider()
	model := p.Transcription("whisper-1")

	var _ provider.TranscriptionModel = model
	var _ provider.Model = model

	if !provider.CanTranscribe(model) {
		t.Error("expected CanTranscribe to return true")
	}
}

func TestSpeechModel_ModelInfo(t *testing.T) {
	p := NewProvider()
	model := p.Speech("tts-1")

	if model.Provider() != "openai" {
		t.Errorf("expected provider 'openai', got %s", model.Provider())
	}
	if model.ModelID() != "tts-1" {
		t.Errorf("expected model ID 'tts-1', got %s", model.ModelID())
	}
}

func TestSpeechModel_Interface(t *testing.T) {
	p := NewProvider()
	model := p.Speech("tts-1")

	var _ provider.SpeechModel = model
	var _ provider.Model = model

	if !provider.CanSynthesize(model) {
		t.Error("expected CanSynthesize to return true")
	}
}

func TestImageOptions(t *testing.T) {
	p := NewProvider()

	model := p.Image("dall-e-3",
		ImageQuality("hd"),
		ImageSize("1024x1024"),
		ImageFormat("png"),
		ImageUser("test-user"),
	)

	if model.ModelID() != "dall-e-3" {
		t.Errorf("expected model ID 'dall-e-3', got %s", model.ModelID())
	}
}

func TestTranscriptionOptions(t *testing.T) {
	p := NewProvider()

	model := p.Transcription("whisper-1",
		TranscriptionLanguage("en"),
		TranscriptionPrompt("Transcribe this audio"),
		TranscriptionTemperature(0.5),
		TranscriptionTimestampGranularity("word"),
	)

	if model.ModelID() != "whisper-1" {
		t.Errorf("expected model ID 'whisper-1', got %s", model.ModelID())
	}
}

func TestSpeechOptions(t *testing.T) {
	p := NewProvider()

	model := p.Speech("tts-1",
		SpeechVoice("alloy"),
		SpeechFormat("mp3"),
		SpeechSpeed(1.0),
		SpeechInstructions("Speak slowly"),
	)

	if model.ModelID() != "tts-1" {
		t.Errorf("expected model ID 'tts-1', got %s", model.ModelID())
	}
}

func TestConvenienceMethods(t *testing.T) {
	p := NewProvider()

	t.Run("DALLE2", func(t *testing.T) {
		model := p.DALLE2()
		if model.ModelID() != "dall-e-2" {
			t.Errorf("expected 'dall-e-2', got %s", model.ModelID())
		}
	})

	t.Run("DALLE3", func(t *testing.T) {
		model := p.DALLE3()
		if model.ModelID() != "dall-e-3" {
			t.Errorf("expected 'dall-e-3', got %s", model.ModelID())
		}
	})

	t.Run("GPTImage1", func(t *testing.T) {
		model := p.GPTImage1()
		if model.ModelID() != "gpt-image-1" {
			t.Errorf("expected 'gpt-image-1', got %s", model.ModelID())
		}
	})

	t.Run("Whisper1", func(t *testing.T) {
		model := p.Whisper1()
		if model.ModelID() != "whisper-1" {
			t.Errorf("expected 'whisper-1', got %s", model.ModelID())
		}
	})

	t.Run("TTS1", func(t *testing.T) {
		model := p.TTS1()
		if model.ModelID() != "tts-1" {
			t.Errorf("expected 'tts-1', got %s", model.ModelID())
		}
	})

	t.Run("TTS1HD", func(t *testing.T) {
		model := p.TTS1HD()
		if model.ModelID() != "tts-1-hd" {
			t.Errorf("expected 'tts-1-hd', got %s", model.ModelID())
		}
	})

	t.Run("GPT4oMiniTTS", func(t *testing.T) {
		model := p.GPT4oMiniTTS()
		if model.ModelID() != "gpt-4o-mini-tts" {
			t.Errorf("expected 'gpt-4o-mini-tts', got %s", model.ModelID())
		}
	})
}
