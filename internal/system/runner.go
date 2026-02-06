// Package system runs offline system test scenarios and produces debug bundles.
package system

import (
	"fmt"

	"github.com/klabo/tinyclaw/internal/bundles"
)

// bundleWriter is the subset of bundles.Writer used by RunScenario.
type bundleWriter interface {
	Dir() string
	WriteFail(msg string) error
	Close(status string) error
}

// newBundleWriter is overridden in tests to inject failures.
var newBundleWriter = func(baseDir, id, scenario string) (bundleWriter, error) {
	return bundles.NewWriter(baseDir, id, scenario)
}

// RunScenario runs a named scenario and writes results to a bundle under baseDir.
// Returns the bundle directory path and any error.
func RunScenario(baseDir, scenario string) (string, error) {
	w, err := newBundleWriter(baseDir, scenario, scenario)
	if err != nil {
		return "", fmt.Errorf("system: create bundle: %w", err)
	}
	msg := "runner not yet implemented"
	if err := w.WriteFail(msg); err != nil {
		return "", fmt.Errorf("system: write fail: %w", err)
	}
	if err := w.Close("fail"); err != nil {
		return "", fmt.Errorf("system: close bundle: %w", err)
	}
	return w.Dir(), nil
}
