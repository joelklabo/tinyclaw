package nostr

import (
	"context"
	"fmt"
	"sync"
	"testing"

	gonostr "github.com/nbd-wtf/go-nostr"
)

// --- mock relay ---

type mockRelay struct {
	mu         sync.Mutex
	published  []gonostr.Event
	publishErr error
	subEvents  []*gonostr.Event
	subErr     error
	queryEvts  []*gonostr.Event
	queryErr   error
	closed     bool
}

func (m *mockRelay) Publish(_ context.Context, event gonostr.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.publishErr != nil {
		return m.publishErr
	}
	m.published = append(m.published, event)
	return nil
}

func (m *mockRelay) Subscribe(ctx context.Context, _ gonostr.Filters, _ ...gonostr.SubscriptionOption) (*gonostr.Subscription, error) {
	if m.subErr != nil {
		return nil, m.subErr
	}
	ch := make(chan *gonostr.Event, len(m.subEvents))
	for _, ev := range m.subEvents {
		ch <- ev
	}
	close(ch)
	return &gonostr.Subscription{
		Events:  ch,
		Context: ctx,
	}, nil
}

func (m *mockRelay) QuerySync(_ context.Context, _ gonostr.Filter) ([]*gonostr.Event, error) {
	return m.queryEvts, m.queryErr
}

func (m *mockRelay) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

// withMockRelayConnect temporarily overrides relayConnectFunc for a test.
func withMockRelayConnect(fn func(ctx context.Context, url string) (relay, error)) func() {
	orig := relayConnectFunc
	relayConnectFunc = fn
	return func() { relayConnectFunc = orig }
}

// --- NewLiveClient tests ---

