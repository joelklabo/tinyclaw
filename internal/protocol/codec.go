// Package protocol implements line-delimited JSON framing for stdio communication.
package protocol

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Frame is a single protocol message as raw JSON bytes.
type Frame struct {
	Raw json.RawMessage
}

// Decoder reads line-delimited JSON frames from a reader.
type Decoder struct {
	scanner *bufio.Scanner
}

// NewDecoder creates a Decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{scanner: bufio.NewScanner(r)}
}

// Decode reads the next frame. Returns io.EOF when no more frames are available.
func (d *Decoder) Decode() (Frame, error) {
	for d.scanner.Scan() {
		line := d.scanner.Bytes()
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if !json.Valid(line) {
			return Frame{}, fmt.Errorf("invalid JSON frame: %s", line)
		}
		// Copy to avoid scanner buffer reuse
		raw := make([]byte, len(line))
		copy(raw, line)
		return Frame{Raw: json.RawMessage(raw)}, nil
	}
	if err := d.scanner.Err(); err != nil {
		return Frame{}, err
	}
	return Frame{}, io.EOF
}

// Encoder writes line-delimited JSON frames to a writer.
type Encoder struct {
	w io.Writer
}

// NewEncoder creates an Encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

// Encode writes a single frame followed by a newline.
// Returns an error if the raw JSON contains embedded newlines.
func (e *Encoder) Encode(f Frame) error {
	if bytes.Contains(f.Raw, []byte("\n")) {
		return fmt.Errorf("frame contains embedded newline")
	}
	_, err := e.w.Write(append(f.Raw, '\n'))
	return err
}
