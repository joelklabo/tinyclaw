package scenario

import (
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestAssertOps_Match(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundPost, Content: "hi"},
		{Kind: plugin.OutboundEdit, Content: "updated"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundPost},
		{Kind: plugin.OutboundEdit},
	}
	if err := AssertOps(actual, expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertOps_CountMismatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundPost},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundPost},
		{Kind: plugin.OutboundEdit},
	}
	err := AssertOps(actual, expected)
	if err == nil {
		t.Fatal("expected error for count mismatch")
	}
}

func TestAssertOps_KindMismatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundEdit},
		{Kind: plugin.OutboundPost},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundPost},
		{Kind: plugin.OutboundEdit},
	}
	err := AssertOps(actual, expected)
	if err == nil {
		t.Fatal("expected error for kind mismatch")
	}
}

func TestAssertOps_Empty(t *testing.T) {
	if err := AssertOps(nil, nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertOps_ContentMatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundPost, Content: "hello"},
		{Kind: plugin.OutboundEdit, Content: "updated"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundPost, Content: "hello"},
		{Kind: plugin.OutboundEdit, Content: "updated"},
	}
	if err := AssertOps(actual, expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertOps_ContentMismatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundPost, Content: "wrong"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundPost, Content: "expected"},
	}
	err := AssertOps(actual, expected)
	if err == nil {
		t.Fatal("expected error for content mismatch")
	}
}

func TestAssertOps_ContentNotCheckedWhenEmpty(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundPost, Content: "anything"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundPost},
	}
	if err := AssertOps(actual, expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
