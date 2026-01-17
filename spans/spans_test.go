package spans_test

import (
	"context"
	"errors"
	"testing"

	"github.com/amp-labs/amp-common/spans"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// setupTestTracer creates a test tracer and exporter for testing spans.
func setupTestTracer() (*trace.TracerProvider, *tracetest.InMemoryExporter) {
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)
	return tp, exporter
}

// TestWithTracer tests the WithTracer function.
func TestWithTracer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tp, _ := setupTestTracer()
	tracer := tp.Tracer("test-tracer")

	// Store tracer in context
	ctx = spans.WithTracer(ctx, tracer)

	// Verify tracer can be retrieved
	retrieved, found := spans.TracerFromContext(ctx)
	require.True(t, found, "tracer should be found in context")
	assert.Equal(t, tracer, retrieved, "retrieved tracer should match")
}

// TestTracerFromContext tests the TracerFromContext function.
func TestTracerFromContext(t *testing.T) {
	t.Parallel()

	t.Run("tracer exists", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		tp, _ := setupTestTracer()
		tracer := tp.Tracer("test-tracer")

		ctx = spans.WithTracer(ctx, tracer)

		retrieved, found := spans.TracerFromContext(ctx)
		assert.True(t, found, "tracer should be found")
		assert.Equal(t, tracer, retrieved, "tracer should match")
	})

	t.Run("tracer does not exist", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		retrieved, found := spans.TracerFromContext(ctx)
		assert.False(t, found, "tracer should not be found")
		assert.Nil(t, retrieved, "retrieved tracer should be nil")
	})
}

// TestSetTracer tests the SetTracer function.
func TestSetTracer(t *testing.T) {
	t.Parallel()

	tp, _ := setupTestTracer()
	tracer := tp.Tracer("test-tracer")

	var capturedKey any
	var capturedValue any

	setter := func(key any, value any) {
		capturedKey = key
		capturedValue = value
	}

	spans.SetTracer(tracer, setter)

	assert.Equal(t, spans.TracerKey, capturedKey, "key should be TracerKey")
	assert.Equal(t, tracer, capturedValue, "value should be the tracer")
}

// TestStart tests the Start function and StartOrchestrator.
func TestStart(t *testing.T) {
	t.Parallel()

	t.Run("with tracer", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		executed := false

		spans.Start(ctx, "test-span").Enter(func(ctx context.Context, span otelTrace.Span) {
			executed = true
			assert.NotNil(t, span, "span should not be nil")
		})

		assert.True(t, executed, "function should have been executed")

		// Verify span was created
		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.Equal(t, "test-span", spanData[0].Name, "span name should match")
		assert.Equal(t, codes.Ok, spanData[0].Status.Code, "span should have OK status")
	})

	t.Run("without tracer", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		executed := false

		spans.Start(ctx, "test-span").Enter(func(ctx context.Context, span otelTrace.Span) {
			executed = true
		})

		assert.True(t, executed, "function should have been executed even without tracer")
	})

	t.Run("nil function", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		spans.Start(ctx, "test-span").Enter(nil)
		// Should not panic
	})
}

// TestStartErr tests the StartErr function and StartErrorOrchestrator.
func TestStartErr(t *testing.T) {
	t.Parallel()

	t.Run("with error", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		expectedErr := errors.New("test error")

		err := spans.StartErr(ctx, "test-span-err").Enter(func(ctx context.Context, span otelTrace.Span) error {
			return expectedErr
		})

		assert.Equal(t, expectedErr, err, "should return the error")

		// Verify span was created with error
		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.Equal(t, "test-span-err", spanData[0].Name, "span name should match")
		assert.Equal(t, codes.Error, spanData[0].Status.Code, "span should have Error status")
		assert.Contains(t, spanData[0].Status.Description, "test error", "error message should be in status")
	})

	t.Run("without error", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		err := spans.StartErr(ctx, "test-span-success").Enter(func(ctx context.Context, span otelTrace.Span) error {
			return nil
		})

		assert.NoError(t, err, "should not return an error")

		// Verify span was created with OK status
		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.Equal(t, codes.Ok, spanData[0].Status.Code, "span should have OK status")
	})

	t.Run("nil function", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		err := spans.StartErr(ctx, "test-span").Enter(nil)
		assert.NoError(t, err, "should return nil for nil function")
	})
}

// TestStartVal tests the StartVal function and StartValueOrchestrator.
func TestStartVal(t *testing.T) {
	t.Parallel()

	t.Run("returns value", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		result := spans.StartVal[string](ctx, "test-span-val").Enter(func(ctx context.Context, span otelTrace.Span) string {
			return "test-value"
		})

		assert.Equal(t, "test-value", result, "should return the value")

		// Verify span was created
		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.Equal(t, "test-span-val", spanData[0].Name, "span name should match")
		assert.Equal(t, codes.Ok, spanData[0].Status.Code, "span should have OK status")
	})

	t.Run("nil function returns zero value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		result := spans.StartVal[string](ctx, "test-span").Enter(nil)
		assert.Equal(t, "", result, "should return zero value for nil function")
	})
}

