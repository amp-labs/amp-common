package spans

import "context"

// Start creates an orchestrator for executing a function that takes a context and span
// but returns nothing. Use this for side-effect operations like logging or sending events.
//
// The orchestrator executes the function within an OpenTelemetry span if a tracer is
// configured in the context via WithTracer(). If no tracer is present, the function
// executes normally without creating a span.
//
// The function signature expected by Enter() is: func(context.Context, trace.Span)
//
// Options can be provided to customize span behavior such as attributes, kind, status
// messages, and decorators. See WithAttribute, WithSpanKind, WithSuccessMessage, etc.
//
// Example:
//
//	spans.Start(ctx, "send-notification",
//	    spans.WithAttribute("recipient", attribute.StringValue(email)),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    sendEmail(ctx, email)
//	})
func Start(
	ctx context.Context, name string, opts ...Option,
) *StartOrchestrator {
	return &StartOrchestrator{
		ctx:  ctx,
		name: name,
		opts: opts,
	}
}

// StartErr creates an orchestrator for executing a function that takes a context and span
// and returns an error. Use this for operations that can fail but don't return a value.
//
// The orchestrator executes the function within an OpenTelemetry span if a tracer is
// configured in the context via WithTracer(). If no tracer is present, the function
// executes normally without creating a span.
//
// Errors returned by the function are automatically recorded in the span with an Error status.
//
// The function signature expected by Enter() is: func(context.Context, trace.Span) error
//
// Options can be provided to customize span behavior such as attributes, kind, status
// messages, and decorators. See WithAttribute, WithSpanKind, WithErrorMessage, etc.
//
// Example:
//
//	err := spans.StartErr(ctx, "validate-input",
//	    spans.WithErrorMessage("Validation failed"),
//	).Enter(func(ctx context.Context, span trace.Span) error {
//	    return validateData(input)
//	})
func StartErr(
	ctx context.Context, name string, opts ...Option,
) *StartErrorOrchestrator {
	return &StartErrorOrchestrator{
		ctx:  ctx,
		name: name,
		opts: opts,
	}
}

// StartVal creates an orchestrator for executing a function that takes a context and span
// and returns a typed value. Use this for operations that produce a result but cannot fail.
//
// The orchestrator executes the function within an OpenTelemetry span if a tracer is
// configured in the context via WithTracer(). If no tracer is present, the function
// executes normally without creating a span.
//
// The function signature expected by Enter() is: func(context.Context, trace.Span) T
//
// If the wrapped function panics, the panic is recorded in the span and re-raised.
//
// Options can be provided to customize span behavior such as attributes, kind, status
// messages, and decorators. See WithAttribute, WithSpanKind, WithSuccessMessage, etc.
//
// Example:
//
//	config := spans.StartVal[AppConfig](ctx, "load-config",
//	    spans.WithSuccessMessage("Configuration loaded"),
//	).Enter(func(ctx context.Context, span trace.Span) AppConfig {
//	    return loadConfigFromEnv()
//	})
func StartVal[Value any](
	ctx context.Context, name string, opts ...Option,
) *StartValueOrchestrator[Value] {
	return &StartValueOrchestrator[Value]{
		ctx:  ctx,
		name: name,
		opts: opts,
	}
}

// StartValErr creates an orchestrator for executing a function that takes a context and span
// and returns both a typed value and an error. This is the most common pattern for fallible
// operations that produce results.
//
// The orchestrator executes the function within an OpenTelemetry span if a tracer is
// configured in the context via WithTracer(). If no tracer is present, the function
// executes normally without creating a span.
//
// Errors returned by the function are automatically recorded in the span with an Error status.
//
// The function signature expected by Enter() is: func(context.Context, trace.Span) (T, error)
//
// Options can be provided to customize span behavior such as attributes, kind, status
// messages, and decorators. See WithAttribute, WithSpanKind, WithErrorMessage, etc.
//
// Example:
//
//	user, err := spans.StartValErr[User](ctx, "fetch-user",
//	    spans.WithAttribute("user_id", attribute.StringValue(id)),
//	    spans.WithErrorMessage("User fetch failed"),
//	).Enter(func(ctx context.Context, span trace.Span) (User, error) {
//	    return db.GetUser(ctx, id)
//	})
func StartValErr[Value any](
	ctx context.Context, name string, opts ...Option,
) *StartValueErrorOrchestrator[Value] {
	return &StartValueErrorOrchestrator[Value]{
		ctx:  ctx,
		name: name,
		opts: opts,
	}
}
