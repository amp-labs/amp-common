# Package: contexts

Context utilities including lifecycle management, type-safe value storage, and atomic context updates.

## Usage

```go
// Type-safe context values
ctx = contexts.WithValue[myKey, string](ctx, key, "value")
val, ok := contexts.GetValue[myKey, string](ctx, key)

// Ensure non-nil context
ctx = contexts.EnsureContext(maybeNilCtx, anotherCtx)

// Check if context is alive
if contexts.IsContextAlive(ctx) {
    // Context not cancelled
}
```

## Common Patterns

- `WithValue/GetValue` - Type-safe alternative to context.WithValue/Value
- `EnsureContext()` - Returns first non-nil context or Background()
- `IsContextAlive()` - Non-blocking check if context is cancelled
- Package also includes: atomic context updates, lifecycle management, multi-value storage

## Gotchas

- Thread-safe context value storage with type safety
- `IsContextAlive()` is non-blocking (uses select with default)

## Related

- Used by `lazy`, `stage`, and other packages for context storage
