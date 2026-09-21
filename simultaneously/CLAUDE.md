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
exec := simultaneously.NewDefaultExecutor(3)
defer exec.Close()
err := simultaneously.DoWithExecutor(exec, tasks...)

// Ambient executor: put one pool under a whole subtree of work.
// Both calls below run on exec.
ctx = simultaneously.WithExecutor(ctx, exec)
err := simultaneously.DoCtx(ctx, 0, tasks...)
out, err := simultaneously.MapSliceCtx(ctx, 0, vals, xform)
```

## Common Patterns

- `Job` - Alias for `func(ctx context.Context) error`; the unit of work every Do* takes.
  Being an alias (not a defined type), `[]Job` and `[]func(ctx context.Context) error`
  are the same type and spread interchangeably.
- `Do()` / `DoCtx()` - Run functions with max concurrency limit
- Returns first error encountered
- Cancels remaining functions on error (via context)
- Automatic panic recovery (panics converted to errors)
- `maxConcurrent < 1` means unlimited parallelism
- Semaphore-based concurrency control

## Ambient Executors

`WithExecutor(ctx, exec)` attaches an executor to the context. Every
context-taking entry point -- `DoCtx` and the whole `MapXCtx`/`FlatMapXCtx`
family -- then runs its work on it instead of building a throwaway one. One
resolver, `resolveExecutor`, holds this rule for all of them; add new entry
points through it rather than calling `newDefaultExecutor` directly.

- The executor is **not closed** by the call that finds it -- whoever attached
  it owns its lifetime
- `maxConcurrent` is **ignored** when an ambient executor is present; the
  executor's own limit governs
- The context-free variants (`Do`, `MapSlice`, ...) start from
  `context.Background()` and never see one; the `*WithExecutor` variants use
  what they were handed

## Gotchas

- Functions should check context for cancellation
- Panics don't crash process (recovered and returned as errors)
- All functions cancelled if any fails
- Executor can be reused across batches
- **Nested calls share an ambient executor and can deadlock**: a job's context
  still carries the executor, so a nested `DoCtx`/`MapXCtx` queues onto the pool
  its caller is occupying. Only context cancellation breaks the cycle. Size the
  executor above the peak nesting depth, or hand the inner level its own
- A panicking custom `Executor` is recovered in `collector.launch` and reported
  as that callback's error; an `Executor` that returns from `GoContext` without
  ever calling `done` will still hang the call

## Related

- Package has README with detailed examples
