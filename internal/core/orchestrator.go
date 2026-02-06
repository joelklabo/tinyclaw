package core

import (
	"context"
	"fmt"

	"github.com/klabo/tinyclaw/internal/bundles"
	"github.com/klabo/tinyclaw/internal/plugin"
)

// Router routes an inbound event to an agent profile name.
type Router interface {
	Route(event plugin.InboundEvent) (string, error)
}

// ContextBuilder assembles context items for a run.
type ContextBuilder interface {
	Build(ctx context.Context, event plugin.InboundEvent) ([]plugin.ContextItem, error)
}

// Orchestrator coordinates a single run: transport -> router -> context -> harness -> transport.
type Orchestrator struct {
	transport plugin.Transport
	harness   plugin.Harness
	router    Router
	ctxBuild  ContextBuilder
	bundle    *bundles.Writer
}

// NewOrchestrator creates an Orchestrator with the given dependencies.
func NewOrchestrator(
	transport plugin.Transport,
	harness plugin.Harness,
	router Router,
	ctxBuild ContextBuilder,
	bundle *bundles.Writer,
) *Orchestrator {
	return &Orchestrator{
		transport: transport,
		harness:   harness,
		router:    router,
		ctxBuild:  ctxBuild,
		bundle:    bundle,
	}
}

// Run processes a single inbound event through the full pipeline.
// It advances the state machine, records transitions and events to the bundle,
// and streams harness events to the transport.
func (o *Orchestrator) Run(ctx context.Context, event plugin.InboundEvent) error {
	m := NewMachine()

	// Ingress -> Routed
	profile, err := o.router.Route(event)
	if err != nil {
		return o.fail(m, fmt.Errorf("route: %w", err))
	}
	advance(m, Routed)
	o.recordTransitions(m)

	_ = profile // will be used when routing is fully implemented

	// Routed -> ContextBuilt
	items, err := o.ctxBuild.Build(ctx, event)
	if err != nil {
		return o.fail(m, fmt.Errorf("context build: %w", err))
	}
	advance(m, ContextBuilt)
	o.recordTransitions(m)

	// ContextBuilt -> Running
	req := plugin.RunRequest{
		Event:   event,
		Context: items,
	}
	eventCh, err := o.harness.Start(ctx, req)
	if err != nil {
		return o.fail(m, fmt.Errorf("harness start: %w", err))
	}
	advance(m, Running)
	o.recordTransitions(m)

	// Stream harness events
	for re := range eventCh {
		o.bundle.AppendJSONL("frames.jsonl", re)
		if err := o.mapToTransport(ctx, re); err != nil {
			return o.fail(m, fmt.Errorf("transport: %w", err))
		}
	}

	// Running -> Completed
	advance(m, Completed)
	o.recordTransitions(m)
	o.bundle.Close("pass")
	return nil
}

// advance moves the machine to the given state, panicking on invalid
// transitions (which indicate a bug in the orchestrator, not a runtime error).
func advance(m *Machine, to State) {
	if err := m.Advance(to); err != nil {
		panic("orchestrator: " + err.Error())
	}
}

// fail records a failure and writes FAIL.md to the bundle.
func (o *Orchestrator) fail(m *Machine, err error) error {
	m.Advance(Failed)
	o.recordTransitions(m)
	o.bundle.WriteFail(err.Error())
	o.bundle.Close("fail")
	return err
}

// recordTransitions writes all transitions to the bundle.
func (o *Orchestrator) recordTransitions(m *Machine) {
	for _, t := range m.Transitions() {
		o.bundle.AppendJSONL("transitions.jsonl", t)
	}
}

// mapToTransport maps a RunEvent to a transport operation.
func (o *Orchestrator) mapToTransport(ctx context.Context, re plugin.RunEvent) error {
	switch re.Kind {
	case "delta":
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind: "edit",
			Data: re.Data,
		})
	case "final":
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind: "post",
			Data: re.Data,
		})
	case "fault":
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind: "post",
			Data: re.Data,
		})
	case "status":
		return o.transport.Post(ctx, plugin.OutboundOp{
			Kind: "typing",
			Data: re.Data,
		})
	default:
		return nil
	}
}
