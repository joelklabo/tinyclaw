package claudecode

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// --- mock types ---

// mockRunner returns a preset io.ReadCloser and error from Run.
type mockRunner struct {
	output io.ReadCloser
	err    error
}

func (m *mockRunner) Run(_ context.Context, _ string) (io.ReadCloser, error) {
	return m.output, m.err
}

// trackingCloser wraps an io.ReadCloser and records whether Close was called.
type trackingCloser struct {
	io.ReadCloser
	closed bool
}

func (tc *trackingCloser) Close() error {
	tc.closed = true
	return tc.ReadCloser.Close()
}

// trackingRunner returns a trackingCloser from Run.
type trackingRunner struct {
	closer *trackingCloser
}

func (tr *trackingRunner) Run(_ context.Context, _ string) (io.ReadCloser, error) {
	return tr.closer, nil
}

// funcRunner allows an inline function to act as a Runner.
type funcRunner struct {
	fn func(ctx context.Context, prompt string) (io.ReadCloser, error)
}

func (fr *funcRunner) Run(ctx context.Context, prompt string) (io.ReadCloser, error) {
	return fr.fn(ctx, prompt)
}

// blockingReader returns data on the first Read, then blocks until ctx is done.
type blockingReader struct {
	data    string
	readOne bool
	ctx     context.Context
}

func (br *blockingReader) Read(p []byte) (int, error) {
	if !br.readOne {
		br.readOne = true
		n := copy(p, br.data)
		return n, nil
	}
	<-br.ctx.Done()
	return 0, br.ctx.Err()
}

func (br *blockingReader) Close() error { return nil }

// --- helpers ---

func readerFromLines(lines ...string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(strings.Join(lines, "\n") + "\n"))
}

func collectEvents(t *testing.T, ch <-chan plugin.RunEvent) []plugin.RunEvent {
	t.Helper()
	var events []plugin.RunEvent
	for ev := range ch {
		events = append(events, ev)
	}
	return events
}

// --- tests ---

