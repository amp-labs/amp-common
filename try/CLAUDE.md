# Package: try

Result type for error handling - encapsulates value or error.

## Usage

```go
// Create Try from operation
result := try.Try[string]{Value: "success", Error: nil}

// Check success/failure
if result.IsSuccess() {
    val := result.Value
}

// Get with fallback
val := result.GetOrElse("default")

// Chain operations
result2 := try.Map(result, func(s string) (int, error) {
    return strconv.Atoi(s)
})
```

## Common Patterns

- Alternative to `(value, error)` return pattern
- Use `Map()` to transform successful values
- Use `FlatMap()` to chain operations returning Try
- `GetOrElse()` provides fallback values

## Gotchas

- Inspired by functional programming patterns (Scala, Rust Result)
