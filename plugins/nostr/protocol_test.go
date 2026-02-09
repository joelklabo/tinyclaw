package nostr

import (
	"testing"

	gonostr "github.com/nbd-wtf/go-nostr"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestEncodeDecodeStatus(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundStatus, Phase: "thinking"}
	ev, err := EncodeOutbound(op, "recipientpk", "run-1", "session-1", "prompt-ev-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if ev.Kind != KindStatus {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindStatus)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Kind != plugin.OutboundStatus {
		t.Fatalf("decoded kind = %q, want %q", decoded.Kind, plugin.OutboundStatus)
	}
	if decoded.Phase != "thinking" {
		t.Fatalf("decoded phase = %q, want %q", decoded.Phase, "thinking")
	}
}

func TestEncodeDecodeDelta(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundDelta, Content: "partial text", Seq: 42}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if ev.Kind != KindDelta {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindDelta)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Kind != plugin.OutboundDelta {
		t.Fatalf("decoded kind = %q, want %q", decoded.Kind, plugin.OutboundDelta)
	}
	if decoded.Content != "partial text" {
		t.Fatalf("decoded content = %q, want %q", decoded.Content, "partial text")
	}
	if decoded.Seq != 42 {
		t.Fatalf("decoded seq = %d, want %d", decoded.Seq, 42)
	}
}

func TestEncodeDecodeTool(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundTool, Tool: "bash"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if ev.Kind != KindToolCall {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindToolCall)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Kind != plugin.OutboundTool {
		t.Fatalf("decoded kind = %q, want %q", decoded.Kind, plugin.OutboundTool)
	}
	if decoded.Tool != "bash" {
		t.Fatalf("decoded tool = %q, want %q", decoded.Tool, "bash")
	}
}

func TestEncodeDecodeResponse(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundResponse, Content: "final answer"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if ev.Kind != KindResponse {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindResponse)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Kind != plugin.OutboundResponse {
		t.Fatalf("decoded kind = %q, want %q", decoded.Kind, plugin.OutboundResponse)
	}
	if decoded.Content != "final answer" {
		t.Fatalf("decoded content = %q, want %q", decoded.Content, "final answer")
	}
}

func TestEncodeDecodeError(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundError, Content: "auth failed", Fault: "auth"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if ev.Kind != KindError {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindError)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Kind != plugin.OutboundError {
		t.Fatalf("decoded kind = %q, want %q", decoded.Kind, plugin.OutboundError)
	}
	if decoded.Content != "auth failed" {
		t.Fatalf("decoded content = %q, want %q", decoded.Content, "auth failed")
	}
	if decoded.Fault != "auth" {
		t.Fatalf("decoded fault = %q, want %q", decoded.Fault, "auth")
	}
}

func TestEncodeDecodeErrorDefaultsFault(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundError, Content: "boom"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Fault != "fatal" {
		t.Fatalf("decoded fault = %q, want %q", decoded.Fault, "fatal")
	}
}

func TestEncodePromptDecodeInbound(t *testing.T) {
	ev, err := EncodePrompt("hello world", "low", "recipientpk", "run-1", "session-1")
	if err != nil {
		t.Fatalf("encode prompt: %v", err)
	}
	if ev.Kind != KindPrompt {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindPrompt)
	}
	// Simulate signing: set PubKey and ID.
	ev.PubKey = "userpubkey123"
	ev.ID = "event-id-123"

	inbound, err := DecodeInbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if inbound.Type != plugin.InboundMessage {
		t.Fatalf("type = %q, want %q", inbound.Type, plugin.InboundMessage)
	}
	if inbound.Content != "hello world" {
		t.Fatalf("content = %q, want %q", inbound.Content, "hello world")
	}
	if inbound.ChannelID != "session-1" {
		t.Fatalf("channel = %q, want %q", inbound.ChannelID, "session-1")
	}
	if inbound.AuthorID != "userpubkey123" {
		t.Fatalf("author = %q, want %q", inbound.AuthorID, "userpubkey123")
	}
	if inbound.MessageID != "event-id-123" {
		t.Fatalf("message id = %q, want %q", inbound.MessageID, "event-id-123")
	}
}

func TestDecodeInboundWrongKind(t *testing.T) {
	ev := makeTestEvent(KindResponse, `{"text":"hi"}`)
	_, err := DecodeInbound(&ev)
	if err == nil {
		t.Fatal("expected error for wrong kind")
	}
}

func TestDecodeInboundBadJSON(t *testing.T) {
	ev := makeTestEvent(KindPrompt, "not json")
	_, err := DecodeInbound(&ev)
	if err == nil {
		t.Fatal("expected error for bad JSON")
	}
}

func TestDecodeOutboundBadJSON(t *testing.T) {
	kinds := []int{KindStatus, KindDelta, KindToolCall, KindResponse, KindError}
	for _, k := range kinds {
		ev := makeTestEvent(k, "not json")
		_, err := DecodeOutbound(&ev)
		if err == nil {
			t.Fatalf("expected error for bad JSON on kind %d", k)
		}
	}
}

