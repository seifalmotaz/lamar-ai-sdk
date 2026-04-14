package ndjson

import (
	"bufio"
	"encoding/json"
	"io"
)

type Reader struct {
	scanner *bufio.Scanner
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

func (r *Reader) Read(v any) error {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	line := r.scanner.Bytes()
	if len(line) == 0 {
		return io.EOF
	}

	return json.Unmarshal(line, v)
}

func (r *Reader) ReadRaw() ([]byte, error) {
	if !r.scanner.Scan() {
		if err := r.scanner.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	line := r.scanner.Bytes()
	if len(line) == 0 {
		return nil, io.EOF
	}

	return line, nil
}
