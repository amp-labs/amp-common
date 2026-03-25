# Package: pointer

Utilities for working with pointers - creating pointers from values and safely dereferencing them.

## Usage

```go
// Create pointer from literal
name := pointer.To("John")  // *string

// Safe dereferencing
val, ok := pointer.Value(name)       // "John", true
val := pointer.ValueOrDefault(name, "default")
val := pointer.ValueOrZero(name)     // returns zero value if nil
```

## Common Patterns

- `To()` - Convert literals to pointers (useful for API fields)
- `Value()` - Safe dereference with ok-check
- `ValueOrDefault()` - Dereference with fallback
- `ValueOrPanic()` - Use when nil is a programming error

## Gotchas

- `ValueOrPanic()` should only be used when nil represents a bug
- Use `ValueOrZero()` when zero value is acceptable fallback
