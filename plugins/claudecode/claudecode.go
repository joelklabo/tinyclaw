// Package claudecode implements a Harness that drives Claude Code
// via stream-json output parsing.
package claudecode

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Runner is the interface for executing Claude Code (allows mocking).
type Runner interface {
	Run(ctx context.Context, prompt string) (io.ReadCloser, error)
}

// Harness implements plugin.Harness using Claude Code.
type Harness struct {
	runner Runner
}

// New creates a new Claude Code harness.
func New(runner Runner) (*Harness, error) {
	if runner == nil {
		return nil, fmt.Errorf("claudecode: runner must not be nil")
	}
	return &Harness{runner: runner}, nil
}

// streamEvent is a single line from Claude Code's stream-json output.
type streamEvent struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content,omitempty"`
	Result  string          `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
	Tool    string          `json:"tool,omitempty"`
	Message json.RawMessage `json:"message,omitempty"`
}

const maxContextChars = 50000

func formatContext(items []plugin.ContextItem) string {
	if len(items) == 0 {
		return ""
	}
	sorted := make([]plugin.ContextItem, len(items))
	copy(sorted, items)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	var b strings.Builder
	b.WriteString("[Context]\n")
	total := 0
	for _, item := range sorted {
		entry := fmt.Sprintf("--- %s ---\n%s\n\n", item.Name, item.Content)
		if total+len(entry) > maxContextChars {
			break
		}
		b.WriteString(entry)
		total += len(entry)
	}
	return b.String()
}

// Start runs Claude Code and emits parsed RunEvents on the returned channel.
func (h *Harness) Start(ctx context.Context, req plugin.RunRequest) (<-chan plugin.RunEvent, error) {
	prompt := req.Scenario
	if req.Event.Content != "" {
		prompt = req.Event.Content
	}
	if ctxStr := formatContext(req.Context); ctxStr != "" {
		prompt = ctxStr + "[Message]\n" + prompt
	}

	reader, err := h.runner.Run(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("claudecode: run failed: %w", err)
	}

	ch := make(chan plugin.RunEvent)
	go func() {
		defer close(ch)
		defer reader.Close()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			line := bytes.TrimSpace(scanner.Bytes())
			if len(line) == 0 {
				continue
			}

			var se streamEvent
			if err := json.Unmarshal(line, &se); err != nil {
				ev := plugin.RunEvent{
					Kind:    plugin.RunEventFault,
					Fault:   "parse_error",
					Message: err.Error(),
				}
				select {
				case <-ctx.Done():
					return
				case ch <- ev:
				}
				continue
			}

			ev := convertEvent(se)
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

var authFailurePatterns = []string{
	"invalid api key",
	"authentication failed",
	"quota exceeded",
	"rate limit",
	"billing",
	"unauthorized",
	"forbidden",
}

func convertEvent(se streamEvent) plugin.RunEvent {
	switch se.Type {
	case "status":
		return plugin.RunEvent{
			Kind:  plugin.RunEventStatus,
			Phase: jsonString(se.Content),
		}

	case "content", "delta":
		return plugin.RunEvent{
			Kind:    plugin.RunEventDelta,
			Content: jsonString(se.Content),
		}

	case "tool_use":
		return plugin.RunEvent{
			Kind:    plugin.RunEventTool,
			Tool:    se.Tool,
			Message: jsonString(se.Message),
		}

	case "error":
		if isAuthFailure(se.Error) {
			return plugin.RunEvent{
				Kind:    plugin.RunEventFault,
				Fault:   "auth",
				Message: se.Error,
			}
		}
		return plugin.RunEvent{
			Kind:    plugin.RunEventFault,
			Fault:   "error",
			Message: se.Error,
		}

	case "result":
		content := se.Result
		if content == "" {
			content = jsonString(se.Content)
		}
		return plugin.RunEvent{
			Kind:    plugin.RunEventFinal,
			Content: content,
		}

	default:
		return plugin.RunEvent{
			Kind:  plugin.RunEventStatus,
			Phase: se.Type,
		}
	}
}

func isAuthFailure(errMsg string) bool {
	lower := strings.ToLower(errMsg)
	for _, pat := range authFailurePatterns {
		if strings.Contains(lower, pat) {
			return true
		}
	}
	return false
}

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
