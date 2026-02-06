package shutdown

import (
	"context"
	"sync/atomic"
	"syscall"
	"testing"
	"time"
)

func TestManagerCancelOnSignal(t *testing.T) {
	m := New(5 * time.Second)
	ctx := m.Context()

	// Send SIGINT to ourselves
	if err := syscall.Kill(syscall.Getpid(), syscall.SIGINT); err != nil {
		t.Fatal(err)
	}

	select {
	case <-ctx.Done():
		// expected
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled after SIGINT")
	}
	m.Wait()
}

func TestManagerCleanupRuns(t *testing.T) {
	m := New(5 * time.Second)

	var ran atomic.Bool
	m.Register(func() {
		ran.Store(true)
	})

	// Trigger shutdown
	m.Shutdown()
	m.Wait()

	if !ran.Load() {
		t.Fatal("cleanup function did not run")
	}
}

func TestManagerCleanupOrder(t *testing.T) {
	m := New(5 * time.Second)

	var order []int
	m.Register(func() { order = append(order, 1) })
	m.Register(func() { order = append(order, 2) })

	m.Shutdown()
	m.Wait()

	// Cleanup should run in reverse registration order (LIFO)
	if len(order) != 2 || order[0] != 2 || order[1] != 1 {
		t.Fatalf("expected cleanup order [2 1], got %v", order)
	}
}

func TestManagerTimeout(t *testing.T) {
	m := New(100 * time.Millisecond)

	m.Register(func() {
		time.Sleep(5 * time.Second) // will be cut short by timeout
	})

	start := time.Now()
	m.Shutdown()
	m.Wait()
	elapsed := time.Since(start)

	if elapsed > 2*time.Second {
		t.Fatalf("shutdown took too long: %v", elapsed)
	}
}

func TestManagerContextCancelledAfterShutdown(t *testing.T) {
	m := New(5 * time.Second)
	ctx := m.Context()

	m.Shutdown()
	m.Wait()

	if ctx.Err() != context.Canceled {
		t.Fatalf("expected context.Canceled, got %v", ctx.Err())
	}
}

func TestManagerShutdownIdempotent(t *testing.T) {
	m := New(5 * time.Second)

	var count atomic.Int32
	m.Register(func() { count.Add(1) })

	m.Shutdown()
	m.Shutdown() // second call should be no-op
	m.Wait()

	if count.Load() != 1 {
		t.Fatalf("cleanup ran %d times, expected 1", count.Load())
	}
}

func TestManagerNoCleanups(t *testing.T) {
	m := New(5 * time.Second)
	m.Shutdown()
	m.Wait()
	// should not panic
}
