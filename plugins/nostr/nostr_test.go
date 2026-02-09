package nostr

import (
	"context"
	"fmt"
	"sync"
	"testing"

	gonostr "github.com/nbd-wtf/go-nostr"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// --- mock client ---

type mockClient struct {
	mu          sync.Mutex
	published   []gonostr.Event
	publishErr  error
	subEvents   []*gonostr.Event
	subErr      error
	queryEvents []*gonostr.Event
	queryErr    error
	closed      bool
}

func (m *mockClient) Publish(_ context.Context, event gonostr.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, event)
	return nil
}

func (m *mockClient) Subscribe(_ context.Context, _ gonostr.Filters) (<-chan *gonostr.Event, error) {
	if m.subErr != nil {
		return nil, m.subErr
	}
	ch := make(chan *gonostr.Event, len(m.subEvents))
	for _, ev := range m.subEvents {
		ch <- ev
	}
	close(ch)
	return ch, nil
}

func (m *mockClient) QuerySync(_ context.Context, _ gonostr.Filter) ([]*gonostr.Event, error) {
	return m.queryEvents, m.queryErr
}

func (m *mockClient) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// testPrivateKey is a deterministic key for tests.
var testPrivateKey = gonostr.GeneratePrivateKey()

func newTestTransport(client Client) *Transport {
	tr, err := New(client, testPrivateKey, "test-session")
	if err != nil {
		panic(err)
	}
	return tr
}

// --- constructor tests ---

func TestNewNilClient(t *testing.T) {
	_, err := New(nil, testPrivateKey, "s")
	if err == nil {
		t.Fatal("expected error for nil client")
	}
}

func TestNewEmptyKey(t *testing.T) {
	_, err := New(&mockClient{}, "", "s")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestNewValidKey(t *testing.T) {
	tr, err := New(&mockClient{}, testPrivateKey, "s")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tr.publicKey == "" {
		t.Fatal("expected non-empty public key")
	}
}

// --- subscribe tests ---

func TestSubscribeDecodesPromptEvents(t *testing.T) {
	promptEv, _ := EncodePrompt("hello", "low", "anypk", "run-1", "test-session")
	promptEv.PubKey = "sender-pk"
	promptEv.ID = "ev-1"

	client := &mockClient{
		subEvents: []*gonostr.Event{&promptEv},
	}
	tr := newTestTransport(client)

	ch, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	var events []plugin.InboundEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].Content != "hello" {
		t.Fatalf("content = %q, want %q", events[0].Content, "hello")
	}
	if events[0].AuthorID != "sender-pk" {
		t.Fatalf("author = %q, want %q", events[0].AuthorID, "sender-pk")
	}
}

func TestSubscribeSkipsMalformedEvents(t *testing.T) {
	badEv := &gonostr.Event{Kind: KindPrompt, Content: "not json"}
	goodEv, _ := EncodePrompt("ok", "", "pk", "run-1", "test-session")
	goodEv.PubKey = "pk"
	goodEv.ID = "ev-2"

	client := &mockClient{
		subEvents: []*gonostr.Event{badEv, &goodEv},
	}
	tr := newTestTransport(client)

	ch, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	var events []plugin.InboundEvent
	for ev := range ch {
		events = append(events, ev)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event (bad one skipped), got %d", len(events))
	}
}

func TestSubscribeError(t *testing.T) {
	client := &mockClient{subErr: fmt.Errorf("relay down")}
	tr := newTestTransport(client)

	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestDoubleSubscribe(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)

	_, err := tr.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("first subscribe: %v", err)
	}
	_, err = tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error on double subscribe")
	}
}

func TestSubscribeAfterClose(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	_ = tr.Close()

	_, err := tr.Subscribe(context.Background())
	if err == nil {
		t.Fatal("expected error subscribing after close")
	}
}

// --- post tests ---

func TestPostStatus(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("prompt-1", "run-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind:  plugin.OutboundStatus,
		Phase: "thinking",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(client.published))
	}
	if client.published[0].Kind != KindStatus {
		t.Fatalf("published kind = %d, want %d", client.published[0].Kind, KindStatus)
	}
}

