package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunTestHappyPath(t *testing.T) {
	scenarioYAML := `
name: test-echo
description: Basic test
inbound_events:
  - type: message
    content: "hello"
harness_events:
  - kind: status
    phase: "thinking"
  - kind: final
    content: "Hello!"
expected_transport_ops:
  - kind: typing
  - kind: post
`
	dir := t.TempDir()
	scenarioFile := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(scenarioFile, []byte(scenarioYAML), 0644); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	configYAML := "bundle_dir: " + dir + "\n"
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	if err := RunTest(scenarioFile, configFile); err != nil {
		t.Fatalf("RunTest: %v", err)
	}
}

func TestRunTestEmptyScenarioFile(t *testing.T) {
	err := RunTest("", "")
	if err == nil {
		t.Fatal("expected error for empty scenario file")
	}
}

func TestRunTestBadConfig(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configFile, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(configFile, 0000); err != nil {
		t.Fatal(err)
	}
	defer os.Chmod(configFile, 0644)

	err := RunTest("test.yaml", configFile)
	if err == nil {
		t.Fatal("expected error for unreadable config")
	}
}

func TestRunTestBadScenario(t *testing.T) {
	err := RunTest("/nonexistent/scenario.yaml", "")
	if err == nil {
		t.Fatal("expected error for missing scenario")
	}
}

func TestRunReplayHappyPath(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundle-test")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatal(err)
	}

	runJSON := `{"id":"test-1","start_time":"2025-01-01T00:00:00Z","scenario":"hello","status":"pass"}`
	if err := os.WriteFile(filepath.Join(bundleDir, "run.json"), []byte(runJSON), 0644); err != nil {
		t.Fatal(err)
	}

	if err := RunReplay(bundleDir); err != nil {
		t.Fatalf("RunReplay: %v", err)
	}
}

func TestRunReplayBadDir(t *testing.T) {
	err := RunReplay("/nonexistent/bundle")
	if err == nil {
		t.Fatal("expected error for missing bundle")
	}
}

func TestRunReplayInvalidBundle(t *testing.T) {
	dir := t.TempDir()
	bundleDir := filepath.Join(dir, "bundle-test")
	if err := os.MkdirAll(bundleDir, 0755); err != nil {
		t.Fatal(err)
	}

	runJSON := `{"id":"test-1","start_time":"2025-01-01T00:00:00Z","scenario":"hello","status":"fail"}`
	if err := os.WriteFile(filepath.Join(bundleDir, "run.json"), []byte(runJSON), 0644); err != nil {
		t.Fatal(err)
	}

	err := RunReplay(bundleDir)
	if err == nil {
		t.Fatal("expected error for invalid bundle (missing FAIL.md)")
	}
}

func TestRunTestRunScenarioError(t *testing.T) {
	// Create a scenario that is valid but use a bundle_dir that causes MkdirAll to fail.
	// Writing a regular file at the bundle_dir path prevents MkdirAll from creating subdirs.
	scenarioYAML := `
name: test-echo
description: Basic test
inbound_events:
  - type: message
    content: "hello"
harness_events:
  - kind: final
    content: "Hello!"
expected_transport_ops:
  - kind: post
`
	dir := t.TempDir()
	scenarioFile := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(scenarioFile, []byte(scenarioYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a file where the bundle writer would try to mkdir, causing MkdirAll to fail.
	blocker := filepath.Join(dir, "blocked")
	if err := os.WriteFile(blocker, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	configYAML := "bundle_dir: " + blocker + "\n"
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	err := RunTest(scenarioFile, configFile)
	if err == nil {
		t.Fatal("expected error for run scenario failure")
	}
}

func TestRunTestAssertionError(t *testing.T) {
	// Create a scenario where expected ops don't match actual ops.
	// The harness produces status->typing and final->post, but we only expect "post".
	scenarioYAML := `
name: test-mismatch
description: Mismatch test
inbound_events:
  - type: message
    content: "hello"
harness_events:
  - kind: status
    phase: "thinking"
  - kind: final
    content: "Hello!"
expected_transport_ops:
  - kind: post
`
	dir := t.TempDir()
	scenarioFile := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(scenarioFile, []byte(scenarioYAML), 0644); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(dir, "config.yaml")
	configYAML := "bundle_dir: " + dir + "\n"
	if err := os.WriteFile(configFile, []byte(configYAML), 0644); err != nil {
		t.Fatal(err)
	}

	err := RunTest(scenarioFile, configFile)
	if err == nil {
		t.Fatal("expected assertion error for mismatched ops")
	}
}
