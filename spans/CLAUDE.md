# Package: spans

Fluent API for creating OpenTelemetry spans with automatic lifecycle management.

## Usage

```go
// Setup tracer in context
ctx = spans.WithTracer(ctx, tracer)

// Create spans with different signatures
spans.Start(ctx, "operation").Enter(func(ctx context.Context, span trace.Span) {
    // Work here
})

result, err := spans.StartValErr[int](ctx, "fetch-data",
    spans.WithAttribute("id", attribute.StringValue("123")),
).Enter(func(ctx context.Context, span trace.Span) (int, error) {
    return fetchData(ctx)
})
```

## Common Patterns

- `Start` - No return value
- `StartErr` - Returns error only
- `StartVal[T]` - Returns value only
- `StartValErr[T]` - Returns value and error
- Automatic panic recovery and error recording

## Gotchas

- All functions receive both context and span
- Span lifecycle managed automatically
- Requires tracer set via `WithTracer()`

## Related

- `telemetry` - OpenTelemetry setup
