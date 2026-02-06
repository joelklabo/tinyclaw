package errors

import (
	"errors"
	"testing"
)

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindAuth, "auth"},
		{KindQuota, "quota"},
		{KindTransient, "transient"},
		{KindFatal, "fatal"},
		{Kind(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestErrorMessage(t *testing.T) {
	e := New(KindAuth, "bad token")
	if got := e.Error(); got != "auth: bad token" {
		t.Fatalf("got %q", got)
	}
}

func TestErrorMessageWithOp(t *testing.T) {
	e := &Error{Kind: KindQuota, Op: "harness.stream", Message: "rate limited"}
	if got := e.Error(); got != "harness.stream: quota: rate limited" {
		t.Fatalf("got %q", got)
	}
}

func TestWrap(t *testing.T) {
	inner := errors.New("connection refused")
	e := Wrap(KindTransient, "transport.send", inner)
	if e.Kind != KindTransient {
		t.Fatal("wrong kind")
	}
	if e.Op != "transport.send" {
		t.Fatal("wrong op")
	}
	if !errors.Is(e, inner) {
		t.Fatal("Unwrap should return inner error")
	}
}

func TestConvenienceConstructors(t *testing.T) {
	a := AuthError("denied")
	if a.Kind != KindAuth || a.Message != "denied" {
		t.Fatal("AuthError broken")
	}
	q := QuotaError("exceeded")
	if q.Kind != KindQuota || q.Message != "exceeded" {
		t.Fatal("QuotaError broken")
	}
	tr := TransientError("timeout")
	if tr.Kind != KindTransient || tr.Message != "timeout" {
		t.Fatal("TransientError broken")
	}
	f := FatalError("panic")
	if f.Kind != KindFatal || f.Message != "panic" {
		t.Fatal("FatalError broken")
	}
}

func TestGetKind(t *testing.T) {
	e := AuthError("test")
	if got := GetKind(e); got != KindAuth {
		t.Fatalf("GetKind = %v, want auth", got)
	}

	plain := errors.New("plain")
	if got := GetKind(plain); got != KindFatal {
		t.Fatalf("GetKind(plain) = %v, want fatal", got)
	}
}

func TestIsAs(t *testing.T) {
	inner := errors.New("root")
	e := Wrap(KindAuth, "op", inner)
	if !Is(e, inner) {
		t.Fatal("Is should match inner")
	}
	var target *Error
	if !As(e, &target) {
		t.Fatal("As should match *Error")
	}
	if target.Kind != KindAuth {
		t.Fatal("As target kind mismatch")
	}
}
