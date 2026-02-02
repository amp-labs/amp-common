# Package: statemachine/validator

Configuration validation for state machine workflows.

## Overview

See [README.md](./README.md) for validation rules and usage.

## Quick Reference

Validates:
- State machine structure
- Transition rules
- Action configurations
- Expression syntax
- State reachability

## Example

```go
// Validate config
errs := validator.Validate(configBytes)
if len(errs) > 0 {
    for _, err := range errs {
        log.Error(err)
    }
}
```

## Related

- [README.md](./README.md) - Full validation documentation
- `statemachine` - Core framework
