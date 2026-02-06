package transportdiscord

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Verify Transport implements plugin.Transport.
var _ plugin.Transport = (*Transport)(nil)

// --- mock client ---

type mockClient struct {
	sent     []sentMsg
	edited   []editedMsg
	handler  func(msg Message)
	subErr   error
	sendErr  error
	editErr  error
	closeErr error
	closed   bool
}

type sentMsg struct {
	channelID string
	content   string
}

type editedMsg struct {
	channelID string
	messageID string
	content   string
}

func (m *mockClient) SendMessage(channelID, content string) (string, error) {
	if m.sendErr != nil {
		return "", m.sendErr
	}
	m.sent = append(m.sent, sentMsg{channelID, content})
	return "msg-1", nil
}

func (m *mockClient) EditMessage(channelID, messageID, content string) error {
	if m.editErr != nil {
		return m.editErr
	}
	m.edited = append(m.edited, editedMsg{channelID, messageID, content})
	return nil
}

func (m *mockClient) SubscribeMessages(handler func(msg Message)) error {
	if m.subErr != nil {
		return m.subErr
	}
	m.handler = handler
	return nil
}

func (m *mockClient) Close() error {
	m.closed = true
	return m.closeErr
}

// simulate delivers a message to the registered handler.
func (m *mockClient) simulate(msg Message) {
	if m.handler != nil {
		m.handler(msg)
	}
}

// --- tests ---

func TestNewValid(t *testing.T) {
	tr, err := New(&mockClient{}, "ch-1")
	if err != nil {
		t.Fatalf("New: %v", err)
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
		t.Fatal("expected error for empty channelID")
	}
}

func TestSubscribeReceivesMessages(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	mc.simulate(Message{ID: "m1", ChannelID: "ch-1", AuthorID: "u1", Content: "hello"})

	select {
	case ev := <-ch:
		if ev.Type != "message" {
			t.Fatalf("type = %q, want message", ev.Type)
		}
		if ev.Data["content"] != "hello" {
			t.Fatalf("content = %v, want hello", ev.Data["content"])
		}
		if ev.Data["author_id"] != "u1" {
			t.Fatalf("author_id = %v, want u1", ev.Data["author_id"])
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
		t.Fatalf("Subscribe: %v", err)
	}

	// Send to a different channel — should be filtered.
	mc.simulate(Message{ID: "m1", ChannelID: "ch-other", AuthorID: "u1", Content: "nope"})

	select {
	case ev := <-ch:
		t.Fatalf("unexpected event: %+v", ev)
	case <-time.After(50 * time.Millisecond):
		// expected: nothing received
	}
}

func TestSubscribeAlreadyClosed(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")
	tr.Close()
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
		t.Fatal(err)
	}
	_, err = tr.Subscribe(ctx)
	if err == nil {
		t.Fatal("expected error for double subscribe")
	}
}

func TestSubscribeClientError(t *testing.T) {
	mc := &mockClient{subErr: fmt.Errorf("boom")}
	tr, _ := New(mc, "ch-1")

	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error from client subscribe failure")
	}

	// After failed subscribe, should be able to try again.
	mc.subErr = nil
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
		t.Fatal(err)
	}

	cancel()

	// Channel should close after context cancellation.
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

