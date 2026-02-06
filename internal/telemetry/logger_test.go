package telemetry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewLogger(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "run-123")
	if err != nil {
		t.Fatal(err)
	}
	defer logger.Close()

	if logger == nil {
		t.Fatal("expected non-nil logger")
	}
}

func TestLoggerWritesToFile(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "run-456")
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("test message", "key", "value")
	logger.Close()

	logFile := filepath.Join(dir, "logs.jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "test message") {
		t.Fatalf("log file missing message: %s", content)
	}
	if !strings.Contains(content, "run-456") {
		t.Fatalf("log file missing runId: %s", content)
	}
}

func TestLoggerJSONL(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "run-789")
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("first")
	logger.Info("second")
	logger.Close()

	logFile := filepath.Join(dir, "logs.jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 log lines, got %d", len(lines))
	}
	// Each line should be valid JSON
	for i, line := range lines {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("line %d not valid JSON: %v", i, err)
		}
	}
}

func TestLoggerIncludesRunId(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "run-abc")
	if err != nil {
		t.Fatal(err)
	}

	logger.Info("check fields")
	logger.Close()

	logFile := filepath.Join(dir, "logs.jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(string(data))), &m); err != nil {
		t.Fatal(err)
	}
	if m["runId"] != "run-abc" {
		t.Fatalf("expected runId %q, got %v", "run-abc", m["runId"])
	}
}

func TestNewLoggerBadDir(t *testing.T) {
	_, err := NewLogger("/nonexistent/path/that/does/not/exist", "run-bad")
	if err == nil {
		t.Fatal("expected error for bad directory")
	}
}

func TestLoggerWarnAndError(t *testing.T) {
	dir := t.TempDir()
	logger, err := NewLogger(dir, "run-warn")
	if err != nil {
		t.Fatal(err)
	}

	logger.Warn("warning msg")
	logger.Error("error msg")
	logger.Close()

	logFile := filepath.Join(dir, "logs.jsonl")
	data, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "warning msg") {
		t.Fatal("missing warn message")
	}
	if !strings.Contains(content, "error msg") {
		t.Fatal("missing error message")
	}
}
