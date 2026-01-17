package spans

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/amp-labs/amp-common/assert"
	"github.com/amp-labs/amp-common/utils"
	"github.com/amp-labs/amp-common/zero"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Option is a function that configures a runner.
// Options are applied when creating orchestrators via Enter, EnterError, EnterValue, or EnterValueError.
type Option func(*runner)

// newRunner creates a new runner with the given tracer, span name, and options.
// The runner is responsible for executing functions within an OpenTelemetry span.
func newRunner(tracer trace.Tracer, spanName string, opts ...Option) *runner {
	r := &runner{
		spanName: spanName,
		spanKind: trace.SpanKindServer,
		tracer:   tracer,
		autoEnd:  true,
	}

	for _, option := range opts {
		if option != nil {
			option(r)
		}
	}

	return r
}

// runner manages the execution of a function within an OpenTelemetry span.
// It handles span lifecycle, error recording, panic recovery, and status reporting.
type runner struct {
	// spanName is the name of the OpenTelemetry span.
	spanName string
	// success is the custom success message for the span status (optional).
	success string
	// failure is the custom error message prefix for the span status (optional).
	failure string
	// autoEnd controls whether to automatically end the span. If false, you must do this manually.
	autoEnd bool
	// spanKind is the OpenTelemetry span kind (default: SpanKindServer).
	spanKind trace.SpanKind
	// tracer is the OpenTelemetry tracer used to create spans.
	tracer trace.Tracer

	// sso are span start options passed to tracer.Start().
	sso []trace.SpanStartOption
	// seo are span end options passed to span.End().
	seo []trace.SpanEndOption

	// decorate are functions called to decorate the span after creation.
	decorate []func(span trace.Span)
}

// runWithSpan executes the given function within an OpenTelemetry span.
// It handles:
//   - Span creation and lifecycle management
//   - Panic recovery with stack traces
//   - Error recording and status setting
//   - Custom success/failure messages
//   - Span decoration
//
// If the tracer is nil, the function is executed without creating a span.
// If the span is not recording, the function is executed but span operations are skipped.
func (r *runner) runWithSpan(
	ctx context.Context,
	operation func(ctx context.Context, span trace.Span) (any, error),
) (valOut any, errOut error) {
	if r == nil || r.tracer == nil {
		return operation(ctx, trace.SpanFromContext(ctx))
	}

	opts := make([]trace.SpanStartOption, len(r.sso)+1)

	copy(opts, r.sso)
	opts[len(r.sso)] = trace.WithSpanKind(r.spanKind)

	ctx, span := r.tracer.Start(ctx, r.spanName, opts...) //nolint:spancheck

	defer func() {
		if r.autoEnd {
			defer span.End(r.seo...)
		}

		if panicErr := recover(); panicErr != nil {
			span.SetAttributes(attribute.KeyValue{
				Key:   "panic",
				Value: attribute.Int64Value(1),
			})

			err := utils.GetPanicRecoveryError(panicErr, debug.Stack())

			if errOut == nil {
				errOut = err
			} else {
				errOut = errors.Join(errOut, err)
			}

			r.setErrorStatus(r.autoEnd, span, errOut)

			panic(panicErr)
		}
	}()

	if span.IsRecording() && len(r.decorate) > 0 {
		for _, decorate := range r.decorate {
			if decorate != nil {
				decorate(span)
			}
		}
	}

	val, err := operation(ctx, span)
	if err != nil {
		span.RecordError(err)
		r.setErrorStatus(r.autoEnd, span, err)
	} else {
		r.setSuccessStatus(r.autoEnd, span)
	}

	return val, err
}

// setErrorStatus sets the span status to error with an optional custom message prefix.
func (r *runner) setErrorStatus(autoEnd bool, span trace.Span, err error) {
	if !autoEnd {
		return
	}

	if len(r.failure) > 0 {
		span.SetStatus(codes.Error, fmt.Sprintf("%s: %s", r.failure, err.Error()))
	} else {
		span.SetStatus(codes.Error, err.Error())
	}
}

// setSuccessStatus sets the span status to OK with an optional custom message.
func (r *runner) setSuccessStatus(autoEnd bool, span trace.Span) {
	if !autoEnd {
		return
	}

	if len(r.success) > 0 {
		span.SetStatus(codes.Ok, r.success)
	} else {
		span.SetStatus(codes.Ok, "ok")
	}
}

// invoke executes a function within an OpenTelemetry span if a tracer is found in the context.
// If no tracer is found, the function is executed without creating a span, and a metric is
// incremented to track this instrumentation gap.
//
// This is the internal function used by all orchestrator Begin() methods.
func invoke[T any](
	ctx context.Context, name string,
	call func(ctx context.Context, span trace.Span) (T, error), opts ...Option,
) (T, error) {
	tracer, found := TracerFromContext(ctx)
	if !found {
		spanWithoutTracerCounter.WithLabelValues(name).Inc()

		return call(ctx, trace.SpanFromContext(ctx))
	}

	r := newRunner(tracer, name, opts...)

	ret, err := r.runWithSpan(ctx, func(ctx context.Context, span trace.Span) (any, error) {
		return call(ctx, span)
	})
	if err != nil {
		return zero.Value[T](), err
	}

	value, err := assert.Type[T](ret)
	if err != nil {
		return zero.Value[T](), err
	}

	return value, nil
}
