# Package: lazy

Thread-safe lazy initialization of values.

## Usage

```go
// Simple lazy value
expensive := lazy.New(func() *Database {
    return connectToDatabase()
})
db := expensive.Get()  // Initialized once, cached thereafter

// With context
lazyVal := lazy.NewCtx(func(ctx context.Context) string {
    return loadConfig(ctx)
})
val := lazyVal.Get(ctx)
```

## Common Patterns

- `Of[T]` - Simple lazy init
- `OfErr[T]` - Lazy init with errors (errors NOT memoized)
- `OfCtx[T]` - Context-aware with overrides
- `OfCtxErr[T]` - Context-aware with errors

## Gotchas

- Initialized at most once (thread-safe)
- Panics reset initialization state for retry
- Context types support testing overrides and named values
- Errors in OfErr/OfCtxErr are NOT cached - will retry on next Get()

## Related

- Used by `stage` package for environment detection
