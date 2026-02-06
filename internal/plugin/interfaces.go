// Package plugin defines the core plugin interfaces for tinyclaw.
package plugin

import "context"

// Transport receives inbound events and sends outbound operations.
type Transport interface {
	Subscribe(ctx context.Context) (<-chan InboundEvent, error)
	Post(ctx context.Context, op OutboundOp) error
	Close() error
}

// Harness runs an agent and emits run events.
type Harness interface {
	Start(ctx context.Context, req RunRequest) (<-chan RunEvent, error)
	Close() error
}

// ContextProvider builds context manifests from workspace files.
type ContextProvider interface {
	Name() string
	Gather(ctx context.Context, req ContextRequest) ([]ContextItem, error)
}

// InboundEvent represents an event received from a transport.
type InboundEvent struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

// OutboundOp represents an operation sent to a transport.
// Kind is one of: post, edit, upload, typing.
type OutboundOp struct {
	Kind string         `json:"kind"`
	Data map[string]any `json:"data"`
}

// RunRequest is the input to a harness run.
type RunRequest struct {
	Scenario string         `json:"scenario"`
	Event    InboundEvent   `json:"event"`
	Context  []ContextItem  `json:"context"`
	Metadata map[string]any `json:"metadata"`
}

// RunEvent is an event emitted by a harness during a run.
// Kind is one of: status, delta, tool, fault, final.
type RunEvent struct {
	Kind string         `json:"kind"`
	Data map[string]any `json:"data"`
}

// ContextRequest is the input to a context provider.
type ContextRequest struct {
	WorkDir string         `json:"work_dir"`
	Hints   map[string]any `json:"hints"`
}

// ContextItem is a single item in a context manifest.
type ContextItem struct {
	Name    string `json:"name"`
	Content string `json:"content"`
	Source  string `json:"source"`
}
