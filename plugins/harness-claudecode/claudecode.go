// Package harnessclaudecode implements a Harness that drives Claude Code
// via stream-json output parsing. The actual CLI invocation is behind the
// Runner interface for testability.
package harnessclaudecode

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Runner is the interface for executing Claude Code (allows mocking).
type Runner interface {
	// Run invokes Claude Code and returns the raw stream-json lines.
	Run(ctx context.Context, prompt string) ([]byte, error)
}

// Harness implements plugin.Harness using Claude Code.
type Harness struct {
	runner Runner
}

// New creates a new Claude Code harness.
func New(runner Runner) (*Harness, error) {
	if runner == nil {
		return nil, fmt.Errorf("harness-claudecode: runner must not be nil")
	}
	return &Harness{runner: runner}, nil
}

// streamEvent is a single line from Claude Code's stream-json output.
type streamEvent struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
	Error   string          `json:"error,omitempty"`
	Tool    string          `json:"tool,omitempty"`
	Message string          `json:"message,omitempty"`
}

// Start runs Claude Code and emits parsed RunEvents on the returned channel.
func (h *Harness) Start(ctx context.Context, req plugin.RunRequest) (<-chan plugin.RunEvent, error) {
	prompt := req.Scenario
	if ev, ok := req.Event.Data["content"].(string); ok && ev != "" {
		prompt = ev
	}

	raw, err := h.runner.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("harness-claudecode: run failed: %w", err)
	}

	events, err := parseStream(raw)
	if err != nil {
		return nil, fmt.Errorf("harness-claudecode: parse failed: %w", err)
	}

	ch := make(chan plugin.RunEvent)
	go func() {
		defer close(ch)
		for _, ev := range events {
			select {
			case <-ctx.Done():
				return
			case ch <- ev:
			}
		}
	}()
	return ch, nil
}

// Close is a no-op for the Claude Code harness.
func (h *Harness) Close() error {
	return nil
}

// authFailurePatterns are error strings that indicate auth or quota failures.
var authFailurePatterns = []string{
	"invalid api key",
	"authentication failed",
	"quota exceeded",
	"rate limit",
	"billing",
	"unauthorized",
	"forbidden",
}

// parseStream parses stream-json output into RunEvents.
func parseStream(data []byte) ([]plugin.RunEvent, error) {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return nil, fmt.Errorf("empty stream data")
	}
	lines := strings.Split(trimmed, "\n")

	var events []plugin.RunEvent
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var se streamEvent
		if err := json.Unmarshal([]byte(line), &se); err != nil {
			return nil, fmt.Errorf("invalid stream line: %w", err)
		}

		ev := convertEvent(se)
		events = append(events, ev)
	}

	return events, nil
}

// convertEvent maps a streamEvent to a plugin.RunEvent.
func convertEvent(se streamEvent) plugin.RunEvent {
	switch se.Type {
	case "status":
		return plugin.RunEvent{
			Kind: "status",
			Data: map[string]any{"phase": jsonString(se.Content)},
		}

	case "content", "delta":
		return plugin.RunEvent{
			Kind: "delta",
			Data: map[string]any{"content": jsonString(se.Content)},
		}

	case "tool_use":
		return plugin.RunEvent{
			Kind: "tool",
			Data: map[string]any{"tool": se.Tool, "message": se.Message},
		}

	case "error":
		if isAuthFailure(se.Error) {
			return plugin.RunEvent{
				Kind: "fault",
				Data: map[string]any{"kind": "auth", "message": se.Error},
			}
		}
		return plugin.RunEvent{
			Kind: "fault",
			Data: map[string]any{"kind": "error", "message": se.Error},
		}

	case "result":
		return plugin.RunEvent{
			Kind: "final",
			Data: map[string]any{"content": jsonString(se.Content)},
		}

	default:
		return plugin.RunEvent{
			Kind: "status",
			Data: map[string]any{"type": se.Type, "raw": string(se.Content)},
		}
	}
}

// isAuthFailure checks if an error message matches known auth/quota failure patterns.
func isAuthFailure(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	for _, pat := range authFailurePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

// jsonString extracts a JSON string value from raw JSON, or returns the raw text.
func jsonString(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return string(raw)
	}
	return s
}
