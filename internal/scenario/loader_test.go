package scenario

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestParse_Valid(t *testing.T) {
	yaml := `
name: hello-test
description: basic test
inbound_events:
  - type: message
    content: hello
    channel_id: ch1
    author_id: user1
harness_events:
  - kind: final
    content: world
    phase: respond
    tool: echo
    message: done
    fault: ""
expected_transport_ops:
  - kind: post
`
	sc, err := Parse([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Name != "hello-test" {
		t.Errorf("Name = %q, want %q", sc.Name, "hello-test")
	}
	if sc.Description != "basic test" {
		t.Errorf("Description = %q, want %q", sc.Description, "basic test")
	}
	if len(sc.InboundEvents) != 1 {
		t.Fatalf("InboundEvents len = %d, want 1", len(sc.InboundEvents))
	}
	ie := sc.InboundEvents[0]
	if ie.Type != plugin.InboundMessage {
		t.Errorf("InboundEvent.Type = %q, want %q", ie.Type, plugin.InboundMessage)
	}
	if ie.Content != "hello" {
		t.Errorf("InboundEvent.Content = %q, want %q", ie.Content, "hello")
	}
	if ie.ChannelID != "ch1" {
		t.Errorf("InboundEvent.ChannelID = %q, want %q", ie.ChannelID, "ch1")
	}
	if ie.AuthorID != "user1" {
		t.Errorf("InboundEvent.AuthorID = %q, want %q", ie.AuthorID, "user1")
	}
	if len(sc.HarnessEvents) != 1 {
		t.Fatalf("HarnessEvents len = %d, want 1", len(sc.HarnessEvents))
	}
	he := sc.HarnessEvents[0]
	if he.Kind != plugin.RunEventFinal {
		t.Errorf("HarnessEvent.Kind = %q, want %q", he.Kind, plugin.RunEventFinal)
	}
	if he.Content != "world" {
		t.Errorf("HarnessEvent.Content = %q, want %q", he.Content, "world")
	}
	if he.Phase != "respond" {
		t.Errorf("HarnessEvent.Phase = %q, want %q", he.Phase, "respond")
	}
	if he.Tool != "echo" {
		t.Errorf("HarnessEvent.Tool = %q, want %q", he.Tool, "echo")
	}
	if he.Message != "done" {
		t.Errorf("HarnessEvent.Message = %q, want %q", he.Message, "done")
	}
	if he.Fault != "" {
		t.Errorf("HarnessEvent.Fault = %q, want empty", he.Fault)
	}

	if len(sc.ExpectedOps) != 1 {
		t.Fatalf("ExpectedOps len = %d, want 1", len(sc.ExpectedOps))
	}
	if sc.ExpectedOps[0].Kind != plugin.OutboundPost {
		t.Errorf("ExpectedOp.Kind = %q, want %q", sc.ExpectedOps[0].Kind, plugin.OutboundPost)
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := Parse([]byte(":::bad yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestParse_MissingName(t *testing.T) {
	yaml := `
inbound_events:
  - type: message
    content: hi
harness_events:
  - kind: final
    content: bye
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestParse_NoInboundEvents(t *testing.T) {
	yaml := `
name: test
harness_events:
  - kind: final
    content: bye
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for no inbound events")
	}
}

func TestParse_NoHarnessEvents(t *testing.T) {
	yaml := `
name: test
inbound_events:
  - type: message
    content: hi
`
	_, err := Parse([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for no harness events")
	}
}

func TestLoadFile_Valid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scenario.yaml")
	yaml := `
name: file-test
inbound_events:
  - type: message
    content: hi
harness_events:
  - kind: final
    content: bye
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	sc, err := LoadFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sc.Name != "file-test" {
		t.Errorf("Name = %q, want %q", sc.Name, "file-test")
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/scenario.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}
