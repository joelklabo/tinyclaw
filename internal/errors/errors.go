// Package errors defines the tinyclaw error taxonomy.
//
// All errors carry structured fields for inclusion in debug bundles.
package errors

import (
	"errors"
	"fmt"
)

// Standard sentinel checks re-exported for convenience.
var (
	Is = errors.Is
	As = errors.As
)

// Kind classifies an error for routing and display.
type Kind int

const (
	KindAuth      Kind = iota + 1 // authentication / authorization failure
	KindQuota                     // rate-limit / quota exceeded
	KindTransient                 // retryable network / service error
	KindFatal                     // unrecoverable error
)

func (k Kind) String() string {
	switch k {
	case KindAuth:
		return "auth"
	case KindQuota:
		return "quota"
	case KindTransient:
		return "transient"
	case KindFatal:
		return "fatal"
	default:
		return "unknown"
	}
}

// Error is a structured error that carries a Kind and optional metadata.
type Error struct {
	Kind    Kind
	Message string
	Op      string // operation that failed
	Err     error  // underlying error, if any
}

func (e *Error) Error() string {
	if e.Op != "" {
		return fmt.Sprintf("%s: %s: %s", e.Op, e.Kind, e.Message)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func (e *Error) Unwrap() error {
	return e.Err
}

// New creates a new Error with the given kind and message.
func New(kind Kind, msg string) *Error {
	return &Error{Kind: kind, Message: msg}
}

// Wrap creates a new Error wrapping an existing error.
func Wrap(kind Kind, op string, err error) *Error {
	return &Error{Kind: kind, Op: op, Err: err, Message: err.Error()}
}

// AuthError creates an auth-kind error.
func AuthError(msg string) *Error {
	return New(KindAuth, msg)
}

// QuotaError creates a quota-kind error.
func QuotaError(msg string) *Error {
	return New(KindQuota, msg)
}

// TransientError creates a transient-kind error.
func TransientError(msg string) *Error {
	return New(KindTransient, msg)
}

// FatalError creates a fatal-kind error.
func FatalError(msg string) *Error {
	return New(KindFatal, msg)
}

// GetKind returns the Kind of an error if it is an *Error, or KindFatal otherwise.
func GetKind(err error) Kind {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind
	}
	return KindFatal
}
