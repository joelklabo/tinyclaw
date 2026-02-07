package scenario

import (
	"fmt"
	"strings"

	"github.com/klabo/tinyclaw/internal/plugin"
)

// AssertOps compares actual transport ops against expected ops from a scenario.
func AssertOps(actual []plugin.OutboundOp, expected []ExpectedOp) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("op count mismatch: got %d, want %d", len(actual), len(expected))
	}
	var mismatches []string
	for i, exp := range expected {
		if actual[i].Kind != exp.Kind {
			mismatches = append(mismatches, fmt.Sprintf("op[%d]: got kind %q, want %q", i, actual[i].Kind, exp.Kind))
		}
		if exp.Content != "" && actual[i].Content != exp.Content {
			mismatches = append(mismatches, fmt.Sprintf("op[%d]: got content %q, want %q", i, actual[i].Content, exp.Content))
		}
	}
	if len(mismatches) > 0 {
		return fmt.Errorf("op mismatches:\n  %s", strings.Join(mismatches, "\n  "))
	}
	return nil
}
