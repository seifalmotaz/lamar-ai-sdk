package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/seifalmotaz/lamar-sdk/provider"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	Headers    map[string]string
}

func NewClient(baseURL string, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{
		BaseURL:    strings.TrimSuffix(baseURL, "/"),
		HTTPClient: httpClient,
		Headers:    make(map[string]string),
	}
}

func (c *Client) SetHeader(key, value string) {
	c.Headers[key] = value
}

func (c *Client) Do(ctx context.Context, method, path string, body, result any) error {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return &provider.Error{
				Code:    provider.CodeParseError,
				Message: "failed to marshal request body",
				Cause:   err,
			}
		}
		reqBody = strings.NewReader(string(data))
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return &provider.Error{
			Code:    provider.CodeInvalidRequest,
			Message: "failed to create request",
			Cause:   err,
		}
	}

	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return provider.ErrContextCanceled
		}
		if ctx.Err() == context.DeadlineExceeded {
			return provider.ErrAPITimeout
		}
		return &provider.Error{
			Code:    provider.CodeAPITimeout,
			Message: "request failed",
			Cause:   err,
		}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &provider.Error{
			Code:    provider.CodeParseError,
			Message: "failed to read response body",
			Cause:   err,
		}
	}

	if resp.StatusCode >= 400 {
		return mapError(resp, respBody)
	}

	if result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, result); err != nil {
			return &provider.Error{
				Code:    provider.CodeParseError,
				Message: "failed to unmarshal response",
				Cause:   err,
			}
		}
	}

	return nil
}

func (c *Client) Get(ctx context.Context, path string, result any) error {
	return c.Do(ctx, http.MethodGet, path, nil, result)
}

func (c *Client) Post(ctx context.Context, path string, body, result any) error {
	return c.Do(ctx, http.MethodPost, path, body, result)
}

func (c *Client) Delete(ctx context.Context, path string, result any) error {
	return c.Do(ctx, http.MethodDelete, path, nil, result)
}

// DoStream performs a streaming HTTP request and returns the response body
// for reading Server-Sent Events. The caller is responsible for closing
// the returned ReadCloser.
func (c *Client) DoStream(ctx context.Context, method, path string, body any) (io.ReadCloser, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, &provider.Error{
				Code:    provider.CodeParseError,
				Message: "failed to marshal request body",
				Cause:   err,
			}
		}
		reqBody = strings.NewReader(string(data))
	}

	url := c.BaseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, &provider.Error{
			Code:    provider.CodeInvalidRequest,
			Message: "failed to create request",
			Cause:   err,
		}
	}

	for key, value := range c.Headers {
		req.Header.Set(key, value)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		if ctx.Err() == context.Canceled {
			return nil, provider.ErrContextCanceled
		}
		if ctx.Err() == context.DeadlineExceeded {
			return nil, provider.ErrAPITimeout
		}
		return nil, &provider.Error{
			Code:    provider.CodeAPITimeout,
			Message: "request failed",
			Cause:   err,
		}
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, &provider.Error{
				Code:    provider.CodeParseError,
				Message: "failed to read error response body",
				Cause:   err,
			}
		}
		return nil, mapError(resp, respBody)
	}

	return resp.Body, nil
}

type ErrorBody struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code,omitempty"`
}

func mapError(resp *http.Response, body []byte) *provider.Error {
	var errBody ErrorBody
	code := mapStatusCodeToCode(resp.StatusCode)

	if err := json.Unmarshal(body, &errBody); err == nil && errBody.Error.Message != "" {
		return &provider.Error{
			Code:       code,
			Message:    errBody.Error.Message,
			Provider:   "openai",
			StatusCode: resp.StatusCode,
			RetryAfter: parseRetryAfter(resp.Header),
		}
	}

	return &provider.Error{
		Code:       code,
		Message:    fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(body)),
		Provider:   "openai",
		StatusCode: resp.StatusCode,
		RetryAfter: parseRetryAfter(resp.Header),
	}
}

func mapStatusCodeToCode(statusCode int) provider.ErrorCode {
	switch statusCode {
	case 400:
		return provider.CodeInvalidRequest
	case 401:
		return provider.CodeAuthenticationFailed
	case 403:
		return provider.CodeAuthenticationFailed
	case 404:
		return provider.CodeModelNotFound
	case 429:
		return provider.CodeRateLimited
	case 500, 502, 503, 504:
		return provider.CodeAPITimeout
	default:
		return provider.CodeUnknown
	}
}

func parseRetryAfter(headers http.Header) time.Duration {
	retryAfter := headers.Get("Retry-After")
	if retryAfter == "" {
		return 0
	}

	if seconds, err := time.ParseDuration(retryAfter + "s"); err == nil {
		return seconds
	}

	return 0
}
