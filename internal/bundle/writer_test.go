package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewWriter(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "abc", "demo")
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}

	// Dir should end with bundle-abc
	if !strings.HasSuffix(w.Dir(), "bundle-abc") {
		t.Fatalf("Dir = %q, want suffix bundle-abc", w.Dir())
	}

	// run.json should exist
	data, err := os.ReadFile(filepath.Join(w.Dir(), "run.json"))
	if err != nil {
		t.Fatalf("read run.json: %v", err)
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("parse run.json: %v", err)
	}
	if meta.ID != "abc" {
		t.Errorf("meta.ID = %q, want abc", meta.ID)
	}
	if meta.Scenario != "demo" {
		t.Errorf("meta.Scenario = %q, want demo", meta.Scenario)
	}
	if meta.Status != "running" {
		t.Errorf("meta.Status = %q, want running", meta.Status)
	}
}

func TestNewWriterBadDir(t *testing.T) {
	// Use a file path as base so MkdirAll fails.
	base := t.TempDir()
	blocker := filepath.Join(base, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := NewWriter(blocker, "id", "sc")
	if err == nil {
		t.Fatal("expected error for bad dir")
	}
	if !strings.Contains(err.Error(), "bundle: mkdir") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewWriterMetaWriteError(t *testing.T) {
	base := t.TempDir()
	// Pre-create bundle-id/run.json as a directory so WriteFile fails.
	dir := filepath.Join(base, "bundle-id")
	if err := os.MkdirAll(filepath.Join(dir, "run.json"), 0755); err != nil {
		t.Fatal(err)
	}
	_, err := NewWriter(base, "id", "sc")
	if err == nil {
		t.Fatal("expected error when run.json is a directory")
	}
}

func TestDirMeta(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "t1", "s1")
	if err != nil {
		t.Fatal(err)
	}
	if w.Dir() == "" {
		t.Error("Dir() should not be empty")
	}
	m := w.Meta()
	if m.ID != "t1" {
		t.Errorf("Meta().ID = %q, want t1", m.ID)
	}
}

func TestAppendJSONL(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "jl", "sc")
	if err != nil {
		t.Fatal(err)
	}

	type rec struct {
		N int `json:"n"`
	}
	for i := 0; i < 3; i++ {
		if err := w.AppendJSONL("events.jsonl", rec{N: i}); err != nil {
			t.Fatalf("AppendJSONL(%d): %v", i, err)
		}
	}

	// Content is only guaranteed after Close.
	if err := w.Close("pass"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(w.Dir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}
	for i, line := range lines {
		var r rec
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("line %d: %v", i, err)
		}
		if r.N != i {
			t.Errorf("line %d: N = %d, want %d", i, r.N, i)
		}
	}
}

func TestAppendJSONLMarshalError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "me", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("error")

	err = w.AppendJSONL("events.jsonl", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "bundle: marshal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendJSONLOpenError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "oe", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("error")

	// Create a directory where the file would be.
	if err := os.MkdirAll(filepath.Join(w.Dir(), "events.jsonl"), 0755); err != nil {
		t.Fatal(err)
	}
	err = w.AppendJSONL("events.jsonl", "hello")
	if err == nil {
		t.Fatal("expected open error")
	}
	if !strings.Contains(err.Error(), "bundle: events.jsonl") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteJSON(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "wj", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("pass")

	payload := map[string]string{"key": "value"}
	if err := w.WriteJSON("ctx.json", payload); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(w.Dir(), "ctx.json"))
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse ctx.json: %v", err)
	}
	if got["key"] != "value" {
		t.Errorf("got %v, want key=value", got)
	}
}

func TestWriteJSONMarshalError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "wjm", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("error")

	err = w.WriteJSON("bad.json", make(chan int))
	if err == nil {
		t.Fatal("expected marshal error")
	}
	if !strings.Contains(err.Error(), "bundle: marshal") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteJSONWriteError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "wjw", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("error")

	// Create a directory at the target path so WriteFile fails.
	if err := os.MkdirAll(filepath.Join(w.Dir(), "ctx.json"), 0755); err != nil {
		t.Fatal(err)
	}
	err = w.WriteJSON("ctx.json", "hello")
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "bundle: write ctx.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteFail(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "wf", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("fail")

	if err := w.WriteFail("something broke"); err != nil {
		t.Fatalf("WriteFail: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(w.Dir(), "FAIL.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "something broke") {
		t.Errorf("FAIL.md content = %q, missing message", string(data))
	}
	if !strings.HasPrefix(string(data), "# Run Failed") {
		t.Errorf("FAIL.md should start with # Run Failed")
	}
}

