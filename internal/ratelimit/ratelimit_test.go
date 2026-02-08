package ratelimit

import (
	"sync"
	"testing"
	"time"
)

func TestAllowUnderLimit(t *testing.T) {
	l := New(3, time.Minute)
	if !l.Allow("user-1") {
		t.Fatal("expected Allow to return true for first request")
	}
	if !l.Allow("user-1") {
		t.Fatal("expected Allow to return true for second request")
	}
	if !l.Allow("user-1") {
		t.Fatal("expected Allow to return true for third request")
	}
}

func TestAllowOverLimit(t *testing.T) {
	l := New(2, time.Minute)
	l.Allow("user-1")
	l.Allow("user-1")
	if l.Allow("user-1") {
		t.Fatal("expected Allow to return false when over limit")
	}
}

func TestAllowAfterWindowExpiry(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	l := New(2, time.Minute)
	l.nowFn = func() time.Time { return now }

	l.Allow("user-1")
	l.Allow("user-1")

	if l.Allow("user-1") {
		t.Fatal("expected false while window is full")
	}

	// Advance past the window
	now = now.Add(2 * time.Minute)

	if !l.Allow("user-1") {
		t.Fatal("expected true after window expiry")
	}
}

func TestNewInvalidParams(t *testing.T) {
	if l := New(0, time.Minute); l != nil {
		t.Fatal("expected nil for maxPerUser=0")
	}
	if l := New(-1, time.Minute); l != nil {
		t.Fatal("expected nil for maxPerUser=-1")
	}
	if l := New(5, 0); l != nil {
		t.Fatal("expected nil for window=0")
	}
	if l := New(5, -time.Second); l != nil {
		t.Fatal("expected nil for negative window")
	}
}

func TestMultipleUsers(t *testing.T) {
	l := New(1, time.Minute)
	if !l.Allow("user-1") {
		t.Fatal("expected true for user-1")
	}
	if !l.Allow("user-2") {
		t.Fatal("expected true for user-2")
	}
	if l.Allow("user-1") {
		t.Fatal("expected false for user-1 over limit")
	}
	if l.Allow("user-2") {
		t.Fatal("expected false for user-2 over limit")
	}
}

func TestConcurrentAccess(t *testing.T) {
	l := New(1000, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.Allow("user-1")
		}()
	}
	wg.Wait()
}

func TestPruneOldEntries(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	l := New(3, time.Minute)
	l.nowFn = func() time.Time { return now }

	l.Allow("user-1") // t=0
	l.Allow("user-1") // t=0

	// Advance 30 seconds — still within window
	now = now.Add(30 * time.Second)
	l.Allow("user-1") // t=30s — now at limit

	if l.Allow("user-1") {
		t.Fatal("expected false at limit")
	}

	// Advance 31 more seconds — first two entries expire (they were at t=0, window is 1min)
	now = now.Add(31 * time.Second)

	if !l.Allow("user-1") {
		t.Fatal("expected true after old entries pruned")
	}
}

func TestAllowExactlyAtLimit(t *testing.T) {
	l := New(1, time.Minute)
	if !l.Allow("user-1") {
		t.Fatal("expected true for first request")
	}
	if l.Allow("user-1") {
		t.Fatal("expected false for second request at limit")
	}
}
