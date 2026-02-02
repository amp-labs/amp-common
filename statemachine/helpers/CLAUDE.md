# Package: statemachine/helpers

High-level convenience functions for common state machine patterns.

## Overview

See [README.md](./README.md) for full documentation.

## Quick Reference

Opinionated helpers with automatic:
- Capability checking
- Graceful fallbacks
- Standardized error handling
- Built-in logging

## Example

```go
// Sampling with fallback
explanation, err := helpers.SampleWithFallback(
    ctx, samplingClient,
    "Explain error: " + msg,
    "Operation failed. Check logs.",
)

// Elicitation with defaults
answers, err := helpers.ElicitWithDefaults(ctx, client, questions, defaults)
```

## Related

- [README.md](./README.md) - Full helper documentation
- `statemachine/actions` - Lower-level action library
