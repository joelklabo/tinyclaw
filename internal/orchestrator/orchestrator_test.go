package orchestrator

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// --- test doubles ---

type stubTransport struct {
	ops    []plugin.OutboundOp
	postFn func(plugin.OutboundOp) error
}

func (s *stubTransport) Subscribe(context.Context) (<-chan plugin.InboundEvent, error) {
	return nil, nil
}

func (s *stubTransport) Post(_ context.Context, op plugin.OutboundOp) error {
	s.ops = append(s.ops, op)
	if s.postFn != nil {
		return s.postFn(op)
	}
	return nil
}

func (s *stubTransport) Close() error { return nil }

type stubHarness struct {
	req     plugin.RunRequest
	events  []plugin.RunEvent
	startFn func() (<-chan plugin.RunEvent, error)
}

func (s *stubHarness) Start(_ context.Context, req plugin.RunRequest) (<-chan plugin.RunEvent, error) {
	s.req = req
	if s.startFn != nil {
		return s.startFn()
	}
	ch := make(chan plugin.RunEvent, len(s.events))
	for _, e := range s.events {
		ch <- e
	}
	close(ch)
	return ch, nil
}

func (s *stubHarness) Close() error { return nil }

// stubBundle is an in-memory BundleWriter stub for tests.
type stubBundle struct {
	appendErr error
	writeErr  error
	closeErr  error
	lines     []any
	failMsg   string
	status    string
}

func (b *stubBundle) AppendJSONL(_ string, v any) error {
	if b.appendErr != nil {
		return b.appendErr
	}
	b.lines = append(b.lines, v)
	return nil
}

func (b *stubBundle) WriteFail(msg string) error {
	if b.writeErr != nil {
		return b.writeErr
	}
	b.failMsg = msg
	return nil
}

func (b *stubBundle) Close(status string) error {
	b.status = status
	return b.closeErr
}

// --- routing tests ---

