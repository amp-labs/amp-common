# Package: validate

Unified validation framework for types implementing validation interfaces.

## Usage

```go
// Implement validation interface
type MyConfig struct { Port int }

func (c MyConfig) Validate() error {
    if c.Port < 1 || c.Port > 65535 {
        return fmt.Errorf("invalid port")
    }
    return nil
}

// Validate using framework
config := MyConfig{Port: 8080}
err := validate.Validate(ctx, config)  // Calls Validate() if implemented
```

## Common Patterns

- Supports `HasValidate` and `HasValidateWithContext` interfaces
- Automatic panic recovery during validation
- Exposes Prometheus metrics for validation operations
- Use `ValidateWithoutContext()` for non-context validation

## Gotchas

- Validation errors wrapped with `errors.ErrValidation`
- Panics during validation are caught and returned as errors
- Metrics track validation calls, failures, and panics

## Related

- `errors` - Sentinel error ErrValidation
