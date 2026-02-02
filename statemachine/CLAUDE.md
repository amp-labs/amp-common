# Package: statemachine

Generic declarative state machine framework for building workflows.

## Overview

See [README.md](./README.md) for comprehensive documentation.

## Quick Start

```go
// Create engine with config
engine, err := statemachine.NewEngine(ctx, configBytes, factory)

// Execute workflow
result, err := engine.Execute(ctx, initialData)
```

## Key Components

- **Engine** - State machine orchestration
- **States** - Action, Conditional, Final
- **Transitions** - Rule-based state transitions
- **Actions** - Composable action system
- **Context** - Thread-safe data carrier
- **Config Loading** - Pluggable loaders
- **Observability** - Metrics, logging, tracing

## Related

- [README.md](./README.md) - Full documentation
- `statemachine/actions` - Action library
- `statemachine/helpers` - Helper functions
- `statemachine/testing` - Testing utilities
- `statemachine/validator` - Config validation
- `statemachine/visualizer` - Visualization tools
