# State Machine Testing Helpers

The testing package provides rich test utilities that make testing state machines intuitive, comprehensive, and maintainable.

## Features

- **TestEngine** - Enhanced engine wrapper with execution tracing and assertions
- **Test Fixtures** - Mock clients, common configs, and test data helpers
- **Test Scenarios** - Pre-built scenarios for common patterns
- **Fluent Matchers** - Readable assertion DSL
- **Execution Tracing** - Record and inspect state machine execution

## Installation

```go
import smtesting "github.com/amp-labs/server/builder-mcp/statemachine/testing"
```

## Quick Start

### Basic Test with TestEngine

```go
func TestMyWorkflow(t *testing.T) {
    // Create config
    config := &statemachine.Config{
        Name:         "test",
        InitialState: "start",
        FinalStates:  []string{"end"},
        States: []statemachine.StateConfig{
            {Name: "start", Type: "action"},
            {Name: "end", Type: "final"},
        },
        Transitions: []statemachine.TransitionConfig{
            {From: "start", To: "end", Condition: "always"},
        },
    }

    // Create test engine
    engine := smtesting.NewTestEngine(t, config)

    // Execute
    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(map[string]any{
        "input": "test data",
    })

    err := engine.Execute(ctx, smCtx)
    require.NoError(t, err)

    // Assert execution
    engine.AssertStateVisited("start")
    engine.AssertTransitionTaken("start", "end")
    engine.AssertFinalState("end")
}
```

### Using Common Test Configs

```go
func TestLinearWorkflow(t *testing.T) {
    // Use pre-built config
    config := smtesting.CommonTestConfigs.Linear()
    engine := smtesting.NewTestEngine(t, config)

    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(nil)

    err := engine.Execute(ctx, smCtx)
    require.NoError(t, err)
}
```

## TestEngine

The `TestEngine` wraps the state machine engine with testing capabilities:

### Creating a TestEngine

```go
config := &statemachine.Config{...}
engine := smtesting.NewTestEngine(t, config)
```

### Assertions

#### Assert State Visit

```go
engine.AssertStateVisited("my_state")
// Fails test if state was not visited
```

#### Assert Transition

```go
engine.AssertTransitionTaken("state_a", "state_b")
// Fails test if transition did not occur
```

#### Assert Final State

```go
engine.AssertFinalState("complete")
// Fails test if final state doesn't match
```

#### Assert Context Values

```go
engine.AssertContextValue("result", "success")
// Fails test if context value doesn't match
```

#### Assert Execution Time

```go
engine.AssertExecutionTime(100 * time.Millisecond)
// Fails test if execution took longer
```

### Execution Trace

Access the full execution trace for custom assertions:

```go
trace := engine.GetTrace()

for _, entry := range trace {
    fmt.Printf("State: %s, Duration: %s\n", entry.State, entry.Duration)
    if entry.Error != nil {
        fmt.Printf("Error: %v\n", entry.Error)
    }
}
```

## Test Fixtures

### Mock Sampling Client

```go
responses := map[string]string{
    "What is the capital of France?": "Paris",
    "What is 2+2?": "4",
}

mockClient := smtesting.NewMockSamplingClient(responses)
response, _ := mockClient.Sample("What is the capital of France?")
// response = "Paris"
```

### Mock Elicitation Client

```go
responses := map[string]any{
    "Choose a color": "blue",
    "Enter a number": 42,
}

mockClient := smtesting.NewMockElicitationClient(responses)
answer, _ := mockClient.Elicit("Choose a color")
// answer = "blue"
```

### Create Test Context

```go
ctx := smtesting.CreateTestContext(map[string]any{
    "user_id": "123",
    "account": "acme",
})

// Context has test-session and test-project IDs
```

### Common Test Configs

Pre-built configs for common patterns:

```go
// Linear: start -> middle -> end
config := smtesting.CommonTestConfigs.Linear()

// Branching: start -> success | failure
config := smtesting.CommonTestConfigs.Branching()

// Loop: start -> retry (loop) -> complete
config := smtesting.CommonTestConfigs.Loop()

// Complex: Multiple states with error handling
config := smtesting.CommonTestConfigs.Complex()
```

## Test Scenarios

Test scenarios encapsulate complete test cases:

### Using Built-in Scenarios

```go
func TestScenarios(t *testing.T) {
    scenarios := []smtesting.TestScenario{
        smtesting.LinearWorkflowScenario(),
        smtesting.BranchingWorkflowScenario(),
        smtesting.RetryScenario(),
        smtesting.ErrorRecoveryScenario(),
    }

    for _, scenario := range scenarios {
        smtesting.RunScenario(t, scenario)
    }
}
```

### Creating Custom Scenarios

```go
scenario := smtesting.TestScenario{
    Name: "My Custom Scenario",
    Config: &statemachine.Config{...},
    InitialContext: map[string]any{
        "input": "test",
    },
    MockResponses: map[string]any{
        "question": "answer",
    },
    Assertions: []smtesting.Assertion{
        // Custom assertions
    },
}

smtesting.RunScenario(t, scenario)
```

## Fluent Matchers

Matchers provide a readable assertion DSL:

### Basic Matchers

```go
matcher := smtesting.StateWasVisited("process")
matched, err := matcher.Match(engine)
// matched = true if state was visited
```

### Available Matchers

