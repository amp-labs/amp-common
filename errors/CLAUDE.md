# Package: errors

Error utilities with collection support for managing multiple errors.

## Usage

```go
// Collect errors from multiple operations
errs := &errors.Collection{}
errs.Add(operation1())
errs.Add(operation2())
if errs.HasError() {
    return errs.GetError()  // Returns joined error
}

// Safe collection with panic recovery
err := errors.Collect(func(errs *errors.Collection) {
    errs.Add(riskyOperation())
    errs.Add(anotherOperation())
})
```

## Common Patterns

- `Collection` - Accumulate multiple errors
- `Collect()` - Safe wrapper with panic recovery
- Returns nil if no errors, single error if one, joined if multiple
- Common sentinel errors: ErrNotImplemented, ErrWrongType, ErrValidation

## Gotchas

- Collection is NOT thread-safe
- Nil errors are automatically ignored
- Panics in Collect() are wrapped as ErrPanicRecovery with stack trace
