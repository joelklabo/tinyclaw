// Package ratelimit implements a sliding window rate limiter per user.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter implements a sliding window rate limiter per user.
type Limiter struct {
	mu         sync.Mutex
	windows    map[string][]time.Time
	maxPerUser int
	window     time.Duration
	nowFn      func() time.Time
}

// New creates a Limiter. If maxPerUser <= 0 or window <= 0, returns nil.
func New(maxPerUser int, window time.Duration) *Limiter {
	if maxPerUser <= 0 || window <= 0 {
		return nil
	}
	return &Limiter{
		windows:    make(map[string][]time.Time),
		maxPerUser: maxPerUser,
		window:     window,
		nowFn:      time.Now,
	}
}

// Allow returns true if the user is under the rate limit, and records the request.
func (l *Limiter) Allow(userID string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.nowFn()
	cutoff := now.Add(-l.window)

	// Prune old entries
	timestamps := l.windows[userID]
	var fresh []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			fresh = append(fresh, ts)
		}
	}

	if len(fresh) >= l.maxPerUser {
		l.windows[userID] = fresh
		return false
	}

	l.windows[userID] = append(fresh, now)
	return true
}
