# Package: assert

Type assertion utilities with error handling.

## Usage

```go
// Safe type assertion
val, err := assert.Type[string](someInterface)
if err != nil {
    // Handle wrong type error
}

// Use with any interface{}
result, err := assert.Type[*MyStruct](data)
```

## Common Patterns

- Safe type assertions without panic
- Returns `errors.ErrWrongType` on mismatch
- Useful when working with interface{} or any types

## Gotchas

- Unlike Go's type assertion (x.(T)), this returns error instead of panicking
- Provides clear error messages showing expected vs actual types
