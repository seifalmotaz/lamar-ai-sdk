package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

func TestTranscriptionModel_Transcribe(t *testing.T) {
	t.Run("successful transcription", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("missing Authorization header")
			}
			if r.URL.Path != "/audio/transcriptions" {
				t.Errorf("path = %q, want %q", r.URL.Path, "/audio/transcriptions")
			}

			contentType := r.Header.Get("Content-Type")
			if !strings.HasPrefix(contentType, "multipart/form-data") {
				t.Errorf("Content-Type = %q, want multipart/form-data", contentType)
			}

			reader, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("failed to create multipart reader: %v", err)
			}

			var foundModel, foundFile bool
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("error reading part: %v", err)
				}

				switch part.FormName() {
				case "model":
					data, _ := io.ReadAll(part)
					if string(data) != "whisper-1" {
						t.Errorf("model = %q, want whisper-1", string(data))
					}
					foundModel = true
				case "file":
					data, _ := io.ReadAll(part)
					if string(data) != "test-audio-data" {
						t.Errorf("file content = %q, want test-audio-data", string(data))
					}
					foundFile = true
				}
			}

			if !foundModel {
				t.Error("model field not found in multipart form")
			}
			if !foundFile {
				t.Error("file field not found in multipart form")
			}

			resp := TranscriptionResponse{
				Text:     "Hello, world!",
				Language: "en",
				Duration: 5.5,
				Segments: []struct {
					Text  string  `json:"text"`
					Start float64 `json:"start"`
					End   float64 `json:"end"`
				}{
					{Text: "Hello,", Start: 0.0, End: 0.5},
					{Text: " world!", Start: 0.5, End: 1.0},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Transcription("whisper-1")

		result, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Text != "Hello, world!" {
			t.Errorf("Text = %q, want %q", result.Text, "Hello, world!")
		}
		if result.Language != "en" {
			t.Errorf("Language = %q, want %q", result.Language, "en")
		}
		if result.Duration != 5.5 {
			t.Errorf("Duration = %v, want 5.5", result.Duration)
		}
		if len(result.Segments) != 2 {
			t.Errorf("Segments count = %d, want 2", len(result.Segments))
		}
	})

	t.Run("transcription with word-level timestamps", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := TranscriptionResponse{
				Text:     "Hello world",
				Language: "en",
				Duration: 2.0,
				Words: []struct {
					Word  string  `json:"word"`
					Start float64 `json:"start"`
					End   float64 `json:"end"`
				}{
					{Word: "Hello", Start: 0.0, End: 0.5},
					{Word: "world", Start: 0.6, End: 1.0},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Transcription("whisper-1")

		result, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Segments) != 2 {
			t.Errorf("Segments count = %d, want 2", len(result.Segments))
		}
		if result.Segments[0].Text != "Hello" {
			t.Errorf("First word = %q, want Hello", result.Segments[0].Text)
		}
		if result.Segments[1].Text != "world" {
			t.Errorf("Second word = %q, want world", result.Segments[1].Text)
		}
	})

	t.Run("transcription with language option", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reader, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("failed to create multipart reader: %v", err)
			}

			var foundLanguage bool
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("error reading part: %v", err)
				}

				if part.FormName() == "language" {
					data, _ := io.ReadAll(part)
					if string(data) != "es" {
						t.Errorf("language = %q, want es", string(data))
					}
					foundLanguage = true
				}
			}

			if !foundLanguage {
				t.Error("language field not found")
			}

			resp := TranscriptionResponse{Text: "Hola mundo", Language: "es"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Transcription("whisper-1", TranscriptionLanguage("es"))

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("transcription with prompt option", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reader, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("failed to create multipart reader: %v", err)
			}

			var foundPrompt bool
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("error reading part: %v", err)
				}

				if part.FormName() == "prompt" {
					data, _ := io.ReadAll(part)
					if string(data) != "A conversation about weather" {
						t.Errorf("prompt = %q, want 'A conversation about weather'", string(data))
					}
					foundPrompt = true
				}
			}

			if !foundPrompt {
				t.Error("prompt field not found")
			}

			resp := TranscriptionResponse{Text: "The weather is nice today"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Transcription("whisper-1", TranscriptionPrompt("A conversation about weather"))

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("request options override config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reader, err := r.MultipartReader()
			if err != nil {
				t.Fatalf("failed to create multipart reader: %v", err)
			}

			var foundLanguage, foundPrompt bool
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("error reading part: %v", err)
				}

				if part.FormName() == "language" {
					data, _ := io.ReadAll(part)
					if string(data) != "request-lang" {
						t.Errorf("language = %q, want request-lang", string(data))
					}
					foundLanguage = true
				}
				if part.FormName() == "prompt" {
					data, _ := io.ReadAll(part)
					if string(data) != "request-prompt" {
						t.Errorf("prompt = %q, want request-prompt", string(data))
					}
					foundPrompt = true
				}
			}

			if !foundLanguage {
				t.Error("language field not found")
			}
			if !foundPrompt {
				t.Error("prompt field not found")
			}

			resp := TranscriptionResponse{Text: "test"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Transcription("whisper-1",
			TranscriptionLanguage("config-lang"),
			TranscriptionPrompt("config-prompt"),
		)

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
			Language:  "request-lang",
			Prompt:    "request-prompt",
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
		model := p.Transcription("whisper-1")

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
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
		model := p.Transcription("invalid-model")

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
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
			w.Header().Set("Retry-After", "60")
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
		model := p.Transcription("whisper-1")

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
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
		if providerErr.RetryAfter != 60*time.Second {
			t.Errorf("RetryAfter = %v, want 60s", providerErr.RetryAfter)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		p := NewProvider(APIKey("test-key"))
		model := p.Transcription("whisper-1")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := model.Transcribe(ctx, &provider.TranscriptionRequest{
			Audio:     []byte("test-audio-data"),
			MediaType: "audio/mp3",
		})
		if err == nil {
			t.Fatal("expected error for canceled context")
		}
	})

	t.Run("different media types", func(t *testing.T) {
		tests := []struct {
			mediaType string
			expectExt string
		}{
			{"audio/mp3", "mp3"},
			{"audio/mpeg", "mp3"},
			{"audio/mp4", "mp4"},
			{"audio/m4a", "mp4"},
			{"audio/wav", "wav"},
			{"audio/webm", "webm"},
			{"audio/ogg", "oga"},
			{"audio/unknown", "mp3"},
		}

		for _, tt := range tests {
			t.Run(tt.mediaType, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					reader, err := r.MultipartReader()
					if err != nil {
						t.Fatalf("failed to create multipart reader: %v", err)
					}

					for {
						part, err := reader.NextPart()
						if err == io.EOF {
							break
						}
						if err != nil {
							t.Fatalf("error reading part: %v", err)
						}

						if part.FormName() == "file" {
							filename := part.FileName()
							if !strings.HasSuffix(filename, "."+tt.expectExt) {
								t.Errorf("filename = %q, expected extension .%s", filename, tt.expectExt)
							}
						}
					}

					resp := TranscriptionResponse{Text: "test"}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(resp)
				}))
				defer server.Close()

				p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
				model := p.Transcription("whisper-1")

				_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
					Audio:     []byte("test-audio-data"),
					MediaType: tt.mediaType,
				})
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			})
		}
	})
}

