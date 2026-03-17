package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

func TestImageModel_GenerateImage(t *testing.T) {
	t.Run("successful generation with base64 response", func(t *testing.T) {
		imageData := []byte("fake-image-data")
		b64Image := base64.StdEncoding.EncodeToString(imageData)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer test-key" {
				t.Error("missing Authorization header")
			}
			if r.URL.Path != "/images/generations" {
				t.Errorf("path = %q, want %q", r.URL.Path, "/images/generations")
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", r.Header.Get("Content-Type"))
			}

			var req map[string]any
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("failed to decode request: %v", err)
			}
			if req["model"] != "dall-e-3" {
				t.Errorf("model = %q, want %q", req["model"], "dall-e-3")
			}
			if req["prompt"] != "a beautiful sunset" {
				t.Errorf("prompt = %q, want %q", req["prompt"], "a beautiful sunset")
			}
			if req["response_format"] != "b64_json" {
				t.Errorf("response_format = %q, want b64_json", req["response_format"])
			}

			resp := ImageGenerationResponse{
				Created: 1234567890,
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{
					{B64JSON: b64Image, RevisedPrompt: "a stunning sunset over mountains"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-3")

		result, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "a beautiful sunset",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Images) != 1 {
			t.Fatalf("expected 1 image, got %d", len(result.Images))
		}
		if string(result.Images[0]) != string(imageData) {
			t.Errorf("image data mismatch")
		}
		if result.RevisedPrompts[0] != "a stunning sunset over mountains" {
			t.Errorf("revised prompt = %q, want %q", result.RevisedPrompts[0], "a stunning sunset over mountains")
		}
	})

	t.Run("generation with URL response format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if _, ok := req["response_format"]; ok {
				t.Errorf("response_format should not be set for URL format, got %v", req["response_format"])
			}

			resp := ImageGenerationResponse{
				Created: 1234567890,
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{
					{URL: "https://example.com/image.png"},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-2")

		result, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
			Format: "url",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Images) != 0 {
			t.Errorf("expected 0 images for URL format (URLs not decoded), got %d", len(result.Images))
		}
	})

	t.Run("generation with size and quality options", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["size"] != "1792x1024" {
				t.Errorf("size = %q, want %q", req["size"], "1792x1024")
			}
			if req["quality"] != "hd" {
				t.Errorf("quality = %q, want %q", req["quality"], "hd")
			}

			imageData := base64.StdEncoding.EncodeToString([]byte("test"))
			resp := ImageGenerationResponse{
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{{B64JSON: imageData}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-3", ImageSize("1792x1024"), ImageQuality("hd"))

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("generation with N parameter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["n"] != float64(3) {
				t.Errorf("n = %v, want 3", req["n"])
			}

			imageData := base64.StdEncoding.EncodeToString([]byte("test"))
			resp := ImageGenerationResponse{
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{
					{B64JSON: imageData},
					{B64JSON: imageData},
					{B64JSON: imageData},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-2")

		result, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
			N:      3,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(result.Images) != 3 {
			t.Errorf("expected 3 images, got %d", len(result.Images))
		}
	})

	t.Run("generation with usage (GPT-Image-1)", func(t *testing.T) {
		imageData := base64.StdEncoding.EncodeToString([]byte("test"))
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ImageGenerationResponse{
				Created: 1234567890,
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{{B64JSON: imageData}},
				Usage: &struct {
					InputTokens        int `json:"input_tokens,omitempty"`
					OutputTokens       int `json:"output_tokens,omitempty"`
					TotalTokens        int `json:"total_tokens,omitempty"`
					InputTokensDetails *struct {
						ImageTokens int `json:"image_tokens,omitempty"`
						TextTokens  int `json:"text_tokens,omitempty"`
					} `json:"input_tokens_details,omitempty"`
				}{
					InputTokens:  150,
					OutputTokens: 50,
					TotalTokens:  200,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("gpt-image-1")

		result, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Usage.TotalTokens != 200 {
			t.Errorf("TotalTokens = %d, want 200", result.Usage.TotalTokens)
		}
		if result.Usage.InputTokens != 150 {
			t.Errorf("InputTokens = %d, want 150", result.Usage.InputTokens)
		}
		if result.Usage.OutputTokens != 50 {
			t.Errorf("OutputTokens = %d, want 50", result.Usage.OutputTokens)
		}
	})

	t.Run("image editing not supported", func(t *testing.T) {
		p := NewProvider(APIKey("test-key"))
		model := p.Image("dall-e-3")

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
			Files:  []provider.ImageFile{{Data: []byte("image-data")}},
		})

		if err == nil {
			t.Fatal("expected error for files parameter, got nil")
		}

		var providerErr *provider.Error
		if !provider.IsError(err, &providerErr) {
			t.Fatalf("expected provider.Error, got %T", err)
		}
		if providerErr.Code != provider.CodeInvalidRequest {
			t.Errorf("error code = %v, want %v", providerErr.Code, provider.CodeInvalidRequest)
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
					Code:    "invalid_api_key",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("invalid-key"), BaseURL(server.URL))
		model := p.Image("dall-e-3")

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeAuthenticationFailed {
			t.Errorf("error code = %v, want %v", code, provider.CodeAuthenticationFailed)
		}
	})

	t.Run("error handling - 429 rate limited", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "30")
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
		model := p.Image("dall-e-3")

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
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
		if providerErr.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", providerErr.RetryAfter)
		}
	})

	t.Run("error handling - 404 model not found", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(ErrorResponse{
				Error: ErrorDetail{
					Message: "The model 'invalid-model' does not exist",
					Type:    "invalid_request_error",
				},
			})
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("invalid-model")

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
		})
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeModelNotFound {
			t.Errorf("error code = %v, want %v", code, provider.CodeModelNotFound)
		}
	})

	t.Run("error handling - invalid base64 response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := ImageGenerationResponse{
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{{B64JSON: "not-valid-base64!!!"}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-3")

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
		})
		if err == nil {
			t.Fatal("expected error for invalid base64, got nil")
		}

		code := provider.ErrorCodeOf(err)
		if code != provider.CodeParseError {
			t.Errorf("error code = %v, want %v", code, provider.CodeParseError)
		}
	})

	t.Run("config options applied", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["quality"] != "hd" {
				t.Errorf("quality = %q, want hd", req["quality"])
			}
			if req["size"] != "1024x1024" {
				t.Errorf("size = %q, want 1024x1024", req["size"])
			}
			if req["user"] != "test-user" {
				t.Errorf("user = %q, want test-user", req["user"])
			}

			imageData := base64.StdEncoding.EncodeToString([]byte("test"))
			resp := ImageGenerationResponse{
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{{B64JSON: imageData}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-3",
			ImageQuality("hd"),
			ImageSize("1024x1024"),
			ImageUser("test-user"),
		)

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt: "test",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("request overrides config", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req map[string]any
			json.NewDecoder(r.Body).Decode(&req)

			if req["quality"] != "request-quality" {
				t.Errorf("quality = %q, want request-quality", req["quality"])
			}
			if req["size"] != "request-size" {
				t.Errorf("size = %q, want request-size", req["size"])
			}

			imageData := base64.StdEncoding.EncodeToString([]byte("test"))
			resp := ImageGenerationResponse{
				Data: []struct {
					B64JSON       string `json:"b64_json,omitempty"`
					URL           string `json:"url,omitempty"`
					RevisedPrompt string `json:"revised_prompt,omitempty"`
				}{{B64JSON: imageData}},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		p := NewProvider(APIKey("test-key"), BaseURL(server.URL))
		model := p.Image("dall-e-3",
			ImageQuality("config-quality"),
			ImageSize("config-size"),
		)

		_, err := model.GenerateImage(context.Background(), &provider.ImageRequest{
			Prompt:  "test",
			Quality: "request-quality",
			Size:    "request-size",
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}
