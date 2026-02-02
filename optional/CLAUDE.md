# Package: optional

Type-safe Optional/Maybe type for representing values that may or may not be present.

## Usage

```go
import "github.com/amp-labs/amp-common/optional"

// Create optionals
some := optional.Some(42)
none := optional.None[int]()

// Safe access
if val, ok := some.Get(); ok {
    fmt.Println(val)
}

// With fallbacks
val := some.GetOrElse(0)
val := some.GetOrElseFunc(func() int { return expensive() })

// Transformations
mapped := optional.Map(some, func(x int) (string, error) {
    return fmt.Sprint(x), nil
})

// Iteration (Go 1.23+ range support)
for val := range some.All() {
    fmt.Println(val)
}
```

## Common Patterns

- `Some()` / `None()` - Create optional values
- `Get()` - Safe extraction with ok-check
- `GetOrElse()` / `GetOrElseFunc()` - Fallback values
- `Map()` / `FlatMap()` - Transform optionals
- `Filter()` - Conditional values
- Supports iteration with `All()`

## Gotchas

- `GetOrPanic()` panics if empty (use sparingly)
- JSON serialization supported
- Empty optionals are conceptually sets of size 0