func TestMediaTypeToExtension(t *testing.T) {
	tests := []struct {
		mediaType string
		expected  string
	}{
		{"audio/mp3", "mp3"},
		{"audio/mpeg", "mp3"},
		{"audio/mp4", "mp4"},
		{"audio/m4a", "mp4"},
		{"audio/wav", "wav"},
		{"audio/webm", "webm"},
		{"audio/ogg", "oga"},
		{"unknown/type", "mp3"},
	}

	for _, tt := range tests {
		t.Run(tt.mediaType, func(t *testing.T) {
			result := mediaTypeToExtension(tt.mediaType)
			if result != tt.expected {
				t.Errorf("mediaTypeToExtension(%q) = %q, want %q", tt.mediaType, result, tt.expected)
			}
		})
	}
}

func TestTranscription_MultipartFormValidation(t *testing.T) {
	t.Run("verifies all form fields", func(t *testing.T) {
		var receivedFields map[string]string
		var receivedFile bool

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			contentType := r.Header.Get("Content-Type")
			_, params, err := parseMediaType(contentType)
			if err != nil {
				t.Fatalf("failed to parse content type: %v", err)
			}

			boundary := params["boundary"]
			reader := multipart.NewReader(r.Body, boundary)

			receivedFields = make(map[string]string)
			for {
				part, err := reader.NextPart()
				if err == io.EOF {
					break
				}
				if err != nil {
					t.Fatalf("error reading part: %v", err)
				}

				data, _ := io.ReadAll(part)
				if part.FormName() == "file" {
					receivedFile = true
				} else {
					receivedFields[part.FormName()] = string(data)
				}
			}

			resp := TranscriptionResponse{Text: "test"}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Transcription("whisper-1",
			TranscriptionLanguage("en"),
			TranscriptionPrompt("test prompt"),
			TranscriptionTemperature(0.5),
			TranscriptionTimestampGranularity("word"),
		)

		_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
			Audio:     []byte("audio-data"),
			MediaType: "audio/mp3",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !receivedFile {
			t.Error("file field not found")
		}
		if receivedFields["model"] != "whisper-1" {
			t.Errorf("model = %q, want whisper-1", receivedFields["model"])
		}
		if receivedFields["language"] != "en" {
			t.Errorf("language = %q, want en", receivedFields["language"])
		}
		if receivedFields["prompt"] != "test prompt" {
			t.Errorf("prompt = %q, want 'test prompt'", receivedFields["prompt"])
		}
		if receivedFields["response_format"] != "verbose_json" {
			t.Errorf("response_format = %q, want verbose_json", receivedFields["response_format"])
		}
		if receivedFields["temperature"] != "0.5" {
			t.Errorf("temperature = %q, want 0.5", receivedFields["temperature"])
		}
	})
}