func TestNew_Valid(t *testing.T) {
	h, err := New(&mockRunner{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestNew_NilRunner(t *testing.T) {
	_, err := New(nil)
	if err == nil {
		t.Fatal("expected error for nil runner")
	}
	if !strings.Contains(err.Error(), "runner must not be nil") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestStart_EventTypes(t *testing.T) {
	tests := []struct {
		name    string
		line    string
		count   int
		kind    plugin.RunEventKind
		content string
		phase   string
		tool    string
		message string
		fault   string
	}{
		{
			name:  "StatusEvent",
			line:  `{"type":"status","content":"thinking"}`,
			count: 1,
			kind:  plugin.RunEventStatus,
			phase: "thinking",
		},
		{
			name:    "DeltaContentType",
			line:    `{"type":"content","content":"hello"}`,
			count:   1,
			kind:    plugin.RunEventDelta,
			content: "hello",
		},
		{
			name:    "DeltaDeltaType",
			line:    `{"type":"delta","content":"world"}`,
			count:   1,
			kind:    plugin.RunEventDelta,
			content: "world",
		},
		{
			name:    "ToolUseEvent",
			line:    `{"type":"tool_use","tool":"bash","message":"running ls"}`,
			count:   1,
			kind:    plugin.RunEventTool,
			tool:    "bash",
			message: "running ls",
		},
		{
			name:    "ResultEvent",
			line:    `{"type":"result","content":"done"}`,
			count:   1,
			kind:    plugin.RunEventFinal,
			content: "done",
		},
		{
			name:    "ErrorEvent",
			line:    `{"type":"error","error":"something broke"}`,
			count:   1,
			kind:    plugin.RunEventFault,
			fault:   "error",
			message: "something broke",
		},
		{
			name:  "UnknownType",
			line:  `{"type":"mystery","content":"data"}`,
			count: 1,
			kind:  plugin.RunEventStatus,
			phase: "mystery",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := &mockRunner{output: readerFromLines(tt.line)}
			h, _ := New(runner)
			ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			events := collectEvents(t, ch)
			if len(events) != tt.count {
				t.Fatalf("expected %d event(s), got %d", tt.count, len(events))
			}
			ev := events[0]
			if ev.Kind != tt.kind {
				t.Errorf("expected kind=%s, got %q", tt.kind, ev.Kind)
			}
			if tt.content != "" && ev.Content != tt.content {
				t.Errorf("expected content=%q, got %q", tt.content, ev.Content)
			}
			if tt.phase != "" && ev.Phase != tt.phase {
				t.Errorf("expected phase=%q, got %q", tt.phase, ev.Phase)
			}
			if tt.tool != "" && ev.Tool != tt.tool {
				t.Errorf("expected tool=%q, got %q", tt.tool, ev.Tool)
			}
			if tt.message != "" && ev.Message != tt.message {
				t.Errorf("expected message=%q, got %q", tt.message, ev.Message)
			}
			if tt.fault != "" && ev.Fault != tt.fault {
				t.Errorf("expected fault=%q, got %q", tt.fault, ev.Fault)
			}
		})
	}
}

func TestStart_AuthFailurePatterns(t *testing.T) {
	patterns := []string{
		"Invalid API Key",
		"Authentication Failed",
		"Quota exceeded for model",
		"Rate limit hit",
		"Billing issue on account",
		"Unauthorized access",
		"Forbidden resource",
	}
	for _, pat := range patterns {
		t.Run(pat, func(t *testing.T) {
			line := `{"type":"error","error":"` + pat + `"}`
			runner := &mockRunner{output: readerFromLines(line)}
			h, _ := New(runner)
			ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			events := collectEvents(t, ch)
			if len(events) != 1 {
				t.Fatalf("expected 1 event, got %d", len(events))
			}
			if events[0].Kind != plugin.RunEventFault {
				t.Errorf("expected kind=fault, got %q", events[0].Kind)
			}
			if events[0].Fault != "auth" {
				t.Errorf("expected fault=auth, got %q", events[0].Fault)
			}
		})
	}
}

func TestStart_FullStream(t *testing.T) {
	runner := &mockRunner{
		output: readerFromLines(
			`{"type":"status","content":"starting"}`,
			`{"type":"content","content":"hello"}`,
			`{"type":"delta","content":"world"}`,
			`{"type":"tool_use","tool":"bash","message":"ls"}`,
			`{"type":"result","content":"final answer"}`,
		),
	}
	h, _ := New(runner)
	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events := collectEvents(t, ch)
	if len(events) != 5 {
		t.Fatalf("expected 5 events, got %d", len(events))
	}
	if events[0].Kind != plugin.RunEventStatus {
		t.Errorf("event 0: expected kind=status, got %q", events[0].Kind)
	}
	if events[1].Kind != plugin.RunEventDelta {
		t.Errorf("event 1: expected kind=delta, got %q", events[1].Kind)
	}
	if events[2].Kind != plugin.RunEventDelta {
		t.Errorf("event 2: expected kind=delta, got %q", events[2].Kind)
	}
	if events[3].Kind != plugin.RunEventTool {
		t.Errorf("event 3: expected kind=tool, got %q", events[3].Kind)
	}
	if events[4].Kind != plugin.RunEventFinal {
		t.Errorf("event 4: expected kind=final, got %q", events[4].Kind)
	}
}

func TestStart_RunnerError(t *testing.T) {
	runner := &mockRunner{err: errors.New("exec failed")}
	h, _ := New(runner)
	_, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err == nil {
		t.Fatal("expected error from runner")
	}
	if !strings.Contains(err.Error(), "run failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStart_EmptyStream(t *testing.T) {
	runner := &mockRunner{
		output: readerFromLines(""),
	}
	h, _ := New(runner)
	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events := collectEvents(t, ch)
	if len(events) != 0 {
		t.Fatalf("expected 0 events, got %d", len(events))
	}
}

func TestStart_InvalidJSON(t *testing.T) {
	runner := &mockRunner{
		output: readerFromLines(`not json at all`),
	}
	h, _ := New(runner)
	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events := collectEvents(t, ch)
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Kind != plugin.RunEventFault {
		t.Errorf("expected kind=fault, got %q", events[0].Kind)
	}
	if events[0].Fault != "parse_error" {
		t.Errorf("expected fault=parse_error, got %q", events[0].Fault)
	}
}

func TestStart_MalformedJSONMidStream(t *testing.T) {
	runner := &mockRunner{
		output: readerFromLines(
			`{"type":"status","content":"ok"}`,
			`{bad json`,
			`{"type":"result","content":"end"}`,
		),
	}
	h, _ := New(runner)
	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events := collectEvents(t, ch)
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Kind != plugin.RunEventStatus {
		t.Errorf("event 0: expected kind=status, got %q", events[0].Kind)
	}
	if events[1].Kind != plugin.RunEventFault {
		t.Errorf("event 1: expected kind=fault, got %q", events[1].Kind)
	}
	if events[1].Fault != "parse_error" {
		t.Errorf("event 1: expected fault=parse_error, got %q", events[1].Fault)
	}
	if events[2].Kind != plugin.RunEventFinal {
		t.Errorf("event 2: expected kind=final, got %q", events[2].Kind)
	}
}

func TestStart_BlankLinesBetweenEvents(t *testing.T) {
	runner := &mockRunner{
		output: readerFromLines(
			`{"type":"status","content":"a"}`,
			"",
			"   ",
			`{"type":"result","content":"b"}`,
		),
	}
	h, _ := New(runner)
	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	events := collectEvents(t, ch)
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Phase != "a" {
		t.Errorf("expected phase=a, got %q", events[0].Phase)
	}
	if events[1].Content != "b" {
		t.Errorf("expected content=b, got %q", events[1].Content)
	}
}

func TestStart_UsesEventContentAsPrompt(t *testing.T) {
	var capturedPrompt string
	runner := &funcRunner{fn: func(_ context.Context, prompt string) (io.ReadCloser, error) {
		capturedPrompt = prompt
		return readerFromLines(`{"type":"result","content":"ok"}`), nil
	}}
	h, _ := New(runner)
	req := plugin.RunRequest{
		Scenario: "fallback scenario",
		Event:    plugin.InboundEvent{Content: "user message"},
	}
	ch, err := h.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	collectEvents(t, ch)
	if capturedPrompt != "user message" {
		t.Errorf("expected prompt='user message', got %q", capturedPrompt)
	}
}

func TestStart_UsesScenarioWhenEventContentEmpty(t *testing.T) {
	var capturedPrompt string
	runner := &funcRunner{fn: func(_ context.Context, prompt string) (io.ReadCloser, error) {
		capturedPrompt = prompt
		return readerFromLines(`{"type":"result","content":"ok"}`), nil
	}}
	h, _ := New(runner)
	req := plugin.RunRequest{
		Scenario: "scenario text",
		Event:    plugin.InboundEvent{Content: ""},
	}
	ch, err := h.Start(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	collectEvents(t, ch)
	if capturedPrompt != "scenario text" {
		t.Errorf("expected prompt='scenario text', got %q", capturedPrompt)
	}
}

func TestStart_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	br := &blockingReader{
		data: "{\"type\":\"status\",\"content\":\"start\"}\n",
		ctx:  ctx,
	}
	runner := &mockRunner{output: br}
	h, _ := New(runner)
	ch, err := h.Start(ctx, plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Read the first event.
	select {
	case ev := <-ch:
		if ev.Kind != plugin.RunEventStatus {
			t.Errorf("expected kind=status, got %q", ev.Kind)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for first event")
	}
	// Cancel context; goroutine should exit.
	cancel()
	select {
	case _, ok := <-ch:
		if ok {
			// Might get one more event; drain it.
			<-ch
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for channel close after cancel")
	}
}

func TestStart_ContextCancelClosesReader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	inner := &blockingReader{
		data: "{\"type\":\"status\",\"content\":\"go\"}\n",
		ctx:  ctx,
	}
	tc := &trackingCloser{ReadCloser: struct {
		io.Reader
		io.Closer
	}{inner, io.NopCloser(nil)}}
	runner := &trackingRunner{closer: tc}
	h, _ := New(runner)
	ch, err := h.Start(ctx, plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Drain the first event.
	<-ch
	cancel()
	// Wait for the channel to close.
	for range ch {
	}
	if !tc.closed {
		t.Error("expected reader to be closed after context cancel")
	}
}

func TestStart_ContextCancelDuringFaultSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	// The reader returns invalid JSON, but the channel is blocked (no receiver),
	// and then we cancel the context.
	data := "not json\n"
	br := &blockingReader{data: data, ctx: ctx}
	runner := &mockRunner{output: br}
	h, _ := New(runner)
	ch, err := h.Start(ctx, plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Don't read from ch immediately — let the goroutine try to send the fault.
	// Give goroutine time to reach the select.
	time.Sleep(50 * time.Millisecond)
	cancel()
	// Now drain channel.
	timeout := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for channel close")
		}
	}
}

func TestStart_ContextCancelDuringValidEventSend(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	data := "{\"type\":\"status\",\"content\":\"hi\"}\n"
	br := &blockingReader{data: data, ctx: ctx}
	runner := &mockRunner{output: br}
	h, _ := New(runner)
	ch, err := h.Start(ctx, plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Don't read from ch — let goroutine block on send.
	time.Sleep(50 * time.Millisecond)
	cancel()
	timeout := time.After(2 * time.Second)
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		case <-timeout:
			t.Fatal("timed out waiting for channel close")
		}
	}
}

func TestStart_ReaderClosedOnNormalCompletion(t *testing.T) {
	inner := readerFromLines(`{"type":"result","content":"ok"}`)
	tc := &trackingCloser{ReadCloser: inner}
	runner := &trackingRunner{closer: tc}
	h, _ := New(runner)
	ch, err := h.Start(context.Background(), plugin.RunRequest{Scenario: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	collectEvents(t, ch)
	if !tc.closed {
		t.Error("expected reader to be closed after normal completion")
	}
}

func TestClose(t *testing.T) {
	h, _ := New(&mockRunner{})
	if err := h.Close(); err != nil {
		t.Fatalf("expected nil error from Close, got %v", err)
	}
}

func TestJsonString_Empty(t *testing.T) {
	result := jsonString(nil)
	if result != "" {
		t.Errorf("expected empty string for nil input, got %q", result)
	}
	result = jsonString(json.RawMessage{})
	if result != "" {
		t.Errorf("expected empty string for empty input, got %q", result)
	}
}

func TestJsonString_NonString(t *testing.T) {
	raw := json.RawMessage(`42`)
	result := jsonString(raw)
	if result != "42" {
		t.Errorf("expected '42' for non-string JSON, got %q", result)
	}

	raw = json.RawMessage(`{"key":"val"}`)
	result = jsonString(raw)
	if result != `{"key":"val"}` {
		t.Errorf("expected raw JSON for object, got %q", result)
	}
}

func TestIsAuthFailure_NonMatch(t *testing.T) {
	if isAuthFailure("something random happened") {
		t.Error("expected false for non-auth error")
	}
	if isAuthFailure("") {
		t.Error("expected false for empty string")
	}
}

func TestLive(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to run live Claude test")
	}
	t.Log("LIVE test placeholder — no live runner configured in unit tests")
}
