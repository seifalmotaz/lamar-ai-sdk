package openai

import (
	"context"
	"encoding/base64"

	"github.com/seifalmotaz/lamar-ai-sdk/provider"
)

type ImageModel struct {
	id       string
	provider *Provider
	config   ImageConfig
}

type ImageConfig struct {
	Quality     string
	Size        string
	Background  string
	Format      string
	Compression int
	User        string
}

type ImageOption func(*ImageConfig)

func ImageQuality(quality string) ImageOption {
	return func(c *ImageConfig) { c.Quality = quality }
}

func ImageSize(size string) ImageOption {
	return func(c *ImageConfig) { c.Size = size }
}

func ImageBackground(background string) ImageOption {
	return func(c *ImageConfig) { c.Background = background }
}

func ImageFormat(format string) ImageOption {
	return func(c *ImageConfig) { c.Format = format }
}

func ImageCompression(compression int) ImageOption {
	return func(c *ImageConfig) { c.Compression = compression }
}

func ImageUser(user string) ImageOption {
	return func(c *ImageConfig) { c.User = user }
}

func NewImageModel(id string, p *Provider, opts ...ImageOption) *ImageModel {
	cfg := &ImageConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ImageModel{
		id:       id,
		provider: p,
		config:   *cfg,
	}
}

func (m *ImageModel) Provider() string { return "openai" }
func (m *ImageModel) ModelID() string  { return m.id }

func (m *ImageModel) MaxImagesPerCall() int {
	switch m.id {
	case "dall-e-2":
		return 10
	case "dall-e-3":
		return 1
	case "gpt-image-1", "gpt-image-1-mini":
		return 10
	default:
		return 1
	}
}

func (m *ImageModel) GenerateImage(ctx context.Context, req *provider.ImageRequest) (*provider.ImageResult, error) {
	if len(req.Files) > 0 {
		return nil, &provider.Error{
			Code:    provider.CodeInvalidRequest,
			Message: "image editing (files parameter) is not yet supported; use GenerateImage with prompt only",
		}
	}

	body := map[string]any{
		"model":  m.id,
		"prompt": req.Prompt,
	}

	if req.N > 0 {
		body["n"] = req.N
	}
	if req.Size != "" {
		body["size"] = req.Size
	} else if m.config.Size != "" {
		body["size"] = m.config.Size
	}
	if req.Quality != "" {
		body["quality"] = req.Quality
	} else if m.config.Quality != "" {
		body["quality"] = m.config.Quality
	}
	if req.Format != "" {
		if req.Format != "url" {
			body["response_format"] = "b64_json"
		}
	} else {
		body["response_format"] = "b64_json"
	}
	if m.config.User != "" {
		body["user"] = m.config.User
	}

	var resp ImageGenerationResponse
	if err := m.provider.client.Post(ctx, "/images/generations", body, &resp); err != nil {
		return nil, err
	}

	var images [][]byte
	revisedPrompts := make([]string, len(resp.Data))
	for i, img := range resp.Data {
		if img.B64JSON != "" {
			data, err := base64.StdEncoding.DecodeString(img.B64JSON)
			if err != nil {
				return nil, &provider.Error{
					Code:    provider.CodeParseError,
					Message: "failed to decode base64 image",
					Cause:   err,
				}
			}
			images = append(images, data)
		}
		if img.RevisedPrompt != "" {
			revisedPrompts[i] = img.RevisedPrompt
		}
	}

	result := &provider.ImageResult{
		Images:         images,
		RevisedPrompts: revisedPrompts,
	}

	if resp.Usage != nil {
		result.Usage = provider.ImageUsage{
			InputTokens:  resp.Usage.InputTokens,
			OutputTokens: resp.Usage.OutputTokens,
			TotalTokens:  resp.Usage.TotalTokens,
		}
	}

	return result, nil
}

type ImageGenerationResponse struct {
	Created int64 `json:"created"`
	Data    []struct {
		B64JSON       string `json:"b64_json,omitempty"`
		URL           string `json:"url,omitempty"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
	Usage *struct {
		InputTokens        int `json:"input_tokens,omitempty"`
		OutputTokens       int `json:"output_tokens,omitempty"`
		TotalTokens        int `json:"total_tokens,omitempty"`
		InputTokensDetails *struct {
			ImageTokens int `json:"image_tokens,omitempty"`
			TextTokens  int `json:"text_tokens,omitempty"`
		} `json:"input_tokens_details,omitempty"`
	} `json:"usage,omitempty"`
}