func parseMediaType(contentType string) (string, map[string]string, error) {
	parts := strings.SplitN(contentType, ";", 2)
	if len(parts) == 1 {
		return parts[0], nil, nil
	}
	mediaType := strings.TrimSpace(parts[0])
	params := make(map[string]string)
	for _, part := range strings.Split(parts[1], ";") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) == 2 {
			params[kv[0]] = strings.Trim(kv[1], `"`)
		}
	}
	return mediaType, params, nil
}

func TestTranscription_TemperatureFormatting(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("failed to create multipart reader: %v", err)
		}

		var foundTemp bool
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading part: %v", err)
			}

			if part.FormName() == "temperature" {
				data, _ := io.ReadAll(part)
				tempStr := string(data)
				if !strings.Contains(tempStr, ".") {
					t.Errorf("temperature = %q, expected decimal format", tempStr)
				}
				if tempStr != "0.75" {
					t.Errorf("temperature = %q, want 0.75", tempStr)
				}
				foundTemp = true
			}
		}

		if !foundTemp {
			t.Error("temperature field not found")
		}

		resp := TranscriptionResponse{Text: "test"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Transcription("whisper-1", TranscriptionTemperature(0.75))

	_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
		Audio:     []byte("test"),
		MediaType: "audio/mp3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranscription_EmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := TranscriptionResponse{
			Text: "",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Transcription("whisper-1")

	result, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
		Audio:     []byte("test"),
		MediaType: "audio/mp3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Text != "" {
		t.Errorf("Text = %q, want empty", result.Text)
	}
}

func TestTranscription_LargeAudioData(t *testing.T) {
	largeAudio := bytes.Repeat([]byte("x"), 1024*1024)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reader, err := r.MultipartReader()
		if err != nil {
			t.Fatalf("failed to create multipart reader: %v", err)
		}

		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatalf("error reading part: %v", err)
			}

			if part.FormName() == "file" {
				data, err := io.ReadAll(part)
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				if len(data) != len(largeAudio) {
					t.Errorf("file size = %d, want %d", len(data), len(largeAudio))
				}
			}
		}

		resp := TranscriptionResponse{Text: "transcribed"}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
	model := p.Transcription("whisper-1")

	_, err := model.Transcribe(context.Background(), &provider.TranscriptionRequest{
		Audio:     largeAudio,
		MediaType: "audio/mp3",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
