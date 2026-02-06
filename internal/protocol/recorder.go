package protocol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// RecordedFrame is a single protocol frame with metadata, written to frames.jsonl.
type RecordedFrame struct {
	RunID     string          `json:"runId"`
	Direction string          `json:"direction"`
	PluginID  string          `json:"pluginId"`
	Timestamp time.Time       `json:"timestamp"`
	Frame     json.RawMessage `json:"frame"`
}

// Recorder writes protocol frames to a frames.jsonl file in a bundle directory.
type Recorder struct {
	runID string
	file  *os.File
	enc   *json.Encoder
}

// NewRecorder creates a Recorder that writes to frames.jsonl in the given directory.
func NewRecorder(dir string, runID string) (*Recorder, error) {
	path := filepath.Join(dir, "frames.jsonl")
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("creating frames.jsonl: %w", err)
	}
	return &Recorder{
		runID: runID,
		file:  f,
		enc:   json.NewEncoder(f),
	}, nil
}

// Record writes a frame entry to the file.
func (r *Recorder) Record(direction, pluginID string, frame Frame) error {
	entry := RecordedFrame{
		RunID:     r.runID,
		Direction: direction,
		PluginID:  pluginID,
		Timestamp: time.Now(),
		Frame:     frame.Raw,
	}
	return r.enc.Encode(entry)
}

// Close flushes and closes the underlying file.
func (r *Recorder) Close() error {
	return r.file.Close()
}
