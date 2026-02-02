# Package: closer

Resource management utilities for io.Closer with collectors, close-once wrappers, and panic handling.

## Usage

```go
// Collect multiple closers
closer := closer.NewCloser()
closer.Add(file)
closer.Add(conn)
defer closer.Close()  // Closes all, collects errors

// Close only once (thread-safe)
onceCloser := closer.CloseOnce(resource)
defer onceCloser.Close()  // Safe to call multiple times

// Handle panics during close
safeCloser := closer.HandlePanic(riskyResource)

// Custom cleanup as closer
cleanup := closer.CustomCloser(func() error {
    return cleanupResources()
})
```

## Common Patterns

- `Closer` - Collector for multiple io.Closer instances
- `CloseOnce` - Thread-safe single-close wrapper
- `HandlePanic` - Converts close panics to errors
- `CustomCloser` - Wrap any cleanup function as io.Closer

## Gotchas

- `Closer` collector closes all even if some return errors
- Errors are collected and joined
- Thread-safe wrappers provided
