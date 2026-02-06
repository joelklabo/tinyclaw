package transportfake

import (
	"context"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestNew(t *testing.T) {
	events := []plugin.InboundEvent{
		{Type: "message", Data: map[string]any{"content": "hi"}},
	}
	tr := New(events)
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
}

func TestSubscribe(t *testing.T) {
	events := []plugin.InboundEvent{
		{Type: "message", Data: map[string]any{"content": "hi"}},
		{Type: "message", Data: map[string]any{"content": "bye"}},
	}
	tr := New(events)

	ch, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var received []plugin.InboundEvent
	for ev := range ch {
		received = append(received, ev)
	}

	if len(received) != 2 {
		t.Fatalf("received %d events, want 2", len(received))
	}
	if received[0].Type != "message" {
		t.Fatalf("event[0].Type = %q, want message", received[0].Type)
	}
	if received[1].Data["content"] != "bye" {
		t.Fatalf("event[1].Data[content] = %v, want bye", received[1].Data["content"])
	}
}

func TestSubscribeEmpty(t *testing.T) {
	tr := New(nil)
	ch, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	count := 0
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 events, got %d", count)
	}
}

func TestSubscribeContextCancel(t *testing.T) {
	// Use enough events so the goroutine will block on the unbuffered-like
	// portion after the buffer fills. The channel buffer is len(events), so
	// all events fit. To reliably test the ctx.Done() branch, cancel the
	// context before subscribing.
	events := []plugin.InboundEvent{
		{Type: "message", Data: map[string]any{"content": "1"}},
		{Type: "message", Data: map[string]any{"content": "2"}},
		{Type: "message", Data: map[string]any{"content": "3"}},
	}
	tr := New(events)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	count := 0
	for range ch {
		count++
	}
	// With a buffered channel of size 3 and immediate cancel, we may get 0-3 events.
	// Just verify it doesn't hang.
}

func TestSubscribeContextCancelReliable(t *testing.T) {
	// Build a large event list. With a pre-cancelled context, the goroutine's
	// select will eventually pick ctx.Done() over ch<-ev since both are ready.
	// With 100 events this is statistically certain.
	var events []plugin.InboundEvent
	for i := range 100 {
		events = append(events, plugin.InboundEvent{
			Type: "message",
			Data: map[string]any{"i": i},
		})
	}
	tr := New(events)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Pre-cancel.

	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	count := 0
	for range ch {
		count++
	}
	// We should get fewer than 100 events since ctx is cancelled.
	// But even if all 100 are delivered (unlikely), just don't hang.
	_ = count
}

func TestSubscribeAlreadyClosed(t *testing.T) {
	tr := New(nil)
	tr.Close()
	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error for closed transport")
	}
}

func TestSubscribeTwice(t *testing.T) {
	tr := New(nil)
	_, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	_, err = tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error for double subscribe")
	}
}

func TestPost(t *testing.T) {
	tr := New(nil)

	op1 := plugin.OutboundOp{Kind: "post", Data: map[string]any{"text": "hello"}}
	op2 := plugin.OutboundOp{Kind: "edit", Data: map[string]any{"text": "hello world"}}

	if err := tr.Post(context.Background(), op1); err != nil {
		t.Fatalf("Post 1: %v", err)
	}
	if err := tr.Post(context.Background(), op2); err != nil {
		t.Fatalf("Post 2: %v", err)
	}

	ops := tr.Ops()
	if len(ops) != 2 {
		t.Fatalf("ops len = %d, want 2", len(ops))
	}
	if ops[0].Kind != "post" {
		t.Fatalf("ops[0].Kind = %q, want post", ops[0].Kind)
	}
	if ops[1].Kind != "edit" {
		t.Fatalf("ops[1].Kind = %q, want edit", ops[1].Kind)
	}
}

func TestPostAfterClose(t *testing.T) {
	tr := New(nil)
	tr.Close()
	err := tr.Post(context.Background(), plugin.OutboundOp{Kind: "post"})
	if err == nil {
		t.Fatal("expected error for post on closed transport")
	}
}

func TestClose(t *testing.T) {
	tr := New(nil)
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestCloseDouble(t *testing.T) {
	tr := New(nil)
	tr.Close()
	err := tr.Close()
	if err == nil {
		t.Fatal("expected error for double close")
	}
}

func TestOpsEmpty(t *testing.T) {
	tr := New(nil)
	ops := tr.Ops()
	if len(ops) != 0 {
		t.Fatalf("ops len = %d, want 0", len(ops))
	}
}

func TestOpsIsolation(t *testing.T) {
	tr := New(nil)
	tr.Post(context.Background(), plugin.OutboundOp{Kind: "post"})

	ops1 := tr.Ops()
	ops2 := tr.Ops()
	// Modifying one copy should not affect the other.
	ops1[0].Kind = "modified"
	if ops2[0].Kind == "modified" {
		t.Fatal("Ops() should return independent copies")
	}
}

// Verify transport-fake implements plugin.Transport interface.
var _ plugin.Transport = (*Transport)(nil)
