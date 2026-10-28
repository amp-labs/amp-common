package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// TestTraceContextFields checks that a line logged inside a span names the span and whether
// its trace was sampled, and that a line outside any span names none of them.
func TestTraceContextFields(t *testing.T) { //nolint:paralleltest // ConfigureLoggingWithOptions sets the default
	traceID := trace.TraceID{0x75, 0xc2, 0x7c, 0x4d, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}
	spanID := trace.SpanID{0x8c, 0xf9, 0xbb, 0x6d, 1, 2, 3, 4}

	inSpan := func(flags trace.TraceFlags) context.Context {
		return trace.ContextWithSpanContext(t.Context(), trace.NewSpanContext(trace.SpanContextConfig{
			TraceID:    traceID,
			SpanID:     spanID,
			TraceFlags: flags,
		}))
	}

	tests := []struct {
		name        string
		ctx         context.Context //nolint:containedctx
		wantTrace   bool
		wantSampled bool
	}{
		{name: "sampled span", ctx: inSpan(trace.FlagsSampled), wantTrace: true, wantSampled: true},
		{name: "unsampled span", ctx: inSpan(0), wantTrace: true, wantSampled: false},
		{name: "no span", ctx: t.Context(), wantTrace: false},
	}

	for _, testCase := range tests { //nolint:paralleltest
		t.Run(testCase.name, func(t *testing.T) {
			var buf bytes.Buffer

			ConfigureLoggingWithOptions(Options{
				Subsystem: "test",
				JSON:      true,
				Output:    &buf,
			})

			Get(testCase.ctx).Info("traced line")

			var line map[string]any

			require.NoError(t, json.Unmarshal(buf.Bytes(), &line))

			if !testCase.wantTrace {
				assert.NotContains(t, line, "trace_id")
				assert.NotContains(t, line, "span_id")
				assert.NotContains(t, line, "trace_sampled")

				return
			}

			assert.Equal(t, traceID.String(), line["trace_id"])
			assert.Equal(t, spanID.String(), line["span_id"])
			assert.Equal(t, testCase.wantSampled, line["trace_sampled"])
		})
	}
}
