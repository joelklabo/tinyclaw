// Package transportfake implements a fake Transport for testing.
package transportfake

import (
	"context"
	"fmt"
	"sync"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Transport is a fake transport that emits scripted inbound events
// and records all outbound operations.
type Transport struct {
	mu         sync.Mutex
	events     []plugin.InboundEvent
	ops        []plugin.OutboundOp
	closed     bool
	subscribed bool
}

// New creates a new fake Transport with the given scripted inbound events.
func New(events []plugin.InboundEvent) *Transport {
	copied := make([]plugin.InboundEvent, len(events))
	copy(copied, events)
	return &Transport{events: copied}
}

// Subscribe returns a channel that emits the scripted events then closes.
func (t *Transport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport-fake: already closed")
	}
	if t.subscribed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport-fake: already subscribed")
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

// Post records an outbound operation.
func (t *Transport) Post(ctx context.Context, op plugin.OutboundOp) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport-fake: already closed")
	}
	t.ops = append(t.ops, op)
	return nil
}

// Close marks the transport as closed.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport-fake: already closed")
	}
	t.closed = true
	return nil
}

// Ops returns a copy of all recorded outbound operations.
func (t *Transport) Ops() []plugin.OutboundOp {
	t.mu.Lock()
	defer t.mu.Unlock()
	copied := make([]plugin.OutboundOp, len(t.ops))
	copy(copied, t.ops)
	return copied
}
