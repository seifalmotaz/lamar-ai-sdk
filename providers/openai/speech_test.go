package openai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestSpeechModel_Synthesize(t *testing.T) {
	t.Run("successful synthesis", func(t *testing.T) {
		audioData := []byte("fake-mp3-audio-data")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("missing Authorization header")
			}
			if r.URL.Path != "/audio/speech" {
				t.Errorf("path = %q, want %q", r.URL.Path, "/audio/speech")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
			}

			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req["model"] != "tts-1" {
				t.Errorf("model = %q, want tts-1", req["model"])
			}
			if req["input"] != "Hello, world!" {
				t.Errorf("input = %q, want 'Hello, world!'", req["input"])
			}
			if req["voice"] != "alloy" {
				t.Errorf("voice = %q, want alloy", req["voice"])
			}
			if req["response_format"] != "mp3" {
				t.Errorf("response_format = %q, want mp3", req["response_format"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(audioData)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		result, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "Hello, world!",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(result.Audio) != string(audioData) {
			t.Errorf("audio data mismatch")
		}
		if result.MediaType != "audio/mpeg" {
			t.Errorf("MediaType = %q, want audio/mpeg", result.MediaType)
		}
	})

	t.Run("synthesis with custom voice", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["voice"] != "nova" {
				t.Errorf("voice = %q, want nova", req["voice"])
			}
			if req["response_format"] != "mp3" {
				t.Errorf("response_format = %v, want mp3", req["response_format"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1", SpeechVoice("nova"))

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("synthesis request voice overrides config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["voice"] != "request-voice" {
				t.Errorf("voice = %q, want request-voice", req["voice"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1", SpeechVoice("config-voice"))

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text:  "test",
			Voice: "request-voice",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("synthesis with different formats", func(t *testing.T) {
		tests := []struct {
			format    string
			mediaType string
		}{
			{"mp3", "audio/mpeg"},
			{"opus", "audio/opus"},
			{"aac", "audio/aac"},
			{"flac", "audio/flac"},
			{"wav", "audio/wav"},
			{"pcm", "audio/pcm"},
		}

		for _, tt := range tests {
			t.Run(tt.format, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req map[string]any
					json.NewDecoder(r.Body).Decode(&req)

					if req["response_format"] != tt.format {
						t.Errorf("response_format = %q, want %q", req["response_format"], tt.format)
					}

					w.Header().Set("Content-Type", tt.mediaType)
					w.Write([]byte("audio-data"))
				}))
				defer server.Close()

				p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
				model := p.Speech("tts-1", SpeechFormat(tt.format))

				result, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
					Text: "test",
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if result.MediaType != tt.mediaType {
					t.Errorf("MediaType = %q, want %q", result.MediaType, tt.mediaType)
				}
			})
		}
	})

	t.Run("synthesis request format overrides config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["response_format"] != "wav" {
				t.Errorf("response_format = %q, want wav", req["response_format"])
			}

			w.Header().Set("Content-Type", "audio/wav")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1", SpeechFormat("mp3"))

		result, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text:   "test",
			Format: "wav",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.MediaType != "audio/wav" {
			t.Errorf("MediaType = %q, want audio/wav", result.MediaType)
		}
	})

	t.Run("synthesis with speed parameter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["speed"] != float64(1.5) {
				t.Errorf("speed = %v, want 1.5", req["speed"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1", SpeechSpeed(1.5))

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("synthesis with instructions (GPT-4o-mini-TTS)", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["instructions"] != "Speak slowly and clearly" {
				t.Errorf("instructions = %q, want 'Speak slowly and clearly'", req["instructions"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("gpt-4o-mini-tts", SpeechInstructions("Speak slowly and clearly"))

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("request parameters override config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["voice"] != "request-voice" {
				t.Errorf("voice = %q, want request-voice", req["voice"])
			}
			if req["response_format"] != "opus" {
				t.Errorf("response_format = %q, want opus", req["response_format"])
			}
			if req["speed"] != float64(2.0) {
				t.Errorf("speed = %v, want 2.0", req["speed"])
			}
			if req["instructions"] != "request-instructions" {
				t.Errorf("instructions = %q, want request-instructions", req["instructions"])
			}

			w.Header().Set("Content-Type", "audio/opus")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("gpt-4o-mini-tts",
			SpeechVoice("config-voice"),
			SpeechFormat("mp3"),
			SpeechSpeed(1.0),
			SpeechInstructions("config-instructions"),
		)

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text:         "test",
			Voice:        "request-voice",
			Format:       "opus",
			Speed:        2.0,
			Instructions: "request-instructions",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("error handling - 401 authentication failed", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Incorrect API key provided",
					Type:    "invalid_request_error",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("invalid-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeAuthenticationFailed {
			t.Errorf("error code = %v, want %v", code, provider.CodeAuthenticationFailed)
		}
	})

	t.Run("error handling - 404 model not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Model not found",
					Type:    "invalid_request_error",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("invalid-model")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeModelNotFound {
			t.Errorf("error code = %v, want %v", code, provider.CodeModelNotFound)
		}
	})

	t.Run("error handling - 429 rate limited", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "45")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Rate limit exceeded",
					Type:    "rate_limit_error",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var providerErr *provider.Error
		if !provider.IsError(err, &providerErr) {
			t.Fatalf("expected provider.Error, got %T", err)
		}
		if providerErr.Code != provider.CodeRateLimited {
			t.Errorf("error code = %v, want %v", providerErr.Code, provider.CodeRateLimited)
		}
		if providerErr.RetryAfter != 45*time.Second {
			t.Errorf("RetryAfter = %v, want 45s", providerErr.RetryAfter)
		}
	})

	t.Run("error handling - 400 invalid request", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Text input is too long",
					Type:    "invalid_request_error",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeInvalidRequest {
			t.Errorf("error code = %v, want %v", code, provider.CodeInvalidRequest)
		}
	})

	t.Run("error handling - 500 server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Internal server error",
					Type:    "server_error",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeAPITimeout {
			t.Errorf("error code = %v, want %v", code, provider.CodeAPITimeout)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		p := NewProvider(APIKey("test-key"))
		model := p.Speech("tts-1")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := model.Synthesize(ctx, &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error for canceled context")
		}
	})

	t.Run("large audio response", func(t *testing.T) {
		largeAudio := make([]byte, 1024*1024)
		for i := range largeAudio {
			largeAudio[i] = byte(i % 256)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write(largeAudio)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		result, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Audio) != len(largeAudio) {
			t.Errorf("audio size = %d, want %d", len(result.Audio), len(largeAudio))
		}
	})
}

func TestFormatToMediaType(t *testing.T) {
	tests := []struct {
		format    string
		mediaType string
	}{
		{"mp3", "audio/mpeg"},
		{"opus", "audio/opus"},
		{"aac", "audio/aac"},
		{"flac", "audio/flac"},
		{"wav", "audio/wav"},
		{"pcm", "audio/pcm"},
		{"unknown", "audio/mpeg"},
	}

	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			result := formatToMediaType(tt.format)
			if result != tt.mediaType {
				t.Errorf("formatToMediaType(%q) = %q, want %q", tt.format, result, tt.mediaType)
			}
		})
	}
}

func TestSpeechModel_Options(t *testing.T) {
	t.Run("all options applied", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["voice"] != "echo" {
				t.Errorf("voice = %q, want echo", req["voice"])
			}
			if req["response_format"] != "flac" {
				t.Errorf("response_format = %q, want flac", req["response_format"])
			}
			if req["speed"] != float64(0.75) {
				t.Errorf("speed = %v, want 0.75", req["speed"])
			}
			if req["instructions"] != "Speak excitedly" {
				t.Errorf("instructions = %q, want 'Speak excitedly'", req["instructions"])
			}

			w.Header().Set("Content-Type", "audio/flac")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("gpt-4o-mini-tts",
			SpeechVoice("echo"),
			SpeechFormat("flac"),
			SpeechSpeed(0.75),
			SpeechInstructions("Speak excitedly"),
		)

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestSpeechModel_ErrorParsing(t *testing.T) {
	t.Run("parse error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "Voice 'invalid-voice' is not supported",
					Type:    "invalid_request_error",
					Code:    "invalid_voice",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var providerErr *provider.Error
		if !provider.IsError(err, &providerErr) {
			t.Fatalf("expected provider.Error, got %T", err)
		}
		if providerErr.Message != "Voice 'invalid-voice' is not supported" {
			t.Errorf("error message = %q, want 'Voice 'invalid-voice' is not supported'", providerErr.Message)
		}
	})

	t.Run("parse non-JSON error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusInternalServerError)
			io.WriteString(w, "Internal Server Error")
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		var providerErr *provider.Error
		if !provider.IsError(err, &providerErr) {
			t.Fatalf("expected provider.Error, got %T", err)
		}
		if providerErr.StatusCode != 500 {
			t.Errorf("StatusCode = %d, want 500", providerErr.StatusCode)
		}
	})
}

func TestSpeechModel_DefaultVoices(t *testing.T) {
	t.Run("default voice when not specified", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["voice"] != "alloy" {
				t.Errorf("voice = %q, want alloy (default)", req["voice"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("default format when not specified", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["response_format"] != "mp3" {
				t.Errorf("response_format = %q, want mp3 (default)", req["response_format"])
			}

			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("audio-data"))
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Speech("tts-1")

		_, err := model.Synthesize(context.Background(), &provider.SpeechRequest{
			Text: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