// TestStartValErr tests the StartValErr function and StartValueErrorOrchestrator.
func TestStartValErr(t *testing.T) {
	t.Parallel()

	t.Run("returns value without error", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		result, err := spans.StartValErr[int](ctx, "test-span-val-err").Enter(
			func(ctx context.Context, span otelTrace.Span) (int, error) {
				return 42, nil
			},
		)

		assert.NoError(t, err, "should not return an error")
		assert.Equal(t, 42, result, "should return the value")

		// Verify span was created
		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.Equal(t, codes.Ok, spanData[0].Status.Code, "span should have OK status")
	})

	t.Run("returns error", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		expectedErr := errors.New("test error")

		result, err := spans.StartValErr[int](ctx, "test-span-val-err").Enter(
			func(ctx context.Context, span otelTrace.Span) (int, error) {
				return 0, expectedErr
			},
		)

		assert.Equal(t, expectedErr, err, "should return the error")
		assert.Equal(t, 0, result, "should return zero value on error")

		// Verify span was created with error
		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.Equal(t, codes.Error, spanData[0].Status.Code, "span should have Error status")
	})

	t.Run("nil function returns zero value", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()
		result, err := spans.StartValErr[string](ctx, "test-span").Enter(nil)
		assert.NoError(t, err, "should not return an error for nil function")
		assert.Equal(t, "", result, "should return zero value for nil function")
	})
}

// TestWithName tests the WithName option.
func TestWithName(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	spans.Start(ctx, "original-name", spans.WithName("overridden-name")).Enter(
		func(ctx context.Context, span otelTrace.Span) {},
	)

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")
	assert.Equal(t, "overridden-name", spanData[0].Name, "span name should be overridden")
}

// TestWithAttribute tests the WithAttribute option.
func TestWithAttribute(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	spans.Start(ctx, "test-span",
		spans.WithAttribute("test.key", attribute.StringValue("test-value")),
		spans.WithAttribute("test.number", attribute.IntValue(42)),
	).Enter(func(ctx context.Context, span otelTrace.Span) {})

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")

	// Find the attributes
	attrs := spanData[0].Attributes
	var foundString, foundInt bool
	for _, attr := range attrs {
		if string(attr.Key) == "test.key" && attr.Value.AsString() == "test-value" {
			foundString = true
		}
		if string(attr.Key) == "test.number" && attr.Value.AsInt64() == 42 {
			foundInt = true
		}
	}

	assert.True(t, foundString, "should have string attribute")
	assert.True(t, foundInt, "should have int attribute")
}

// TestWithSpanKind tests the WithSpanKind option.
func TestWithSpanKind(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	spans.Start(ctx, "test-span", spans.WithSpanKind(otelTrace.SpanKindClient)).Enter(
		func(ctx context.Context, span otelTrace.Span) {},
	)

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")
	assert.Equal(t, otelTrace.SpanKindClient, spanData[0].SpanKind, "span kind should be Client")
}

// TestWithSuccessMessage tests the WithSuccessMessage option.
func TestWithSuccessMessage(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	spans.Start(ctx, "test-span", spans.WithSuccessMessage("custom success message")).Enter(
		func(ctx context.Context, span otelTrace.Span) {},
	)

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")
	assert.Equal(t, codes.Ok, spanData[0].Status.Code, "span should have OK status")
	// Note: Status description may not be captured by InMemoryExporter in all SDK versions
}

// TestWithErrorMessage tests the WithErrorMessage option.
func TestWithErrorMessage(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	expectedErr := errors.New("underlying error")

	_ = spans.StartErr(ctx, "test-span", spans.WithErrorMessage("Operation failed")).Enter(
		func(ctx context.Context, span otelTrace.Span) error {
			return expectedErr
		},
	)

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")
	assert.Equal(t, codes.Error, spanData[0].Status.Code, "span should have Error status")
	assert.Contains(t, spanData[0].Status.Description, "Operation failed", "should contain custom error prefix")
	assert.Contains(t, spanData[0].Status.Description, "underlying error", "should contain actual error message")
}

// TestWithSpanStartOptions tests the WithSpanStartOptions option.
func TestWithSpanStartOptions(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	spans.Start(ctx, "test-span",
		spans.WithSpanStartOptions(
			otelTrace.WithAttributes(
				attribute.String("custom.attr", "value"),
			),
		),
	).Enter(func(ctx context.Context, span otelTrace.Span) {})

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")

	// Find the attribute
	var found bool
	for _, attr := range spanData[0].Attributes {
		if string(attr.Key) == "custom.attr" && attr.Value.AsString() == "value" {
			found = true
			break
		}
	}

	assert.True(t, found, "should have custom attribute")
}