func TestSubscribeContextDoneInHandler(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel before any messages arrive.

	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a message after cancellation — should not block.
	mc.simulate(Message{ID: "m1", ChannelID: "ch-1", AuthorID: "u1", Content: "late"})

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

func TestPostSend(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{
		Kind: "post",
		Data: map[string]any{"content": "hello world"},
	}
	if err := tr.Post(context.Background(), op); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if len(mc.sent) != 1 {
		t.Fatalf("sent %d messages, want 1", len(mc.sent))
	}
	if mc.sent[0].content != "hello world" {
		t.Fatalf("content = %q, want hello world", mc.sent[0].content)
	}
	if mc.sent[0].channelID != "ch-1" {
		t.Fatalf("channelID = %q, want ch-1", mc.sent[0].channelID)
	}
}

func TestPostSendCustomChannel(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{
		Kind: "post",
		Data: map[string]any{"content": "hi", "channel_id": "ch-2"},
	}
	if err := tr.Post(context.Background(), op); err != nil {
		t.Fatalf("Post: %v", err)
	}
	if mc.sent[0].channelID != "ch-2" {
		t.Fatalf("channelID = %q, want ch-2", mc.sent[0].channelID)
	}
}

func TestPostSendError(t *testing.T) {
	mc := &mockClient{sendErr: fmt.Errorf("send fail")}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{Kind: "post", Data: map[string]any{"content": "hi"}}
	err := tr.Post(context.Background(), op)
	if err == nil {
		t.Fatal("expected error from send failure")
	}
}

func TestPostEdit(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{
		Kind: "edit",
		Data: map[string]any{"content": "updated", "message_id": "msg-1"},
	}
	if err := tr.Post(context.Background(), op); err != nil {
		t.Fatalf("Post edit: %v", err)
	}
	if len(mc.edited) != 1 {
		t.Fatalf("edited %d messages, want 1", len(mc.edited))
	}
	if mc.edited[0].content != "updated" {
		t.Fatalf("content = %q, want updated", mc.edited[0].content)
	}
	if mc.edited[0].messageID != "msg-1" {
		t.Fatalf("messageID = %q, want msg-1", mc.edited[0].messageID)
	}
}

func TestPostEditCustomChannel(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{
		Kind: "edit",
		Data: map[string]any{"content": "x", "message_id": "m1", "channel_id": "ch-3"},
	}
	if err := tr.Post(context.Background(), op); err != nil {
		t.Fatalf("Post edit: %v", err)
	}
	if mc.edited[0].channelID != "ch-3" {
		t.Fatalf("channelID = %q, want ch-3", mc.edited[0].channelID)
	}
}

func TestPostEditMissingMessageID(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{
		Kind: "edit",
		Data: map[string]any{"content": "updated"},
	}
	err := tr.Post(context.Background(), op)
	if err == nil {
		t.Fatal("expected error for missing message_id")
	}
}

func TestPostEditError(t *testing.T) {
	mc := &mockClient{editErr: fmt.Errorf("edit fail")}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{
		Kind: "edit",
		Data: map[string]any{"content": "x", "message_id": "m1"},
	}
	err := tr.Post(context.Background(), op)
	if err == nil {
		t.Fatal("expected error from edit failure")
	}
}

func TestPostUnsupportedKind(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	op := plugin.OutboundOp{Kind: "upload", Data: map[string]any{}}
	err := tr.Post(context.Background(), op)
	if err == nil {
		t.Fatal("expected error for unsupported kind")
	}
}

func TestPostAfterClose(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")
	tr.Close()

	op := plugin.OutboundOp{Kind: "post", Data: map[string]any{"content": "hi"}}
	err := tr.Post(context.Background(), op)
	if err == nil {
		t.Fatal("expected error for post on closed transport")
	}
}

func TestPostContextCancelled(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	op := plugin.OutboundOp{Kind: "post", Data: map[string]any{"content": "hi"}}
	err := tr.Post(ctx, op)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestClose(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")
	if err := tr.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if !mc.closed {
		t.Fatal("expected client to be closed")
	}
}

func TestCloseDouble(t *testing.T) {
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")
	tr.Close()
	err := tr.Close()
	if err == nil {
		t.Fatal("expected error for double close")
	}
}

func TestCloseClientError(t *testing.T) {
	mc := &mockClient{closeErr: fmt.Errorf("close fail")}
	tr, _ := New(mc, "ch-1")
	err := tr.Close()
	if err == nil {
		t.Fatal("expected error from client close failure")
	}
}

func TestSubscribeContextCancelWhileForwarding(t *testing.T) {
	// Covers the inner ctx.Done() branch in the forwarding goroutine.
	// We fill the out channel, then cancel context, then push a message.
	mc := &mockClient{}
	tr, _ := New(mc, "ch-1")

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := tr.Subscribe(ctx)
	if err != nil {
		t.Fatal(err)
	}

	// Fill the out channel buffer (capacity 64).
	for i := 0; i < 64; i++ {
		mc.simulate(Message{ID: fmt.Sprintf("m%d", i), ChannelID: "ch-1", AuthorID: "u1", Content: "fill"})
	}
	// Give the forwarding goroutine time to process.
	time.Sleep(50 * time.Millisecond)

	// Cancel context. The forwarding goroutine will hit ctx.Done() in the
	// inner select on the next event it tries to forward.
	cancel()

	// Push one more message — it may arrive on internal but ctx is done,
	// so the inner select picks ctx.Done().
	mc.simulate(Message{ID: "extra", ChannelID: "ch-1", AuthorID: "u1", Content: "extra"})

	// Drain and verify close.
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

func TestLiveDiscord(t *testing.T) {
	if os.Getenv("LIVE") != "1" || os.Getenv("DISCORD_TOKEN") == "" {
		t.Skip("requires LIVE=1 and DISCORD_TOKEN")
	}
	// A live integration test would go here using a real discordgo client.
	// For now this is a placeholder for manual integration testing.
	t.Log("live Discord test not yet implemented")
}
