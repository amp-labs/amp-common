# Package: utils

Miscellaneous utilities for common operations (JSON, channels, context, sleep, dedup, panic recovery, UUID, etc.).

## Usage

```go
// JSON conversion
jsonMap, err := utils.ToJSONMap(myStruct)

// Dedup slices
unique := utils.Dedup([]string{"a", "b", "a"})  // ["a", "b"]

// Panic recovery
err := utils.GetPanicRecoveryError(recovered, stack)

// UUID validation
if utils.IsValidUUID(str) { /* ... */ }

// Sleep with context
utils.SleepWithContext(ctx, 5*time.Second)
```

## Common Patterns

- `ToJSONMap()` - Convert struct to map[string]any
- `Dedup()` - Remove duplicates from slices
- `GetPanicRecoveryError()` - Convert panic to error with stack
- `IsValidUUID()` - UUID validation
- `SleepWithContext()` - Context-aware sleep
- Also includes: Pushd (change dir), Ticker, Grep, Func utilities

## Gotchas

- Mixed bag of utilities - check individual functions for thread safety
- Most are standalone helper functions
