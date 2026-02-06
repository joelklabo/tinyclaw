// Package core implements the tinyclaw state machine and orchestrator.
package core

import (
	"fmt"
	"time"
)

// State represents a phase of an agent run.
type State int

const (
	Ingress      State = iota // initial state: event received
	Routed                    // event routed to an agent profile
	ContextBuilt              // context manifest assembled
	Running                   // harness is running
	Completed                 // run finished successfully
	Failed                    // run finished with an error
)

func (s State) String() string {
	switch s {
	case Ingress:
		return "ingress"
	case Routed:
		return "routed"
	case ContextBuilt:
		return "context_built"
	case Running:
		return "running"
	case Completed:
		return "completed"
	case Failed:
		return "failed"
	default:
		return "unknown"
	}
}

// Transition records a single state change.
type Transition struct {
	From      State     `json:"from"`
	To        State     `json:"to"`
	Timestamp time.Time `json:"timestamp"`
}

// validTransitions defines the allowed state transitions.
var validTransitions = map[State][]State{
	Ingress:      {Routed},
	Routed:       {ContextBuilt},
	ContextBuilt: {Running},
	Running:      {Completed, Failed},
}

// Machine tracks the current state and records transitions.
type Machine struct {
	current     State
	transitions []Transition
	now         func() time.Time
}

// NewMachine creates a Machine starting in the Ingress state.
func NewMachine() *Machine {
	return &Machine{
		current: Ingress,
		now:     time.Now,
	}
}

// Current returns the current state.
func (m *Machine) Current() State {
	return m.current
}

// Transitions returns a copy of all recorded transitions.
func (m *Machine) Transitions() []Transition {
	out := make([]Transition, len(m.transitions))
	copy(out, m.transitions)
	return out
}

// Advance moves the machine to the given state.
// Returns an error if the transition is not valid.
func (m *Machine) Advance(to State) error {
	allowed := validTransitions[m.current]
	for _, s := range allowed {
		if s == to {
			t := Transition{
				From:      m.current,
				To:        to,
				Timestamp: m.now(),
			}
			m.transitions = append(m.transitions, t)
			m.current = to
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %s -> %s", m.current, to)
}
