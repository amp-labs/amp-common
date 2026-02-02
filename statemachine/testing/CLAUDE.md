# Package: statemachine/testing

Testing utilities for state machine workflows.

## Overview

See [README.md](./README.md) for comprehensive testing patterns.

## Quick Reference

Test utilities provide:
- Mock action implementations
- Test context builders
- Assertion helpers
- Workflow execution testing

## Example

```go
// Create test engine
engine, err := statemachine.NewEngine(ctx, testConfig, factory)

// Execute and verify
result, err := engine.Execute(ctx, testData)
assert.NoError(t, err)
assert.Equal(t, "expected_state", result.CurrentState)
```

## Related

- [README.md](./README.md) - Full testing documentation
- `statemachine` - Core framework
