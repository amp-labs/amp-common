package spans

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
	"go.opentelemetry.io/otel/trace"
)

// contextKey is a unique type for storing values in context to avoid collisions.
type contextKey string

// TracerKey is the context key used to store the OpenTelemetry tracer.
const TracerKey contextKey = "tracer"

// WithTracer stores an OpenTelemetry tracer in the context.
// This tracer will be used by Enter*, EnterError, EnterValue*, and EnterValueError
// functions to create spans when executing traced functions.
//
// If no tracer is found in the context, the orchestrator functions will execute
// the wrapped function without creating spans.
//
// Example:
//
//	ctx = spans.WithTracer(ctx, otel.Tracer("my-service"))
func WithTracer(ctx context.Context, tracer trace.Tracer) context.Context {
	return contexts.WithValue[contextKey, trace.Tracer](ctx, TracerKey, tracer)
}

// SetTracer stores an OpenTelemetry tracer using a provided setter function.
// This is useful when working with context abstractions that provide their own
// value-setting mechanisms instead of directly using context.Context.
//
// The setter function should accept a key and value to store in its underlying context.
func SetTracer(tracer trace.Tracer, setter func(key any, value any)) {
	setter(TracerKey, tracer)
}

// TracerFromContext retrieves the OpenTelemetry tracer from the context.
// Returns the tracer and true if found, or nil and false if not present.
//
// This function is typically used internally by the orchestrator but can be
// used to check if a tracer is configured in the current context.
func TracerFromContext(ctx context.Context) (trace.Tracer, bool) {
	return contexts.GetValue[contextKey, trace.Tracer](ctx, TracerKey)
}
