package spans

import (
	"context"

	"github.com/amp-labs/amp-common/zero"
	"go.opentelemetry.io/otel/trace"
)

// StartOrchestrator orchestrates the execution of a function that takes a context
// and returns nothing. Create via spans.Start().
type StartOrchestrator struct {
	ctx  context.Context //nolint:containedctx
	name string
	opts []Option
}

// Enter executes the given function within an OpenTelemetry span.
// The function signature is: func(context.Context, trace.Span)
//
// Panics are recovered, recorded in the span with stack traces, and re-raised.
// This ensures panic information is captured in telemetry before propagating.
//
// Example:
//
//	spans.Start(ctx, "log-event").Enter(func(ctx context.Context, span trace.Span) {
//	    logger.Info("Event occurred")
//	})
func (o *StartOrchestrator) Enter(f func(ctx context.Context, span trace.Span)) {
	if f == nil {
		return
	}

	_, err := invoke[struct{}](o.ctx, o.name, func(ctx context.Context, span trace.Span) (struct{}, error) {
		f(ctx, span)

		return struct{}{}, nil
	}, o.opts...)
	if err != nil {
		panic(err)
	}
}

// StartErrorOrchestrator orchestrates the execution of a function that takes a context
// and returns an error. Create via spans.StartError().
type StartErrorOrchestrator struct {
	ctx  context.Context //nolint:containedctx
	name string
	opts []Option
}

// Enter executes the given function within an OpenTelemetry span.
// The function signature is: func(context.Context, trace.Span) error
//
// Returns the error from the wrapped function, if any.
// Errors are automatically recorded in the span with an Error status.
//
// Example:
//
//	err := spans.StartErr(ctx, "validate-input",
//	    spans.WithErrorMessage("Input validation failed"),
//	).Enter(func(ctx context.Context, span trace.Span) error {
//	    return validateInput(data)
//	})
func (o *StartErrorOrchestrator) Enter(f func(ctx context.Context, span trace.Span) error) error {
	if f == nil {
		return nil
	}

	_, err := invoke[struct{}](o.ctx, o.name, func(ctx context.Context, span trace.Span) (struct{}, error) {
		funcErr := f(ctx, span)

		return struct{}{}, funcErr
	}, o.opts...)

	return err
}

// StartValueOrchestrator orchestrates the execution of a function that takes a context
// and returns a value. Create via spans.StartValue() or spans.StartValueError().
type StartValueOrchestrator[T any] struct {
	ctx  context.Context //nolint:containedctx
	name string
	opts []Option
}

// Enter executes the given function within an OpenTelemetry span.
// The function signature is: func(context.Context, trace.Span) T
//
// Returns the value from the wrapped function.
// Panics are recovered, recorded in the span, and re-raised.
//
// Example:
//
//	config := spans.StartVal[AppConfig](ctx, "load-config").Enter(func(ctx context.Context, span trace.Span) AppConfig {
//	    return loadConfig()
//	})
func (o *StartValueOrchestrator[T]) Enter(f func(ctx context.Context, span trace.Span) T) T {
	if f == nil {
		return zero.Value[T]()
	}

	value, err := invoke[T](o.ctx, o.name, func(ctx context.Context, span trace.Span) (T, error) {
		return f(ctx, span), nil
	}, o.opts...)
	if err != nil {
		panic(err)
	}

	return value
}

// StartValueErrorOrchestrator orchestrates the execution of a function that takes a context
// and returns both a value and an error. Create via spans.StartValErr().
type StartValueErrorOrchestrator[T any] struct {
	ctx  context.Context //nolint:containedctx
	name string
	opts []Option
}

// Enter executes the given function within an OpenTelemetry span.
// The function signature is: func(context.Context, trace.Span) (T, error)
//
// Returns the value and error from the wrapped function.
// Errors are automatically recorded in the span with an Error status.
// Panics are recovered, recorded in the span, and re-raised.
//
// Example:
//
//	user, err := spans.StartValErr[User](ctx, "fetch-user",
//	    spans.WithAttribute("user_id", attribute.StringValue(id)),
//	).Enter(func(ctx context.Context, span trace.Span) (User, error) {
//	    return database.GetUser(ctx, id)
//	})
func (o *StartValueErrorOrchestrator[T]) Enter(f func(ctx context.Context, span trace.Span) (T, error)) (T, error) {
	if f == nil {
		return zero.Value[T](), nil
	}

	return invoke[T](o.ctx, o.name, f, o.opts...)
}
