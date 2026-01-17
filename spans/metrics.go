package spans

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// spanWithoutTracerCounter tracks the number of times a span was attempted
// without a tracer in the context. This helps identify instrumentation gaps
// where spans.WithTracer() may not have been called.
//
// Metric name: amp_spans_without_tracer_total
// Labels:
//   - span_name: The name of the span that was attempted
//
// Example PromQL query:
//   sum by (span_name) (rate(amp_spans_without_tracer_total[5m]))
var spanWithoutTracerCounter = promauto.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "amp",
		Subsystem: "spans",
		Name:      "without_tracer_total",
		Help:      "Total number of span executions without a tracer in context",
	},
	[]string{"span_name"},
)