func TestRouteExactChannel(t *testing.T) {
	o := New(Params{
		Routing: Config{
			Rules: []Rule{{Channel: "ch1", Profile: "profile-a"}},
		},
		Bundle: &stubBundle{},
	})

	got, err := o.route(plugin.InboundEvent{ChannelID: "ch1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "profile-a" {
		t.Fatalf("got %q, want %q", got, "profile-a")
	}
}

func TestRoutePrefixMatch(t *testing.T) {
	o := New(Params{
		Routing: Config{
			Rules: []Rule{{Prefix: "!ask", Profile: "profile-b"}},
		},
		Bundle: &stubBundle{},
	})

	got, err := o.route(plugin.InboundEvent{Content: "!ask me anything"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "profile-b" {
		t.Fatalf("got %q, want %q", got, "profile-b")
	}
}

func TestRouteChannelBeatsPrefix(t *testing.T) {
	o := New(Params{
		Routing: Config{
			Rules: []Rule{
				{Prefix: "!ask", Profile: "prefix-profile"},
				{Channel: "ch1", Profile: "channel-profile"},
			},
		},
		Bundle: &stubBundle{},
	})

	got, err := o.route(plugin.InboundEvent{ChannelID: "ch1", Content: "!ask me"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "channel-profile" {
		t.Fatalf("got %q, want %q", got, "channel-profile")
	}
}

func TestRouteDefaultFallback(t *testing.T) {
	o := New(Params{
		Routing: Config{Default: "fallback"},
		Bundle:  &stubBundle{},
	})

	got, err := o.route(plugin.InboundEvent{Content: "random message"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "fallback" {
		t.Fatalf("got %q, want %q", got, "fallback")
	}
}

func TestRouteNoMatchNoDefault(t *testing.T) {
	o := New(Params{
		Routing: Config{
			Rules: []Rule{{Channel: "ch1", Profile: "p"}},
		},
		Bundle: &stubBundle{},
	})

	_, err := o.route(plugin.InboundEvent{ChannelID: "ch-other"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- event mapping tests (table-driven) ---

func TestMapToTransport(t *testing.T) {
	tests := []struct {
		name    string
		input   plugin.RunEvent
		wantOps []plugin.OutboundOp
	}{
		{
			name:  "status maps to OutboundStatus",
			input: plugin.RunEvent{Kind: plugin.RunEventStatus, Phase: "thinking"},
			wantOps: []plugin.OutboundOp{
				{Kind: plugin.OutboundStatus, Phase: "thinking"},
			},
		},
		{
			name:  "delta maps to OutboundDelta with seq",
			input: plugin.RunEvent{Kind: plugin.RunEventDelta, Content: "chunk"},
			wantOps: []plugin.OutboundOp{
				{Kind: plugin.OutboundDelta, Content: "chunk", Seq: 1},
			},
		},
		{
			name:  "tool maps to OutboundTool",
			input: plugin.RunEvent{Kind: plugin.RunEventTool, Tool: "bash"},
			wantOps: []plugin.OutboundOp{
				{Kind: plugin.OutboundTool, Tool: "bash"},
			},
		},
		{
			name:  "final maps to OutboundResponse",
			input: plugin.RunEvent{Kind: plugin.RunEventFinal, Content: "done"},
			wantOps: []plugin.OutboundOp{
				{Kind: plugin.OutboundResponse, Content: "done"},
			},
		},
		{
			name:  "fault maps to OutboundError",
			input: plugin.RunEvent{Kind: plugin.RunEventFault, Message: "oops", Fault: "fatal"},
			wantOps: []plugin.OutboundOp{
				{Kind: plugin.OutboundError, Content: "oops", Fault: "fatal"},
			},
		},
		{
			name:    "unknown kind produces no ops",
			input:   plugin.RunEvent{Kind: "unknown-kind"},
			wantOps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &stubTransport{}
			o := New(Params{Transport: tr, Bundle: &stubBundle{}})

			err := o.mapToTransport(context.Background(), tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tr.ops) != len(tt.wantOps) {
				t.Fatalf("got %d ops, want %d: %+v", len(tr.ops), len(tt.wantOps), tr.ops)
			}
			for i, want := range tt.wantOps {
				if tr.ops[i].Kind != want.Kind {
					t.Errorf("ops[%d].Kind = %q, want %q", i, tr.ops[i].Kind, want.Kind)
				}
				if tr.ops[i].Content != want.Content {
					t.Errorf("ops[%d].Content = %q, want %q", i, tr.ops[i].Content, want.Content)
				}
				if tr.ops[i].Phase != want.Phase {
					t.Errorf("ops[%d].Phase = %q, want %q", i, tr.ops[i].Phase, want.Phase)
				}
				if tr.ops[i].Tool != want.Tool {
					t.Errorf("ops[%d].Tool = %q, want %q", i, tr.ops[i].Tool, want.Tool)
				}
				if tr.ops[i].Fault != want.Fault {
					t.Errorf("ops[%d].Fault = %q, want %q", i, tr.ops[i].Fault, want.Fault)
				}
				if tr.ops[i].Seq != want.Seq {
					t.Errorf("ops[%d].Seq = %d, want %d", i, tr.ops[i].Seq, want.Seq)
				}
			}
		})
	}
}

// --- pipeline tests ---

func TestRunHappyPath(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventDelta, Content: "hi"},
		{Kind: plugin.RunEventFinal, Content: "done"},
	}}

	bw := &stubBundle{}
	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: bw})

	event := plugin.InboundEvent{Content: "hello"}
	ctx := context.Background()
	err := o.Run(ctx, event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should have 2 ops: delta + response
	if len(tr.ops) != 2 {
		t.Fatalf("expected 2 transport ops, got %d: %+v", len(tr.ops), tr.ops)
	}
	if tr.ops[0].Kind != plugin.OutboundDelta {
		t.Fatalf("expected delta, got %q", tr.ops[0].Kind)
	}
	if tr.ops[1].Kind != plugin.OutboundResponse {
		t.Fatalf("expected response, got %q", tr.ops[1].Kind)
	}
	if tr.ops[1].Content != "done" {
		t.Fatalf("expected content %q, got %q", "done", tr.ops[1].Content)
	}
}

func TestRunHarnessStartError(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{startFn: func() (<-chan plugin.RunEvent, error) {
		return nil, errors.New("harness boom")
	}}

	bw := &stubBundle{}
	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: bw})

	err := o.Run(context.Background(), plugin.InboundEvent{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if bw.failMsg == "" {
		t.Fatal("expected fail message to be written")
	}
	if bw.status != "fail" {
		t.Fatalf("expected bundle status fail, got %q", bw.status)
	}
}

func TestRunTransportPostError(t *testing.T) {
	postErr := errors.New("post boom")
	tr := &stubTransport{postFn: func(_ plugin.OutboundOp) error {
		return postErr
	}}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventFinal, Content: "result"},
	}}

	bw := &stubBundle{}
	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: bw})

	err := o.Run(context.Background(), plugin.InboundEvent{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if bw.failMsg == "" {
		t.Fatal("expected fail message to be written")
	}
}

func TestRunRouteError(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{}
	bw := &stubBundle{}
	o := New(Params{Transport: tr, Harness: h, Bundle: bw}) // no rules, no default

	err := o.Run(context.Background(), plugin.InboundEvent{}, nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if bw.failMsg == "" {
		t.Fatal("expected fail message to be written")
	}
}

// --- profile and context passing ---

func TestProfilePassedToHarness(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{}}

	o := New(Params{
		Transport: tr,
		Harness:   h,
		Routing: Config{
			Rules: []Rule{{Channel: "ch1", Profile: "special-agent"}},
		},
		Bundle: &stubBundle{},
	})

	event := plugin.InboundEvent{ChannelID: "ch1", Content: "hi"}
	err := o.Run(context.Background(), event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.req.Profile != "special-agent" {
		t.Fatalf("profile: got %q, want %q", h.req.Profile, "special-agent")
	}
}

func TestContextItemsPassed(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{}}

	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: &stubBundle{}})

	items := []plugin.ContextItem{
		{Name: "readme", Content: "hello world", Source: "test", Priority: 1},
	}
	err := o.Run(context.Background(), plugin.InboundEvent{}, items)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(h.req.Context) != 1 || h.req.Context[0].Name != "readme" {
		t.Fatalf("context: got %+v", h.req.Context)
	}
}

