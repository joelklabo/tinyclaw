package discord

import (
	"fmt"
	"os"
	"testing"

	"github.com/bwmarrin/discordgo"
)

// mockSession implements the session interface for testing.
type mockSession struct {
	sendFn     func(channelID, content string) (*discordgo.Message, error)
	editFn     func(channelID, messageID, content string) (*discordgo.Message, error)
	typingFn   func(channelID string) error
	addHandler func(handler interface{}) func()
	closeFn    func() error
}

func (m *mockSession) ChannelMessageSend(channelID, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.sendFn != nil {
		return m.sendFn(channelID, content)
	}
	return &discordgo.Message{ID: "msg-1"}, nil
}

func (m *mockSession) ChannelMessageEdit(channelID, messageID, content string, _ ...discordgo.RequestOption) (*discordgo.Message, error) {
	if m.editFn != nil {
		return m.editFn(channelID, messageID, content)
	}
	return &discordgo.Message{ID: messageID}, nil
}

func (m *mockSession) ChannelTyping(channelID string, _ ...discordgo.RequestOption) error {
	if m.typingFn != nil {
		return m.typingFn(channelID)
	}
	return nil
}

func (m *mockSession) AddHandler(handler interface{}) func() {
	if m.addHandler != nil {
		return m.addHandler(handler)
	}
	return func() {}
}

func (m *mockSession) Close() error {
	if m.closeFn != nil {
		return m.closeFn()
	}
	return nil
}

func TestLiveClientSendMessage(t *testing.T) {
	ms := &mockSession{
		sendFn: func(ch, content string) (*discordgo.Message, error) {
			return &discordgo.Message{ID: "sent-1"}, nil
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")
	id, err := c.SendMessage("ch-1", "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != "sent-1" {
		t.Fatalf("got id %q, want %q", id, "sent-1")
	}
}

func TestLiveClientSendMessageError(t *testing.T) {
	ms := &mockSession{
		sendFn: func(string, string) (*discordgo.Message, error) {
			return nil, fmt.Errorf("send fail")
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")
	_, err := c.SendMessage("ch-1", "hello")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientEditMessage(t *testing.T) {
	ms := &mockSession{}
	c := newLiveClientFromSession(ms, "bot-1")
	err := c.EditMessage("ch-1", "msg-1", "updated")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLiveClientEditMessageError(t *testing.T) {
	ms := &mockSession{
		editFn: func(string, string, string) (*discordgo.Message, error) {
			return nil, fmt.Errorf("edit fail")
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")
	err := c.EditMessage("ch-1", "msg-1", "updated")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientChannelTyping(t *testing.T) {
	ms := &mockSession{}
	c := newLiveClientFromSession(ms, "bot-1")
	err := c.ChannelTyping("ch-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLiveClientChannelTypingError(t *testing.T) {
	ms := &mockSession{
		typingFn: func(string) error {
			return fmt.Errorf("typing fail")
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")
	err := c.ChannelTyping("ch-1")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestLiveClientSubscribeMessages(t *testing.T) {
	var capturedHandler interface{}
	ms := &mockSession{
		addHandler: func(handler interface{}) func() {
			capturedHandler = handler
			return func() {}
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")

	var received Message
	err := c.SubscribeMessages(func(msg Message) {
		received = msg
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedHandler == nil {
		t.Fatal("expected handler to be registered")
	}

	// Invoke the handler with a non-bot message.
	fn := capturedHandler.(func(*discordgo.Session, *discordgo.MessageCreate))
	fn(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "m1",
			ChannelID: "ch-1",
			Content:   "hello",
			Author:    &discordgo.User{ID: "user-1"},
		},
	})

	if received.ID != "m1" {
		t.Fatalf("got id %q, want %q", received.ID, "m1")
	}
	if received.Content != "hello" {
		t.Fatalf("got content %q, want %q", received.Content, "hello")
	}
}

func TestLiveClientSubscribeFiltersBotMessages(t *testing.T) {
	var capturedHandler interface{}
	ms := &mockSession{
		addHandler: func(handler interface{}) func() {
			capturedHandler = handler
			return func() {}
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")

	called := false
	_ = c.SubscribeMessages(func(msg Message) {
		called = true
	})

	fn := capturedHandler.(func(*discordgo.Session, *discordgo.MessageCreate))
	fn(nil, &discordgo.MessageCreate{
		Message: &discordgo.Message{
			ID:        "m1",
			ChannelID: "ch-1",
			Content:   "hello",
			Author:    &discordgo.User{ID: "bot-1"},
		},
	})

	if called {
		t.Fatal("handler should not be called for bot's own messages")
	}
}

func TestLiveClientClose(t *testing.T) {
	removeCalled := false
	ms := &mockSession{
		addHandler: func(interface{}) func() {
			return func() { removeCalled = true }
		},
	}
	c := newLiveClientFromSession(ms, "bot-1")
	_ = c.SubscribeMessages(func(msg Message) {})

	err := c.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !removeCalled {
		t.Fatal("expected remove handler to be called")
	}
}

func TestLiveClientCloseNoHandler(t *testing.T) {
	ms := &mockSession{}
	c := newLiveClientFromSession(ms, "bot-1")
	err := c.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLiveClientCloseError(t *testing.T) {
	ms := &mockSession{
		closeFn: func() error { return fmt.Errorf("close fail") },
	}
	c := newLiveClientFromSession(ms, "bot-1")
	err := c.Close()
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestNewLiveClientLive(t *testing.T) {
	if os.Getenv("LIVE") != "1" {
		t.Skip("set LIVE=1 to run live Discord client tests")
	}
	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		t.Skip("set DISCORD_TOKEN to run live Discord client tests")
	}
	c, err := NewLiveClient(token)
	if err != nil {
		t.Fatalf("NewLiveClient: %v", err)
	}
	defer c.Close()
	t.Logf("connected as bot ID: %s", c.botID)
}
