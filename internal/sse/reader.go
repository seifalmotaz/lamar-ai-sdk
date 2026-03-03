package sse

import (
	"bufio"
	"errors"
	"io"
	"strings"
)

// Event represents a single Server-Sent Event.
type Event struct {
	ID    string // Event ID (from "id:" field)
	Event string // Event type (from "event:" field, defaults to "message")
	Data  string // Event data (from "data:" field)
}

// Done is returned when the stream sends [DONE].
var Done = errors.New("stream done")

// Reader reads SSE events from an io.Reader.
type Reader struct {
	scanner *bufio.Scanner
	done    bool
}

// NewReader creates a new SSE reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// ReadEvent reads the next SSE event.
// Returns Done when the stream sends "[DONE]".
// Returns io.EOF when the underlying stream ends.
func (r *Reader) ReadEvent() (Event, error) {
	if r.done {
		return Event{}, io.EOF
	}

	var event Event
	var dataLines []string

	for r.scanner.Scan() {
		line := r.scanner.Text()

		// Empty line signals end of event
		if line == "" {
			if len(dataLines) > 0 || event.ID != "" || event.Event != "" {
				event.Data = strings.Join(dataLines, "\n")

				// Check for [DONE] marker
				if event.Data == "[DONE]" {
					r.done = true
					return Event{}, Done
				}

				// Set default event type
				if event.Event == "" {
					event.Event = "message"
				}

				return event, nil
			}
			continue
		}

		// Parse field
		field, value, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		// Remove leading space from value
		value = strings.TrimPrefix(value, " ")

		switch field {
		case "id":
			event.ID = value
		case "event":
			event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "retry":
			// Ignore retry field for now
		}
	}

	if err := r.scanner.Err(); err != nil {
		return Event{}, err
	}

	// Handle last event if stream ends without blank line
	if len(dataLines) > 0 || event.ID != "" || event.Event != "" {
		event.Data = strings.Join(dataLines, "\n")
		if event.Event == "" {
			event.Event = "message"
		}
		return event, nil
	}

	return Event{}, io.EOF
}

// ReadAll reads all events until Done or EOF.
// This is a convenience method for testing and simple use cases.
func (r *Reader) ReadAll() ([]Event, error) {
	var events []Event
	for {
		event, err := r.ReadEvent()
		if err != nil {
			if errors.Is(err, Done) || errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		events = append(events, event)
	}
	return events, nil
}
