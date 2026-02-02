# Package: bgworker

Background worker pool management with graceful lifecycle control.

## Usage

```go
// Submit work to background pool
task := bgworker.Submit(ctx, func() {
    // Background work here
})
result := task.Wait()  // Wait for completion

// Fire and forget
err := bgworker.Go(ctx, func() {
    // Background work
})
```

## Common Patterns

- Lazy-initialized global worker pool
- Pool size from BACKGROUND_WORKER_COUNT env (default: 10)
- `Submit()` - Returns task handle for waiting
- `Go()` - Fire and forget
- Auto-stops on shutdown

## Gotchas

- Uses global pool (initialized once)
- Integrates with shutdown package for graceful stop
- Pool stops and waits for all tasks on shutdown

## Related

- `shutdown` - Graceful shutdown integration
- `lazy` - Lazy pool initialization
