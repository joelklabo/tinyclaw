package nostr

import (
	"context"

	gonostr "github.com/nbd-wtf/go-nostr"
)

// Client abstracts Nostr relay operations for testability.
type Client interface {
	Publish(ctx context.Context, event gonostr.Event) error
	Subscribe(ctx context.Context, filters gonostr.Filters) (<-chan *gonostr.Event, error)
	QuerySync(ctx context.Context, filter gonostr.Filter) ([]*gonostr.Event, error)
	Close() error
}

// LiveClient connects to Nostr relays using go-nostr.
type LiveClient struct {
	relays []*gonostr.Relay
	urls   []string
}

// NewLiveClient creates a LiveClient connected to the given relay URLs.
func NewLiveClient(ctx context.Context, urls []string) (*LiveClient, error) {
	var relays []*gonostr.Relay
	for _, url := range urls {
		relay, err := gonostr.RelayConnect(ctx, url)
		if err != nil {
			// Close any already-connected relays.
			for _, r := range relays {
				r.Close()
			}
			return nil, err
		}
		relays = append(relays, relay)
	}
	return &LiveClient{relays: relays, urls: urls}, nil
}

// Publish sends an event to all connected relays.
func (c *LiveClient) Publish(ctx context.Context, event gonostr.Event) error {
	var lastErr error
	for _, relay := range c.relays {
		if err := relay.Publish(ctx, event); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// Subscribe opens subscriptions on all relays and merges events into a single channel.
func (c *LiveClient) Subscribe(ctx context.Context, filters gonostr.Filters) (<-chan *gonostr.Event, error) {
	out := make(chan *gonostr.Event, 64)

	for _, relay := range c.relays {
		sub, err := relay.Subscribe(ctx, filters)
		if err != nil {
			close(out)
			return nil, err
		}
		go func() {
			for ev := range sub.Events {
				select {
				case out <- ev:
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		<-ctx.Done()
		close(out)
	}()

	return out, nil
}

// QuerySync queries events from the first relay.
func (c *LiveClient) QuerySync(ctx context.Context, filter gonostr.Filter) ([]*gonostr.Event, error) {
	if len(c.relays) == 0 {
		return nil, nil
	}
	return c.relays[0].QuerySync(ctx, filter)
}

// Close disconnects from all relays.
func (c *LiveClient) Close() error {
	for _, relay := range c.relays {
		relay.Close()
	}
	return nil
}
