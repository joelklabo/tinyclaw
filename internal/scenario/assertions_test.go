package scenario

import (
	"testing"

	"github.com/klabo/tinyclaw/internal/plugin"
)

func TestAssertOps_Match(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundResponse, Content: "hi"},
		{Kind: plugin.OutboundDelta, Content: "updated"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundResponse},
		{Kind: plugin.OutboundDelta},
	}
	if err := AssertOps(actual, expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertOps_CountMismatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundResponse},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundResponse},
		{Kind: plugin.OutboundDelta},
	}
	err := AssertOps(actual, expected)
	if err == nil {
		t.Fatal("expected error for count mismatch")
	}
}

func TestAssertOps_KindMismatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundDelta},
		{Kind: plugin.OutboundResponse},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundResponse},
		{Kind: plugin.OutboundDelta},
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
		{Kind: plugin.OutboundResponse, Content: "hello"},
		{Kind: plugin.OutboundDelta, Content: "updated"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundResponse, Content: "hello"},
		{Kind: plugin.OutboundDelta, Content: "updated"},
	}
	if err := AssertOps(actual, expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAssertOps_ContentMismatch(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundResponse, Content: "wrong"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundResponse, Content: "expected"},
	}
	err := AssertOps(actual, expected)
	if err == nil {
		t.Fatal("expected error for content mismatch")
	}
}

func TestAssertOps_ContentNotCheckedWhenEmpty(t *testing.T) {
	actual := []plugin.OutboundOp{
		{Kind: plugin.OutboundResponse, Content: "anything"},
	}
	expected := []ExpectedOp{
		{Kind: plugin.OutboundResponse},
	}
	if err := AssertOps(actual, expected); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
