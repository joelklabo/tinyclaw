// Package nostr implements a Transport backed by Nostr relays.
package nostr

import (
	"encoding/json"
	"fmt"
	"strconv"

	gonostr "github.com/nbd-wtf/go-nostr"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Nostr event kinds for AI conversations.
const (
	KindPrompt   = 5800
	KindResponse = 5801
	KindToolCall = 5802
	KindError    = 5803
	KindStatus   = 25800
	KindDelta    = 25801
)

// PromptContent is the JSON content of a kind:5800 event.
type PromptContent struct {
	Message  string `json:"message"`
	Thinking string `json:"thinking,omitempty"`
}

// ResponseContent is the JSON content of a kind:5801 event.
type ResponseContent struct {
	Text  string         `json:"text"`
	Usage map[string]any `json:"usage,omitempty"`
}

// StatusContent is the JSON content of a kind:25800 event.
type StatusContent struct {
	State string `json:"state"`
}

// DeltaContent is the JSON content of a kind:25801 event.
type DeltaContent struct {
	Text string `json:"text"`
	Seq  int    `json:"seq"`
}

// ToolCallContent is the JSON content of a kind:5802 event.
type ToolCallContent struct {
	Name   string         `json:"name"`
	Phase  string         `json:"phase"`
	Args   map[string]any `json:"args,omitempty"`
	Output string         `json:"output,omitempty"`
}

// ErrorContent is the JSON content of a kind:5803 event.
type ErrorContent struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
}

// BaseTags returns the common tags for all events in a conversation.
func BaseTags(recipientPubkey, runID, sessionKey string) gonostr.Tags {
	return gonostr.Tags{
		{"p", recipientPubkey},
		{"r", runID},
		{"s", sessionKey},
	}
}

// ResponseTags returns tags for response events (5801, 5802, 5803, 25800, 25801)
// which reference the root prompt event.
func ResponseTags(recipientPubkey, runID, sessionKey, promptEventID string) gonostr.Tags {
	tags := BaseTags(recipientPubkey, runID, sessionKey)
	tags = append(tags, gonostr.Tag{"e", promptEventID, "", "root"})
	return tags
}

// EncodeOutbound converts an OutboundOp to a Nostr event. The event is not
// signed; the caller must set PubKey and call Sign().
func EncodeOutbound(op plugin.OutboundOp, recipientPubkey, runID, sessionKey, promptEventID string) (gonostr.Event, error) {
	tags := ResponseTags(recipientPubkey, runID, sessionKey, promptEventID)

	var kind int
	var content []byte
	var err error

	switch op.Kind {
	case plugin.OutboundStatus:
		kind = KindStatus
		tags = append(tags, gonostr.Tag{"state", op.Phase})
		content, err = json.Marshal(StatusContent{State: op.Phase})

	case plugin.OutboundDelta:
		kind = KindDelta
		tags = append(tags, gonostr.Tag{"seq", strconv.Itoa(op.Seq)})
		content, err = json.Marshal(DeltaContent{Text: op.Content, Seq: op.Seq})

	case plugin.OutboundTool:
		kind = KindToolCall
		tags = append(tags, gonostr.Tag{"tool", op.Tool})
		tags = append(tags, gonostr.Tag{"phase", "start"})
		content, err = json.Marshal(ToolCallContent{Name: op.Tool, Phase: "start"})

	case plugin.OutboundResponse:
		kind = KindResponse
		content, err = json.Marshal(ResponseContent{Text: op.Content})

	case plugin.OutboundError:
		kind = KindError
		faultKind := op.Fault
		if faultKind == "" {
			faultKind = "fatal"
		}
		tags = append(tags, gonostr.Tag{"error_kind", faultKind})
		content, err = json.Marshal(ErrorContent{Kind: faultKind, Message: op.Content})

	default:
		return gonostr.Event{}, fmt.Errorf("nostr: unsupported outbound op kind %q", op.Kind)
	}

	if err != nil {
		return gonostr.Event{}, fmt.Errorf("nostr: marshal content: %w", err)
	}

	return gonostr.Event{
		Kind:      kind,
		Tags:      tags,
		Content:   string(content),
		CreatedAt: gonostr.Now(),
	}, nil
}

