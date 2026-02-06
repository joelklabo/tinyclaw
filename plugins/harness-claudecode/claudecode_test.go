package harnessclaudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Verify Harness implements plugin.Harness.
var _ plugin.Harness = (*Harness)(nil)

// --- mock runner ---

type mockRunner struct {
	output []byte
	err    error
}

func (m *mockRunner) Run(_ context.Context, _ string) ([]byte, error) {
	return m.output, m.err
}

// --- fixture helpers ---

func fixtureStream(events ...streamEvent) []byte {
	var lines []string
	for _, ev := range events {
		b, _ := json.Marshal(ev)
		lines = append(lines, string(b))
	}
	return []byte(joinLines(lines))
}

func joinLines(lines []string) string {
	result := ""
	for i, l := range lines {
		if i > 0 {
			result += "\n"
		}
		result += l
	}
	return result
}

// --- tests ---

func TestNewValid(t *testing.T) {
	h, err := New(&mockRunner{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestNewNilRunner(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatal("expected error for nil runner")
	}
}

func TestStartEmitsStatus(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "status", Content: json.RawMessage(`"thinking"`)},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []plugin.RunEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != "status" {
		t.Fatalf("kind = %q, want status", events[0].Kind)
	}
	if events[0].Data["phase"] != "thinking" {
		t.Fatalf("phase = %v, want thinking", events[0].Data["phase"])
	}
}

func TestStartEmitsDelta(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "content", Content: json.RawMessage(`"hello world"`)},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ev := <-ch
	if ev.Kind != "delta" {
		t.Fatalf("kind = %q, want delta", ev.Kind)
	}
	if ev.Data["content"] != "hello world" {
		t.Fatalf("content = %v, want hello world", ev.Data["content"])
	}
}

func TestStartEmitsDeltaType(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "delta", Content: json.RawMessage(`"chunk"`)},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ev := <-ch
	if ev.Kind != "delta" {
		t.Fatalf("kind = %q, want delta", ev.Kind)
	}
}

func TestStartEmitsToolUse(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "tool_use", Tool: "bash", Message: "running ls"},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ev := <-ch
	if ev.Kind != "tool" {
		t.Fatalf("kind = %q, want tool", ev.Kind)
	}
	if ev.Data["tool"] != "bash" {
		t.Fatalf("tool = %v, want bash", ev.Data["tool"])
	}
}

func TestStartEmitsResult(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "result", Content: json.RawMessage(`"done"`)},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ev := <-ch
	if ev.Kind != "final" {
		t.Fatalf("kind = %q, want final", ev.Kind)
	}
	if ev.Data["content"] != "done" {
		t.Fatalf("content = %v, want done", ev.Data["content"])
	}
}

func TestStartEmitsErrorFault(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "error", Error: "something went wrong"},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ev := <-ch
	if ev.Kind != "fault" {
		t.Fatalf("kind = %q, want fault", ev.Kind)
	}
	if ev.Data["kind"] != "error" {
		t.Fatalf("fault kind = %v, want error", ev.Data["kind"])
	}
}

func TestStartDetectsAuthFailure(t *testing.T) {
	patterns := []string{
		"Invalid API Key provided",
		"Authentication Failed for user",
		"Quota exceeded for this month",
		"Rate limit reached, try again",
		"Billing issue on account",
		"Unauthorized access attempt",
		"Forbidden: insufficient permissions",
	}
	for _, pat := range patterns {
		data := fixtureStream(streamEvent{Type: "error", Error: pat})
		h, _ := New(&mockRunner{output: data})

		ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
		if err != nil {
			t.Fatalf("Start with %q: %v", pat, err)
		}
		ev := <-ch
		if ev.Kind != "fault" {
			t.Fatalf("pattern %q: kind = %q, want fault", pat, ev.Kind)
		}
		if ev.Data["kind"] != "auth" {
			t.Fatalf("pattern %q: fault kind = %v, want auth", pat, ev.Data["kind"])
		}
	}
}

func TestStartUnknownType(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "unknown_event", Content: json.RawMessage(`{"foo":"bar"}`)},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	ev := <-ch
	if ev.Kind != "status" {
		t.Fatalf("kind = %q, want status for unknown type", ev.Kind)
	}
	if ev.Data["type"] != "unknown_event" {
		t.Fatalf("type = %v, want unknown_event", ev.Data["type"])
	}
}

