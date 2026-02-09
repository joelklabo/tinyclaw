package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/klabo/tinyclaw/internal/memory"
	"github.com/klabo/tinyclaw/internal/orchestrator"
	"github.com/klabo/tinyclaw/internal/plugin"
	"github.com/klabo/tinyclaw/internal/ratelimit"
	"github.com/klabo/tinyclaw/plugins/claudecode"
	"github.com/klabo/tinyclaw/plugins/nostr"
	"github.com/klabo/tinyclaw/plugins/openclaw"
)

// connectFunc creates a Nostr transport.
// Overridable for testing.
var connectFunc = defaultConnect

func defaultConnect(privateKey, sessionKey string, relayURLs []string) (plugin.Transport, error) {
	ctx := context.Background()
	client, err := nostr.NewLiveClient(ctx, relayURLs)
	if err != nil {
		return nil, fmt.Errorf("nostr connect: %w", err)
	}
	transport, err := nostr.New(client, privateKey, sessionKey)
	if err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("nostr transport: %w", err)
	}
	return transport, nil
}

// RunBot wires real dependencies and runs the serve loop.
func RunBot(cmd Command) error {
	cfg, err := Load(cmd.ConfigFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg = FromEnv(cfg)

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: ParseLogLevel(cfg.LogLevel),
	}))

	transport, err := connectFunc(cmd.PrivateKey, cmd.SessionKey, cmd.Relays)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	params := buildServeParams(cmd, cfg, transport, logger)

	logger.Info("tinyclaw bot starting",
		"relays", cmd.Relays,
		"workdir", cmd.WorkDir,
		"bundle_dir", cfg.BundleDir,
	)

	err = RunServe(ctx, params)

	if closeErr := transport.Close(); closeErr != nil {
		logger.Warn("close transport", "err", closeErr.Error())
	}

	if err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	return nil
}

// buildServeParams creates the ServeParams from the command, config, and transport.
func buildServeParams(cmd Command, cfg Config, transport plugin.Transport, logger *slog.Logger) ServeParams {
	provider := openclaw.New(openclaw.Options{})

	mem := memory.New(time.Hour, 20)
	limiter := ratelimit.New(5, 60*time.Second)

	return ServeParams{
		Transport: transport,
		NewHarness: func() (plugin.Harness, error) {
			runner := claudecode.NewExecRunner(cmd.WorkDir)
			runner.SystemPrompt = cfg.SystemPrompt
			return claudecode.New(runner)
		},
		Context:     provider,
		Memory:      mem,
		RateLimiter: limiter,
		WorkDir:     cmd.WorkDir,
		BundleDir:   cfg.BundleDir,
		Routing: orchestrator.Config{
			Default: "default",
		},
		Logger: logger,
		IDFunc: newRunIDFunc(),
	}
}

func newRunIDFunc() func() string {
	var n int64
	return func() string {
		n++
		return fmt.Sprintf("live-%d", n)
	}
}

// parseRelayList splits a comma-separated list of relay URLs.
func parseRelayList(s string) []string {
	var relays []string
	for _, r := range strings.Split(s, ",") {
		r = strings.TrimSpace(r)
		if r != "" {
			relays = append(relays, r)
		}
	}
	return relays
}
