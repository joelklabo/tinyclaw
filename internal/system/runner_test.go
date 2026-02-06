package system

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/klabo/tinyclaw/internal/bundles"
)

func TestRunScenarioBadBaseDir(t *testing.T) {
	_, err := RunScenario("/dev/null/impossible", "test")
	if err == nil {
		t.Fatal("expected error for invalid base dir")
	}
}

func TestRunScenarioWriteFailError(t *testing.T) {
	orig := newBundleWriter
	t.Cleanup(func() { newBundleWriter = orig })

	newBundleWriter = func(_, _, _ string) (bundleWriter, error) {
		return &failWriter{failOn: "writefail"}, nil
	}
	_, err := RunScenario(t.TempDir(), "test")
	if err == nil {
		t.Fatal("expected error from WriteFail")
	}
}

func TestRunScenarioCloseError(t *testing.T) {
	orig := newBundleWriter
	t.Cleanup(func() { newBundleWriter = orig })

	newBundleWriter = func(_, _, _ string) (bundleWriter, error) {
		return &failWriter{failOn: "close"}, nil
	}
	_, err := RunScenario(t.TempDir(), "test")
	if err == nil {
		t.Fatal("expected error from Close")
	}
}

func TestRunScenarioProducesBundle(t *testing.T) {
	baseDir := t.TempDir()
	scenario := "echo-hello"

	dir, err := RunScenario(baseDir, scenario)
	if err != nil {
		t.Fatalf("RunScenario returned error: %v", err)
	}

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("bundle dir does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Fatal("expected bundle path to be a directory")
	}

	runData, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		t.Fatalf("run.json not found: %v", err)
	}
	var meta bundles.RunMeta
	if err := json.Unmarshal(runData, &meta); err != nil {
		t.Fatalf("run.json invalid JSON: %v", err)
	}
	if meta.Scenario != scenario {
		t.Fatalf("run.json scenario = %q, want %q", meta.Scenario, scenario)
	}
	if meta.Status != "fail" {
		t.Fatalf("run.json status = %q, want fail", meta.Status)
	}

	failData, err := os.ReadFile(filepath.Join(dir, "FAIL.md"))
	if err != nil {
		t.Fatalf("FAIL.md not found: %v", err)
	}
	if len(failData) == 0 {
		t.Fatal("FAIL.md is empty")
	}
}

// failWriter is a test double that fails on the specified operation.
type failWriter struct {
	failOn string
}

func (f *failWriter) Dir() string { return "/tmp/fake-bundle" }

func (f *failWriter) WriteFail(_ string) error {
	if f.failOn == "writefail" {
		return errors.New("injected WriteFail error")
	}
	return nil
}

func (f *failWriter) Close(_ string) error {
	if f.failOn == "close" {
		return errors.New("injected Close error")
	}
	return nil
}
