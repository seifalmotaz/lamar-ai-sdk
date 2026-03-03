package sse

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestReader_SingleEvent(t *testing.T) {
	input := "data: {\"message\":\"hello\"}\n\n"
	reader := NewReader(strings.NewReader(input))

	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Data != `{"message":"hello"}` {
		t.Errorf("expected data %q, got %q", `{"message":"hello"}`, event.Data)
	}
	if event.Event != "message" {
		t.Errorf("expected event %q, got %q", "message", event.Event)
	}
}

func TestReader_MultipleEvents(t *testing.T) {
	input := "data: first\n\ndata: second\n\ndata: third\n\n"
	reader := NewReader(strings.NewReader(input))

	events, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	expectedData := []string{"first", "second", "third"}
	for i, event := range events {
		if event.Data != expectedData[i] {
			t.Errorf("event %d: expected data %q, got %q", i, expectedData[i], event.Data)
		}
	}
}

func TestReader_Done(t *testing.T) {
	input := "data: {\"content\":\"hello\"}\n\ndata: [DONE]\n\n"
	reader := NewReader(strings.NewReader(input))

	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error on first event: %v", err)
	}
	if event.Data != `{"content":"hello"}` {
		t.Errorf("expected data %q, got %q", `{"content":"hello"}`, event.Data)
	}

	_, err = reader.ReadEvent()
	if !errors.Is(err, Done) {
		t.Errorf("expected Done error, got %v", err)
	}
}

func TestReader_EventType(t *testing.T) {
	input := "event: custom\ndata: payload\n\n"
	reader := NewReader(strings.NewReader(input))

	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Event != "custom" {
		t.Errorf("expected event type %q, got %q", "custom", event.Event)
	}
	if event.Data != "payload" {
		t.Errorf("expected data %q, got %q", "payload", event.Data)
	}
}

func TestReader_EventID(t *testing.T) {
	input := "id: 123\ndata: payload\n\n"
	reader := NewReader(strings.NewReader(input))

	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.ID != "123" {
		t.Errorf("expected id %q, got %q", "123", event.ID)
	}
}

func TestReader_MultilineData(t *testing.T) {
	input := "data: line1\ndata: line2\n\ndata: single\n\n"
	reader := NewReader(strings.NewReader(input))

	events, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	if events[0].Data != "line1\nline2" {
		t.Errorf("expected multiline data %q, got %q", "line1\nline2", events[0].Data)
	}
	if events[1].Data != "single" {
		t.Errorf("expected data %q, got %q", "single", events[1].Data)
	}
}

func TestReader_EOFOnEmptyInput(t *testing.T) {
	reader := NewReader(strings.NewReader(""))

	_, err := reader.ReadEvent()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected EOF error, got %v", err)
	}
}

func TestReader_AfterDone(t *testing.T) {
	input := "data: [DONE]\n\n"
	reader := NewReader(strings.NewReader(input))

	_, err := reader.ReadEvent()
	if !errors.Is(err, Done) {
		t.Errorf("expected Done error, got %v", err)
	}

	// Second call should return EOF
	_, err = reader.ReadEvent()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected EOF error after Done, got %v", err)
	}
}

func TestReader_SpaceAfterColon(t *testing.T) {
	input := "data: value with spaces\n\n"
	reader := NewReader(strings.NewReader(input))

	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// SSE spec: one leading space is trimmed from the value
	if event.Data != "value with spaces" {
		t.Errorf("expected data %q, got %q", "value with spaces", event.Data)
	}
}

func TestReader_IgnoresUnknownFields(t *testing.T) {
	input := "retry: 5000\ndata: payload\n\n"
	reader := NewReader(strings.NewReader(input))

	event, err := reader.ReadEvent()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if event.Data != "payload" {
		t.Errorf("expected data %q, got %q", "payload", event.Data)
	}
}

func TestReader_OpenAISample(t *testing.T) {
	input := `data: {"id":"chatcmpl-xxx","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-xxx","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`
	reader := NewReader(strings.NewReader(input))

	events, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	if !strings.Contains(events[0].Data, `"role":"assistant"`) {
		t.Errorf("first event should contain role")
	}
	if !strings.Contains(events[1].Data, `"content":"Hello"`) {
		t.Errorf("second event should contain content")
	}
	if !strings.Contains(events[2].Data, `"finish_reason":"stop"`) {
		t.Errorf("third event should contain finish_reason")
	}
}