// --- bundle error logging ---

func TestAppendFrameErrorLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventFinal, Content: "result"},
	}}

	bw := &stubBundle{appendErr: errors.New("append fail")}
	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: bw, Logger: logger})

	_ = o.Run(context.Background(), plugin.InboundEvent{}, nil)

	logged := buf.String()
	if logged == "" {
		t.Fatal("expected log output about append errors, got nothing")
	}
}

// --- fail path ---

func TestFailWritesBundleFailMsg(t *testing.T) {
	bw := &stubBundle{}
	o := New(Params{Bundle: bw})

	returnedErr := o.fail(errors.New("test failure"))
	if returnedErr == nil || returnedErr.Error() != "test failure" {
		t.Fatalf("expected 'test failure', got %v", returnedErr)
	}

	if bw.failMsg != "test failure" {
		t.Fatalf("expected fail msg 'test failure', got %q", bw.failMsg)
	}
	if bw.status != "fail" {
		t.Fatalf("expected status 'fail', got %q", bw.status)
	}
}

func TestFailBundleWriteError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	bw := &stubBundle{writeErr: errors.New("write fail"), closeErr: errors.New("close fail")}
	o := New(Params{Bundle: bw, Logger: logger})
	_ = o.fail(errors.New("some error"))

	logged := buf.String()
	if logged == "" {
		t.Fatal("expected log output about bundle write errors")
	}
}

// --- nil logger ---

