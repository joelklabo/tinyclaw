package router

import (
	"testing"
	"time"
)

func TestDelegationTrackerHappyPath(t *testing.T) {
	dt := NewDelegationTracker(3, 10, time.Minute)
	for i := 0; i < 3; i++ {
		if err := dt.RecordHop("conv-1"); err != nil {
			t.Fatalf("hop %d: %v", i+1, err)
		}
	}
	if dt.HopCount("conv-1") != 3 {
		t.Fatalf("hop count = %d, want 3", dt.HopCount("conv-1"))
	}
}

func TestDelegationTrackerMaxHops(t *testing.T) {
	dt := NewDelegationTracker(2, 10, time.Minute)
	dt.RecordHop("conv-1")
	dt.RecordHop("conv-1")
	err := dt.RecordHop("conv-1")
	if err == nil {
		t.Fatal("expected error when exceeding max hops")
	}
}

func TestDelegationTrackerRateLimit(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	dt := NewDelegationTracker(100, 2, time.Minute)
	dt.now = func() time.Time { return now }

	dt.RecordHop("conv-1")
	dt.RecordHop("conv-1")
	err := dt.RecordHop("conv-1")
	if err == nil {
		t.Fatal("expected rate limit error")
	}
}

func TestDelegationTrackerRateLimitWindowExpiry(t *testing.T) {
	now := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	dt := NewDelegationTracker(100, 2, time.Minute)
	dt.now = func() time.Time { return now }

	dt.RecordHop("conv-1")
	dt.RecordHop("conv-1")

	// Move time past the window.
	now = now.Add(2 * time.Minute)
	if err := dt.RecordHop("conv-1"); err != nil {
		t.Fatalf("expected success after window expiry: %v", err)
	}
}

func TestDelegationTrackerReset(t *testing.T) {
	dt := NewDelegationTracker(2, 10, time.Minute)
	dt.RecordHop("conv-1")
	dt.RecordHop("conv-1")

	dt.Reset("conv-1")

	if dt.HopCount("conv-1") != 0 {
		t.Fatalf("hop count after reset = %d, want 0", dt.HopCount("conv-1"))
	}

	// Should be able to hop again after reset.
	if err := dt.RecordHop("conv-1"); err != nil {
		t.Fatalf("hop after reset: %v", err)
	}
}

func TestDelegationTrackerIndependentConversations(t *testing.T) {
	dt := NewDelegationTracker(2, 10, time.Minute)
	dt.RecordHop("conv-1")
	dt.RecordHop("conv-1")

	// conv-2 should be independent.
	if err := dt.RecordHop("conv-2"); err != nil {
		t.Fatalf("conv-2 should be independent: %v", err)
	}
	if dt.HopCount("conv-2") != 1 {
		t.Fatalf("conv-2 hop count = %d, want 1", dt.HopCount("conv-2"))
	}
}

func TestDelegationTrackerHopCountUnknown(t *testing.T) {
	dt := NewDelegationTracker(5, 10, time.Minute)
	if dt.HopCount("unknown") != 0 {
		t.Fatalf("hop count for unknown = %d, want 0", dt.HopCount("unknown"))
	}
}

func TestDelegationTrackerResetUnknown(t *testing.T) {
	dt := NewDelegationTracker(5, 10, time.Minute)
	// Should not panic.
	dt.Reset("unknown")
}