func TestStartFullStream(t *testing.T) {
	data := fixtureStream(
		streamEvent{Type: "status", Content: json.RawMessage(`"thinking"`)},
		streamEvent{Type: "content", Content: json.RawMessage(`"Hello "`)},
		streamEvent{Type: "content", Content: json.RawMessage(`"world"`)},
		streamEvent{Type: "tool_use", Tool: "bash", Message: "ls"},
		streamEvent{Type: "result", Content: json.RawMessage(`"all done"`)},
	)
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []plugin.RunEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	if events[0].Kind != "status" {
		t.Errorf("event 0: kind = %q, want status", events[0].Kind)
	}
	if events[1].Kind != "delta" {
		t.Errorf("event 1: kind = %q, want delta", events[1].Kind)
	}
	if events[3].Kind != "tool" {
		t.Errorf("event 3: kind = %q, want tool", events[3].Kind)
	}
	if events[4].Kind != "final" {
		t.Errorf("event 4: kind = %q, want final", events[4].Kind)
	}
}

func TestStartRunnerError(t *testing.T) {
	h, _ := New(&mockRunner{err: fmt.Errorf("exec failed")})

	_, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err == nil {
		t.Fatal("expected error from runner failure")
	}
}

func TestStartEmptyStream(t *testing.T) {
	h, _ := New(&mockRunner{output: []byte("")})

	_, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err == nil {
		t.Fatal("expected error for empty stream")
	}
}

func TestStartInvalidJSON(t *testing.T) {
	h, _ := New(&mockRunner{output: []byte("not json\n")})

	_, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestStartBlankLines(t *testing.T) {
	// Stream with blank lines between valid JSON lines.
	line1, _ := json.Marshal(streamEvent{Type: "status", Content: json.RawMessage(`"thinking"`)})
	line2, _ := json.Marshal(streamEvent{Type: "result", Content: json.RawMessage(`"ok"`)})
	data := []byte(string(line1) + "\n\n\n" + string(line2))
	h, _ := New(&mockRunner{output: data})

	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	var events []plugin.RunEvent
	for ev := range ch {
		events = append(events, ev)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestStartUsesEventContent(t *testing.T) {
	// When RunRequest.Event.Data["content"] is set, it should be used as prompt.
	// We verify indirectly that the runner receives the call without error.
	data := fixtureStream(
		streamEvent{Type: "result", Content: json.RawMessage(`"ok"`)},
	)
	h, _ := New(&mockRunner{output: data})

	req := plugin.RunRequest{
		Scenario: "default",
		Event: plugin.InboundEvent{
			Type: "message",
			Data: map[string]any{"content": "user question"},
		},
	}
	ch, err := h.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	for range ch {
	}
}

func TestStartContextCancellation(t *testing.T) {
	// Create a stream with many events.
	var events []streamEvent
	for i := 0; i < 100; i++ {
		events = append(events, streamEvent{Type: "content", Content: json.RawMessage(`"chunk"`)})
	}
	data := fixtureStream(events...)
	h, _ := New(&mockRunner{output: data})

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := h.Start(ctx, plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatal(err)
	}

	// Read one then cancel.
	<-ch
	cancel()

	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("channel not closed after cancel")
	}
}

func TestClose(t *testing.T) {
	h, _ := New(&mockRunner{})
	if err := h.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
}

func TestJsonStringEmpty(t *testing.T) {
	result := jsonString(nil)
	if result != "" {
		t.Fatalf("expected empty string, got %q", result)
	}
}

func TestJsonStringNonString(t *testing.T) {
	result := jsonString(json.RawMessage(`42`))
	if result != "42" {
		t.Fatalf("expected 42, got %q", result)
	}
}

func TestIsAuthFailureNonMatch(t *testing.T) {
	if isAuthFailure("connection timeout") {
		t.Fatal("expected false for non-auth error")
	}
}

func TestParseStreamOnlyBlanks(t *testing.T) {
	// After TrimSpace, "\n\n\n" becomes "" which hits the "empty stream data" path.
	_, err := parseStream([]byte("\n\n\n"))
	if err == nil {
		t.Fatal("expected error for stream with only blank lines")
	}
}

func TestLiveClaude(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("requires LIVE=1")
	}
	// A live integration test would shell out to the actual claude CLI.
	t.Log("live Claude Code test not yet implemented")
}