func TestNilLoggerNoPanic(t *testing.T) {
	o := New(Params{Bundle: &stubBundle{}})
	if o.logger == nil {
		t.Fatal("expected non-nil logger")
	}
	// Should not panic
	o.logger.Info("test")
}

// --- logPhase ---

func TestLogPhaseWritesToBundle(t *testing.T) {
	bw := &stubBundle{}
	o := New(Params{Bundle: bw})

	o.logPhase("routed")

	if len(bw.lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(bw.lines))
	}
}

func TestLogPhaseErrorLogged(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))

	bw := &stubBundle{appendErr: errors.New("append fail")}
	o := New(Params{Bundle: bw, Logger: logger})
	o.logPhase("test-phase")

	if buf.String() == "" {
		t.Fatal("expected log output about phase append error")
	}
}

// --- event passed in RunRequest ---

func TestEventPassedToHarness(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{}}

	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: &stubBundle{}})

	event := plugin.InboundEvent{
		Type:      plugin.InboundMessage,
		Content:   "test content",
		ChannelID: "ch1",
		AuthorID:  "author1",
		MessageID: "msg1",
	}
	err := o.Run(context.Background(), event, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.req.Event.Content != "test content" {
		t.Fatalf("event content: got %q, want %q", h.req.Event.Content, "test content")
	}
	if h.req.Event.ChannelID != "ch1" {
		t.Fatalf("event channel: got %q, want %q", h.req.Event.ChannelID, "ch1")
	}
}

// --- dual channel+prefix rule ---

func TestRouteDualChannelAndPrefix(t *testing.T) {
	o := New(Params{
		Routing: Config{
			Rules: []Rule{{Channel: "ch1", Prefix: "!ask", Profile: "dual-profile"}},
		},
		Bundle: &stubBundle{},
	})

	got, err := o.route(plugin.InboundEvent{ChannelID: "ch1", Content: "!ask something"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "dual-profile" {
		t.Fatalf("got %q, want %q", got, "dual-profile")
	}
}

// --- delta buffering tests ---

func TestDeltaBuffering(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventDelta, Content: "Hello "},
		{Kind: plugin.RunEventDelta, Content: "world"},
		{Kind: plugin.RunEventFinal, Content: "Hello world"},
	}}

	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: &stubBundle{}})

	err := o.Run(context.Background(), plugin.InboundEvent{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 deltas + 1 response = 3 ops
	if len(tr.ops) != 3 {
		t.Fatalf("expected 3 transport ops, got %d: %+v", len(tr.ops), tr.ops)
	}
	if tr.ops[0].Kind != plugin.OutboundDelta {
		t.Fatalf("expected delta, got %q", tr.ops[0].Kind)
	}
	if tr.ops[1].Kind != plugin.OutboundDelta {
		t.Fatalf("expected delta, got %q", tr.ops[1].Kind)
	}
	if tr.ops[2].Kind != plugin.OutboundResponse {
		t.Fatalf("expected response, got %q", tr.ops[2].Kind)
	}
	if tr.ops[2].Content != "Hello world" {
		t.Fatalf("expected content %q, got %q", "Hello world", tr.ops[2].Content)
	}
}

func TestDeltaBufferUsedWhenFinalEmpty(t *testing.T) {
	tr := &stubTransport{}
	h := &stubHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventDelta, Content: "buffered "},
		{Kind: plugin.RunEventDelta, Content: "content"},
		{Kind: plugin.RunEventFinal, Content: ""},
	}}

	o := New(Params{Transport: tr, Harness: h, Routing: Config{Default: "agent"}, Bundle: &stubBundle{}})

	err := o.Run(context.Background(), plugin.InboundEvent{}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 2 deltas + 1 response = 3 ops
	if len(tr.ops) != 3 {
		t.Fatalf("expected 3 transport ops, got %d: %+v", len(tr.ops), tr.ops)
	}
	// The final response uses buffered content when its own content is empty.
	if tr.ops[2].Content != "buffered content" {
		t.Fatalf("expected content %q, got %q", "buffered content", tr.ops[2].Content)
	}
}
