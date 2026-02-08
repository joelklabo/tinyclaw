// Package orchestrator coordinates a single agent run through the pipeline.
package orchestrator

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// Rule maps a condition to a target profile.
type Rule struct {
	Channel string `json:"channel"`
	Prefix  string `json:"prefix"`
	Profile string `json:"profile"`
}

// Config holds the routing configuration.
type Config struct {
	Rules   []Rule `json:"rules"`
	Default string `json:"default"`
}

// BundleWriter is the interface the orchestrator requires for recording run data.
type BundleWriter interface {
	AppendJSONL(filename string, v any) error
	WriteFail(msg string) error
	Close(status string) error
}

// Params holds dependencies for creating an Orchestrator.
type Params struct {
	Transport plugin.Transport
	Harness   plugin.Harness
	Routing   Config
	Bundle    BundleWriter
	Logger    *slog.Logger
}

// Orchestrator coordinates a single run: route -> harness -> transport.
type Orchestrator struct {
	transport   plugin.Transport
	harness     plugin.Harness
	routing     Config
	bundle      BundleWriter
	logger      *slog.Logger
	nowFn       func() time.Time
	deltaBuffer strings.Builder
}

// New creates an Orchestrator with the given dependencies.
// A nil Logger uses a discard handler.
func New(p Params) *Orchestrator {
	logger := p.Logger
	if logger == nil {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	return &Orchestrator{
		transport: p.Transport,
		harness:   p.Harness,
		routing:   p.Routing,
		bundle:    p.Bundle,
		logger:    logger,
		nowFn:     time.Now,
	}
}

// Run processes a single inbound event through the full pipeline.
func (o *Orchestrator) Run(ctx context.Context, event plugin.InboundEvent, contextItems []plugin.ContextItem) error {
	// Route
	profile, err := o.route(event)
	if err != nil {
		return o.fail(fmt.Errorf("route: %w", err))
	}
	o.logPhase("routed")

	// Start harness
	req := plugin.RunRequest{
		Profile: profile,
		Event:   event,
		Context: contextItems,
	}
	eventCh, err := o.harness.Start(ctx, req)
	if err != nil {
		return o.fail(fmt.Errorf("harness start: %w", err))
	}

	// Stream harness events
	for re := range eventCh {
		if appendErr := o.bundle.AppendJSONL("frames.jsonl", re); appendErr != nil {
			o.logger.Warn("append frame", "err", appendErr.Error())
		}
		if err := o.mapToTransport(ctx, re); err != nil {
			return o.fail(fmt.Errorf("transport: %w", err))
		}
	}

	return nil
}

// route determines the target agent profile for an event.
func (o *Orchestrator) route(event plugin.InboundEvent) (string, error) {
	channel := event.ChannelID
	text := event.Content

	bestProfile := ""
	bestScore := 0

	for _, rule := range o.routing.Rules {
		if rule.Channel != "" && rule.Channel == channel && 2 > bestScore {
			bestProfile = rule.Profile
			bestScore = 2
		} else if rule.Prefix != "" && strings.HasPrefix(text, rule.Prefix) && 1 > bestScore {
			bestProfile = rule.Profile
			bestScore = 1
		}
	}

	if bestScore > 0 {
		return bestProfile, nil
	}

	if o.routing.Default != "" {
		return o.routing.Default, nil
	}

	return "", fmt.Errorf("router: no matching rule for event")
}

func (o *Orchestrator) logPhase(phase string) {
	entry := struct {
		Phase string `json:"phase"`
		Time  string `json:"time"`
	}{
		Phase: phase,
		Time:  o.nowFn().UTC().Format(time.RFC3339),
	}
	if err := o.bundle.AppendJSONL("phases.jsonl", entry); err != nil {
		o.logger.Warn("append phase", "err", err.Error())
	}
}

// fail records a failure and writes FAIL.md to the bundle.
func (o *Orchestrator) fail(err error) error {
	if wErr := o.bundle.WriteFail(err.Error()); wErr != nil {
		o.logger.Error("write fail", "err", wErr.Error())
	}
	if cErr := o.bundle.Close("fail"); cErr != nil {
		o.logger.Error("close bundle", "err", cErr.Error())
	}
	return err
}

// mapToTransport maps a RunEvent to a transport operation.
func (o *Orchestrator) mapToTransport(ctx context.Context, re plugin.RunEvent) error {
	switch re.Kind {
	case plugin.RunEventDelta:
		o.deltaBuffer.WriteString(re.Content)
		return nil
	case plugin.RunEventFinal:
		content := re.Content
		if content == "" && o.deltaBuffer.Len() > 0 {
			content = o.deltaBuffer.String()
		}
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind:    plugin.OutboundPost,
			Content: content,
		})
	case plugin.RunEventFault:
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind:    plugin.OutboundPost,
			Content: re.Message,
		})
	case plugin.RunEventStatus, plugin.RunEventTool:
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind: plugin.OutboundTyping,
		})
	default:
		return nil
	}
}
