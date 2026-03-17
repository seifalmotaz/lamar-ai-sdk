package testutil_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/seifalmotaz/lamar-ai-sdk/internal/testutil"
)

func TestNewMockServer_JSON(t *testing.T) {
	type response struct {
		Message string `json:"message"`
	}

	server := testutil.NewMockServer(t, response{Message: "hello"})
	defer server.Close()

	resp, err := http.Get(server.URL())
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got response
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got.Message != "hello" {
		t.Errorf("message = %q, want %q", got.Message, "hello")
	}
}

func TestNewMockServer_String(t *testing.T) {
	server := testutil.NewMockServer(t, `{"status":"ok"}`)
	defer server.Close()

	resp, err := http.Get(server.URL())
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", resp.Header.Get("Content-Type"))
	}
}

func TestMockServer_SetResponse(t *testing.T) {
	server := testutil.NewMockServer(t, map[string]string{"status": "initial"})
	defer server.Close()

	server.SetResponse(map[string]string{"status": "updated"})

	resp, err := http.Get(server.URL())
	if err != nil {
		t.Fatalf("GET failed: %v", err)
	}
	defer resp.Body.Close()

	var got map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	if got["status"] != "updated" {
		t.Errorf("status = %q, want %q", got["status"], "updated")
	}
}

func TestMockServer_Calls(t *testing.T) {
	server := testutil.NewMockServer(t, map[string]string{"status": "ok"})
	defer server.Close()

	http.Get(server.URL() + "/path1")
	http.Post(server.URL()+"/path2", "application/json", nil)

	calls := server.Calls()
	if len(calls) != 2 {
		t.Fatalf("call count = %d, want 2", len(calls))
	}

	if calls[0].Method != "GET" {
		t.Errorf("calls[0].Method = %q, want GET", calls[0].Method)
	}
	if calls[0].Path != "/path1" {
		t.Errorf("calls[0].Path = %q, want /path1", calls[0].Path)
	}

	if calls[1].Method != "POST" {
		t.Errorf("calls[1].Method = %q, want POST", calls[1].Method)
	}
	if calls[1].Path != "/path2" {
		t.Errorf("calls[1].Path = %q, want /path2", calls[1].Path)
	}
}

func TestMockServer_Reset(t *testing.T) {
	server := testutil.NewMockServer(t, map[string]string{"status": "ok"})
	defer server.Close()

	http.Get(server.URL())

	if len(server.Calls()) != 1 {
		t.Fatalf("call count before reset = %d, want 1", len(server.Calls()))
	}

	server.Reset()

	if len(server.Calls()) != 0 {
		t.Errorf("call count after reset = %d, want 0", len(server.Calls()))
	}
}

func TestMockServer_LastCall(t *testing.T) {
	server := testutil.NewMockServer(t, map[string]string{"status": "ok"})
	defer server.Close()

	http.Get(server.URL() + "/first")
	http.Get(server.URL() + "/last")

	last := server.LastCall()
	if last == nil {
		t.Fatal("LastCall() returned nil")
	}

	if last.Path != "/last" {
		t.Errorf("last.Path = %q, want /last", last.Path)
	}

	server.Reset()

	if server.LastCall() != nil {
		t.Error("LastCall() after reset should return nil")
	}
}

func TestMockServer_Assertions(t *testing.T) {
	server := testutil.NewMockServer(t, map[string]string{"status": "ok"})
	defer server.Close()

	http.Get(server.URL() + "/test")

	t.Run("AssertCallCount", func(t *testing.T) {
		server.AssertCallCount(t, 1)
	})

	t.Run("AssertMethod", func(t *testing.T) {
		server.AssertMethod(t, "GET")
	})

	t.Run("AssertPath", func(t *testing.T) {
		server.AssertPath(t, "/test")
	})

	t.Run("AssertHeader", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL(), nil)
		req.Header.Set("X-Custom", "value")
		http.DefaultClient.Do(req)

		server.AssertHeader(t, "X-Custom", "value")
	})
}

func TestNewJSONResponse(t *testing.T) {
	resp := testutil.NewJSONResponse(http.StatusCreated, map[string]string{"id": "123"})

	if resp.StatusCode != http.StatusCreated {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}
	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", resp.Headers["Content-Type"])
	}
}

func TestNewErrorResponse(t *testing.T) {
	resp := testutil.NewErrorResponse(http.StatusBadRequest, "invalid_request", "The request was invalid")

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("StatusCode = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	if resp.Headers["Content-Type"] != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", resp.Headers["Content-Type"])
	}

	body, ok := resp.Body.(map[string]interface{})
	if !ok {
		t.Fatal("body is not a map")
	}

	errObj, ok := body["error"].(map[string]interface{})
	if !ok {
		t.Fatal("error field is not a map")
	}

	if errObj["code"] != "invalid_request" {
		t.Errorf("error.code = %v, want invalid_request", errObj["code"])
	}
}
