# Package: statemachine/actions

Comprehensive action library for state machine workflows.

## Overview

See [README.md](./README.md) for full documentation.

## Quick Reference

Action categories:
- **Sampling** - AI-powered text generation with fallbacks
- **Elicitation** - User input collection
- **Validation** - Input validation with AI feedback
- **Composition** - Sequence, conditional, parallel execution

## Example

```go
// Register actions
factory.Register("sample_with_fallback", actions.NewSampleWithFallback)

// Use in config
{
  "name": "explain_error",
  "action": "sample_with_fallback",
  "params": {
    "prompt": "Explain: {{error}}",
    "fallback": "An error occurred"
  }
}
```

## Related

- [README.md](./README.md) - Full action documentation
- `statemachine` - Core framework
