package protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDecodeSingleFrame(t *testing.T) {
	input := `{"jsonrpc":"2.0","method":"test"}` + "\n"
	dec := NewDecoder(strings.NewReader(input))
	frame, err := dec.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	if frame.Raw == nil {
		t.Fatal("expected non-nil raw")
	}
	var msg map[string]any
	if err := json.Unmarshal(frame.Raw, &msg); err != nil {
		t.Fatal(err)
	}
	if msg["method"] != "test" {
		t.Fatalf("method = %v, want test", msg["method"])
	}
}

func TestDecodeMultipleFrames(t *testing.T) {
	input := `{"id":1}` + "\n" + `{"id":2}` + "\n"
	dec := NewDecoder(strings.NewReader(input))
	f1, err := dec.Decode()
	if err != nil {
		t.Fatalf("Decode 1: %v", err)
	}
	f2, err := dec.Decode()
	if err != nil {
		t.Fatalf("Decode 2: %v", err)
	}
	var m1, m2 map[string]any
	json.Unmarshal(f1.Raw, &m1)
	json.Unmarshal(f2.Raw, &m2)
	if m1["id"].(float64) != 1 || m2["id"].(float64) != 2 {
		t.Fatal("frame ordering not preserved")
	}
}

func TestDecodeEOF(t *testing.T) {
	dec := NewDecoder(strings.NewReader(""))
	_, err := dec.Decode()
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestDecodeSkipsEmptyLines(t *testing.T) {
	input := "\n\n" + `{"ok":true}` + "\n\n"
	dec := NewDecoder(strings.NewReader(input))
	frame, err := dec.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	var msg map[string]any
	json.Unmarshal(frame.Raw, &msg)
	if msg["ok"] != true {
		t.Fatal("expected ok:true")
	}
	_, err = dec.Decode()
	if err != io.EOF {
		t.Fatalf("expected EOF after single frame, got %v", err)
	}
}

func TestDecodeRejectsInvalidJSON(t *testing.T) {
	input := "not json\n"
	dec := NewDecoder(strings.NewReader(input))
	_, err := dec.Decode()
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if err == io.EOF {
		t.Fatal("expected parse error, not EOF")
	}
}

func TestEncodeSingleFrame(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	msg := json.RawMessage(`{"jsonrpc":"2.0","id":1}`)
	if err := enc.Encode(Frame{Raw: msg}); err != nil {
		t.Fatalf("Encode: %v", err)
	}
	line := buf.String()
	if !strings.HasSuffix(line, "\n") {
		t.Fatal("encoded frame must end with newline")
	}
	trimmed := strings.TrimSuffix(line, "\n")
	if strings.Contains(trimmed, "\n") {
		t.Fatal("encoded frame must not contain embedded newlines")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		t.Fatalf("encoded output not valid JSON: %v", err)
	}
}

func TestEncodeMultipleFrames(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.Encode(Frame{Raw: json.RawMessage(`{"a":1}`)})
	enc.Encode(Frame{Raw: json.RawMessage(`{"b":2}`)})
	lines := strings.Split(strings.TrimSuffix(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestRoundtrip(t *testing.T) {
	original := `{"jsonrpc":"2.0","id":42,"method":"tools/call","params":{"name":"test"}}`
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.Encode(Frame{Raw: json.RawMessage(original)})

	dec := NewDecoder(&buf)
	frame, err := dec.Decode()
	if err != nil {
		t.Fatalf("roundtrip Decode: %v", err)
	}
	// Verify the JSON content is equivalent
	var orig, decoded map[string]any
	json.Unmarshal([]byte(original), &orig)
	json.Unmarshal(frame.Raw, &decoded)
	if orig["id"].(float64) != decoded["id"].(float64) {
		t.Fatal("id mismatch in roundtrip")
	}
	if orig["method"] != decoded["method"] {
		t.Fatal("method mismatch in roundtrip")
	}
}

func TestRoundtripPreservesFieldOrder(t *testing.T) {
	// json.RawMessage preserves the original byte representation
	original := `{"z":1,"a":2,"m":3}`
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	enc.Encode(Frame{Raw: json.RawMessage(original)})

	dec := NewDecoder(&buf)
	frame, err := dec.Decode()
	if err != nil {
		t.Fatal(err)
	}
	if string(frame.Raw) != original {
		t.Fatalf("field order not preserved: got %s, want %s", frame.Raw, original)
	}
}

func TestEncodeRejectsEmbeddedNewlines(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)
	bad := json.RawMessage("{\"a\":\n\"b\"}")
	err := enc.Encode(Frame{Raw: bad})
	if err == nil {
		t.Fatal("expected error for frame with embedded newlines")
	}
}

type errReader struct{ err error }

func (r *errReader) Read([]byte) (int, error) { return 0, r.err }

func TestDecodeReaderError(t *testing.T) {
	readErr := errors.New("read failed")
	dec := NewDecoder(&errReader{err: readErr})
	_, err := dec.Decode()
	if err == nil {
		t.Fatal("expected error from reader")
	}
	if !errors.Is(err, readErr) {
		t.Fatalf("expected read error, got %v", err)
	}
}
