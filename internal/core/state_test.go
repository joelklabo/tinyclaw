package core

import (
	"testing"
	"time"
)

func TestNewMachineStartsAtIngress(t *testing.T) {
	m := NewMachine()
	if m.Current() != Ingress {
		t.Fatalf("initial state = %v, want Ingress", m.Current())
	}
}

func TestHappyPath(t *testing.T) {
	m := NewMachine()
	steps := []State{Routed, ContextBuilt, Running, Completed}
	for _, s := range steps {
		if err := m.Advance(s); err != nil {
			t.Fatalf("Advance(%v) error: %v", s, err)
		}
	}
	if m.Current() != Completed {
		t.Fatalf("final state = %v, want Completed", m.Current())
	}
	if len(m.Transitions()) != 4 {
		t.Fatalf("transitions = %d, want 4", len(m.Transitions()))
	}
}

func TestFailPath(t *testing.T) {
	m := NewMachine()
	for _, s := range []State{Routed, ContextBuilt, Running} {
		if err := m.Advance(s); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Advance(Failed); err != nil {
		t.Fatalf("Advance(Failed) error: %v", err)
	}
	if m.Current() != Failed {
		t.Fatalf("state = %v, want Failed", m.Current())
	}
}

func TestInvalidTransitions(t *testing.T) {
	tests := []struct {
		name string
		from State
		to   State
	}{
		{"ingress to running", Ingress, Running},
		{"ingress to completed", Ingress, Completed},
		{"ingress to failed", Ingress, Failed},
		{"ingress to context_built", Ingress, ContextBuilt},
		{"routed to running", Routed, Running},
		{"routed to ingress", Routed, Ingress},
		{"context_built to completed", ContextBuilt, Completed},
		{"running to ingress", Running, Ingress},
		{"running to routed", Running, Routed},
		{"completed to anything", Completed, Ingress},
		{"failed to anything", Failed, Ingress},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Machine{current: tt.from, now: time.Now}
			if err := m.Advance(tt.to); err == nil {
				t.Fatalf("expected error for %s -> %s", tt.from, tt.to)
			}
		})
	}
}

func TestTransitionsRecordTimestamps(t *testing.T) {
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	m := NewMachine()
	m.now = func() time.Time { return fixed }

	if err := m.Advance(Routed); err != nil {
		t.Fatal(err)
	}
	trans := m.Transitions()
	if len(trans) != 1 {
		t.Fatalf("transitions = %d, want 1", len(trans))
	}
	if trans[0].From != Ingress || trans[0].To != Routed {
		t.Fatalf("transition = %v -> %v, want Ingress -> Routed", trans[0].From, trans[0].To)
	}
	if !trans[0].Timestamp.Equal(fixed) {
		t.Fatalf("timestamp = %v, want %v", trans[0].Timestamp, fixed)
	}
}

func TestTransitionsReturnsCopy(t *testing.T) {
	m := NewMachine()
	if err := m.Advance(Routed); err != nil {
		t.Fatal(err)
	}
	t1 := m.Transitions()
	t2 := m.Transitions()
	t1[0].To = Failed
	if t2[0].To == Failed {
		t.Fatal("Transitions should return a copy, not a reference")
	}
}

func TestStateStrings(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{Ingress, "ingress"},
		{Routed, "routed"},
		{ContextBuilt, "context_built"},
		{Running, "running"},
		{Completed, "completed"},
		{Failed, "failed"},
		{State(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.state.String(); got != tt.want {
			t.Errorf("State(%d).String() = %q, want %q", tt.state, got, tt.want)
		}
	}
}
