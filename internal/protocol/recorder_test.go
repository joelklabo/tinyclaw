package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecorderWritesFrames(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecorder(dir, "run-1")
	if err != nil {
		t.Fatalf("NewRecorder: %v", err)
	}
	frame := Frame{Raw: json.RawMessage(`{"method":"test"}`)}
	if err := rec.Record("inbound", "transport-1", frame); err != nil {
		t.Fatalf("Record: %v", err)
	}
	if err := rec.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "frames.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var entry RecordedFrame
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry.RunID != "run-1" {
		t.Fatalf("runId = %q, want run-1", entry.RunID)
	}
	if entry.Direction != "inbound" {
		t.Fatalf("direction = %q, want inbound", entry.Direction)
	}
	if entry.PluginID != "transport-1" {
		t.Fatalf("pluginId = %q, want transport-1", entry.PluginID)
	}
	if entry.Timestamp.IsZero() {
		t.Fatal("timestamp should not be zero")
	}
}

func TestRecorderMultipleFrames(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecorder(dir, "run-2")
	if err != nil {
		t.Fatal(err)
	}
	rec.Record("inbound", "t1", Frame{Raw: json.RawMessage(`{"a":1}`)})
	rec.Record("outbound", "t2", Frame{Raw: json.RawMessage(`{"b":2}`)})
	rec.Close()

	data, _ := os.ReadFile(filepath.Join(dir, "frames.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var e1, e2 RecordedFrame
	json.Unmarshal([]byte(lines[0]), &e1)
	json.Unmarshal([]byte(lines[1]), &e2)
	if e1.Direction != "inbound" || e2.Direction != "outbound" {
		t.Fatal("frame directions not preserved")
	}
}

func TestRecorderTimestampIncreases(t *testing.T) {
	dir := t.TempDir()
	rec, err := NewRecorder(dir, "run-3")
	if err != nil {
		t.Fatal(err)
	}
	rec.Record("inbound", "t1", Frame{Raw: json.RawMessage(`{"a":1}`)})
	time.Sleep(time.Millisecond)
	rec.Record("outbound", "t1", Frame{Raw: json.RawMessage(`{"b":2}`)})
	rec.Close()

	data, _ := os.ReadFile(filepath.Join(dir, "frames.jsonl"))
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	var e1, e2 RecordedFrame
	json.Unmarshal([]byte(lines[0]), &e1)
	json.Unmarshal([]byte(lines[1]), &e2)
	if !e2.Timestamp.After(e1.Timestamp) {
		t.Fatal("second timestamp should be after first")
	}
}

func TestRecorderBadDir(t *testing.T) {
	_, err := NewRecorder("/nonexistent/path/deep", "run-1")
	if err == nil {
		t.Fatal("expected error for bad directory")
	}
}

func TestRecorderFramePayload(t *testing.T) {
	dir := t.TempDir()
	rec, _ := NewRecorder(dir, "run-4")
	original := `{"jsonrpc":"2.0","method":"tools/call"}`
	rec.Record("outbound", "harness-1", Frame{Raw: json.RawMessage(original)})
	rec.Close()

	data, _ := os.ReadFile(filepath.Join(dir, "frames.jsonl"))
	var entry RecordedFrame
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &entry)
	if string(entry.Frame) != original {
		t.Fatalf("frame payload mismatch: got %s", entry.Frame)
	}
}
