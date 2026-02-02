# Package: zero

Get zero values for generic types and check if values are zero.

## Usage

```go
// Get zero value for a type
defaultInt := zero.Value[int]()      // 0
defaultStr := zero.Value[string]()   // ""

// Check if value is zero
zero.IsZero(0)        // true
zero.IsZero("hello")  // false
```

## Common Patterns

- Use when working with generic functions that need zero values
- `IsZero()` uses reflect.DeepEqual for comparison

## Gotchas

- `IsZero()` uses reflection, so has some performance cost
- Works with all types including structs and pointers
