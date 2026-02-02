# Package: shutdown

Graceful shutdown coordination with signal handling and cleanup hooks.

## Usage

```go
func main() {
    ctx := shutdown.SetupHandler()  // Handles SIGINT, SIGTERM

    shutdown.BeforeShutdown(func() {
        // Cleanup logic here
        db.Close()
    })

    // Run app with ctx
    <-ctx.Done()
    // Hooks have been called, context is cancelled
}
```

## Common Patterns

- `SetupHandler()` - Sets up signal handlers, returns cancellable context
- `BeforeShutdown()` - Register cleanup hooks
- `Shutdown()` - Trigger shutdown programmatically
- Hooks run before context cancellation

## Gotchas

- Context cancelled after all BeforeShutdown hooks complete
- Hooks run in registration order
- Thread-safe hook registration
- Supports SIGINT and SIGTERM

## Related

- `bgworker` - Uses shutdown hooks for pool cleanup