// TestWithSpanDecorator tests the WithSpanDecorator option.
func TestWithSpanDecorator(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	decoratorCalled := false

	spans.Start(ctx, "test-span",
		spans.WithSpanDecorator(func(span otelTrace.Span) {
			decoratorCalled = true
			span.SetAttributes(attribute.String("decorated", "true"))
		}),
	).Enter(func(ctx context.Context, span otelTrace.Span) {})

	assert.True(t, decoratorCalled, "decorator should have been called")

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")

	// Find the decorated attribute
	var found bool
	for _, attr := range spanData[0].Attributes {
		if string(attr.Key) == "decorated" && attr.Value.AsString() == "true" {
			found = true
			break
		}
	}

	assert.True(t, found, "should have decorated attribute")
}

// TestWithAutoEnd tests the WithAutoEnd option.
func TestWithAutoEnd(t *testing.T) {
	t.Parallel()

	t.Run("autoEnd true (default)", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		spans.Start(ctx, "test-span", spans.WithAutoEnd(true)).Enter(
			func(ctx context.Context, span otelTrace.Span) {},
		)

		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.True(t, spanData[0].EndTime.After(spanData[0].StartTime), "span should be ended")
	})

	t.Run("autoEnd false requires manual end", func(t *testing.T) {
		t.Parallel()

		tp, exporter := setupTestTracer()
		defer tp.Shutdown(context.Background()) //nolint:errcheck

		tracer := tp.Tracer("test-tracer")
		ctx := spans.WithTracer(context.Background(), tracer)

		spans.Start(ctx, "test-span", spans.WithAutoEnd(false)).Enter(
			func(ctx context.Context, span otelTrace.Span) {
				defer span.End()
				// Span should be manually ended
			},
		)

		spanData := exporter.GetSpans()
		require.Len(t, spanData, 1, "should have created one span")
		assert.True(t, spanData[0].EndTime.After(spanData[0].StartTime), "span should be ended manually")
	})
}

// TestPanicRecovery tests that panics are recovered and recorded in spans.
func TestPanicRecovery(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	assert.Panics(t, func() {
		spans.Start(ctx, "test-span").Enter(func(ctx context.Context, span otelTrace.Span) {
			panic("test panic")
		})
	}, "should re-panic after recording")

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")

	// Check for panic attribute
	var foundPanic bool
	for _, attr := range spanData[0].Attributes {
		if string(attr.Key) == "panic" && attr.Value.AsInt64() == 1 {
			foundPanic = true
			break
		}
	}

	assert.True(t, foundPanic, "should have panic attribute")
}

// TestMultipleOptions tests using multiple options together.
func TestMultipleOptions(t *testing.T) {
	t.Parallel()

	tp, exporter := setupTestTracer()
	defer tp.Shutdown(context.Background()) //nolint:errcheck

	tracer := tp.Tracer("test-tracer")
	ctx := spans.WithTracer(context.Background(), tracer)

	spans.Start(ctx, "original-name",
		spans.WithName("custom-name"),
		spans.WithAttribute("key1", attribute.StringValue("value1")),
		spans.WithAttribute("key2", attribute.IntValue(123)),
		spans.WithSpanKind(otelTrace.SpanKindClient),
		spans.WithSuccessMessage("all done"),
		spans.WithSpanDecorator(func(span otelTrace.Span) {
			span.SetAttributes(attribute.String("decorated", "yes"))
		}),
	).Enter(func(ctx context.Context, span otelTrace.Span) {})

	spanData := exporter.GetSpans()
	require.Len(t, spanData, 1, "should have created one span")

	// Verify all options took effect
	assert.Equal(t, "custom-name", spanData[0].Name, "name should be overridden")
	assert.Equal(t, otelTrace.SpanKindClient, spanData[0].SpanKind, "span kind should be Client")
	assert.Equal(t, codes.Ok, spanData[0].Status.Code, "status should be Ok")
	// Note: Status description may not be captured by InMemoryExporter in all SDK versions

	// Check attributes
	attrMap := make(map[string]attribute.Value)
	for _, attr := range spanData[0].Attributes {
		attrMap[string(attr.Key)] = attr.Value
	}

	assert.Equal(t, "value1", attrMap["key1"].AsString(), "should have key1 attribute")
	assert.Equal(t, int64(123), attrMap["key2"].AsInt64(), "should have key2 attribute")
	assert.Equal(t, "yes", attrMap["decorated"].AsString(), "should have decorated attribute")
}
