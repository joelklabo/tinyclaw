// Package memory provides per-channel conversation history storage.
package memory

import (
	"sync"
	"time"
)

// Entry represents a single message in a conversation.
type Entry struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// Store holds per-channel conversation history.
type Store struct {
	mu      sync.RWMutex
	convos  map[string][]Entry
	maxAge  time.Duration
	maxSize int
	nowFn   func() time.Time
}

// New creates a Store with the given TTL and max entries per channel.
func New(maxAge time.Duration, maxSize int) *Store {
	return &Store{
		convos:  make(map[string][]Entry),
		maxAge:  maxAge,
		maxSize: maxSize,
		nowFn:   time.Now,
	}
}

// Append adds an entry to the channel's conversation.
func (s *Store) Append(channelID, role, content string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.convos[channelID] = append(s.convos[channelID], Entry{
		Role:      role,
		Content:   content,
		Timestamp: s.nowFn(),
	})
	if len(s.convos[channelID]) > s.maxSize {
		s.convos[channelID] = s.convos[channelID][len(s.convos[channelID])-s.maxSize:]
	}
}

// Recent returns up to n recent entries for the channel, pruning expired ones.
func (s *Store) Recent(channelID string, n int) []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.convos[channelID]
	if len(entries) == 0 {
		return nil
	}
	cutoff := s.nowFn().Add(-s.maxAge)
	var fresh []Entry
	for _, e := range entries {
		if e.Timestamp.After(cutoff) {
			fresh = append(fresh, e)
		}
	}
	s.convos[channelID] = fresh
	if n > len(fresh) {
		n = len(fresh)
	}
	if n == 0 {
		return nil
	}
	result := make([]Entry, n)
	copy(result, fresh[len(fresh)-n:])
	return result
}