func TestWriteFailError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "wfe", "sc")
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close("error")

	// Create a directory at the FAIL.md path so WriteFile fails.
	if err := os.MkdirAll(filepath.Join(w.Dir(), "FAIL.md"), 0755); err != nil {
		t.Fatal(err)
	}
	err = w.WriteFail("oops")
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "bundle: write FAIL.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClose(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "cl", "sc")
	if err != nil {
		t.Fatal(err)
	}

	if err := w.AppendJSONL("events.jsonl", "line1"); err != nil {
		t.Fatal(err)
	}

	if err := w.Close("pass"); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Verify run.json updated.
	data, err := os.ReadFile(filepath.Join(w.Dir(), "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Status != "pass" {
		t.Errorf("status = %q, want pass", meta.Status)
	}
	if meta.EndTime == "" {
		t.Error("EndTime should be set after Close")
	}
}

func TestCloseMetaWriteError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "cle", "sc")
	if err != nil {
		t.Fatal(err)
	}

	// Replace run.json with a directory so the final writeMeta fails.
	runPath := filepath.Join(w.Dir(), "run.json")
	os.Remove(runPath)
	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatal(err)
	}

	err = w.Close("pass")
	if err == nil {
		t.Fatal("expected error from Close when writeMeta fails")
	}
}

func TestCloseFlushError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "fe", "sc")
	if err != nil {
		t.Fatal(err)
	}

	if err := w.AppendJSONL("events.jsonl", "line1"); err != nil {
		t.Fatal(err)
	}

	// Close the underlying file to cause Flush to fail.
	for _, bw := range w.buffers {
		bw.f.Close()
	}

	err = w.Close("pass")
	if err == nil {
		t.Fatal("expected error from Close when Flush fails")
	}
}

func TestCloseFileCloseError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "fce", "sc")
	if err != nil {
		t.Fatal(err)
	}

	// AppendJSONL but with no buffered data, so Flush succeeds but we need
	// to trigger Close error. Write something tiny so flush works, then
	// close the underlying fd so f.Close fails.
	if err := w.AppendJSONL("events.jsonl", "x"); err != nil {
		t.Fatal(err)
	}

	// Flush manually so the buffer is clean, then close the fd.
	for _, bw := range w.buffers {
		bw.w.Flush()
		bw.f.Close() // first close succeeds; second close (in Close) will fail
	}

	err = w.Close("pass")
	// Either the f.Close or writeMeta could fail.
	if err == nil {
		t.Fatal("expected error from Close when f.Close fails")
	}
}

func TestCloseFlushErrorAndMetaError(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "fme", "sc")
	if err != nil {
		t.Fatal(err)
	}

	if err := w.AppendJSONL("events.jsonl", "line1"); err != nil {
		t.Fatal(err)
	}

	// Close underlying file to cause Flush error (firstErr set).
	for _, bw := range w.buffers {
		bw.f.Close()
	}

	// Also make writeMeta fail by replacing run.json with a directory.
	runPath := filepath.Join(w.Dir(), "run.json")
	os.Remove(runPath)
	if err := os.MkdirAll(runPath, 0755); err != nil {
		t.Fatal(err)
	}

	err = w.Close("pass")
	if err == nil {
		t.Fatal("expected error")
	}
	// The firstErr from Flush should be returned (not the writeMeta error).
}

func TestMultipleJSONLFiles(t *testing.T) {
	base := t.TempDir()
	w, err := NewWriter(base, "mf", "sc")
	if err != nil {
		t.Fatal(err)
	}

	if err := w.AppendJSONL("events.jsonl", "e1"); err != nil {
		t.Fatal(err)
	}
	if err := w.AppendJSONL("transport.jsonl", "t1"); err != nil {
		t.Fatal(err)
	}
	if err := w.AppendJSONL("events.jsonl", "e2"); err != nil {
		t.Fatal(err)
	}

	if err := w.Close("pass"); err != nil {
		t.Fatal(err)
	}

	// Verify events.jsonl has 2 lines.
	edata, err := os.ReadFile(filepath.Join(w.Dir(), "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	elines := strings.Split(strings.TrimSpace(string(edata)), "\n")
	if len(elines) != 2 {
		t.Errorf("events.jsonl: got %d lines, want 2", len(elines))
	}

	// Verify transport.jsonl has 1 line.
	tdata, err := os.ReadFile(filepath.Join(w.Dir(), "transport.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	tlines := strings.Split(strings.TrimSpace(string(tdata)), "\n")
	if len(tlines) != 1 {
		t.Errorf("transport.jsonl: got %d lines, want 1", len(tlines))
	}
}

func TestWithNowFunc(t *testing.T) {
	fixed := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return fixed }

	base := t.TempDir()
	w, err := NewWriter(base, "clk", "sc", WithNowFunc(clock))
	if err != nil {
		t.Fatal(err)
	}

	m := w.Meta()
	if m.StartTime != "2025-06-15T12:00:00Z" {
		t.Fatalf("StartTime = %q, want 2025-06-15T12:00:00Z", m.StartTime)
	}

	if err := w.Close("pass"); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(filepath.Join(w.Dir(), "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.EndTime != "2025-06-15T12:00:00Z" {
		t.Fatalf("EndTime = %q, want 2025-06-15T12:00:00Z", meta.EndTime)
	}
}
