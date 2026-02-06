package telemetry

import (
	"context"
	"testing"
)

func TestNewTracer(t *testing.T) {
	tr, err := NewTracer()
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Shutdown(context.Background())

	if tr == nil {
		t.Fatal("expected non-nil tracer")
	}
}

func TestTracerSpanOperations(t *testing.T) {
	tr, err := NewTracer()
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Shutdown(context.Background())

	ops := []string{
		"run.start",
		"run.end",
		"context.build",
		"harness.stream",
		"transport.send",
	}

	ctx := context.Background()
	for _, op := range ops {
		_, span := tr.Start(ctx, op)
		span.End()
	}

	spans := tr.Spans()
	if len(spans) != len(ops) {
		t.Fatalf("expected %d spans, got %d", len(ops), len(spans))
	}

	for i, s := range spans {
		if s.Name != ops[i] {
			t.Fatalf("span %d: expected name %q, got %q", i, ops[i], s.Name)
		}
	}
}

func TestTracerNestedSpans(t *testing.T) {
	tr, err := NewTracer()
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Shutdown(context.Background())

	ctx := context.Background()
	ctx, parent := tr.Start(ctx, "run.start")
	_, child := tr.Start(ctx, "context.build")
	child.End()
	parent.End()

	spans := tr.Spans()
	if len(spans) != 2 {
		t.Fatalf("expected 2 spans, got %d", len(spans))
	}

	// Child should have parent's span ID as parent
	childSpan := spans[0] // child ends first
	parentSpan := spans[1]
	if childSpan.Parent.SpanID() != parentSpan.SpanContext.SpanID() {
		t.Fatal("child span should have parent span as parent")
	}
}

func TestTracerShutdown(t *testing.T) {
	tr, err := NewTracer()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	_, span := tr.Start(ctx, "run.start")
	span.End()

	// Verify spans exist before shutdown
	spans := tr.Spans()
	if len(spans) != 1 {
		t.Fatalf("expected 1 span before shutdown, got %d", len(spans))
	}

	if err := tr.Shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
