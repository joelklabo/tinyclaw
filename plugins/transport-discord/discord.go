// Package transportdiscord implements a Transport backed by the Discord API.
// The actual Discord client is hidden behind the Client interface for testability.
package transportdiscord

import (
	"context"
	"fmt"
	"sync"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Client is the interface for Discord operations (allows mocking).
type Client interface {
	SendMessage(channelID, content string) (string, error)
	EditMessage(channelID, messageID, content string) error
	SubscribeMessages(handler func(msg Message)) error
	Close() error
}

// Message represents an inbound Discord message.
type Message struct {
	ID        string
	ChannelID string
	AuthorID  string
	Content   string
}

// Transport implements plugin.Transport using a Discord Client.
type Transport struct {
	client    Client
	channelID string

	mu         sync.Mutex
	closed     bool
	subscribed bool
}

// New creates a new Discord Transport for the given channel.
func New(client Client, channelID string) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("transport-discord: client must not be nil")
	}
	if channelID == "" {
		return nil, fmt.Errorf("transport-discord: channelID must not be empty")
	}
	return &Transport{client: client, channelID: channelID}, nil
}

// Subscribe returns a channel of inbound events from Discord messages.
func (t *Transport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport-discord: already closed")
	}
	if t.subscribed {
		t.mu.Unlock()
		return nil, fmt.Errorf("transport-discord: already subscribed")
	}
	t.subscribed = true
	t.mu.Unlock()

	out := make(chan plugin.InboundEvent, 64)
	internal := make(chan plugin.InboundEvent, 64)

	err := t.client.SubscribeMessages(func(msg Message) {
		if msg.ChannelID != t.channelID {
			return
		}
		ev := plugin.InboundEvent{
			Type: "message",
			Data: map[string]any{
				"id":         msg.ID,
				"channel_id": msg.ChannelID,
				"author_id":  msg.AuthorID,
				"content":    msg.Content,
			},
		}
		select {
		case <-ctx.Done():
		case internal <- ev:
		}
	})
	if err != nil {
		t.mu.Lock()
		t.subscribed = false
		t.mu.Unlock()
		return nil, fmt.Errorf("transport-discord: subscribe failed: %w", err)
	}

	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-internal:
				out <- ev
			}
		}
	}()

	return out, nil
}

// Post sends an outbound operation to Discord.
// Supported kinds: "post", "edit".
func (t *Transport) Post(ctx context.Context, op plugin.OutboundOp) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("transport-discord: already closed")
	}
	t.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	switch op.Kind {
	case "post":
		content, _ := op.Data["content"].(string)
		channelID, ok := op.Data["channel_id"].(string)
		if !ok || channelID == "" {
			channelID = t.channelID
		}
		_, err := t.client.SendMessage(channelID, content)
		return err

	case "edit":
		content, _ := op.Data["content"].(string)
		messageID, _ := op.Data["message_id"].(string)
		channelID, ok := op.Data["channel_id"].(string)
		if !ok || channelID == "" {
			channelID = t.channelID
		}
		if messageID == "" {
			return fmt.Errorf("transport-discord: edit requires message_id")
		}
		return t.client.EditMessage(channelID, messageID, content)

	default:
		return fmt.Errorf("transport-discord: unsupported op kind %q", op.Kind)
	}
}

// Close shuts down the transport and the underlying client.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("transport-discord: already closed")
	}
	t.closed = true
	return t.client.Close()
}
