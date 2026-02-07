package discord

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// mockClient implements Client for testing.
type mockClient struct {
	mu            sync.Mutex
	sendFn        func(channelID, content string) (string, error)
	editFn        func(channelID, messageID, content string) error
	subscribeFn   func(handler func(msg Message)) error
	closeFn       func() error
	lastHandler   func(msg Message)
	sendCalls     []sendCall
	editCalls     []editCall
}

type sendCall struct {
	channelID, content string
}

type editCall struct {
	channelID, messageID, content string
}

func (m *mockClient) SendMessage(channelID, content string) (string, error) {
	m.mu.Lock()
	m.sendCalls = append(m.sendCalls, sendCall{channelID, content})
	m.mu.Unlock()
	if m.sendFn != nil {
		return m.sendFn(channelID, content)
	}
	return "msg-1", nil
}

func (m *mockClient) EditMessage(channelID, messageID, content string) error {
	m.mu.Lock()
	m.editCalls = append(m.editCalls, editCall{channelID, messageID, content})
	m.mu.Unlock()
	if m.editFn != nil {
		return m.editFn(channelID, messageID, content)
	}
	return nil
}

func (m *mockClient) SubscribeMessages(handler func(msg Message)) error {
	m.mu.Lock()
	m.lastHandler = handler
	m.mu.Unlock()
	if m.subscribeFn != nil {
		return m.subscribeFn(handler)
	}
	return nil
}

func (m *mockClient) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func (m *mockClient) handler() func(msg Message) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastHandler
}

// --- New ---

