package spans_test

import (
	"context"
	"testing"

	"github.com/amp-labs/amp-common/spans"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otelTrace "go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TestWithoutTracerInsideATrace checks that a call whose context carries a span but no tracer
// (a service that never called WithTracer) gets a span of its own, a child of the context's,
// from the global TracerProvider, and that what the callback sets on its span lands there rather
// than on the caller's.
//
//nolint:paralleltest // sets the global TracerProvider
func TestWithoutTracerInsideATrace(t *testing.T) {
	tp, exporter := setupTestTracer()

	otel.SetTracerProvider(tp)
	t.Cleanup(func() { otel.SetTracerProvider(noop.NewTracerProvider()) })

	ctx, parent := tp.Tracer("caller").Start(context.Background(), "request")

	var inner otelTrace.Span

	value, err := spans.StartValErr[int](ctx, "call-token-manager").
		Enter(func(_ context.Context, span otelTrace.Span) (int, error) {
			inner = span
			span.SetAttributes(attribute.Int("status_code", 200))

			return 1, nil
		})
	require.NoError(t, err)
	assert.Equal(t, 1, value)

	parent.End()

	require.NotNil(t, inner)
	assert.NotEqual(t, parent.SpanContext().SpanID(), inner.SpanContext().SpanID(),
		"the callback should get a span of its own, not the caller's")

	byName := map[string]int{}
	for i, span := range exporter.GetSpans() {
		byName[span.Name] = i
	}

	require.Contains(t, byName, "call-token-manager")

	call := exporter.GetSpans()[byName["call-token-manager"]]
	request := exporter.GetSpans()[byName["request"]]

	assert.Equal(t, parent.SpanContext().SpanID(), call.Parent.SpanID())
	assert.Contains(t, call.Attributes, attribute.Int("status_code", 200))
	assert.NotContains(t, request.Attributes, attribute.Int("status_code", 200))
}

// TestWithoutTracerOrTrace checks that a call outside any trace and without a tracer runs, with
// no span started.
func TestWithoutTracerOrTrace(t *testing.T) {
	t.Parallel()

	executed := false

	spans.Start(context.Background(), "background").Enter(func(_ context.Context, span otelTrace.Span) {
		executed = true

		assert.False(t, span.SpanContext().IsValid(), "no span should start outside a trace")
	})

	assert.True(t, executed)
}
