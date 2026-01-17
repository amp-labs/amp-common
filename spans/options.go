package spans

import (
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// WithName overrides the span name from the default name provided to Start/StartErr/StartVal/StartValErr.
//
// This is useful when you want to compute the span name dynamically or conditionally.
//
// Example:
//
//	spans.Start(ctx, "default-name",
//	    spans.WithName(fmt.Sprintf("process-%s", taskType)),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    // span will be named "process-{taskType}"
//	})
func WithName(name string) Option {
	return func(r *runner) {
		r.spanName = name
	}
}

// WithAttribute adds an attribute to the span when it is created.
//
// Attributes are key-value pairs that provide additional context about the span.
// Use OpenTelemetry's attribute types for values: StringValue, IntValue, BoolValue, etc.
//
// Multiple attributes can be added by calling this option multiple times or using
// WithSpanStartOptions with trace.WithAttributes().
//
// Example:
//
//	spans.Start(ctx, "process-payment",
//	    spans.WithAttribute("payment_method", attribute.StringValue("credit_card")),
//	    spans.WithAttribute("amount", attribute.Float64Value(99.99)),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    processPayment(ctx)
//	})
func WithAttribute(key attribute.Key, value attribute.Value) Option {
	return func(r *runner) {
		r.sso = append(r.sso, trace.WithAttributes(attribute.KeyValue{
			Key:   key,
			Value: value,
		}))
	}
}

// WithSpanKind sets the OpenTelemetry span kind, which indicates the role of the span
// in a trace. The default is SpanKindServer.
//
// Common span kinds:
//   - SpanKindServer: The span represents work done by a server (default)
//   - SpanKindClient: The span represents a client call to an external service
//   - SpanKindProducer: The span represents putting a message into a queue
//   - SpanKindConsumer: The span represents receiving a message from a queue
//   - SpanKindInternal: The span represents internal application operations
//
// Example:
//
//	spans.Start(ctx, "fetch-user-api",
//	    spans.WithSpanKind(trace.SpanKindClient),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    callExternalAPI(ctx)
//	})
func WithSpanKind(kind trace.SpanKind) Option {
	return func(r *runner) {
		r.spanKind = kind
	}
}

// WithSuccessMessage sets a custom success message for the span status.
//
// When the wrapped function completes without error, this message is set as the
// span's status description with codes.Ok. If not provided, defaults to "ok".
//
// This is useful for providing human-readable context about what succeeded.
//
// Example:
//
//	spans.Start(ctx, "send-email",
//	    spans.WithSuccessMessage("Email sent successfully to user"),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    sendEmail(ctx)
//	})
func WithSuccessMessage(description string) Option {
	return func(r *runner) {
		r.success = description
	}
}

// WithErrorMessage sets a custom error message prefix for the span status.
//
// When the wrapped function returns an error, this prefix is prepended to the error
// message in the span's status description. If not provided, only the error message is used.
//
// This is useful for providing additional context about what operation failed.
//
// Example:
//
//	err := spans.StartErr(ctx, "validate-input",
//	    spans.WithErrorMessage("Input validation failed"),
//	).Enter(func(ctx context.Context, span trace.Span) error {
//	    return validateInput(data)
//	})
//	// If error occurs, span status will be: "Input validation failed: {error message}"
func WithErrorMessage(description string) Option {
	return func(r *runner) {
		r.failure = description
	}
}

// WithSpanStartOptions provides raw OpenTelemetry span start options.
//
// This is an escape hatch for advanced span configuration that isn't covered by
// the other With* functions. Common uses include adding links, timestamps, or
// multiple attributes at once.
//
// See go.opentelemetry.io/otel/trace.SpanStartOption for available options.
//
// Example:
//
//	spans.Start(ctx, "process-batch",
//	    spans.WithSpanStartOptions(
//	        trace.WithAttributes(
//	            attribute.Int("batch_size", len(items)),
//	            attribute.String("batch_id", batchID),
//	        ),
//	        trace.WithTimestamp(startTime),
//	    ),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    processBatch(ctx, items)
//	})
func WithSpanStartOptions(options ...trace.SpanStartOption) Option {
	return func(r *runner) {
		r.sso = append(r.sso, options...)
	}
}

// WithSpanEndOptions provides raw OpenTelemetry span end options.
//
// This is an escape hatch for advanced span finalization that isn't covered by
// the other With* functions. Common uses include setting a custom end timestamp.
//
// See go.opentelemetry.io/otel/trace.SpanEndOption for available options.
//
// Example:
//
//	spans.Start(ctx, "process-batch",
//	    spans.WithSpanEndOptions(trace.WithTimestamp(endTime)),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    processBatch(ctx)
//	})
func WithSpanEndOptions(options ...trace.SpanEndOption) Option {
	return func(r *runner) {
		r.seo = append(r.seo, options...)
	}
}

// WithSpanDecorator registers a function to decorate the span after creation.
//
// Decorator functions are called after the span is created and before the wrapped
// function executes. They receive the span and can add attributes, events, or perform
// other customizations.
//
// Multiple decorators can be registered and will be executed in order.
//
// Example:
//
//	spans.Start(ctx, "process-request",
//	    spans.WithSpanDecorator(func(span trace.Span) {
//	        span.SetAttributes(
//	            attribute.String("request_id", requestID),
//	            attribute.Int64("start_time", time.Now().Unix()),
//	        )
//	        span.AddEvent("processing started")
//	    }),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    processRequest(ctx)
//	})
func WithSpanDecorator(decorator func(span trace.Span)) Option {
	return func(r *runner) {
		r.decorate = append(r.decorate, decorator)
	}
}

// WithAutoEnd controls whether the span is automatically ended when the wrapped
// function completes. The default is true.
//
// Set to false when you need to manually control span lifecycle, such as for
// long-running operations where you want to end the span at a specific point
// rather than when the function returns.
//
// When autoEnd is false:
//   - You must call span.End() manually
//   - Span status is NOT automatically set (you must call span.SetStatus)
//   - Success/error messages from WithSuccessMessage/WithErrorMessage are ignored
//
// Example:
//
//	spans.Start(ctx, "async-operation",
//	    spans.WithAutoEnd(false),
//	).Enter(func(ctx context.Context, span trace.Span) {
//	    defer span.End() // manual span end
//
//	    startAsyncWork(ctx)
//	    // span continues after function returns
//	})
func WithAutoEnd(autoEnd bool) Option {
	return func(r *runner) {
		r.autoEnd = autoEnd
	}
}
