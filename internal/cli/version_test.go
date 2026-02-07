package cli

import "testing"

func TestVersionDefault(t *testing.T) {
	if Version != "0.0.0-dev" {
		t.Fatalf("expected Version %q, got %q", "0.0.0-dev", Version)
	}
}
