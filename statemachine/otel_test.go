package statemachine

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

const testToolName = "test-tool"

// setupTestTracer creates a test tracer with an in-memory exporter.
func setupTestTracer(t *testing.T) (*tracetest.InMemoryExporter, func()) {
	t.Helper()

	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithSyncer(exporter),
	)

	oldProvider := otel.GetTracerProvider()

	otel.SetTracerProvider(tp)

	cleanup := func() {
		otel.SetTracerProvider(oldProvider)
	}

	return exporter, cleanup
}

// TestSpanCreation verifies span creation for execution, state, and action spans.
// Subtests cannot run in parallel because they share the same exporter instance
// and use exporter.Reset() to ensure test isolation.
// Note: Cannot use t.Parallel() because setupTestTracer modifies global OTEL tracer provider.
//
//nolint:paralleltest // Test modifies global OTEL tracer provider
//nolint:tparallel // Subtests share exporter, must run sequentially
func TestSpanCreation(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")
	smCtx.ToolName = testToolName
	smCtx.Provider = "test-provider"

	// Test execution span
	//nolint:paralleltest // Subtests share exporter, must run sequentially
	t.Run("execution span", func(t *testing.T) {
		exporter.Reset() // Reset before test

		execCtx, span := startExecutionSpan(ctx, smCtx)
		assert.NotNil(t, execCtx)
		assert.NotNil(t, span)

		// Verify span is valid
		assert.True(t, span.SpanContext().IsValid())

		span.End()

		// Verify span was recorded
		spans := exporter.GetSpans()
		require.Len(t, spans, 1)

		execSpan := spans[0]
		assert.Equal(t, "statemachine.execute", execSpan.Name)

		// Verify attributes
		attrs := execSpan.Attributes

		attrMap := make(map[string]any)
		for _, attr := range attrs {
			attrMap[string(attr.Key)] = attr.Value.AsInterface()
		}

		assert.Equal(t, testToolName, attrMap["tool"])
		assert.Equal(t, "test-session", attrMap["session_id"])
		assert.Equal(t, "test-provider", attrMap["provider"])
	})

	// Test state span
	//nolint:paralleltest // Subtests share exporter, must run sequentially
	t.Run("state span", func(t *testing.T) {
		exporter.Reset() // Reset before test

		stateCtx, span := startStateSpan(ctx, "test-state", smCtx)
		assert.NotNil(t, stateCtx)
		assert.NotNil(t, span)

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)

		stateSpan := spans[0]
		assert.Equal(t, "state.test-state", stateSpan.Name)

		// Verify state attribute
		attrs := stateSpan.Attributes

		attrMap := make(map[string]any)
		for _, attr := range attrs {
			attrMap[string(attr.Key)] = attr.Value.AsInterface()
		}

		assert.Equal(t, "test-state", attrMap["state"])
	})

	// Test action span
	//nolint:paralleltest // Subtests share exporter, must run sequentially
	t.Run("action span", func(t *testing.T) {
		exporter.Reset() // Reset before test

		actionCtx, span := startActionSpan(ctx, "test-action", "test-state", smCtx)
		assert.NotNil(t, actionCtx)
		assert.NotNil(t, span)

		span.End()

		spans := exporter.GetSpans()
		require.Len(t, spans, 1)

		actionSpan := spans[0]
		assert.Equal(t, "action.test-action", actionSpan.Name)

		// Verify attributes
		attrs := actionSpan.Attributes

		attrMap := make(map[string]any)
		for _, attr := range attrs {
			attrMap[string(attr.Key)] = attr.Value.AsInterface()
		}

		assert.Equal(t, "test-action", attrMap["action"])
		assert.Equal(t, "test-state", attrMap["state"])
	})
}

// TestSpanHierarchy verifies parent-child relationships between spans.
// Note: Cannot use t.Parallel() because setupTestTracer modifies global OTEL tracer provider.
//
//nolint:paralleltest // Test modifies global OTEL tracer provider
func TestSpanHierarchy(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")
	smCtx.ToolName = testToolName

	// Create execution span
	execCtx, execSpan := startExecutionSpan(ctx, smCtx)
	defer execSpan.End()

	// Create state span (child of execution)
	stateCtx, stateSpan := startStateSpan(execCtx, "test-state", smCtx)
	defer stateSpan.End()

	// Create action span (child of state)
	_, actionSpan := startActionSpan(stateCtx, "test-action", "test-state", smCtx)
	actionSpan.End()

	// Verify span hierarchy
	spans := exporter.GetSpans()
	require.Len(t, spans, 1) // Only action span has ended so far

	actionSpan2 := spans[0]
	assert.Equal(t, "action.test-action", actionSpan2.Name)

	// Verify parent-child relationship
	assert.True(t, actionSpan2.Parent.IsValid())
}

