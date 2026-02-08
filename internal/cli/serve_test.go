package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/klabo/tinyclaw/internal/orchestrator"
	"github.com/klabo/tinyclaw/internal/plugin"
	"github.com/klabo/tinyclaw/plugins/openclaw"
)

// --- stubs ---

type stubServeTransport struct {
	events  []plugin.InboundEvent
	subErr  error
	mu      sync.Mutex
	postOps []plugin.OutboundOp
}

func (s *stubServeTransport) Subscribe(_ context.Context) (<-chan plugin.InboundEvent, error) {
	if s.subErr != nil {
		return nil, s.subErr
	}
	ch := make(chan plugin.InboundEvent, len(s.events))
	for _, ev := range s.events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (s *stubServeTransport) Post(_ context.Context, op plugin.OutboundOp) error {
	s.mu.Lock()
	s.postOps = append(s.postOps, op)
	s.mu.Unlock()
	return nil
}

func (s *stubServeTransport) Close() error { return nil }

type stubServeHarness struct {
	events []plugin.RunEvent
	err    error
}

func (s *stubServeHarness) Start(_ context.Context, _ plugin.RunRequest) (<-chan plugin.RunEvent, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan plugin.RunEvent, len(s.events))
	for _, ev := range s.events {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (s *stubServeHarness) Close() error { return nil }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func counter() func() string {
	var n atomic.Int64
	return func() string {
		return fmt.Sprintf("run-%d", n.Add(1))
	}
}

// --- tests ---

func TestRunServeHappyPath(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}
	h := &stubServeHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventFinal, Content: "response"},
	}}

	bundleDir := t.TempDir()
	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return h, nil },
		WorkDir:    t.TempDir(),
		BundleDir:  bundleDir,
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tr.postOps) == 0 {
		t.Fatal("expected at least one post op")
	}
}

func TestRunServeSubscribeError(t *testing.T) {
	tr := &stubServeTransport{subErr: fmt.Errorf("subscribe fail")}

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return &stubServeHarness{}, nil },
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunServeConcurrent(t *testing.T) {
	events := make([]plugin.InboundEvent, 10)
	for i := range events {
		events[i] = plugin.InboundEvent{
			Type:      plugin.InboundMessage,
			Content:   fmt.Sprintf("msg-%d", i),
			ChannelID: "ch-1",
		}
	}
	tr := &stubServeTransport{events: events}

	var count atomic.Int64
	newHarness := func() (plugin.Harness, error) {
		count.Add(1)
		return &stubServeHarness{events: []plugin.RunEvent{
			{Kind: plugin.RunEventFinal, Content: "done"},
		}}, nil
	}

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: newHarness,
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count.Load() != 10 {
		t.Fatalf("expected 10 harness creates, got %d", count.Load())
	}
}

func TestRunServeContextCancel(t *testing.T) {
	// Use a transport that blocks until context is cancelled.
	blockingTransport := &blockingServeTransport{}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- RunServe(ctx, ServeParams{
			Transport:  blockingTransport,
			NewHarness: func() (plugin.Harness, error) { return &stubServeHarness{}, nil },
			WorkDir:    t.TempDir(),
			BundleDir:  t.TempDir(),
			Routing:    orchestrator.Config{Default: "default"},
			Logger:     testLogger(),
			IDFunc:     counter(),
		})
	}()

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("RunServe did not exit after context cancel")
	}
}

type blockingServeTransport struct{}

func (b *blockingServeTransport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	ch := make(chan plugin.InboundEvent)
	go func() {
		<-ctx.Done()
		close(ch)
	}()
	return ch, nil
}

func (b *blockingServeTransport) Post(context.Context, plugin.OutboundOp) error { return nil }
func (b *blockingServeTransport) Close() error                                  { return nil }

func TestRunServeNilContext(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}
	h := &stubServeHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventFinal, Content: "response"},
	}}

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return h, nil },
		Context:    nil, // no openclaw provider
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunServeWithOpenclaw(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}
	h := &stubServeHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventFinal, Content: "response"},
	}}

	// Provider with no .openclaw dir returns nil items, no error.
	provider := openclaw.New(openclaw.Options{})

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return h, nil },
		Context:    provider,
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunServeHarnessFactoryError(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return nil, fmt.Errorf("harness fail") },
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	// RunServe itself doesn't return harness errors — they're logged and the event is skipped.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunServeBundleCreateError(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}

	// Point BundleDir to a path that can't be created.
	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return &stubServeHarness{}, nil },
		WorkDir:    t.TempDir(),
		BundleDir:  "/dev/null/impossible",
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	// RunServe itself returns nil; the bundle error is logged and event is skipped.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunServeOrchestratorRunError(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}
	h := &stubServeHarness{err: fmt.Errorf("harness start fail")}

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return h, nil },
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     testLogger(),
		IDFunc:     counter(),
	})
	// Error is logged, not returned.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunServeNilLogger(t *testing.T) {
	tr := &stubServeTransport{
		events: []plugin.InboundEvent{
			{Type: plugin.InboundMessage, Content: "hello", ChannelID: "ch-1"},
		},
	}
	h := &stubServeHarness{events: []plugin.RunEvent{
		{Kind: plugin.RunEventFinal, Content: "done"},
	}}

	err := RunServe(context.Background(), ServeParams{
		Transport:  tr,
		NewHarness: func() (plugin.Harness, error) { return h, nil },
		WorkDir:    t.TempDir(),
		BundleDir:  t.TempDir(),
		Routing:    orchestrator.Config{Default: "default"},
		Logger:     nil,
		IDFunc:     counter(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
