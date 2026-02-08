// Package discord implements a Transport backed by the Discord API.
package discord

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
	ChannelTyping(channelID string) error
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
	client         Client
	channels       map[string]bool
	defaultChannel string

	mu         sync.Mutex
	closed     bool
	subscribed bool
}

// New creates a new Discord Transport for the given channels.
func New(client Client, channelIDs ...string) (*Transport, error) {
	if client == nil {
		return nil, fmt.Errorf("discord: client must not be nil")
	}
	if len(channelIDs) == 0 {
		return nil, fmt.Errorf("discord: at least one channelID required")
	}
	channels := make(map[string]bool, len(channelIDs))
	for _, id := range channelIDs {
		if id == "" {
			return nil, fmt.Errorf("discord: channelID must not be empty")
		}
		channels[id] = true
	}
	return &Transport{
		client:         client,
		channels:       channels,
		defaultChannel: channelIDs[0],
	}, nil
}

// Subscribe returns a channel of inbound events from Discord messages.
func (t *Transport) Subscribe(ctx context.Context) (<-chan plugin.InboundEvent, error) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil, fmt.Errorf("discord: already closed")
	}
	if t.subscribed {
		t.mu.Unlock()
		return nil, fmt.Errorf("discord: already subscribed")
	}
	t.subscribed = true
	t.mu.Unlock()

	out := make(chan plugin.InboundEvent, 64)
	internal := make(chan plugin.InboundEvent, 64)

	err := t.client.SubscribeMessages(func(msg Message) {
		if !t.channels[msg.ChannelID] {
			return
		}
		ev := plugin.InboundEvent{
			Type:      plugin.InboundMessage,
			Content:   msg.Content,
			ChannelID: msg.ChannelID,
			AuthorID:  msg.AuthorID,
			MessageID: msg.ID,
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
		return nil, fmt.Errorf("discord: subscribe failed: %w", err)
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
func (t *Transport) Post(ctx context.Context, op plugin.OutboundOp) error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return fmt.Errorf("discord: already closed")
	}
	t.mu.Unlock()

	if err := ctx.Err(); err != nil {
		return err
	}

	switch op.Kind {
	case plugin.OutboundPost:
		content := op.Content
		channelID := op.ChannelID
		if channelID == "" {
			channelID = t.defaultChannel
		}
		chunks := chunk(content, 2000)
		if len(chunks) == 0 {
			_, err := t.client.SendMessage(channelID, content)
			return err
		}
		for _, c := range chunks {
			if _, err := t.client.SendMessage(channelID, c); err != nil {
				return err
			}
		}
		return nil

	case plugin.OutboundEdit:
		content := op.Content
		messageID := op.MessageID
		channelID := op.ChannelID
		if channelID == "" {
			channelID = t.defaultChannel
		}
		if messageID == "" {
			return fmt.Errorf("discord: edit requires message_id")
		}
		return t.client.EditMessage(channelID, messageID, content)

	case plugin.OutboundTyping:
		channelID := op.ChannelID
		if channelID == "" {
			channelID = t.defaultChannel
		}
		return t.client.ChannelTyping(channelID)

	default:
		return fmt.Errorf("discord: unsupported op kind %q", op.Kind)
	}
}

// Close shuts down the transport and the underlying client.
func (t *Transport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return fmt.Errorf("discord: already closed")
	}
	t.closed = true
	return t.client.Close()
}
