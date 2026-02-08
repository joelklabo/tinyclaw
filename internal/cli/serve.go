package cli

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/klabo/tinyclaw/internal/bundle"
	"github.com/klabo/tinyclaw/internal/memory"
	"github.com/klabo/tinyclaw/internal/orchestrator"
	"github.com/klabo/tinyclaw/internal/plugin"
	"github.com/klabo/tinyclaw/internal/ratelimit"
	"github.com/klabo/tinyclaw/plugins/openclaw"
)

// ServeParams holds the injected dependencies for RunServe.
type ServeParams struct {
	Transport   plugin.Transport
	NewHarness  func() (plugin.Harness, error)
	Context     *openclaw.Provider
	Memory      *memory.Store
	RateLimiter *ratelimit.Limiter
	WorkDir     string
	BundleDir   string
	Routing     orchestrator.Config
	Logger      *slog.Logger
	IDFunc      func() string
}

// RunServe runs the main event loop: subscribe to transport events,
// run each through the harness pipeline concurrently.
func RunServe(ctx context.Context, p ServeParams) error {
	logger := p.Logger
	if logger == nil {
		logger = slog.Default()
	}

	ch, err := p.Transport.Subscribe(ctx)
	if err != nil {
		return fmt.Errorf("serve: subscribe: %w", err)
	}

	const maxConcurrent = 5
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for event := range ch {
		sem <- struct{}{}
		wg.Add(1)
		go func(ev plugin.InboundEvent) {
			defer wg.Done()
			defer func() { <-sem }()
			handleEvent(ctx, ev, p, logger)
		}(event)
	}

	wg.Wait()
	return nil
}

func handleEvent(ctx context.Context, ev plugin.InboundEvent, p ServeParams, logger *slog.Logger) {
	// Rate limiting
	if p.RateLimiter != nil && !p.RateLimiter.Allow(ev.AuthorID) {
		_ = p.Transport.Post(ctx, plugin.OutboundOp{
			Kind:      plugin.OutboundPost,
			Content:   "You're sending messages too fast. Please wait a moment.",
			ChannelID: ev.ChannelID,
		})
		return
	}

	id := p.IDFunc()
	logger.Info("handling event", "run_id", id, "channel", ev.ChannelID)

	// Store user message in memory
	if p.Memory != nil {
		p.Memory.Append(ev.ChannelID, "user", ev.Content)
	}

	// Gather openclaw context (non-fatal on error).
	var items []plugin.ContextItem
	if p.Context != nil {
		var err error
		items, err = p.Context.Gather(ctx, p.WorkDir)
		if err != nil {
			logger.Warn("gather context", "err", err.Error())
		}
	}

	// Gather memory context
	if p.Memory != nil {
		entries := p.Memory.Recent(ev.ChannelID, 10)
		for _, e := range entries {
			items = append(items, plugin.ContextItem{
				Name:    fmt.Sprintf("memory_%s", e.Role),
				Content: e.Content,
				Source:  "memory",
			})
		}
	}

	bw, err := bundle.NewWriter(p.BundleDir, id, "live")
	if err != nil {
		logger.Error("create bundle", "err", err.Error())
		return
	}

	harness, err := p.NewHarness()
	if err != nil {
		logger.Error("create harness", "err", err.Error())
		_ = bw.WriteFail(err.Error())
		_ = bw.Close("fail")
		_ = p.Transport.Post(ctx, plugin.OutboundOp{
			Kind:      plugin.OutboundPost,
			Content:   "Sorry, something went wrong processing your message.",
			ChannelID: ev.ChannelID,
		})
		return
	}
	defer harness.Close()

	o := orchestrator.New(orchestrator.Params{
		Transport: p.Transport,
		Harness:   harness,
		Routing:   p.Routing,
		Bundle:    bw,
		Logger:    logger,
	})

	if err := o.Run(ctx, ev, items); err != nil {
		logger.Error("run", "run_id", id, "err", err.Error())
		_ = p.Transport.Post(ctx, plugin.OutboundOp{
			Kind:      plugin.OutboundPost,
			Content:   "Sorry, something went wrong processing your message.",
			ChannelID: ev.ChannelID,
		})
		return
	}

	if err := bw.Close("pass"); err != nil {
		logger.Warn("close bundle", "run_id", id, "err", err.Error())
	}

	logger.Info("run complete", "run_id", id)
}