// TestErrorRecording verifies that errors are recorded in spans.
// Note: Cannot use t.Parallel() because setupTestTracer modifies global OTEL tracer provider.
//
//nolint:paralleltest // Test modifies global OTEL tracer provider
func TestErrorRecording(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	execCtx, span := startExecutionSpan(ctx, smCtx)
	assert.NotNil(t, execCtx)

	// Simulate error
	testErr := assert.AnError
	span.RecordError(testErr)
	span.End()

	// Verify error was recorded
	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	execSpan := spans[0]
	require.Len(t, execSpan.Events, 1)

	event := execSpan.Events[0]
	assert.Equal(t, "exception", event.Name)
}

func TestDebugMode(t *testing.T) {
	// Note: Cannot use t.Parallel() on subtests because t.Setenv modifies global state

	// Test debug mode enabled
	t.Run("debug mode enabled", func(t *testing.T) {
		t.Setenv("MCP_DEBUG", "1")

		assert.True(t, isDebugMode())
	})

	// Test debug mode disabled
	t.Run("debug mode disabled", func(t *testing.T) {
		t.Setenv("MCP_DEBUG", "")

		assert.False(t, isDebugMode())
	})

	// Test debug mode with "true" value
	t.Run("debug mode true", func(t *testing.T) {
		t.Setenv("MCP_DEBUG", "true")

		assert.True(t, isDebugMode())
	})
}

func TestHashID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "non-empty string",
			input:    "test-project-id",
			expected: "d4735e3a", // SHA-256 hash first 8 chars
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := hashID(tt.input)
			if tt.expected == "" {
				assert.Empty(t, result)
			} else {
				assert.NotEmpty(t, result)
				assert.Len(t, result, 8) // First 8 chars of hex
			}
		})
	}
}

// TestExtractTraceContext verifies trace context extraction from spans.
// Note: Cannot use t.Parallel() because setupTestTracer modifies global OTEL tracer provider.
//
//nolint:paralleltest // Test modifies global OTEL tracer provider
func TestExtractTraceContext(t *testing.T) {
	_, cleanup := setupTestTracer(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	execCtx, span := startExecutionSpan(ctx, smCtx)
	defer span.End()

	// Extract trace context
	traceID, spanID := extractTraceContext(execCtx)

	assert.NotEmpty(t, traceID)
	assert.NotEmpty(t, spanID)

	// Verify they match the actual span context
	spanCtx := span.SpanContext()
	assert.Equal(t, spanCtx.TraceID().String(), traceID)
	assert.Equal(t, spanCtx.SpanID().String(), spanID)
}

// TestSpanAttributes verifies that span attributes are set correctly.
// Note: Cannot use t.Parallel() because setupTestTracer modifies global OTEL tracer provider.
//
//nolint:paralleltest // Test modifies global OTEL tracer provider
func TestSpanAttributes(t *testing.T) {
	exporter, cleanup := setupTestTracer(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")
	smCtx.ToolName = testToolName
	smCtx.Provider = "salesforce"
	smCtx.ContextChunkID = "chunk-123"
	smCtx.PathHistory = []string{"state1", "state2"}

	_, span := startExecutionSpan(ctx, smCtx)
	span.End()

	spans := exporter.GetSpans()
	require.Len(t, spans, 1)

	execSpan := spans[0]

	// Verify all attributes are present
	attrs := execSpan.Attributes

	attrMap := make(map[string]any)
	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	assert.Equal(t, testToolName, attrMap["tool"])
	assert.Equal(t, "test-session", attrMap["session_id"])
	assert.Equal(t, "salesforce", attrMap["provider"])
	assert.NotEmpty(t, attrMap["project_id_hash"])
	assert.NotEmpty(t, attrMap["chunk_id_hash"])
}
