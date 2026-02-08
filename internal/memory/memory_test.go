package memory

import (
	"sync"
	"testing"
	"time"
)

func TestAppendAndRecent(t *testing.T) {
	s := New(time.Hour, 100)
	s.Append("ch-1", "user", "hello")
	s.Append("ch-1", "assistant", "hi there")

	entries := s.Recent("ch-1", 10)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Role != "user" || entries[0].Content != "hello" {
		t.Fatalf("unexpected entry 0: %+v", entries[0])
	}
	if entries[1].Role != "assistant" || entries[1].Content != "hi there" {
		t.Fatalf("unexpected entry 1: %+v", entries[1])
	}
}

func TestRecentPrunesExpired(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	s := New(time.Hour, 100)
	s.nowFn = func() time.Time { return now }

	s.Append("ch-1", "user", "old message")

	// Advance time past expiry
	now = now.Add(2 * time.Hour)

	s.Append("ch-1", "user", "new message")

	entries := s.Recent("ch-1", 10)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after pruning, got %d", len(entries))
	}
	if entries[0].Content != "new message" {
		t.Fatalf("expected new message, got %q", entries[0].Content)
	}
}

func TestRecentMaxEntries(t *testing.T) {
	s := New(time.Hour, 100)
	for i := 0; i < 10; i++ {
		s.Append("ch-1", "user", "msg")
	}

	entries := s.Recent("ch-1", 3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestAppendMaxSize(t *testing.T) {
	s := New(time.Hour, 3)
	for i := 0; i < 5; i++ {
		s.Append("ch-1", "user", "msg")
	}

	// Internal store should be capped at 3
	s.mu.RLock()
	n := len(s.convos["ch-1"])
	s.mu.RUnlock()
	if n != 3 {
		t.Fatalf("expected 3 stored entries, got %d", n)
	}

	entries := s.Recent("ch-1", 10)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestConcurrentAccess(t *testing.T) {
	s := New(time.Hour, 1000)
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			s.Append("ch-1", "user", "hello")
		}()
		go func() {
			defer wg.Done()
			s.Recent("ch-1", 5)
		}()
	}
	wg.Wait()
}

func TestEmptyChannel(t *testing.T) {
	s := New(time.Hour, 100)
	entries := s.Recent("nonexistent", 10)
	if entries != nil {
		t.Fatalf("expected nil for empty channel, got %v", entries)
	}
}

func TestMultipleChannels(t *testing.T) {
	s := New(time.Hour, 100)
	s.Append("ch-1", "user", "msg-1")
	s.Append("ch-2", "user", "msg-2")

	e1 := s.Recent("ch-1", 10)
	e2 := s.Recent("ch-2", 10)

	if len(e1) != 1 || e1[0].Content != "msg-1" {
		t.Fatalf("ch-1: unexpected %+v", e1)
	}
	if len(e2) != 1 || e2[0].Content != "msg-2" {
		t.Fatalf("ch-2: unexpected %+v", e2)
	}
}

func TestRecentMoreThanAvailable(t *testing.T) {
	s := New(time.Hour, 100)
	s.Append("ch-1", "user", "only one")

	entries := s.Recent("ch-1", 100)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Content != "only one" {
		t.Fatalf("unexpected content: %q", entries[0].Content)
	}
}

func TestRecentReturnsLastN(t *testing.T) {
	s := New(time.Hour, 100)
	s.Append("ch-1", "user", "first")
	s.Append("ch-1", "user", "second")
	s.Append("ch-1", "user", "third")

	entries := s.Recent("ch-1", 2)
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Content != "second" {
		t.Fatalf("expected 'second', got %q", entries[0].Content)
	}
	if entries[1].Content != "third" {
		t.Fatalf("expected 'third', got %q", entries[1].Content)
	}
}

func TestRecentAllExpired(t *testing.T) {
	now := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	s := New(time.Hour, 100)
	s.nowFn = func() time.Time { return now }

	s.Append("ch-1", "user", "old")

	now = now.Add(2 * time.Hour)

	entries := s.Recent("ch-1", 10)
	if entries != nil {
		t.Fatalf("expected nil after all expired, got %v", entries)
	}
}

func TestRecentZeroN(t *testing.T) {
	s := New(time.Hour, 100)
	s.Append("ch-1", "user", "msg")

	entries := s.Recent("ch-1", 0)
	if entries != nil {
		t.Fatalf("expected nil for n=0, got %v", entries)
	}
}
