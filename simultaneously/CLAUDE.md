# Package: simultaneously

Run functions concurrently with controlled parallelism, context cancellation, and panic recovery.

## Usage

```go
import "github.com/amp-labs/amp-common/simultaneously"

// Run functions in parallel (limit concurrency to 3)
err := simultaneously.Do(3,
    func(ctx context.Context) error { return task1(ctx) },
    func(ctx context.Context) error { return task2(ctx) },
    func(ctx context.Context) error { return task3(ctx) },
    func(ctx context.Context) error { return task4(ctx) },
)

// With custom context
err := simultaneously.DoCtx(ctx, 3, task1, task2, task3)

// Reusable executor
exec := simultaneously.NewDefaultExecutor(3, 10)
err := simultaneously.DoWithExecutor(exec, tasks...)
defer exec.Close()
```

## Common Patterns

- `Do()` / `DoCtx()` - Run functions with max concurrency limit
- Returns first error encountered
- Cancels remaining functions on error (via context)
- Automatic panic recovery (panics converted to errors)
- `maxConcurrent < 1` means unlimited parallelism
- Semaphore-based concurrency control

## Gotchas

- Functions should check context for cancellation
- Panics don't crash process (recovered and returned as errors)
- All functions cancelled if any fails
- Executor can be reused across batches

## Related

- Package has README with detailed examples