func TestDecodeOutboundUnknownKind(t *testing.T) {
	ev := makeTestEvent(9999, `{}`)
	_, err := DecodeOutbound(&ev)
	if err == nil {
		t.Fatal("expected error for unknown kind")
	}
}

func TestEncodeOutboundUnsupportedKind(t *testing.T) {
	op := plugin.OutboundOp{Kind: "bogus"}
	_, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err == nil {
		t.Fatal("expected error for unsupported kind")
	}
}

func TestBaseTags(t *testing.T) {
	tags := BaseTags("pk", "run-1", "session-1")
	if len(tags) != 3 {
		t.Fatalf("expected 3 tags, got %d", len(tags))
	}
	if tags[0][0] != "p" || tags[0][1] != "pk" {
		t.Errorf("tag[0] = %v", tags[0])
	}
	if tags[1][0] != "r" || tags[1][1] != "run-1" {
		t.Errorf("tag[1] = %v", tags[1])
	}
	if tags[2][0] != "s" || tags[2][1] != "session-1" {
		t.Errorf("tag[2] = %v", tags[2])
	}
}

func TestResponseTags(t *testing.T) {
	tags := ResponseTags("pk", "run-1", "session-1", "prompt-ev-1")
	if len(tags) != 4 {
		t.Fatalf("expected 4 tags, got %d", len(tags))
	}
	eTag := tags[3]
	if eTag[0] != "e" || eTag[1] != "prompt-ev-1" || eTag[3] != "root" {
		t.Errorf("e tag = %v", eTag)
	}
}

func TestEncodeStatusHasStateTags(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundStatus, Phase: "tool_use"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	found := getTagValue(ev.Tags, "state")
	if found != "tool_use" {
		t.Fatalf("state tag = %q, want %q", found, "tool_use")
	}
}

func TestEncodeDeltaHasSeqTag(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundDelta, Content: "x", Seq: 7}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	found := getTagValue(ev.Tags, "seq")
	if found != "7" {
		t.Fatalf("seq tag = %q, want %q", found, "7")
	}
}

func TestEncodeToolHasToolAndPhaseTags(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundTool, Tool: "read"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if getTagValue(ev.Tags, "tool") != "read" {
		t.Fatalf("tool tag = %q", getTagValue(ev.Tags, "tool"))
	}
	if getTagValue(ev.Tags, "phase") != "start" {
		t.Fatalf("phase tag = %q", getTagValue(ev.Tags, "phase"))
	}
}

func TestEncodeErrorHasErrorKindTag(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundError, Content: "x", Fault: "quota"}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	if getTagValue(ev.Tags, "error_kind") != "quota" {
		t.Fatalf("error_kind tag = %q", getTagValue(ev.Tags, "error_kind"))
	}
}

func TestEncodeDecodeEmptyContent(t *testing.T) {
	op := plugin.OutboundOp{Kind: plugin.OutboundResponse, Content: ""}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Content != "" {
		t.Fatalf("expected empty content, got %q", decoded.Content)
	}
}

func TestEncodeDecodeLongContent(t *testing.T) {
	long := make([]byte, 100000)
	for i := range long {
		long[i] = 'x'
	}
	op := plugin.OutboundOp{Kind: plugin.OutboundResponse, Content: string(long)}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Content != string(long) {
		t.Fatalf("content length = %d, want %d", len(decoded.Content), len(long))
	}
}

func TestEncodeDecodeSpecialChars(t *testing.T) {
	content := "Hello \"world\" \n\t 日本語 🎉 <script>alert('xss')</script>"
	op := plugin.OutboundOp{Kind: plugin.OutboundResponse, Content: content}
	ev, err := EncodeOutbound(op, "pk", "run-1", "s-1", "p-1")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	decoded, err := DecodeOutbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Content != content {
		t.Fatalf("decoded content = %q, want %q", decoded.Content, content)
	}
}

func TestEncodePromptEmptyMessage(t *testing.T) {
	ev, err := EncodePrompt("", "", "pk", "run-1", "session-1")
	if err != nil {
		t.Fatalf("encode prompt: %v", err)
	}
	if ev.Kind != KindPrompt {
		t.Fatalf("kind = %d, want %d", ev.Kind, KindPrompt)
	}
	ev.PubKey = "pk"
	ev.ID = "ev-empty"
	inbound, err := DecodeInbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if inbound.Content != "" {
		t.Fatalf("content = %q, want empty", inbound.Content)
	}
}

func TestEncodePromptSpecialCharacters(t *testing.T) {
	msg := "Hello \"world\" \n\t 日本語 🎉 <script>alert('xss')</script>"
	ev, err := EncodePrompt(msg, "think\ning", "pk", "run-1", "session-1")
	if err != nil {
		t.Fatalf("encode prompt: %v", err)
	}
	ev.PubKey = "pk"
	ev.ID = "ev-special"
	inbound, err := DecodeInbound(&ev)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if inbound.Content != msg {
		t.Fatalf("content = %q, want %q", inbound.Content, msg)
	}
}

// --- helpers ---

func makeTestEvent(kind int, content string) gonostr.Event {
	return gonostr.Event{Kind: kind, Content: content}
}
