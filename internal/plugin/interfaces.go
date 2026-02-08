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

// InboundEventType is the type of inbound transport event.
type InboundEventType string

const (
	InboundMessage InboundEventType = "message"
)

// InboundEvent represents an event received from a transport.
type InboundEvent struct {
	Type      InboundEventType `json:"type"`
	Content   string           `json:"content,omitempty"`
	ChannelID string           `json:"channel_id,omitempty"`
	AuthorID  string           `json:"author_id,omitempty"`
	MessageID string           `json:"message_id,omitempty"`
}

// OutboundOpKind is the kind of outbound transport operation.
type OutboundOpKind string

const (
	OutboundPost   OutboundOpKind = "post"
	OutboundEdit   OutboundOpKind = "edit"
	OutboundTyping OutboundOpKind = "typing"
)

// OutboundOp represents an operation sent to a transport.
type OutboundOp struct {
	Kind      OutboundOpKind `json:"kind"`
	Content   string         `json:"content,omitempty"`
	ChannelID string         `json:"channel_id,omitempty"`
	MessageID string         `json:"message_id,omitempty"`
}

// RunRequest is the input to a harness run.
type RunRequest struct {
	Scenario string        `json:"scenario"`
	Profile  string        `json:"profile"`
	Event    InboundEvent  `json:"event"`
	Context  []ContextItem `json:"context"`
}

// RunEventKind is the kind of harness run event.
type RunEventKind string

const (
	RunEventStatus RunEventKind = "status"
	RunEventDelta  RunEventKind = "delta"
	RunEventTool   RunEventKind = "tool"
	RunEventFault  RunEventKind = "fault"
	RunEventFinal  RunEventKind = "final"
)

// RunEvent is an event emitted by a harness during a run.
type RunEvent struct {
	Kind    RunEventKind `json:"kind" yaml:"kind"`
	Content string       `json:"content,omitempty" yaml:"content"`
	Phase   string       `json:"phase,omitempty" yaml:"phase"`
	Tool    string       `json:"tool,omitempty" yaml:"tool"`
	Message string       `json:"message,omitempty" yaml:"message"`
	Fault   string       `json:"fault,omitempty" yaml:"fault"`
}

// ContextItem is a single item in a context manifest.
type ContextItem struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Source   string `json:"source"`
	Priority int    `json:"priority"`
}
