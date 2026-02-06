package version

import "testing"

func TestVersionDefault(t *testing.T) {
	if Version != "0.0.0-dev" {
		t.Fatalf("expected 0.0.0-dev, got %s", Version)
	}
}