func TestNewLiveClient_SingleRelay(t *testing.T) {
	mr := &mockRelay{}
	restore := withMockRelayConnect(func(_ context.Context, url string) (relay, error) {
		if url != "wss://relay1" {
			t.Fatalf("unexpected url: %s", url)
		}
		return mr, nil
	})
	defer restore()

	c, err := NewLiveClient(context.Background(), []string{"wss://relay1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.relays) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(c.relays))
	}
}

func TestNewLiveClient_MultipleRelays(t *testing.T) {
	var connected []string
	restore := withMockRelayConnect(func(_ context.Context, url string) (relay, error) {
		connected = append(connected, url)
		return &mockRelay{}, nil
	})
	defer restore()

	c, err := NewLiveClient(context.Background(), []string{"wss://r1", "wss://r2", "wss://r3"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.relays) != 3 {
		t.Fatalf("expected 3 relays, got %d", len(c.relays))
	}
	if len(connected) != 3 {
		t.Fatalf("expected 3 connect calls, got %d", len(connected))
	}
}

func TestNewLiveClient_FirstRelayFails(t *testing.T) {
	restore := withMockRelayConnect(func(_ context.Context, _ string) (relay, error) {
		return nil, fmt.Errorf("connection refused")
	})
	defer restore()

	_, err := NewLiveClient(context.Background(), []string{"wss://bad"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewLiveClient_SecondRelayFails_ClosesFirst(t *testing.T) {
	first := &mockRelay{}
	call := 0
	restore := withMockRelayConnect(func(_ context.Context, _ string) (relay, error) {
		call++
		if call == 1 {
			return first, nil
		}
		return nil, fmt.Errorf("second relay down")
	})
	defer restore()

	_, err := NewLiveClient(context.Background(), []string{"wss://r1", "wss://r2"})
	if err == nil {
		t.Fatal("expected error")
	}
	first.mu.Lock()
	defer first.mu.Unlock()
	if !first.closed {
		t.Fatal("expected first relay to be closed on cleanup")
	}
}

func TestNewLiveClient_EmptyURLs(t *testing.T) {
	restore := withMockRelayConnect(func(_ context.Context, _ string) (relay, error) {
		t.Fatal("should not be called")
		return nil, nil
	})
	defer restore()

	c, err := NewLiveClient(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(c.relays) != 0 {
		t.Fatalf("expected 0 relays, got %d", len(c.relays))
	}
}

// --- Publish tests ---

func TestLiveClient_Publish_Success(t *testing.T) {
	mr := &mockRelay{}
	c := &LiveClient{relays: []relay{mr}}

	ev := gonostr.Event{Kind: 1, Content: "test"}
	err := c.Publish(context.Background(), ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	mr.mu.Lock()
	defer mr.mu.Unlock()
	if len(mr.published) != 1 {
		t.Fatalf("expected 1 published, got %d", len(mr.published))
	}
}

func TestLiveClient_Publish_MultipleRelays(t *testing.T) {
	r1 := &mockRelay{}
	r2 := &mockRelay{}
	c := &LiveClient{relays: []relay{r1, r2}}

	ev := gonostr.Event{Kind: 1, Content: "multi"}
	err := c.Publish(context.Background(), ev)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r1.mu.Lock()
	if len(r1.published) != 1 {
		t.Fatalf("r1: expected 1 published, got %d", len(r1.published))
	}
	r1.mu.Unlock()
	r2.mu.Lock()
	if len(r2.published) != 1 {
		t.Fatalf("r2: expected 1 published, got %d", len(r2.published))
	}
	r2.mu.Unlock()
}

func TestLiveClient_Publish_PartialFailure(t *testing.T) {
	r1 := &mockRelay{}
	r2 := &mockRelay{publishErr: fmt.Errorf("relay2 down")}
	c := &LiveClient{relays: []relay{r1, r2}}

	ev := gonostr.Event{Kind: 1, Content: "test"}
	err := c.Publish(context.Background(), ev)
	if err == nil {
		t.Fatal("expected error from partial failure")
	}
	// First relay should still have received the event.
	r1.mu.Lock()
	defer r1.mu.Unlock()
	if len(r1.published) != 1 {
		t.Fatalf("r1: expected 1 published, got %d", len(r1.published))
	}
}

func TestLiveClient_Publish_AllFail(t *testing.T) {
	r1 := &mockRelay{publishErr: fmt.Errorf("fail1")}
	r2 := &mockRelay{publishErr: fmt.Errorf("fail2")}
	c := &LiveClient{relays: []relay{r1, r2}}

	ev := gonostr.Event{Kind: 1, Content: "test"}
	err := c.Publish(context.Background(), ev)
	if err == nil {
		t.Fatal("expected error when all relays fail")
	}
}

func TestLiveClient_Publish_NoRelays(t *testing.T) {
	c := &LiveClient{relays: nil}
	err := c.Publish(context.Background(), gonostr.Event{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Subscribe tests ---

func TestLiveClient_Subscribe_Success(t *testing.T) {
	ev := &gonostr.Event{Kind: 1, Content: "hello"}
	mr := &mockRelay{subEvents: []*gonostr.Event{ev}}
	c := &LiveClient{relays: []relay{mr}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := c.Subscribe(ctx, gonostr.Filters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := <-ch
	if got.Content != "hello" {
		t.Fatalf("content = %q, want %q", got.Content, "hello")
	}
	cancel()
}

func TestLiveClient_Subscribe_Error(t *testing.T) {
	mr := &mockRelay{subErr: fmt.Errorf("sub failed")}
	c := &LiveClient{relays: []relay{mr}}

	_, err := c.Subscribe(context.Background(), gonostr.Filters{})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClient_Subscribe_NoRelays(t *testing.T) {
	c := &LiveClient{relays: nil}

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := c.Subscribe(ctx, gonostr.Filters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cancel()
	// Channel should close after context cancel.
	for range ch {
	}
}

// --- QuerySync tests ---

func TestLiveClient_QuerySync_Success(t *testing.T) {
	ev := &gonostr.Event{Kind: 1, Content: "found"}
	mr := &mockRelay{queryEvts: []*gonostr.Event{ev}}
	c := &LiveClient{relays: []relay{mr}}

	results, err := c.QuerySync(context.Background(), gonostr.Filter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Content != "found" {
		t.Fatalf("content = %q, want %q", results[0].Content, "found")
	}
}

func TestLiveClient_QuerySync_NoRelays(t *testing.T) {
	c := &LiveClient{relays: nil}

	results, err := c.QuerySync(context.Background(), gonostr.Filter{})
	if err != nil {
		t.Fatal("unexpected error")
	}
	if results != nil {
		t.Fatalf("expected nil results, got %v", results)
	}
}

func TestLiveClient_QuerySync_Error(t *testing.T) {
	mr := &mockRelay{queryErr: fmt.Errorf("query failed")}
	c := &LiveClient{relays: []relay{mr}}

	_, err := c.QuerySync(context.Background(), gonostr.Filter{})
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- Close tests ---

func TestLiveClient_Close(t *testing.T) {
	r1 := &mockRelay{}
	r2 := &mockRelay{}
	c := &LiveClient{relays: []relay{r1, r2}}

	err := c.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	r1.mu.Lock()
	if !r1.closed {
		t.Fatal("expected r1 to be closed")
	}
	r1.mu.Unlock()
	r2.mu.Lock()
	if !r2.closed {
		t.Fatal("expected r2 to be closed")
	}
	r2.mu.Unlock()
}

func TestLiveClient_Close_NoRelays(t *testing.T) {
	c := &LiveClient{relays: nil}
	err := c.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
