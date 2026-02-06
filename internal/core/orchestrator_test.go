package core

import (
	"context"
	"errors"
	"testing"

	"github.com/klabo/tinyclaw/internal/bundles"
	"github.com/klabo/tinyclaw/internal/plugin"
)

// --- test doubles ---

type stubRouter struct {
	profile string
	err     error
}

func (r *stubRouter) Route(_ plugin.InboundEvent) (string, error) {
	return r.profile, r.err
}

type stubContextBuilder struct {
	items []plugin.ContextItem
	err   error
}

func (b *stubContextBuilder) Build(_ context.Context, _ plugin.InboundEvent) ([]plugin.ContextItem, error) {
	return b.items, b.err
}

type stubTransport struct {
	events []plugin.InboundEvent
	ops    []plugin.OutboundOp
	subErr error
	postFn func(plugin.OutboundOp) error
}

func (t *stubTransport) Subscribe(_ context.Context) (<-chan plugin.InboundEvent, error) {
	if t.subErr != nil {
		return nil, t.subErr
	}
	ch := make(chan plugin.InboundEvent, len(t.events))
	for _, e := range t.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (t *stubTransport) Post(_ context.Context, op plugin.OutboundOp) error {
	t.ops = append(t.ops, op)
	if t.postFn != nil {
		return t.postFn(op)
	}
	return nil
}

func (t *stubTransport) Close() error { return nil }

type stubHarness struct {
	events []plugin.RunEvent
	err    error
}

func (h *stubHarness) Start(_ context.Context, _ plugin.RunRequest) (<-chan plugin.RunEvent, error) {
	if h.err != nil {
		return nil, h.err
	}
	ch := make(chan plugin.RunEvent, len(h.events))
	for _, e := range h.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (h *stubHarness) Close() error { return nil }

func makeBundle(t *testing.T) *bundles.Writer {
	t.Helper()
	w, err := bundles.NewWriter(t.TempDir(), "test", "test-scenario")
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func makeEvent() plugin.InboundEvent {
	return plugin.InboundEvent{
		Type: "message",
		Data: map[string]any{"text": "hello"},
	}
}

// --- tests ---

func TestOrchestratorHappyPath(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: "status", Data: map[string]any{"status": "thinking"}},
		{Kind: "delta", Data: map[string]any{"text": "hi"}},
		{Kind: "tool", Data: map[string]any{"name": "bash"}},
		{Kind: "final", Data: map[string]any{"text": "done"}},
	}}
	r := &stubRouter{profile: "default"}
	cb := &stubContextBuilder{}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	if err := o.Run(context.Background(), makeEvent()); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	// transport should have received: typing, edit, final post (tool is not sent)
	if len(tr.ops) != 3 {
		t.Fatalf("transport ops = %d, want 3", len(tr.ops))
	}
	if tr.ops[0].Kind != "typing" {
		t.Fatalf("op[0].Kind = %q, want typing", tr.ops[0].Kind)
	}
	if tr.ops[1].Kind != "edit" {
		t.Fatalf("op[1].Kind = %q, want edit", tr.ops[1].Kind)
	}
	if tr.ops[2].Kind != "post" {
		t.Fatalf("op[2].Kind = %q, want post", tr.ops[2].Kind)
	}
}

func TestOrchestratorRouteError(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{}
	r := &stubRouter{err: errors.New("no match")}
	cb := &stubContextBuilder{}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	err := o.Run(context.Background(), makeEvent())
	if err == nil {
		t.Fatal("expected route error")
	}
}

func TestOrchestratorContextBuildError(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{}
	r := &stubRouter{profile: "default"}
	cb := &stubContextBuilder{err: errors.New("no files")}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	err := o.Run(context.Background(), makeEvent())
	if err == nil {
		t.Fatal("expected context build error")
	}
}

func TestOrchestratorHarnessStartError(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{err: errors.New("harness fail")}
	r := &stubRouter{profile: "default"}
	cb := &stubContextBuilder{}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	err := o.Run(context.Background(), makeEvent())
	if err == nil {
		t.Fatal("expected harness start error")
	}
}

func TestOrchestratorTransportPostError(t *testing.T) {
	tr := &stubTransport{
		postFn: func(_ plugin.OutboundOp) error {
			return errors.New("post fail")
		},
	}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: "delta", Data: map[string]any{"text": "hi"}},
	}}
	r := &stubRouter{profile: "default"}
	cb := &stubContextBuilder{}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	err := o.Run(context.Background(), makeEvent())
	if err == nil {
		t.Fatal("expected transport post error")
	}
}

func TestOrchestratorFaultEvent(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: "fault", Data: map[string]any{"kind": "auth", "message": "denied"}},
	}}
	r := &stubRouter{profile: "default"}
	cb := &stubContextBuilder{}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	if err := o.Run(context.Background(), makeEvent()); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	if len(tr.ops) != 1 {
		t.Fatalf("transport ops = %d, want 1", len(tr.ops))
	}
	if tr.ops[0].Kind != "post" {
		t.Fatalf("op.Kind = %q, want post", tr.ops[0].Kind)
	}
}

func TestAdvancePanicsOnInvalidTransition(t *testing.T) {
	m := NewMachine() // starts at Ingress
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on invalid transition")
		}
	}()
	advance(m, Completed) // invalid: Ingress -> Completed
}

func TestOrchestratorUnknownEventKind(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: "custom", Data: map[string]any{"x": 1}},
	}}
	r := &stubRouter{profile: "default"}
	cb := &stubContextBuilder{}
	b := makeBundle(t)

	o := NewOrchestrator(tr, h, r, cb, b)
	if err := o.Run(context.Background(), makeEvent()); err != nil {
		t.Fatalf("Run error: %v", err)
	}
	// Unknown event kinds should not produce transport ops.
	if len(tr.ops) != 0 {
		t.Fatalf("transport ops = %d, want 0", len(tr.ops))
	}
}
