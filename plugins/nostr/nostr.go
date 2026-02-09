package nostr

import (
	"context"
	"fmt"
	"sync"

	gonostr "github.com/nbd-wtf/go-nostr"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Transport implements plugin.Transport using Nostr relays.
type Transport struct {
	client     Client
	privateKey string
	publicKey  string
	sessionKey string

	mu         sync.Mutex
	closed     bool
	subscribed bool

	// promptEventID tracks the current prompt event for threading.
	promptEventID string
	runID         string
}

// New creates a Nostr Transport.
func New(client Client, privateKey, sessionKey string) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("nostr: client must not be nil")
	}
	if privateKey == "" {
		return nil, fmt.Errorf("nostr: private key must not be empty")
	}
	pubKey, err := gonostr.GetPublicKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("nostr: derive public key: %w", err)
	}
	return &Transport{
		client:     client,
		privateKey: privateKey,
		publicKey:  pubKey,
		sessionKey: sessionKey,
	}, nil
}

// Subscribe returns a channel of inbound events from kind:5800 Nostr events.
func (t *Transport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("nostr: already closed")
	}
	if t.subscribed {
		t.mu.Unlock()
		return nil, fmt.Errorf("nostr: already subscribed")
	}
	t.subscribed = true
	t.mu.Unlock()

	filters := gonostr.Filters{{
		Kinds: []int{KindPrompt},
		Tags: gonostr.TagMap{
			"p": {t.publicKey},
			"s": {t.sessionKey},
		},
		Since: func() *gonostr.Timestamp { ts := gonostr.Now(); return &ts }(),
	}}

	events, err := t.client.Subscribe(ctx, filters)
	if err != nil {
		t.mu.Lock()
		t.subscribed = false
		t.mu.Unlock()
		return nil, fmt.Errorf("nostr: subscribe: %w", err)
	}

	out := make(chan plugin.InboundEvent, 64)
	go func() {
		defer close(out)
		for ev := range events {
			inbound, err := DecodeInbound(ev)
			if err != nil {
				continue // skip malformed events
			}
			select {
			case out <- inbound:
			case <-ctx.Done():
				return
			}
		}
	}()

	return out, nil
}

// Post sends an outbound operation as a Nostr event.
func (t *Transport) Post(ctx context.Context, op plugin.OutboundOp) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("nostr: already closed")
	}
	promptID := t.promptEventID
	runID := t.runID
	t.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	ev, err := EncodeOutbound(op, t.publicKey, runID, t.sessionKey, promptID)
	if err != nil {
		return err
	}

	ev.PubKey = t.publicKey
	if err := ev.Sign(t.privateKey); err != nil {
		return fmt.Errorf("nostr: sign event: %w", err)
	}

	return t.client.Publish(ctx, ev)
}

// SetRunContext sets the current prompt event ID and run ID for threading responses.
func (t *Transport) SetRunContext(promptEventID, runID string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.promptEventID = promptEventID
	t.runID = runID
}

// Close shuts down the transport and the underlying client.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("nostr: already closed")
	}
	t.closed = true
	return t.client.Close()
}
