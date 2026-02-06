// Package harnessreplay implements a Harness plugin that replays a scripted event stream.
package harnessreplay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Replay is a Harness that emits a pre-configured sequence of RunEvents.
type Replay struct {
	events []plugin.RunEvent
}

// New creates a Replay harness from a slice of events.
func New(events []plugin.RunEvent) *Replay {
	return &Replay{events: events}
}

// NewFromJSON creates a Replay harness from JSON-encoded event data.
func NewFromJSON(data []byte) (*Replay, error) {
	var events []plugin.RunEvent
	if err := json.Unmarshal(data, &events); err != nil {
		return nil, fmt.Errorf("parsing replay events: %w", err)
	}
	return New(events), nil
}

// Start begins replaying events on the returned channel.
func (r *Replay) Start(ctx context.Context, _ plugin.RunRequest) (<-chan plugin.RunEvent, error) {
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

// Close is a no-op for the replay harness.
func (r *Replay) Close() error {
	return nil
}
