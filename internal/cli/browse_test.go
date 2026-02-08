package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/klabo/tinyclaw/internal/bundle"
)

func createTestBundle(t *testing.T, dir, id, scenario, status, start, end string) string {
	t.Helper()
	bundleDir := filepath.Join(dir, "bundle-"+id)
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatal(err)
	}
	meta := bundle.Meta{
		ID:        id,
		Scenario:  scenario,
		Status:    status,
		StartTime: start,
		EndTime:   end,
	}
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundleDir, "run.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	return bundleDir
}

func TestRunBrowseListMode(t *testing.T) {
	dir := t.TempDir()
	createTestBundle(t, dir, "test-1", "scenario-a", "pass", "2025-01-01T12:00:00Z", "2025-01-01T12:00:30Z")
	createTestBundle(t, dir, "test-2", "scenario-b", "fail", "2025-01-01T12:01:00Z", "2025-01-01T12:02:00Z")

	err := RunBrowse(Command{Action: ActionBrowse, BundleDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBrowseDetailMode(t *testing.T) {
	dir := t.TempDir()
	bundlePath := createTestBundle(t, dir, "test-1", "scenario-a", "pass", "2025-01-01T12:00:00Z", "2025-01-01T12:00:30Z")

	err := RunBrowse(Command{Action: ActionBrowse, BundlePath: bundlePath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBrowseDetailNoEndTime(t *testing.T) {
	dir := t.TempDir()
	bundlePath := createTestBundle(t, dir, "test-1", "scenario-a", "running", "2025-01-01T12:00:00Z", "")

	err := RunBrowse(Command{Action: ActionBrowse, BundlePath: bundlePath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBrowseEmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := RunBrowse(Command{Action: ActionBrowse, BundleDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunBrowseBadDir(t *testing.T) {
	err := RunBrowse(Command{Action: ActionBrowse, BundleDir: "/nonexistent/path"})
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestRunBrowseDefaultBundleDir(t *testing.T) {
	// With no BundleDir and no config file, it uses default "bundles" dir
	// which won't exist in temp dir context — that's an error from browseList
	err := RunBrowse(Command{Action: ActionBrowse})
	if err == nil {
		t.Fatal("expected error for default bundle dir that doesn't exist")
	}
}

func TestRunBrowseSkipsNonBundles(t *testing.T) {
	dir := t.TempDir()
	// Create a non-bundle directory (no "bundle-" prefix)
	if err := os.MkdirAll(filepath.Join(dir, "other-dir"), 0755); err != nil {
		t.Fatal(err)
	}
	// Create a regular file
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	// Create a bundle dir with invalid run.json
	badBundle := filepath.Join(dir, "bundle-bad")
	if err := os.MkdirAll(badBundle, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(badBundle, "run.json"), []byte("not json"), 0644); err != nil {
		t.Fatal(err)
	}

	err := RunBrowse(Command{Action: ActionBrowse, BundleDir: dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrowseDetailBadBundle(t *testing.T) {
	dir := t.TempDir()
	err := RunBrowse(Command{Action: ActionBrowse, BundlePath: dir})
	if err == nil {
		t.Fatal("expected error for bad bundle")
	}
}

func TestFormatTime(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"2025-01-01T12:30:45Z", "12:30:45"},
		{"not-a-time", "not-a-time"},
	}
	for _, tt := range tests {
		got := formatTime(tt.input)
		if got != tt.want {
			t.Errorf("formatTime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		start, end string
		want       string
	}{
		{"2025-01-01T12:00:00Z", "2025-01-01T12:00:30Z", "30s"},
		{"2025-01-01T12:00:00Z", "2025-01-01T12:05:00Z", "5m"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.start, tt.end)
		if got != tt.want {
			t.Errorf("formatDuration(%q, %q) = %q, want %q", tt.start, tt.end, got, tt.want)
		}
	}
}

func TestFormatDurationInvalid(t *testing.T) {
	tests := []struct {
		start, end string
	}{
		{"bad", "2025-01-01T12:00:00Z"},
		{"2025-01-01T12:00:00Z", "bad"},
		{"bad", "bad"},
	}
	for _, tt := range tests {
		got := formatDuration(tt.start, tt.end)
		if got != "-" {
			t.Errorf("formatDuration(%q, %q) = %q, want %q", tt.start, tt.end, got, "-")
		}
	}
}
