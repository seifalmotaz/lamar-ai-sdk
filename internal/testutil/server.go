package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

type CallRecord struct {
	Method      string
	Path        string
	Headers     http.Header
	Body        []byte
	QueryParams map[string][]string
}

type MockServer struct {
	Server   *httptest.Server
	calls    []CallRecord
	response interface{}
	mu       sync.RWMutex
	t        *testing.T
}

type ResponseConfig struct {
	StatusCode int
	Headers    map[string]string
	Body       interface{}
}

func NewMockServer(t *testing.T, initialResponse interface{}) *MockServer {
	ms := &MockServer{
		t:        t,
		response: initialResponse,
		calls:    make([]CallRecord, 0),
	}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ms.mu.Lock()
		defer ms.mu.Unlock()

		body := make([]byte, 0)
		if r.Body != nil {
			buf := make([]byte, 1024)
			for {
				n, err := r.Body.Read(buf)
				if err != nil {
					break
				}
				body = append(body, buf[:n]...)
			}
		}

		call := CallRecord{
			Method:      r.Method,
			Path:        r.URL.Path,
			Headers:     r.Header.Clone(),
			Body:        body,
			QueryParams: r.URL.Query(),
		}
		ms.calls = append(ms.calls, call)

		if resp, ok := ms.response.(*ResponseConfig); ok {
			for k, v := range resp.Headers {
				w.Header().Set(k, v)
			}
			if resp.StatusCode > 0 {
				w.WriteHeader(resp.StatusCode)
			} else {
				w.WriteHeader(http.StatusOK)
			}
			if resp.Body != nil {
				switch v := resp.Body.(type) {
				case string:
					w.Write([]byte(v))
				case []byte:
					w.Write(v)
				default:
					json.NewEncoder(w).Encode(v)
				}
			}
			return
		}

		switch v := ms.response.(type) {
		case string:
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(v))
		case []byte:
			w.Write(v)
		default:
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(v)
		}
	}))

	return ms
}

func (s *MockServer) URL() string {
	return s.Server.URL
}

func (s *MockServer) SetResponse(response interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.response = response
}

func (s *MockServer) Calls() []CallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	calls := make([]CallRecord, len(s.calls))
	copy(calls, s.calls)
	return calls
}

func (s *MockServer) LastCall() *CallRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) == 0 {
		return nil
	}
	call := s.calls[len(s.calls)-1]
	return &call
}

func (s *MockServer) RequestBodyJSON(v interface{}) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) == 0 {
		return nil
	}
	return json.Unmarshal(s.calls[len(s.calls)-1].Body, v)
}

func (s *MockServer) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = make([]CallRecord, 0)
}

func (s *MockServer) Close() {
	s.Server.Close()
}

func (s *MockServer) AssertCallCount(t *testing.T, expected int) {
	t.Helper()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) != expected {
		t.Errorf("call count = %d, want %d", len(s.calls), expected)
	}
}

func (s *MockServer) AssertMethod(t *testing.T, expected string) {
	t.Helper()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) == 0 {
		t.Fatal("no calls made")
	}
	if s.calls[len(s.calls)-1].Method != expected {
		t.Errorf("method = %s, want %s", s.calls[len(s.calls)-1].Method, expected)
	}
}

func (s *MockServer) AssertPath(t *testing.T, expected string) {
	t.Helper()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) == 0 {
		t.Fatal("no calls made")
	}
	if s.calls[len(s.calls)-1].Path != expected {
		t.Errorf("path = %s, want %s", s.calls[len(s.calls)-1].Path, expected)
	}
}

func (s *MockServer) AssertHeader(t *testing.T, key, expected string) {
	t.Helper()
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.calls) == 0 {
		t.Fatal("no calls made")
	}
	actual := s.calls[len(s.calls)-1].Headers.Get(key)
	if actual != expected {
		t.Errorf("header %s = %s, want %s", key, actual, expected)
	}
}

func NewJSONResponse(statusCode int, body interface{}) *ResponseConfig {
	return &ResponseConfig{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "application/json"},
		Body:       body,
	}
}

func NewStreamResponse(statusCode int, chunks []string) *ResponseConfig {
	return &ResponseConfig{
		StatusCode: statusCode,
		Headers: map[string]string{
			"Content-Type":  "text/event-stream",
			"Cache-Control": "no-cache",
		},
		Body: chunks,
	}
}

func NewErrorResponse(statusCode int, errorCode, errorMsg string) *ResponseConfig {
	return NewJSONResponse(statusCode, map[string]interface{}{
		"error": map[string]interface{}{
			"code":    errorCode,
			"message": errorMsg,
		},
	})
}