// EncodePrompt creates a kind:5800 event from a user message.
func EncodePrompt(message, thinking, recipientPubkey, runID, sessionKey string) (gonostr.Event, error) {
	tags := BaseTags(recipientPubkey, runID, sessionKey)
	content, err := json.Marshal(PromptContent{Message: message, Thinking: thinking})
	if err != nil {
		return gonostr.Event{}, fmt.Errorf("nostr: marshal prompt: %w", err)
	}
	return gonostr.Event{
		Kind:      KindPrompt,
		Tags:      tags,
		Content:   string(content),
		CreatedAt: gonostr.Now(),
	}, nil
}

// DecodeInbound converts a kind:5800 Nostr event to an InboundEvent.
func DecodeInbound(ev *gonostr.Event) (plugin.InboundEvent, error) {
	if ev.Kind != KindPrompt {
		return plugin.InboundEvent{}, fmt.Errorf("nostr: expected kind %d, got %d", KindPrompt, ev.Kind)
	}
	var pc PromptContent
	if err := json.Unmarshal([]byte(ev.Content), &pc); err != nil {
		return plugin.InboundEvent{}, fmt.Errorf("nostr: unmarshal prompt: %w", err)
	}
	sessionKey := getTagValue(ev.Tags, "s")
	return plugin.InboundEvent{
		Type:      plugin.InboundMessage,
		Content:   pc.Message,
		ChannelID: sessionKey,
		AuthorID:  ev.PubKey,
		MessageID: ev.ID,
	}, nil
}

// DecodeOutbound converts a Nostr event back to an OutboundOp (for testing/verification).
func DecodeOutbound(ev *gonostr.Event) (plugin.OutboundOp, error) {
	switch ev.Kind {
	case KindStatus:
		var sc StatusContent
		if err := json.Unmarshal([]byte(ev.Content), &sc); err != nil {
			return plugin.OutboundOp{}, fmt.Errorf("nostr: unmarshal status: %w", err)
		}
		return plugin.OutboundOp{
			Kind:  plugin.OutboundStatus,
			Phase: sc.State,
		}, nil

	case KindDelta:
		var dc DeltaContent
		if err := json.Unmarshal([]byte(ev.Content), &dc); err != nil {
			return plugin.OutboundOp{}, fmt.Errorf("nostr: unmarshal delta: %w", err)
		}
		return plugin.OutboundOp{
			Kind:    plugin.OutboundDelta,
			Content: dc.Text,
			Seq:     dc.Seq,
		}, nil

	case KindToolCall:
		var tc ToolCallContent
		if err := json.Unmarshal([]byte(ev.Content), &tc); err != nil {
			return plugin.OutboundOp{}, fmt.Errorf("nostr: unmarshal tool_call: %w", err)
		}
		return plugin.OutboundOp{
			Kind: plugin.OutboundTool,
			Tool: tc.Name,
		}, nil

	case KindResponse:
		var rc ResponseContent
		if err := json.Unmarshal([]byte(ev.Content), &rc); err != nil {
			return plugin.OutboundOp{}, fmt.Errorf("nostr: unmarshal response: %w", err)
		}
		return plugin.OutboundOp{
			Kind:    plugin.OutboundResponse,
			Content: rc.Text,
		}, nil

	case KindError:
		var ec ErrorContent
		if err := json.Unmarshal([]byte(ev.Content), &ec); err != nil {
			return plugin.OutboundOp{}, fmt.Errorf("nostr: unmarshal error: %w", err)
		}
		return plugin.OutboundOp{
			Kind:    plugin.OutboundError,
			Content: ec.Message,
			Fault:   ec.Kind,
		}, nil

	default:
		return plugin.OutboundOp{}, fmt.Errorf("nostr: unsupported event kind %d", ev.Kind)
	}
}

func getTagValue(tags gonostr.Tags, key string) string {
	for _, tag := range tags {
		if len(tag) >= 2 && tag[0] == key {
			return tag[1]
		}
	}
	return ""
}
