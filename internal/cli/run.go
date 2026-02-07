package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/klabo/tinyclaw/internal/bundle"
	"github.com/klabo/tinyclaw/internal/scenario"
)

// RunTest loads a scenario and runs it through the system runner.
func RunTest(scenarioFile, configFile string) error {
	if scenarioFile == "" {
		return fmt.Errorf("scenario file is required")
	}

	cfg, err := Load(configFile)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	cfg = FromEnv(cfg)

	sc, err := scenario.LoadFile(scenarioFile)
	if err != nil {
		return fmt.Errorf("load scenario: %w", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: ParseLogLevel(cfg.LogLevel),
	}))

	runner := scenario.NewRunner()
	runner.Logger = logger
	dir, ops, err := runner.RunScenario(cfg.BundleDir, sc)
	if err != nil {
		return fmt.Errorf("run scenario: %w", err)
	}

	if err := scenario.AssertOps(ops, sc.ExpectedOps); err != nil {
		return fmt.Errorf("assertion failed: %w", err)
	}

	fmt.Printf("PASS: %s (bundle: %s)\n", sc.Name, dir)
	return nil
}

// RunReplay loads and validates a bundle.
func RunReplay(bundleDir string) error {
	info, err := bundle.LoadBundle(bundleDir)
	if err != nil {
		return fmt.Errorf("load bundle: %w", err)
	}

	if err := bundle.Validate(info); err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	fmt.Printf("Bundle: %s\n", info.Dir)
	fmt.Printf("  ID:       %s\n", info.Meta.ID)
	fmt.Printf("  Scenario: %s\n", info.Meta.Scenario)
	fmt.Printf("  Status:   %s\n", info.Meta.Status)
	fmt.Printf("  Files:    %d\n", len(info.Files))
	return nil
}
