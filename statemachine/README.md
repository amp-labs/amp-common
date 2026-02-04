# State Machine Framework

A generic, reusable state machine framework for building declarative workflows in Go.

## Overview

This package provides a declarative state machine framework that has been extracted from the server repository into amp-common for reusability across projects.

## Architecture

### Core Framework (in amp-common)

The core framework provides:

- **Engine**: State machine orchestration and execution
- **Config Loading**: Flexible configuration loading with pluggable loaders
- **States**: Action, Conditional, and Final state types
- **Transitions**: Rule-based state transitions with expression evaluation
- **Actions**: Composable action system with sequence, conditional support
- **Context**: Thread-safe data carrier for workflow state
- **Observability**: Built-in metrics, logging, and OpenTelemetry tracing
- **Developer Tools**: Validation, visualization, and testing utilities

### Extensibility

The framework is designed to be extended by applications:

**Config Loading:**

```go
// Applications implement ConfigLoader to provide embedded configs
type ConfigLoader interface {
    LoadByName(name string) ([]byte, error)
    ListAvailable() []string
}

// Register your loader
statemachine.SetConfigLoader(myLoader)
```

**Custom Actions:**

```go
// Register custom action builders
factory := statemachine.NewActionFactory()
factory.Register("my_action", func(factory *ActionFactory, name string, params map[string]any) (Action, error) {
    // Create your custom action
    return myAction, nil
})
```

## Usage

### Basic Example

```go
package main

import (
    "context"
    sm "github.com/amp-labs/amp-common/statemachine"
)

func main() {
    // Load configuration (requires a ConfigLoader to be registered)
    config, err := sm.LoadConfig("my_workflow")
    if err != nil {
        panic(err)
    }

    // Create engine
    engine, err := sm.NewEngine(config, nil)
    if err != nil {
        panic(err)
    }

    // Execute workflow
    ctx := context.Background()
    smCtx := sm.NewContext("session-123", "project-456")

    if err := engine.Execute(ctx, smCtx); err != nil {
        panic(err)
    }
}
```

### Custom Actions

```go
// Define your action type
type MyAction struct {
    sm.BaseAction
    // ... your fields
}

func (a *MyAction) Execute(ctx context.Context, smCtx *sm.Context) error {
    // ... your logic
    return nil
}

// Register it with the factory
factory.Register("my_action", func(f *sm.ActionFactory, name string, params map[string]any) (sm.Action, error) {
    return &MyAction{BaseAction: sm.BaseAction{name: name}}, nil
})
```

## Dependencies

This package depends on `github.com/amp-labs/server` for sampling and elicitation packages. This is acceptable because:

1. It's a local replace dependency (not published)
2. The server is the primary consumer
3. The framework remains generic - sampling/elicitation are just example actions

## Migration from Server

If you're migrating code that used `github.com/amp-labs/server/builder-mcp/statemachine`:

1. Update imports: `github.com/amp-labs/server/builder-mcp/statemachine` → `github.com/amp-labs/amp-common/statemachine`
2. If you used embedded configs, implement `ConfigLoader` in your application
3. If you used custom actions with sampling/elicitation, keep those in your application

## Documentation

For detailed documentation, see the original README and docs in the server repository at `/builder-mcp/statemachine/`.
