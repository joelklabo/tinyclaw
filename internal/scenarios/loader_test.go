package scenarios

import (
	"os"
	"path/filepath"
	"testing"
)

const validYAML = `
name: hello-world
description: Basic test
inbound_events:
  - type: message
    data:
      content: "hi"
    delay: 100
harness_events:
  - kind: status
    data:
      status: "thinking"
  - kind: final
    data:
      text: "Hello!"
expected_transport_ops:
  - kind: post
  - kind: edit
expected_context:
  must_include:
    - name: readme
expected_failures:
  - kind: auth
    message_contains: "token"
`

func TestParseValid(t *testing.T) {
	s, err := Parse([]byte(validYAML))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.Name != "hello-world" {
		t.Fatalf("Name = %q, want hello-world", s.Name)
	}
	if s.Description != "Basic test" {
		t.Fatalf("Description = %q, want Basic test", s.Description)
	}
	if len(s.InboundEvents) != 1 {
		t.Fatalf("InboundEvents len = %d, want 1", len(s.InboundEvents))
	}
	if s.InboundEvents[0].Type != "message" {
		t.Fatalf("InboundEvents[0].Type = %q, want message", s.InboundEvents[0].Type)
	}
	if s.InboundEvents[0].Delay != 100 {
		t.Fatalf("InboundEvents[0].Delay = %d, want 100", s.InboundEvents[0].Delay)
	}
	if len(s.HarnessEvents) != 2 {
		t.Fatalf("HarnessEvents len = %d, want 2", len(s.HarnessEvents))
	}
	if s.HarnessEvents[0].Kind != "status" {
		t.Fatalf("HarnessEvents[0].Kind = %q, want status", s.HarnessEvents[0].Kind)
	}
	if len(s.ExpectedOps) != 2 {
		t.Fatalf("ExpectedOps len = %d, want 2", len(s.ExpectedOps))
	}
	if s.ExpectedOps[0].Kind != "post" {
		t.Fatalf("ExpectedOps[0].Kind = %q, want post", s.ExpectedOps[0].Kind)
	}
	if s.ExpectedContext == nil {
		t.Fatal("ExpectedContext should not be nil")
	}
	if len(s.ExpectedContext.MustInclude) != 1 {
		t.Fatalf("MustInclude len = %d, want 1", len(s.ExpectedContext.MustInclude))
	}
	if s.ExpectedContext.MustInclude[0].Name != "readme" {
		t.Fatalf("MustInclude[0].Name = %q, want readme", s.ExpectedContext.MustInclude[0].Name)
	}
	if len(s.ExpectedFailures) != 1 {
		t.Fatalf("ExpectedFailures len = %d, want 1", len(s.ExpectedFailures))
	}
	if s.ExpectedFailures[0].Kind != "auth" {
		t.Fatalf("ExpectedFailures[0].Kind = %q, want auth", s.ExpectedFailures[0].Kind)
	}
	if s.ExpectedFailures[0].MessageContains != "token" {
		t.Fatalf("MessageContains = %q, want token", s.ExpectedFailures[0].MessageContains)
	}
}

func TestParseInvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":::invalid"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParseMissingName(t *testing.T) {
	yaml := `
inbound_events:
  - type: message
    data: {}
harness_events:
  - kind: final
    data: {}
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParseMissingInboundEvents(t *testing.T) {
	yaml := `
name: test
harness_events:
  - kind: final
    data: {}
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing inbound_events")
	}
}

func TestParseMissingHarnessEvents(t *testing.T) {
	yaml := `
name: test
inbound_events:
  - type: message
    data: {}
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing harness_events")
	}
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.yaml")
	if err := os.WriteFile(path, []byte(validYAML), 0644); err != nil {
		t.Fatal(err)
	}

	s, err := LoadFile(path)
	if err != nil {
		t.Fatalf("LoadFile: %v", err)
	}
	if s.Name != "hello-world" {
		t.Fatalf("Name = %q, want hello-world", s.Name)
	}
}

func TestLoadFileMissing(t *testing.T) {
	_, err := LoadFile("/nonexistent/scenario.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadFileInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::bad"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML file")
	}
}