func TestNew(t *testing.T) {
	mc := &mockClient{}
	tr, err := New(mc, "ch-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
}

func TestNewNilClient(t *testing.T) {
	_, err := New(nil, "ch-1")
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestNewEmptyChannel(t *testing.T) {
	_, err := New(&mockClient{}, "")
	if err == nil {
		t.Fatal("expected error for empty channel")
	}
}

// --- Subscribe ---

func TestSubscribeReceivesMessages(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	mc.handler()(Message{ID: "m1", ChannelID: "ch-1", AuthorID: "u1", Content: "hello"})

	select {
	case ev := <-ch:
		if ev.Content != "hello" {
			t.Fatalf("got content %q, want %q", ev.Content, "hello")
		}
		if ev.Type != plugin.InboundMessage {
			t.Fatalf("got type %q, want %q", ev.Type, plugin.InboundMessage)
		}
		if ev.ChannelID != "ch-1" {
			t.Fatalf("got channel %q, want %q", ev.ChannelID, "ch-1")
		}
		if ev.AuthorID != "u1" {
			t.Fatalf("got author %q, want %q", ev.AuthorID, "u1")
		}
		if ev.MessageID != "m1" {
			t.Fatalf("got message id %q, want %q", ev.MessageID, "m1")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestSubscribeFiltersOtherChannels(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	mc.handler()(Message{ID: "m1", ChannelID: "ch-other", Content: "nope"})

	select {
	case ev := <-ch:
		t.Fatalf("unexpected event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// good: filtered out
	}
}

func TestSubscribeAlreadyClosed(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")
	_ = tr.Close()

	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error for closed transport")
	}
}

func TestSubscribeTwice(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("first subscribe error: %v", err)
	}

	_, err = tr.Subscribe(ctx)
	if err == nil {
		t.Fatal("expected error for double subscribe")
	}
}

func TestSubscribeClientError(t *testing.T) {
	mc := &mockClient{
		subscribeFn: func(handler func(msg Message)) error {
			return fmt.Errorf("client fail")
		},
	}
	tr, _ := New(mc, "ch-1")

	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error from client subscribe")
	}

	// subscribed flag should be reset — can subscribe again
	mc.subscribeFn = nil
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err = tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("second subscribe should succeed: %v", err)
	}
}

func TestSubscribeContextCancel(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	cancel()

	// The forwarding goroutine should exit and close `out`.
	select {
	case _, ok := <-ch:
		if ok {
			// may get a stale event; drain
		}
	case <-time.After(time.Second):
		t.Fatal("channel not closed after cancel")
	}
}

func TestSubscribeContextDoneInHandler(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	_, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	cancel()
	// Give forwarding goroutine time to notice cancel.
	time.Sleep(20 * time.Millisecond)

	// Handler should not block when context is done.
	done := make(chan struct{})
	go func() {
		mc.handler()(Message{ID: "m1", ChannelID: "ch-1", Content: "after cancel"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler blocked after context cancel")
	}
}

func TestSubscribeContextCancelWhileForwarding(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("subscribe error: %v", err)
	}

	// Fill the internal buffer (capacity 64)
	for i := 0; i < 64; i++ {
		mc.handler()(Message{ID: fmt.Sprintf("m%d", i), ChannelID: "ch-1", Content: fmt.Sprintf("msg%d", i)})
	}

	// Cancel while forwarding goroutine is busy
	cancel()

	// Push more — should not block because context is cancelled
	done := make(chan struct{})
	go func() {
		mc.handler()(Message{ID: "extra", ChannelID: "ch-1", Content: "extra"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("handler blocked with full buffer after cancel")
	}

	// Drain the out channel
	for range ch {
	}
}

// --- Post ---

func TestPostSend(t *testing.T) {
	tests := []struct {
		name      string
		op        plugin.OutboundOp
		wantCh    string
		wantBody  string
	}{
		{
			name:     "DefaultChannel",
			op:       plugin.OutboundOp{Kind: plugin.OutboundPost, Content: "hi"},
			wantCh:   "ch-1",
			wantBody: "hi",
		},
		{
			name:     "CustomChannel",
			op:       plugin.OutboundOp{Kind: plugin.OutboundPost, Content: "hi", ChannelID: "ch-2"},
			wantCh:   "ch-2",
			wantBody: "hi",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{}
			tr, _ := New(mc, "ch-1")
			if err := tr.Post(context.Background(), tt.op); err != nil {
				t.Fatalf("post error: %v", err)
			}
			mc.mu.Lock()
			defer mc.mu.Unlock()
			if len(mc.sendCalls) != 1 {
				t.Fatalf("expected 1 send call, got %d", len(mc.sendCalls))
			}
			if mc.sendCalls[0].channelID != tt.wantCh {
				t.Fatalf("got channel %q, want %q", mc.sendCalls[0].channelID, tt.wantCh)
			}
			if mc.sendCalls[0].content != tt.wantBody {
				t.Fatalf("got content %q, want %q", mc.sendCalls[0].content, tt.wantBody)
			}
		})
	}
}

func TestPostEdit(t *testing.T) {
	tests := []struct {
		name   string
		op     plugin.OutboundOp
		wantCh string
		wantID string
	}{
		{
			name:   "DefaultChannel",
			op:     plugin.OutboundOp{Kind: plugin.OutboundEdit, Content: "updated", MessageID: "m1"},
			wantCh: "ch-1",
			wantID: "m1",
		},
		{
			name:   "CustomChannel",
			op:     plugin.OutboundOp{Kind: plugin.OutboundEdit, Content: "updated", MessageID: "m1", ChannelID: "ch-2"},
			wantCh: "ch-2",
			wantID: "m1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := &mockClient{}
			tr, _ := New(mc, "ch-1")
			if err := tr.Post(context.Background(), tt.op); err != nil {
				t.Fatalf("post error: %v", err)
			}
			mc.mu.Lock()
			defer mc.mu.Unlock()
			if len(mc.editCalls) != 1 {
				t.Fatalf("expected 1 edit call, got %d", len(mc.editCalls))
			}
			if mc.editCalls[0].channelID != tt.wantCh {
				t.Fatalf("got channel %q, want %q", mc.editCalls[0].channelID, tt.wantCh)
			}
			if mc.editCalls[0].messageID != tt.wantID {
				t.Fatalf("got message id %q, want %q", mc.editCalls[0].messageID, tt.wantID)
			}
		})
	}
}

func TestPostErrors(t *testing.T) {
	tests := []struct {
		name   string
		setup  func() (*Transport, context.Context)
		op     plugin.OutboundOp
	}{
		{
			name: "SendError",
			setup: func() (*Transport, context.Context) {
				mc := &mockClient{sendFn: func(string, string) (string, error) {
					return "", fmt.Errorf("send fail")
				}}
				tr, _ := New(mc, "ch-1")
				return tr, context.Background()
			},
			op: plugin.OutboundOp{Kind: plugin.OutboundPost, Content: "hi"},
		},
		{
			name: "EditMissingMessageID",
			setup: func() (*Transport, context.Context) {
				tr, _ := New(&mockClient{}, "ch-1")
				return tr, context.Background()
			},
			op: plugin.OutboundOp{Kind: plugin.OutboundEdit, Content: "updated"},
		},
		{
			name: "EditError",
			setup: func() (*Transport, context.Context) {
				mc := &mockClient{editFn: func(string, string, string) error {
					return fmt.Errorf("edit fail")
				}}
				tr, _ := New(mc, "ch-1")
				return tr, context.Background()
			},
			op: plugin.OutboundOp{Kind: plugin.OutboundEdit, Content: "x", MessageID: "m1"},
		},
		{
			name: "UnsupportedKind",
			setup: func() (*Transport, context.Context) {
				tr, _ := New(&mockClient{}, "ch-1")
				return tr, context.Background()
			},
			op: plugin.OutboundOp{Kind: "delete"},
		},
		{
			name: "AfterClose",
			setup: func() (*Transport, context.Context) {
				tr, _ := New(&mockClient{}, "ch-1")
				_ = tr.Close()
				return tr, context.Background()
			},
			op: plugin.OutboundOp{Kind: plugin.OutboundPost, Content: "hi"},
		},
		{
			name: "ContextCancelled",
			setup: func() (*Transport, context.Context) {
				tr, _ := New(&mockClient{}, "ch-1")
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return tr, ctx
			},
			op: plugin.OutboundOp{Kind: plugin.OutboundPost, Content: "hi"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr, ctx := tt.setup()
			if err := tr.Post(ctx, tt.op); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestPostSendChunked(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	content := strings.Repeat("a", 3000)
	err := tr.Post(context.Background(), plugin.OutboundOp{Kind: plugin.OutboundPost, Content: content})
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	mc.mu.Lock()
	defer mc.mu.Unlock()
	if len(mc.sendCalls) < 2 {
		t.Fatalf("expected at least 2 send calls for chunked content, got %d", len(mc.sendCalls))
	}
}

func TestPostSendEmptyContent(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{Kind: plugin.OutboundPost, Content: ""})
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	mc.mu.Lock()
	defer mc.mu.Unlock()
	// chunk("", 2000) returns nil, so len(chunks) == 0 -> single SendMessage call
	if len(mc.sendCalls) != 1 {
		t.Fatalf("expected 1 send call for empty content, got %d", len(mc.sendCalls))
	}
}

func TestPostSendChunkedError(t *testing.T) {
	callCount := 0
	mc := &mockClient{
		sendFn: func(channelID, content string) (string, error) {
			callCount++
			if callCount == 2 {
				return "", fmt.Errorf("send fail on chunk 2")
			}
			return "msg-1", nil
		},
	}
	tr, _ := New(mc, "ch-1")

	content := strings.Repeat("a", 3000)
	err := tr.Post(context.Background(), plugin.OutboundOp{Kind: plugin.OutboundPost, Content: content})
	if err == nil {
		t.Fatal("expected error from chunked send")
	}
}

// --- Close ---

func TestClose(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	err := tr.Close()
	if err != nil {
		t.Fatalf("close error: %v", err)
	}
}

func TestDoubleClose(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	_ = tr.Close()
	err := tr.Close()
	if err == nil {
		t.Fatal("expected error for double close")
	}
}

func TestCloseClientError(t *testing.T) {
	mc := &mockClient{
		closeFn: func() error {
			return fmt.Errorf("close fail")
		},
	}
	tr, _ := New(mc, "ch-1")

	err := tr.Close()
	if err == nil {
		t.Fatal("expected error from client close")
	}
}

// --- Live test ---

func TestLiveDiscord(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to run live Discord tests")
	}
	t.Log("live Discord test placeholder — requires bot token and channel ID")
}
