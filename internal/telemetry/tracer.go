package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// Tracer wraps an OpenTelemetry TracerProvider with an in-memory exporter.
type Tracer struct {
	provider *trace.TracerProvider
	exporter *tracetest.InMemoryExporter
	tracer   oteltrace.Tracer
}

// NewTracer creates a Tracer backed by an in-memory span exporter.
func NewTracer() (*Tracer, error) {
	exp := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exp),
	)
	return &Tracer{
		provider: tp,
		exporter: exp,
		tracer:   tp.Tracer("tinyclaw"),
	}, nil
}

// Start begins a new span with the given operation name.
func (t *Tracer) Start(ctx context.Context, operation string) (context.Context, oteltrace.Span) {
	return t.tracer.Start(ctx, operation)
}

// Spans returns all completed spans recorded by the in-memory exporter.
func (t *Tracer) Spans() tracetest.SpanStubs {
	return t.exporter.GetSpans()
}

// Shutdown flushes and shuts down the tracer provider.
func (t *Tracer) Shutdown(ctx context.Context) error {
	return t.provider.Shutdown(ctx)
}
