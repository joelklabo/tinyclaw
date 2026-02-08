package bundle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBundle(t *testing.T) {
	dir := t.TempDir()
	meta := Meta{
		ID:        "run1",
		StartTime: "2025-01-01T00:00:00Z",
		Scenario:  "demo",
		Status:    "pass",
		EndTime:   "2025-01-01T00:01:00Z",
	}
	data, _ := json.Marshal(meta)
	if err := os.WriteFile(filepath.Join(dir, "run.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	// Add an optional file.
	if err := os.WriteFile(filepath.Join(dir, "events.jsonl"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	info, err := LoadBundle(dir)
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if info.Meta.ID != "run1" {
		t.Errorf("ID = %q, want run1", info.Meta.ID)
	}
	if info.Dir != dir {
		t.Errorf("Dir = %q, want %q", info.Dir, dir)
	}
	// Files should include run.json and events.jsonl.
	found := make(map[string]bool)
	for _, f := range info.Files {
		found[f] = true
	}
	if !found["run.json"] {
		t.Error("missing run.json in Files")
	}
	if !found["events.jsonl"] {
		t.Error("missing events.jsonl in Files")
	}
}

func TestLoadBundleMissingDir(t *testing.T) {
	_, err := LoadBundle("/nonexistent/path/bundle-xyz")
	if err == nil {
		t.Fatal("expected error for missing dir")
	}
	if !strings.Contains(err.Error(), "bundle: read run.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadBundleInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "run.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBundle(dir)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "bundle: parse run.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadBundleMissingID(t *testing.T) {
	dir := t.TempDir()
	data, _ := json.Marshal(Meta{Status: "pass"})
	if err := os.WriteFile(filepath.Join(dir, "run.json"), data, 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBundle(dir)
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
	if !strings.Contains(err.Error(), "bundle: run.json missing id") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidatePass(t *testing.T) {
	info := &BundleInfo{
		Meta:  Meta{ID: "ok", Status: "pass"},
		Files: []string{"run.json"},
	}
	if err := Validate(info); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestValidateMissingRequired(t *testing.T) {
	info := &BundleInfo{
		Meta:  Meta{ID: "bad", Status: "pass"},
		Files: []string{},
	}
	err := Validate(info)
	if err == nil {
		t.Fatal("expected error for missing required file")
	}
	if !strings.Contains(err.Error(), "bundle: missing required file run.json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFailWithoutFAIL(t *testing.T) {
	info := &BundleInfo{
		Meta:  Meta{ID: "f1", Status: "fail"},
		Files: []string{"run.json"},
	}
	err := Validate(info)
	if err == nil {
		t.Fatal("expected error for fail status without FAIL.md")
	}
	if !strings.Contains(err.Error(), "bundle: failed run missing FAIL.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateErrorWithoutFAIL(t *testing.T) {
	info := &BundleInfo{
		Meta:  Meta{ID: "e1", Status: "error"},
		Files: []string{"run.json"},
	}
	err := Validate(info)
	if err == nil {
		t.Fatal("expected error for error status without FAIL.md")
	}
	if !strings.Contains(err.Error(), "bundle: failed run missing FAIL.md") {
		t.Fatalf("unexpected error: %v", err)
	}
}
