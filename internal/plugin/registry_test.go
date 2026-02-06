package plugin

import (
	"context"
	"testing"
)

// stubTransport implements Transport for testing.
type stubTransport struct{}

func (s *stubTransport) Subscribe(context.Context) (<-chan InboundEvent, error) { return nil, nil }
func (s *stubTransport) Post(context.Context, OutboundOp) error               { return nil }
func (s *stubTransport) Close() error                                          { return nil }

// stubHarness implements Harness for testing.
type stubHarness struct{}

func (s *stubHarness) Start(context.Context, RunRequest) (<-chan RunEvent, error) { return nil, nil }
func (s *stubHarness) Close() error                                               { return nil }

// stubContext implements ContextProvider for testing.
type stubContext struct{}

func (s *stubContext) Name() string                                                  { return "stub" }
func (s *stubContext) Gather(context.Context, ContextRequest) ([]ContextItem, error) { return nil, nil }

func TestRegistryTransport(t *testing.T) {
	r := NewRegistry()

	err := r.RegisterTransport("fake", func() Transport { return &stubTransport{} })
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	tr, err := r.GetTransport("fake")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if tr == nil {
		t.Fatal("expected non-nil transport")
	}
}

func TestRegistryTransportDuplicate(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterTransport("fake", func() Transport { return &stubTransport{} })
	err := r.RegisterTransport("fake", func() Transport { return &stubTransport{} })
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistryTransportNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetTransport("nope")
	if err == nil {
		t.Fatal("expected error for missing transport")
	}
}

func TestRegistryHarness(t *testing.T) {
	r := NewRegistry()

	err := r.RegisterHarness("replay", func() Harness { return &stubHarness{} })
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	h, err := r.GetHarness("replay")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil harness")
	}
}

func TestRegistryHarnessDuplicate(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterHarness("replay", func() Harness { return &stubHarness{} })
	err := r.RegisterHarness("replay", func() Harness { return &stubHarness{} })
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistryHarnessNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetHarness("nope")
	if err == nil {
		t.Fatal("expected error for missing harness")
	}
}

func TestRegistryContext(t *testing.T) {
	r := NewRegistry()

	err := r.RegisterContext("openclaw", func() ContextProvider { return &stubContext{} })
	if err != nil {
		t.Fatalf("register: %v", err)
	}

	cp, err := r.GetContext("openclaw")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if cp == nil {
		t.Fatal("expected non-nil context provider")
	}
}

func TestRegistryContextDuplicate(t *testing.T) {
	r := NewRegistry()
	_ = r.RegisterContext("openclaw", func() ContextProvider { return &stubContext{} })
	err := r.RegisterContext("openclaw", func() ContextProvider { return &stubContext{} })
	if err == nil {
		t.Fatal("expected error for duplicate registration")
	}
}

func TestRegistryContextNotFound(t *testing.T) {
	r := NewRegistry()
	_, err := r.GetContext("nope")
	if err == nil {
		t.Fatal("expected error for missing context")
	}
}
