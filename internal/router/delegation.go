package router

import (
	"fmt"
	"sync"
	"time"
)

// DelegationTracker prevents delegation loops by tracking hop counts
// and enforcing rate limits per conversation.
type DelegationTracker struct {
	mu       sync.Mutex
	hops     map[string]int       // conversation ID -> hop count
	windows  map[string][]time.Time // conversation ID -> delegation timestamps
	maxHops  int
	maxRate  int           // max delegations per window
	window   time.Duration // rate limit window
	now      func() time.Time
}

// NewDelegationTracker creates a tracker with the given limits.
func NewDelegationTracker(maxHops, maxRate int, window time.Duration) *DelegationTracker {
	return &DelegationTracker{
		hops:    make(map[string]int),
		windows: make(map[string][]time.Time),
		maxHops: maxHops,
		maxRate: maxRate,
		window:  window,
		now:     time.Now,
	}
}

// RecordHop records a delegation hop for a conversation.
// Returns an error if the hop count exceeds the max or the rate limit is exceeded.
func (d *DelegationTracker) RecordHop(conversationID string) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Check hop count.
	count := d.hops[conversationID] + 1
	if count > d.maxHops {
		return fmt.Errorf("delegation: loop detected for %q (hop %d exceeds max %d)", conversationID, count, d.maxHops)
	}

	// Check rate limit.
	now := d.now()
	cutoff := now.Add(-d.window)
	timestamps := d.windows[conversationID]
	var recent []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			recent = append(recent, ts)
		}
	}
	if len(recent) >= d.maxRate {
		return fmt.Errorf("delegation: rate limit exceeded for %q (%d delegations in %v)", conversationID, len(recent), d.window)
	}

	d.hops[conversationID] = count
	d.windows[conversationID] = append(recent, now)
	return nil
}

// Reset clears all tracking state for a conversation.
func (d *DelegationTracker) Reset(conversationID string) {
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.hops, conversationID)
	delete(d.windows, conversationID)
}

// HopCount returns the current hop count for a conversation.
func (d *DelegationTracker) HopCount(conversationID string) int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.hops[conversationID]
}
