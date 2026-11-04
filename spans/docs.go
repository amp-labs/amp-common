// Package spans provides utilities for creating OpenTelemetry spans with a fluent API.
//
// This package simplifies the creation of traced function calls by providing orchestrators
// that handle span lifecycle, error recording, panic recovery, and status reporting.
//
// The package supports four function signatures, all receiving both a context and span:
//   - Start: func(context.Context, trace.Span) - no return value
//   - StartErr: func(context.Context, trace.Span) error - returns error only
//   - StartVal: func(context.Context, trace.Span) T - returns value only
//   - StartValErr: func(context.Context, trace.Span) (T, error) - returns value and error
//
// The tracer is the one WithTracer put in the context. Without one, a call inside a trace (the
// context carries a span) continues it through the global TracerProvider, and a call outside any
// trace runs without a span.
//
// Usage example:
//
//	ctx = spans.WithTracer(ctx, tracer)
//	result, err := spans.StartValErr[int](ctx, "my-operation",
//	    spans.WithAttribute("key", attribute.StringValue("value")),
//	).Enter(func(ctx context.Context, span trace.Span) (int, error) {
//	    return doWork(ctx)
//	})
package spans
