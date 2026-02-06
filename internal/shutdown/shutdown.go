// Package shutdown provides graceful shutdown management with signal handling.
package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Manager coordinates graceful shutdown on SIGINT/SIGTERM.
type Manager struct {
	ctx      context.Context
	cancel   context.CancelFunc
	cleanups []func()
	mu       sync.Mutex
	once     sync.Once
	done     chan struct{}
	timeout  time.Duration
}

// New creates a Manager that cancels its context on SIGINT/SIGTERM.
// The timeout limits how long cleanup functions may run.
func New(timeout time.Duration) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		ctx:     ctx,
		cancel:  cancel,
		done:    make(chan struct{}),
		timeout: timeout,
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case <-sigCh:
			m.Shutdown()
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
	}()

	return m
}

// Context returns a context that is cancelled when shutdown is triggered.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// Register adds a cleanup function to run during shutdown.
// Functions run in reverse registration order (LIFO).
func (m *Manager) Register(fn func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cleanups = append(m.cleanups, fn)
}

// Shutdown triggers the shutdown sequence. It is safe to call multiple times.
func (m *Manager) Shutdown() {
	m.once.Do(func() {
		m.cancel()
		go m.runCleanups()
	})
}

// Wait blocks until all cleanup functions have completed or the timeout expires.
func (m *Manager) Wait() {
	<-m.done
}

func (m *Manager) runCleanups() {
	defer close(m.done)

	m.mu.Lock()
	fns := make([]func(), len(m.cleanups))
	copy(fns, m.cleanups)
	m.mu.Unlock()

	finished := make(chan struct{})
	go func() {
		defer close(finished)
		// Run in reverse order (LIFO)
		for i := len(fns) - 1; i >= 0; i-- {
			fns[i]()
		}
	}()

	select {
	case <-finished:
	case <-time.After(m.timeout):
	}
}
