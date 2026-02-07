package scenario

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/klabo/tinyclaw/internal/bundle"
	"github.com/klabo/tinyclaw/internal/orchestrator"
	"github.com/klabo/tinyclaw/internal/plugin"
)

// Runner executes scenarios and produces bundles.
type Runner struct {
	newBundle    func(baseDir, id, scenario string, opts ...bundle.Option) (*bundle.Writer, error)
	newTransport func([]plugin.InboundEvent) *scriptedTransport
	newHarness   func([]plugin.RunEvent) *scriptedHarness
	Logger       *slog.Logger
}

// NewRunner creates a Runner with default dependencies.
func NewRunner() *Runner {
	return &Runner{
		newBundle: func(baseDir, id, scenario string, opts ...bundle.Option) (*bundle.Writer, error) {
			return bundle.NewWriter(baseDir, id, scenario, opts...)
		},
		newTransport: newScriptedTransport,
		newHarness:   newScriptedHarness,
	}
}

// RunScenario runs a named scenario and writes results to a bundle under baseDir.
// Returns the bundle directory path, the recorded transport ops, and any error.
func (r *Runner) RunScenario(baseDir string, sc *Scenario) (string, []plugin.OutboundOp, error) {
	w, err := r.newBundle(baseDir, sc.Name, sc.Name)
	if err != nil {
		return "", nil, fmt.Errorf("scenario: create bundle: %w", err)
	}

	// Convert scenario inbound events to plugin events.
	inbound := make([]plugin.InboundEvent, len(sc.InboundEvents))
	for i, e := range sc.InboundEvents {
		inbound[i] = plugin.InboundEvent{
			Type:      e.Type,
			Content:   e.Content,
			ChannelID: e.ChannelID,
			AuthorID:  e.AuthorID,
		}
	}

	// Create plugins.
	tr := r.newTransport(inbound)
	hr := r.newHarness(sc.HarnessEvents)
	routing := orchestrator.Config{Default: "default"}

	// Run the orchestrator for each inbound event.
	for _, event := range inbound {
		orch := orchestrator.New(orchestrator.Params{
			Transport: tr,
			Harness:   hr,
			Routing:   routing,
			Bundle:    w,
			Logger:    r.Logger,
		})
		if err := orch.Run(context.Background(), event, nil); err != nil {
			return w.Dir(), tr.Ops(), fmt.Errorf("scenario: run: %w", err)
		}
	}

	if err := w.Close("pass"); err != nil {
		return w.Dir(), tr.Ops(), fmt.Errorf("scenario: close bundle: %w", err)
	}
	return w.Dir(), tr.Ops(), nil
}
