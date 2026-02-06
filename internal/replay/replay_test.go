package replay

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeRunJSON(t *testing.T, dir string, meta BundleMeta) {
	t.Helper()
	data, err := json.Marshal(meta)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "run.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadBundle(t *testing.T) {
	dir := t.TempDir()
	writeRunJSON(t, dir, BundleMeta{
		ID:        "test-1",
		StartTime: "2025-01-01T00:00:00Z",
		Scenario:  "hello-world",
		Status:    "pass",
	})
	// Create some optional files.
	os.WriteFile(filepath.Join(dir, "frames.jsonl"), []byte("{}\n"), 0644)
	os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{}\n"), 0644)

	info, err := LoadBundle(dir)
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if info.Meta.ID != "test-1" {
		t.Fatalf("ID = %q, want test-1", info.Meta.ID)
	}
	if info.Meta.Scenario != "hello-world" {
		t.Fatalf("Scenario = %q, want hello-world", info.Meta.Scenario)
	}
	if info.Dir != dir {
		t.Fatalf("Dir = %q, want %q", info.Dir, dir)
	}
	// Should have run.json + frames.jsonl + events.jsonl.
	if len(info.Files) != 3 {
		t.Fatalf("Files len = %d, want 3", len(info.Files))
	}
}

func TestLoadBundleMissingDir(t *testing.T) {
	_, err := LoadBundle("/nonexistent/bundle")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoadBundleInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "run.json"), []byte("not json"), 0644)
	_, err := LoadBundle(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadBundleMissingID(t *testing.T) {
	dir := t.TempDir()
	writeRunJSON(t, dir, BundleMeta{
		Status: "pass",
	})
	_, err := LoadBundle(dir)
	if err == nil {
		t.Fatal("expected error for missing id")
	}
}

func TestValidatePass(t *testing.T) {
	info := &BundleInfo{
		Files: []string{"run.json"},
		Meta:  BundleMeta{Status: "pass"},
	}
	if err := Validate(info); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	info := &BundleInfo{
		Files: []string{"frames.jsonl"},
		Meta:  BundleMeta{Status: "pass"},
	}
	err := Validate(info)
	if err == nil {
		t.Fatal("expected error for missing required file")
	}
}

func TestValidateFailedRunWithFAIL(t *testing.T) {
	info := &BundleInfo{
		Files: []string{"run.json", "FAIL.md"},
		Meta:  BundleMeta{Status: "fail"},
	}
	if err := Validate(info); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateFailedRunMissingFAIL(t *testing.T) {
	info := &BundleInfo{
		Files: []string{"run.json"},
		Meta:  BundleMeta{Status: "fail"},
	}
	err := Validate(info)
	if err == nil {
		t.Fatal("expected error for failed run missing FAIL.md")
	}
}

func TestValidateErrorStatusMissingFAIL(t *testing.T) {
	info := &BundleInfo{
		Files: []string{"run.json"},
		Meta:  BundleMeta{Status: "error"},
	}
	err := Validate(info)
	if err == nil {
		t.Fatal("expected error for error run missing FAIL.md")
	}
}
