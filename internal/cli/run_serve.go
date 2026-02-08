package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/klabo/tinyclaw/internal/orchestrator"
	"github.com/klabo/tinyclaw/internal/plugin"
	"github.com/klabo/tinyclaw/plugins/claudecode"
	"github.com/klabo/tinyclaw/plugins/discord"
	"github.com/klabo/tinyclaw/plugins/openclaw"
)

// connectFunc creates a Discord client and transport.
// Overridable for testing.
var connectFunc = defaultConnect

func defaultConnect(token, channelID string) (discord.Client, plugin.Transport, error) {
	client, err := discord.NewLiveClient(token)
	if err != nil {
		return nil, nil, fmt.Errorf("discord connect: %w", err)
	}
	transport, err := discord.New(client, channelID)
	if err != nil {
		_ = client.Close()
		return nil, nil, fmt.Errorf("discord transport: %w", err)
	}
	return client, transport, nil
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

	_, transport, err := connectFunc(cmd.Token, cmd.Channels[0])
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	params := buildServeParams(cmd, cfg, transport, logger)

	logger.Info("tinyclaw bot starting",
		"channels", cmd.Channels,
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

	var rules []orchestrator.Rule
	for _, ch := range cmd.Channels {
		rules = append(rules, orchestrator.Rule{
			Channel: ch,
			Profile: "default",
		})
	}

	return ServeParams{
		Transport: transport,
		NewHarness: func() (plugin.Harness, error) {
			runner := claudecode.NewExecRunner(cmd.WorkDir)
			return claudecode.New(runner)
		},
		Context:   provider,
		WorkDir:   cmd.WorkDir,
		BundleDir: cfg.BundleDir,
		Routing: orchestrator.Config{
			Default: "default",
			Rules:   rules,
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