func TestPostDelta(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("prompt-1", "run-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind:    plugin.OutboundDelta,
		Content: "chunk",
		Seq:     5,
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.published) != 1 {
		t.Fatalf("expected 1 published, got %d", len(client.published))
	}
	if client.published[0].Kind != KindDelta {
		t.Fatalf("kind = %d, want %d", client.published[0].Kind, KindDelta)
	}
}

func TestPostResponse(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("prompt-1", "run-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind:    plugin.OutboundResponse,
		Content: "final answer",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.published[0].Kind != KindResponse {
		t.Fatalf("kind = %d, want %d", client.published[0].Kind, KindResponse)
	}
}

func TestPostError(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("prompt-1", "run-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind:    plugin.OutboundError,
		Content: "boom",
		Fault:   "auth",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.published[0].Kind != KindError {
		t.Fatalf("kind = %d, want %d", client.published[0].Kind, KindError)
	}
}

func TestPostTool(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("prompt-1", "run-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind: plugin.OutboundTool,
		Tool: "bash",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	if client.published[0].Kind != KindToolCall {
		t.Fatalf("kind = %d, want %d", client.published[0].Kind, KindToolCall)
	}
}

func TestPostPublishError(t *testing.T) {
	client := &mockClient{publishErr: fmt.Errorf("relay fail")}
	tr := newTestTransport(client)
	tr.SetRunContext("p-1", "r-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind:    plugin.OutboundResponse,
		Content: "hi",
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestPostAfterClose(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	_ = tr.Close()

	err := tr.Post(context.Background(), plugin.OutboundOp{Kind: plugin.OutboundResponse, Content: "x"})
	if err == nil {
		t.Fatal("expected error posting after close")
	}
}

func TestPostCancelledContext(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := tr.Post(ctx, plugin.OutboundOp{Kind: plugin.OutboundResponse, Content: "x"})
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// --- published event is signed ---

func TestPostEventIsSigned(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("p-1", "r-1")

	err := tr.Post(context.Background(), plugin.OutboundOp{
		Kind:    plugin.OutboundResponse,
		Content: "signed",
	})
	if err != nil {
		t.Fatalf("post: %v", err)
	}

	client.mu.Lock()
	defer client.mu.Unlock()
	ev := client.published[0]
	if ev.PubKey == "" {
		t.Fatal("expected non-empty PubKey")
	}
	if ev.Sig == "" {
		t.Fatal("expected non-empty Sig")
	}
	if ev.ID == "" {
		t.Fatal("expected non-empty ID")
	}
	ok, err := ev.CheckSignature()
	if err != nil {
		t.Fatalf("check sig: %v", err)
	}
	if !ok {
		t.Fatal("signature verification failed")
	}
}

// --- close tests ---

func TestClose(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)

	err := tr.Close()
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if !client.closed {
		t.Fatal("expected client to be closed")
	}
}

func TestDoubleClose(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	_ = tr.Close()

	err := tr.Close()
	if err == nil {
		t.Fatal("expected error on double close")
	}
}

// --- SetRunContext ---

func TestSetRunContext(t *testing.T) {
	client := &mockClient{}
	tr := newTestTransport(client)
	tr.SetRunContext("prompt-123", "run-456")

	// Post should use the run context for tags.
	_ = tr.Post(context.Background(), plugin.OutboundOp{
		Kind:  plugin.OutboundStatus,
		Phase: "thinking",
	})

	client.mu.Lock()
	defer client.mu.Unlock()
	if len(client.published) == 0 {
		t.Fatal("expected published event")
	}
	ev := client.published[0]
	runTag := getTagValue(ev.Tags, "r")
	if runTag != "run-456" {
		t.Fatalf("run tag = %q, want %q", runTag, "run-456")
	}
	eTag := getTagValue(ev.Tags, "e")
	if eTag != "prompt-123" {
		t.Fatalf("e tag = %q, want %q", eTag, "prompt-123")
	}
}