```go
// State matchers
smtesting.StateWasVisited("state_name")
smtesting.TransitionWasTaken("from", "to")

// Context matchers
smtesting.ContextContains("key", "value")

// Execution matchers
smtesting.ExecutionCompleted()
smtesting.ExecutionFailed()
smtesting.ExecutionTookLessThan(100 * time.Millisecond)
```

### Combining Matchers

```go
// All must pass
matcher := smtesting.All(
    smtesting.StateWasVisited("start"),
    smtesting.StateWasVisited("end"),
    smtesting.ExecutionCompleted(),
)

// At least one must pass
matcher := smtesting.Any(
    smtesting.StateWasVisited("success"),
    smtesting.StateWasVisited("failure"),
)
```

### Using Matchers in Tests

```go
func TestWithMatchers(t *testing.T) {
    engine := smtesting.NewTestEngine(t, config)

    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(nil)

    _ = engine.Execute(ctx, smCtx)

    // Use matchers
    matchers := []smtesting.Matcher{
        smtesting.StateWasVisited("start"),
        smtesting.TransitionWasTaken("start", "complete"),
        smtesting.ExecutionCompleted(),
    }

    for _, matcher := range matchers {
        matched, err := matcher.Match(engine)
        require.True(t, matched, matcher.Description())
        require.NoError(t, err)
    }
}
```

## Testing Patterns

### Unit Testing Actions

```go
func TestSingleAction(t *testing.T) {
    config := &statemachine.Config{
        Name:         "action_test",
        InitialState: "test_action",
        FinalStates:  []string{"test_action"},
        States: []statemachine.StateConfig{
            {
                Name: "test_action",
                Type: "action",
                Actions: []statemachine.ActionConfig{
                    {Type: "my_action", Name: "test"},
                },
            },
        },
    }

    engine := smtesting.NewTestEngine(t, config)
    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(map[string]any{
        "input": "test",
    })

    err := engine.Execute(ctx, smCtx)
    require.NoError(t, err)

    // Assert action effects
    engine.AssertContextValue("output", "expected")
}
```

### Integration Testing Complete Workflows

```go
func TestCompleteWorkflow(t *testing.T) {
    config := smtesting.CommonTestConfigs.Complex()
    engine := smtesting.NewTestEngine(t, config)

    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(map[string]any{
        "valid": true,
        "success": true,
    })

    err := engine.Execute(ctx, smCtx)
    require.NoError(t, err)

    // Verify complete path
    engine.AssertStateVisited("init")
    engine.AssertStateVisited("validate")
    engine.AssertStateVisited("process")
    engine.AssertFinalState("success")
}
```

### Testing Error Scenarios

```go
func TestErrorHandling(t *testing.T) {
    config := &statemachine.Config{...}
    engine := smtesting.NewTestEngine(t, config)

    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(map[string]any{
        "force_error": true,
    })

    err := engine.Execute(ctx, smCtx)
    // Depending on config, might expect error or error recovery

    // Verify error handling path
    engine.AssertStateVisited("error_handler")
}
```

### Performance Testing

```go
func TestPerformance(t *testing.T) {
    config := smtesting.CommonTestConfigs.Complex()
    engine := smtesting.NewTestEngine(t, config)

    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(nil)

    start := time.Now()
    err := engine.Execute(ctx, smCtx)
    duration := time.Since(start)

    require.NoError(t, err)
    assert.Less(t, duration, 100*time.Millisecond,
        "execution should complete quickly")
}
```

### Snapshot Testing

```go
func TestExecutionSnapshot(t *testing.T) {
    engine := smtesting.NewTestEngine(t, config)

    ctx := context.Background()
    smCtx := smtesting.CreateTestContext(nil)

    _ = engine.Execute(ctx, smCtx)

    // Get execution trace
    trace := engine.GetTrace()

    // Compare with golden snapshot
    // (would use snapshot testing library)
    compareWithSnapshot(t, trace, "workflow_execution.json")
}
```

## Best Practices

1. **Use common configs** - Leverage `CommonTestConfigs` for standard patterns
2. **Test edge cases** - Use scenarios to test error paths and edge cases
3. **Assert liberally** - Use multiple assertions to catch regressions
4. **Mock external services** - Use mock clients for deterministic tests
5. **Check execution traces** - Inspect traces for unexpected behavior
6. **Performance test** - Use `AssertExecutionTime` for critical workflows
7. **Snapshot tests** - Use execution traces for regression detection

## Advanced Usage

### Custom Assertions

```go
func assertCustomBehavior(t *testing.T, engine *smtesting.TestEngine) {
    trace := engine.GetTrace()

    // Custom validation logic
    for _, entry := range trace {
        if entry.Duration > 50*time.Millisecond {
            t.Errorf("State %s took too long: %s", entry.State, entry.Duration)
        }
    }
}
```

### Test Helpers

```go
func createTestEngine(t *testing.T, modifications ...func(*statemachine.Config)) *smtesting.TestEngine {
    config := smtesting.CommonTestConfigs.Linear()

    for _, modify := range modifications {
        modify(config)
    }

    return smtesting.NewTestEngine(t, config)
}
```

## See Also

- [State Machine README](../README.md)
- [Visualizer Package](../visualizer/README.md)
- [Validator Package](../validator/README.md)
- [Examples](../examples/)
