package scenario

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// newScriptedHarnessFromJSON is a test helper that creates a scriptedHarness from JSON.
func newScriptedHarnessFromJSON(data []byte) (*scriptedHarness, error) {
	var events []plugin.RunEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parsing replay events: %w", err)
	}
	return newScriptedHarness(events), nil
}

// --- scriptedTransport tests ---

func TestScriptedTransport_Subscribe(t *testing.T) {
	events := []plugin.InboundEvent{
		{Type: plugin.InboundMessage, Content: "hello"},
		{Type: plugin.InboundMessage, Content: "world"},
	}
	tr := newScriptedTransport(events)
	ch, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	var got []plugin.InboundEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Content != "hello" {
		t.Errorf("event[0].Content = %q, want %q", got[0].Content, "hello")
	}
	if got[1].Content != "world" {
		t.Errorf("event[1].Content = %q, want %q", got[1].Content, "world")
	}
}

func TestScriptedTransport_Post(t *testing.T) {
	tr := newScriptedTransport(nil)
	ctx := context.Background()
	op1 := plugin.OutboundOp{Kind: plugin.OutboundPost, Content: "msg1"}
	op2 := plugin.OutboundOp{Kind: plugin.OutboundEdit, Content: "msg2"}
	if err := tr.Post(ctx, op1); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if err := tr.Post(ctx, op2); err != nil {
		t.Fatalf("Post: %v", err)
	}
	ops := tr.Ops()
	if len(ops) != 2 {
		t.Fatalf("Ops len = %d, want 2", len(ops))
	}
	if ops[0].Kind != plugin.OutboundPost {
		t.Errorf("ops[0].Kind = %q, want %q", ops[0].Kind, plugin.OutboundPost)
	}
	if ops[1].Kind != plugin.OutboundEdit {
		t.Errorf("ops[1].Kind = %q, want %q", ops[1].Kind, plugin.OutboundEdit)
	}
}

func TestScriptedTransport_OpsCopied(t *testing.T) {
	tr := newScriptedTransport(nil)
	_ = tr.Post(context.Background(), plugin.OutboundOp{Kind: plugin.OutboundPost})
	ops1 := tr.Ops()
	ops1[0].Kind = "mutated"
	ops2 := tr.Ops()
	if ops2[0].Kind != plugin.OutboundPost {
		t.Errorf("Ops returned mutable slice; got %q after mutation, want %q", ops2[0].Kind, plugin.OutboundPost)
	}
}

func TestScriptedTransport_Close(t *testing.T) {
	tr := newScriptedTransport(nil)
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestScriptedTransport_DoubleClose(t *testing.T) {
	tr := newScriptedTransport(nil)
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := tr.Close(); err == nil {
		t.Fatal("expected error on double close")
	}
}

func TestScriptedTransport_PostAfterClose(t *testing.T) {
	tr := newScriptedTransport(nil)
	_ = tr.Close()
	if err := tr.Post(context.Background(), plugin.OutboundOp{Kind: plugin.OutboundPost}); err == nil {
		t.Fatal("expected error posting after close")
	}
}

func TestScriptedTransport_SubscribeAfterClose(t *testing.T) {
	tr := newScriptedTransport(nil)
	_ = tr.Close()
	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error subscribing after close")
	}
}

func TestScriptedTransport_SubscribeTwice(t *testing.T) {
	tr := newScriptedTransport([]plugin.InboundEvent{{Type: plugin.InboundMessage, Content: "hi"}})
	_, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("first Subscribe: %v", err)
	}
	_, err = tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error on second subscribe")
	}
}

func TestScriptedTransport_SubscribeContextCancel(t *testing.T) {
	events := []plugin.InboundEvent{
		{Type: plugin.InboundMessage, Content: "a"},
		{Type: plugin.InboundMessage, Content: "b"},
	}
	tr := newScriptedTransport(events)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	cancel()
	// Drain: channel must close (not hang).
	for range ch {
	}
}

func TestNewScriptedTransport_NilEvents(t *testing.T) {
	tr := newScriptedTransport(nil)
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
	ops := tr.Ops()
	if len(ops) != 0 {
		t.Errorf("Ops len = %d, want 0", len(ops))
	}
}

func TestNewScriptedTransport_EmptyEvents(t *testing.T) {
	tr := newScriptedTransport([]plugin.InboundEvent{})
	ch, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("got %d events, want 0", count)
	}
}

// --- scriptedHarness tests ---

func TestScriptedHarness_EmitEvents(t *testing.T) {
	events := []plugin.RunEvent{
		{Kind: plugin.RunEventStatus, Phase: "start"},
		{Kind: plugin.RunEventFinal, Content: "done"},
	}
	hr := newScriptedHarness(events)
	ch, err := hr.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var got []plugin.RunEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	if got[0].Kind != plugin.RunEventStatus {
		t.Errorf("event[0].Kind = %q, want %q", got[0].Kind, plugin.RunEventStatus)
	}
	if got[1].Kind != plugin.RunEventFinal {
		t.Errorf("event[1].Kind = %q, want %q", got[1].Kind, plugin.RunEventFinal)
	}
}

func TestScriptedHarness_EmptyEvents(t *testing.T) {
	hr := newScriptedHarness(nil)
	ch, err := hr.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Errorf("got %d events, want 0", count)
	}
}

func TestScriptedHarness_Close(t *testing.T) {
	hr := newScriptedHarness(nil)
	if err := hr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestScriptedHarness_ContextCancel(t *testing.T) {
	events := []plugin.RunEvent{
		{Kind: plugin.RunEventStatus},
		{Kind: plugin.RunEventDelta},
		{Kind: plugin.RunEventFinal},
	}
	hr := newScriptedHarness(events)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately
	ch, err := hr.Start(ctx, plugin.RunRequest{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var count int
	for range ch {
		count++
	}
	if count >= len(events) {
		t.Errorf("expected fewer than %d events due to cancel, got %d", len(events), count)
	}
}

func TestNewScriptedHarnessFromJSON_Valid(t *testing.T) {
	data := `[{"kind":"final","content":"hello"}]`
	hr, err := newScriptedHarnessFromJSON([]byte(data))
	if err != nil {
		t.Fatalf("newScriptedHarnessFromJSON: %v", err)
	}
	ch, err := hr.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var got []plugin.RunEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1", len(got))
	}
	if got[0].Kind != plugin.RunEventFinal {
		t.Errorf("Kind = %q, want %q", got[0].Kind, plugin.RunEventFinal)
	}
	if got[0].Content != "hello" {
		t.Errorf("Content = %q, want %q", got[0].Content, "hello")
	}
}

func TestNewScriptedHarnessFromJSON_Invalid(t *testing.T) {
	_, err := newScriptedHarnessFromJSON([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
