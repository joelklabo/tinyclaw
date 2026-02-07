// Package bundle writes and loads run bundles.
package bundle

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Meta is the metadata stored in run.json.
type Meta struct {
	ID        string `json:"id"`
	StartTime string `json:"start_time"`
	Scenario  string `json:"scenario"`
	Status    string `json:"status"`
	EndTime   string `json:"end_time,omitempty"`
}

// Option configures a Writer.
type Option func(*Writer)

// WithNowFunc overrides the clock used for timestamps.
func WithNowFunc(fn func() time.Time) Option {
	return func(w *Writer) { w.nowFn = fn }
}

// Writer creates and writes to a run bundle directory.
type Writer struct {
	dir     string
	meta    Meta
	buffers map[string]*bufWriter
	nowFn   func() time.Time
}

// bufWriter wraps a file and a buffered writer for JSONL appending.
type bufWriter struct {
	f *os.File
	w *bufio.Writer
}

// NewWriter creates a new bundle directory and writes the initial run.json.
func NewWriter(baseDir, id, scenario string, opts ...Option) (*Writer, error) {
	dir := filepath.Join(baseDir, "bundle-"+id)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("bundle: mkdir: %w", err)
	}

	w := &Writer{dir: dir, buffers: make(map[string]*bufWriter), nowFn: time.Now}
	for _, o := range opts {
		o(w)
	}

	w.meta = Meta{
		ID:        id,
		StartTime: w.nowFn().UTC().Format(time.RFC3339),
		Scenario:  scenario,
		Status:    "running",
	}

	if err := w.writeMeta(); err != nil {
		return nil, err
	}
	return w, nil
}

// Dir returns the bundle directory path.
func (w *Writer) Dir() string {
	return w.dir
}

// Meta returns a copy of the current run metadata.
func (w *Writer) Meta() Meta {
	return w.meta
}

// AppendJSONL appends a JSON-encoded line to the named file in the bundle.
// Writes are buffered and flushed on Close.
func (w *Writer) AppendJSONL(filename string, v any) error {
	bw, err := w.getBuffer(filename)
	if err != nil {
		return fmt.Errorf("bundle: %s: %w", filename, err)
	}
	if err := json.NewEncoder(bw.w).Encode(v); err != nil {
		return fmt.Errorf("bundle: marshal: %w", err)
	}
	return nil
}

func (w *Writer) getBuffer(filename string) (*bufWriter, error) {
	if bw, ok := w.buffers[filename]; ok {
		return bw, nil
	}
	path := filepath.Join(w.dir, filename)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	bw := &bufWriter{f: f, w: bufio.NewWriterSize(f, 32*1024)}
	w.buffers[filename] = bw
	return bw, nil
}

// WriteJSON writes a single JSON object to the named file in the bundle.
func (w *Writer) WriteJSON(filename string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("bundle: marshal: %w", err)
	}
	data = append(data, '\n')

	path := filepath.Join(w.dir, filename)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("bundle: write %s: %w", filename, err)
	}
	return nil
}

// WriteFail writes a FAIL.md file with the given error description.
func (w *Writer) WriteFail(msg string) error {
	content := "# Run Failed\n\n" + msg + "\n"
	path := filepath.Join(w.dir, "FAIL.md")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("bundle: write FAIL.md: %w", err)
	}
	return nil
}

// Close flushes all buffered writers, closes their files, then finalizes run.json.
func (w *Writer) Close(status string) error {
	var firstErr error
	for _, bw := range w.buffers {
		if err := bw.w.Flush(); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := bw.f.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	w.buffers = make(map[string]*bufWriter)

	w.meta.Status = status
	w.meta.EndTime = w.nowFn().UTC().Format(time.RFC3339)
	if err := w.writeMeta(); err != nil {
		if firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (w *Writer) writeMeta() error {
	return w.WriteJSON("run.json", w.meta)
}
