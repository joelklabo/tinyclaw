package scenario

import (
	"context"
	"fmt"
	"sync"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// scriptedTransport is a transport that emits scripted inbound events
// and records all outbound operations.
type scriptedTransport struct {
	mu         sync.Mutex
	events     []plugin.InboundEvent
	ops        []plugin.OutboundOp
	closed     bool
	subscribed bool
}

func newScriptedTransport(events []plugin.InboundEvent) *scriptedTransport {
	copied := make([]plugin.InboundEvent, len(events))
	copy(copied, events)
	return &scriptedTransport{events: copied}
}

func (t *scriptedTransport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("scripted-transport: already closed")
	}
	if t.subscribed {
		t.mu.Unlock()
		return nil, fmt.Errorf("scripted-transport: already subscribed")
	}
	t.subscribed = true
	events := make([]plugin.InboundEvent, len(t.events))
	copy(events, t.events)
	t.mu.Unlock()

	ch := make(chan plugin.InboundEvent, len(events))
	go func() {
		defer close(ch)
		for _, ev := range events {
			select {
			case <-ctx.Done():
				return
			case ch <- ev:
			}
		}
	}()
	return ch, nil
}

func (t *scriptedTransport) Post(_ context.Context, op plugin.OutboundOp) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("scripted-transport: already closed")
	}
	t.ops = append(t.ops, op)
	return nil
}

func (t *scriptedTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("scripted-transport: already closed")
	}
	t.closed = true
	return nil
}

func (t *scriptedTransport) Ops() []plugin.OutboundOp {
	t.mu.Lock()
	defer t.mu.Unlock()
	copied := make([]plugin.OutboundOp, len(t.ops))
	copy(copied, t.ops)
	return copied
}

// scriptedHarness is a harness that emits a pre-configured sequence of RunEvents.
type scriptedHarness struct {
	events []plugin.RunEvent
}

func newScriptedHarness(events []plugin.RunEvent) *scriptedHarness {
	return &scriptedHarness{events: events}
}

func (r *scriptedHarness) Start(ctx context.Context, _ plugin.RunRequest) (<-chan plugin.RunEvent, error) {
	ch := make(chan plugin.RunEvent)
	go func() {
		defer close(ch)
		for _, ev := range r.events {
			select {
			case <-ctx.Done():
				return
			case ch <- ev:
			}
		}
	}()
	return ch, nil
}

func (r *scriptedHarness) Close() error {
	return nil
}
