package scenario

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/klabo/tinyclaw/internal/bundle"
	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestRunScenario_HappyPath(t *testing.T) {
	sc := &Scenario{
		Name: "happy",
		InboundEvents: []InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch1", AuthorID: "u1"},
		},
		HarnessEvents: []plugin.RunEvent{
			{Kind: plugin.RunEventFinal, Content: "world"},
		},
		ExpectedOps: []ExpectedOp{
			{Kind: plugin.OutboundPost},
		},
	}

	baseDir := t.TempDir()
	r := NewRunner()
	dir, ops, err := r.RunScenario(baseDir, sc)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty bundle dir")
	}
	// Verify bundle directory was created.
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bundle dir does not exist: %v", err)
	}
	// Verify ops match expectations.
	if err := AssertOps(ops, sc.ExpectedOps); err != nil {
		t.Fatalf("AssertOps: %v", err)
	}
}

func TestRunScenario_BundleCreationError(t *testing.T) {
	sc := &Scenario{
		Name: "bundle-err",
		InboundEvents: []InboundEvent{
			{Type: plugin.InboundMessage, Content: "hi"},
		},
		HarnessEvents: []plugin.RunEvent{
			{Kind: plugin.RunEventFinal, Content: "bye"},
		},
	}

	r := &Runner{
		newBundle: func(baseDir, id, scenario string, opts ...bundle.Option) (*bundle.Writer, error) {
			return nil, fmt.Errorf("injected bundle error")
		},
		newTransport: newScriptedTransport,
		newHarness:   newScriptedHarness,
	}
	_, _, err := r.RunScenario(t.TempDir(), sc)
	if err == nil {
		t.Fatal("expected error for bundle creation failure")
	}
}

func TestRunScenario_MultipleEventTypes(t *testing.T) {
	sc := &Scenario{
		Name: "multi-events",
		InboundEvents: []InboundEvent{
			{Type: plugin.InboundMessage, Content: "hi"},
		},
		HarnessEvents: []plugin.RunEvent{
			{Kind: plugin.RunEventStatus, Phase: "start"},
			{Kind: plugin.RunEventDelta, Content: "partial"},
			{Kind: plugin.RunEventFault, Message: "oops"},
			{Kind: plugin.RunEventFinal, Content: "done"},
		},
		ExpectedOps: []ExpectedOp{
			{Kind: plugin.OutboundTyping},
			{Kind: plugin.OutboundPost},
			{Kind: plugin.OutboundPost},
		},
	}

	baseDir := t.TempDir()
	r := NewRunner()
	dir, ops, err := r.RunScenario(baseDir, sc)
	if err != nil {
		t.Fatalf("RunScenario: %v", err)
	}
	if dir == "" {
		t.Fatal("expected non-empty bundle dir")
	}
	if err := AssertOps(ops, sc.ExpectedOps); err != nil {
		t.Fatalf("AssertOps: %v", err)
	}
}

func TestRunScenario_BundleDirBroken(t *testing.T) {
	dir := t.TempDir()
	fakePath := filepath.Join(dir, "not-a-dir")
	if err := os.WriteFile(fakePath, []byte("block"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	sc := &Scenario{
		Name: "broken-dir",
		InboundEvents: []InboundEvent{
			{Type: plugin.InboundMessage, Content: "hi"},
		},
		HarnessEvents: []plugin.RunEvent{
			{Kind: plugin.RunEventFinal, Content: "bye"},
		},
	}

	r := NewRunner()
	_, _, err := r.RunScenario(fakePath, sc)
	if err == nil {
		t.Fatal("expected error when base dir is a file")
	}
}
