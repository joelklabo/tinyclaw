package discord

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// session wraps the subset of discordgo.Session methods used by LiveClient.
type session interface {
	ChannelMessageSend(channelID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageEdit(channelID, messageID, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelTyping(channelID string, options ...discordgo.RequestOption) error
	AddHandler(handler interface{}) func()
	Close() error
}

// LiveClient wraps a discordgo.Session and implements Client.
type LiveClient struct {
	session  session
	botID    string
	removeFn func()
}

// compile-time check
var _ Client = (*LiveClient)(nil)

// NewLiveClient creates a LiveClient by opening a discordgo session.
func NewLiveClient(token string) (*LiveClient, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("discord: create session: %w", err)
	}
	s.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentMessageContent
	if err := s.Open(); err != nil {
		return nil, fmt.Errorf("discord: open session: %w", err)
	}
	return &LiveClient{session: s, botID: s.State.User.ID}, nil
}

// newLiveClientFromSession creates a LiveClient from an existing session (for testing).
func newLiveClientFromSession(s session, botID string) *LiveClient {
	return &LiveClient{session: s, botID: botID}
}

// SendMessage sends a message to a Discord channel.
func (c *LiveClient) SendMessage(channelID, content string) (string, error) {
	msg, err := c.session.ChannelMessageSend(channelID, content)
	if err != nil {
		return "", err
	}
	return msg.ID, nil
}

// EditMessage edits a message in a Discord channel.
func (c *LiveClient) EditMessage(channelID, messageID, content string) error {
	_, err := c.session.ChannelMessageEdit(channelID, messageID, content)
	return err
}

// ChannelTyping triggers a typing indicator in a Discord channel.
func (c *LiveClient) ChannelTyping(channelID string) error {
	return c.session.ChannelTyping(channelID)
}

// SubscribeMessages registers a handler for incoming Discord messages.
func (c *LiveClient) SubscribeMessages(handler func(msg Message)) error {
	c.removeFn = c.session.AddHandler(func(_ *discordgo.Session, m *discordgo.MessageCreate) {
		if m.Author.ID == c.botID {
			return
		}
		handler(Message{
			ID:        m.ID,
			ChannelID: m.ChannelID,
			AuthorID:  m.Author.ID,
			Content:   m.Content,
		})
	})
	return nil
}

// Close removes the message handler and closes the session.
func (c *LiveClient) Close() error {
	if c.removeFn != nil {
		c.removeFn()
	}
	return c.session.Close()
}
