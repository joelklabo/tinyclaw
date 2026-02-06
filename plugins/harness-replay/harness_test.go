package harnessreplay

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestReplayEmitsAllEvents(t *testing.T) {
	events := []plugin.RunEvent{
		{Kind: "status", Data: map[string]any{"phase": "thinking"}},
		{Kind: "delta", Data: map[string]any{"content": "hello"}},
		{Kind: "tool", Data: map[string]any{"toolName": "ui.post"}},
		{Kind: "final", Data: map[string]any{"response": "done"}},
	}
	h := New(events)
	ctx := context.Background()
	ch, err := h.Start(ctx, plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	var got []plugin.RunEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != len(events) {
		t.Fatalf("expected %d events, got %d", len(events), len(got))
	}
	for i, ev := range got {
		if ev.Kind != events[i].Kind {
			t.Errorf("event %d: kind = %q, want %q", i, ev.Kind, events[i].Kind)
		}
	}
}

func TestReplayEmptyEvents(t *testing.T) {
	h := New(nil)
	ch, err := h.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for range ch {
		count++
	}
	if count != 0 {
		t.Fatalf("expected 0 events, got %d", count)
	}
}

func TestReplayClose(t *testing.T) {
	h := New(nil)
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestReplayContextCancellation(t *testing.T) {
	events := make([]plugin.RunEvent, 100)
	for i := range events {
		events[i] = plugin.RunEvent{Kind: "delta", Data: map[string]any{"i": i}}
	}
	h := New(events)
	ctx, cancel := context.WithCancel(context.Background())
	ch, err := h.Start(ctx, plugin.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	// Read one event then cancel
	<-ch
	cancel()
	// Drain remaining events; should not block forever
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after context cancellation")
	}
}

func TestNewFromJSON(t *testing.T) {
	events := []plugin.RunEvent{
		{Kind: "status", Data: map[string]any{"phase": "thinking"}},
		{Kind: "final", Data: map[string]any{"response": "ok"}},
	}
	data, err := json.Marshal(events)
	if err != nil {
		t.Fatal(err)
	}
	h, err := NewFromJSON(data)
	if err != nil {
		t.Fatalf("NewFromJSON: %v", err)
	}
	ch, err := h.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var got []plugin.RunEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
}

func TestNewFromJSONInvalid(t *testing.T) {
	_, err := NewFromJSON([]byte("not json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestReplayFaultEvent(t *testing.T) {
	events := []plugin.RunEvent{
		{Kind: "status", Data: map[string]any{"phase": "thinking"}},
		{Kind: "fault", Data: map[string]any{"kind": "transient", "message": "timeout"}},
		{Kind: "final", Data: map[string]any{"status": "failed"}},
	}
	h := New(events)
	ch, err := h.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var got []plugin.RunEvent
	for ev := range ch {
		got = append(got, ev)
	}
	if len(got) != 3 {
		t.Fatalf("expected 3 events, got %d", len(got))
	}
	if got[1].Kind != "fault" {
		t.Fatalf("expected fault event, got %q", got[1].Kind)
	}
}

func TestReplayMultipleStarts(t *testing.T) {
	events := []plugin.RunEvent{
		{Kind: "final", Data: map[string]any{"response": "ok"}},
	}
	h := New(events)
	// First start
	ch1, err := h.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	for range ch1 {
	}
	// Second start should also work
	ch2, err := h.Start(context.Background(), plugin.RunRequest{})
	if err != nil {
		t.Fatal(err)
	}
	var count int
	for range ch2 {
		count++
	}
	if count != 1 {
		t.Fatalf("expected 1 event on second start, got %d", count)
	}
}
