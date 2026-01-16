# State Machine Actions Library

A comprehensive, composable library of actions for building declarative state machine workflows in the Ampersand builder-mcp.

## Overview

This action library transforms complex nested conditionals into declarative, reusable "lego blocks" that can be easily composed to create sophisticated interactive workflows. Each action is a focused, testable unit that implements a specific pattern (sampling, elicitation, validation, or composition).

## Design Principles

1. **Composition over inheritance** - Complex behaviors built by combining simple actions
2. **Capability checking** - Actions gracefully handle missing capabilities (sampling/elicitation)
3. **Predictable naming** - Consistent context key patterns (`{name}_result`, `{name}_valid`, etc.)
4. **Sensible fallbacks** - Always provide static alternatives when AI is unavailable
5. **Comprehensive documentation** - Every action includes inline docs and examples

## Action Categories

### Sampling Actions (`sampling_actions.go`)

AI-powered text generation with fallback support:

- **SampleWithFallback** - Sample with multiple fallback strategies (static, function, context)
- **SampleForExplanation** - Generate explanations for complex concepts
- **SampleForValidation** - AI-enhanced validation feedback
- **SampleForSuggestions** - Context-aware suggestions
- **SampleWithContext** - Sample with automatic context injection

### Elicitation Actions (`elicitation_actions.go`)

User interaction and form handling:

- **ElicitForm** - Structured form input with field mapping
- **ElicitConfirmation** - Simple yes/no confirmations
- **ElicitWithRetry** - Retry elicitation on validation failure
- **ElicitConditional** - Conditional elicitation based on context
- **ElicitWithSamplingHelp** - Forms with AI-generated help text

### Validation Actions (`validation_actions.go`)

Data validation with rich feedback:

- **ValidateInput** - Custom validators with multiple rules
- **ValidateState** - Multi-field state validation
- **ValidateTransition** - Check if ready for state transitions
- **ValidateWithSampling** - AI-enhanced validation feedback
- **ValidateAndConfirm** - Validate then ask for confirmation

### Composite Actions (`composite_actions.go`)

Advanced composition patterns:

- **TryWithFallback** - Try primary action, fall back on error
- **ValidatedSequence** - Sequential execution with validation between steps
- **ConditionalBranch** - Multi-way branching based on conditions
- **RetryWithBackoff** - Exponential backoff retry logic
- **ParallelWithMerge** - Parallel execution with result merging
- **ProgressiveDisclosure** - Multi-step conditional elicitation

## Quick Start

### Basic Example

```go
import (
    "context"
    "github.com/amp-labs/server/builder-mcp/statemachine"
    "github.com/amp-labs/server/builder-mcp/statemachine/actions"
)

// Create context
ctx := context.Background()
smCtx := statemachine.NewContext("session-id", "project-id")

// Use SampleWithFallback action
action := &actions.SampleWithFallback{
    Name:     "oauth_help",
    Prompt:   "Explain OAuth flow for Salesforce",
    Fallback: "OAuth is an authorization protocol...",
}

if err := action.Execute(ctx, smCtx); err != nil {
    return err
}

// Get result
result, _ := smCtx.GetString("oauth_help_result")
source, _ := smCtx.GetString("oauth_help_source") // "sampling" or "fallback"
```

### Composing Actions

```go
// Create a validation sequence
sequence := &actions.ValidatedSequence{
    Name: "oauth_setup",
    Actions: []actions.Action{
        &actions.ElicitForm{...},      // Get credentials
        &actions.ValidateInput{...},   // Validate format
        &actions.ElicitConfirmation{...}, // Confirm
    },
    Validators: map[int]actions.ValidationFunc{
        0: validateCredentialsFormat,
        1: validateCredentialsLogic,
    },
}

if err := sequence.Execute(ctx, smCtx); err != nil {
    return err
}
```

## Context Data Conventions

All actions follow consistent naming patterns:

| Pattern | Example | Description |
|---------|---------|-------------|
| `{name}_result` | `sample_help_result` | Primary result of action |
| `{name}_valid` | `validate_creds_valid` | Boolean validation result |
| `{name}_feedback` | `validate_creds_feedback` | Human-readable feedback |
| `{name}_confirmed` | `confirm_setup_confirmed` | Boolean confirmation |
| `{name}_source` | `sample_help_source` | Source of result (sampling/fallback) |
| `{name}_errors` | `validate_state_errors` | Array or map of errors |
| `{name}_skipped` | `elicit_advanced_skipped` | Boolean indicating skip |

## Common Patterns

### Pattern: OAuth Setup Flow

```go
// 1. Elicit provider
elicitProvider := &actions.ElicitForm{
    Name:    "provider",
    Request: /* form schema */,
}

// 2. Validate provider
validateProvider := &actions.ValidateTransition{
    Name:           "check_provider",
    RequiredFields: []string{"provider"},
}

// 3. Elicit credentials with AI help
elicitCreds := &actions.ElicitWithSamplingHelp{
    Name:        "credentials",
    Request:     /* form schema */,
    HelpPrompts: /* AI prompts per field */,
    StaticHelp:  /* fallback help */,
}

// 4. Validate and confirm
validateAndConfirm := &actions.ValidateAndConfirm{
    Name:                "final_check",
    DataContext:         "credentials",
    ValidationFunc:      validateOAuthCreds,
    ConfirmationMessage: "Proceed with OAuth setup?",
}
```

See `examples/oauth_setup_example.go` for complete implementation.

### Pattern: Validation with Retry

```go
// Validate with automatic retry on failure
elicitWithRetry := &actions.ElicitWithRetry{
    Name:    "validated_input",
    Request: /* form schema */,
    ValidationFunc: func(data map[string]any) (bool, string) {
        // Validation logic
        if invalid {
            return false, "Error message shown to user"
        }
        return true, ""
    },
    MaxRetries: 3,
}
```

### Pattern: Error Recovery

```go
// Try primary action, fall back gracefully
tryWithFallback := &actions.TryWithFallback{
    Name:           "get_config",
    PrimaryAction:  &actions.SampleWithFallback{...},
    FallbackAction: &actions.ElicitForm{...},
}
```

See `examples/error_recovery_example.go` for complete implementation.

### Pattern: Progressive Disclosure

```go
// Multi-step with conditional advanced settings
progressive := &actions.ProgressiveDisclosure{
    Name: "setup_flow",
    Steps: []actions.DisclosureStep{
        {
            Action:   basicSetupAction,
            Required: true,
        },
        {
            Action:   advancedSetupAction,
            Required: false,
            Condition: func(c *statemachine.Context) bool {
                wantAdvanced, _ := c.GetBool("want_advanced")
                return wantAdvanced
            },
        },
    },
}
```

See `examples/progressive_disclosure_example.go` for complete implementation.

## Testing

### Unit Testing

Use the test helpers in `testing.go`:

```go
func TestMyAction(t *testing.T) {
    // Create mock context
    ctx := actions.MockContext("test-session", "test-project", map[string]any{
        "initial_data": "value",
    })

    // Execute action
    action := &actions.SampleWithFallback{...}
    err := action.Execute(context.Background(), ctx)

    // Assert results
    actions.AssertContextString(t, ctx, "result_key", "expected_value")
    actions.AssertContextBool(t, ctx, "valid_key", true)
}
```

### Test Cases

Use `RunActionTestCases` for comprehensive testing:

```go
actions.RunActionTestCases(t, []actions.ActionTestCase{
    {
        Name:   "valid input",
        Action: myAction,
        InitialContext: map[string]any{
            "input": "valid",
        },
        ExpectError:  false,
        ExpectedKeys: []string{"result", "valid"},
        Validate: func(t *testing.T, ctx *statemachine.Context) {
            // Custom validation
        },
    },
})
```

## Debugging

Use debugging helpers in `debug.go`:

```go
// Dump context state
fmt.Println(actions.DumpContext(smCtx))

// Trace action execution
tracer := actions.NewActionTracer()
err := tracer.TraceAction("MyAction", smCtx, func() error {
    return myAction.Execute(ctx, smCtx)
})
tracer.PrintTraces()

// Compare contexts
fmt.Println(actions.CompareContexts(beforeCtx, afterCtx))

// Add breakpoint for debugging
breakpoint := actions.BreakpointAction("checkpoint-1")
breakpoint.Execute(ctx, smCtx)
```

## Best Practices

### When to Use Which Action

- **Use SampleWithFallback** when you want AI-generated content with a static fallback
- **Use ElicitForm** when you need structured user input
- **Use ValidateInput** for data validation with custom logic
- **Use TryWithFallback** for graceful error recovery
- **Use ValidatedSequence** for multi-step workflows with validation
- **Use ProgressiveDisclosure** for conditional multi-step forms

### Naming Conventions

- **Action names**: Use descriptive, snake_case names (`oauth_credentials`, `validate_config`)
- **Context keys**: Always prefix with action name (`oauth_credentials_result`)
- **Validation functions**: Name clearly (`validateOAuthFormat`, `checkRequiredFields`)

### Error Handling

- Always check if data exists in context before using it
- Use proper error wrapping with `fmt.Errorf("%w", err)`
- Provide helpful error messages for users
- Use graceful degradation (fallbacks) when capabilities are unavailable

### Performance

- Use `Async: true` in ValidateInput for parallel validation
- Use ParallelWithMerge for independent operations
- Avoid deep nesting - compose actions instead
- Cache expensive operations in context

## Migration Guide

### Converting Nested Conditionals to Actions

**Before:**
```go
if sampling.CanSample(ctx) {
    resp, err := sampling.Sample(ctx, req)
    if err != nil {
        if fallbackFunc != nil {
            result, err = fallbackFunc(ctx, smCtx)
        } else {
            result = staticFallback
        }
    } else {
        result = resp.Content
    }
} else {
    result = staticFallback
}
smCtx.Set("result", result)
```

**After:**
```go
action := &actions.SampleWithFallback{
    Name:         "generate_content",
    Prompt:       "...",
    Fallback:     staticFallback,
    FallbackFunc: fallbackFunc,
}
action.Execute(ctx, smCtx)
result, _ := smCtx.GetString("generate_content_result")
```

### Common Pitfalls

1. **Don't mix action and manual context manipulation** - Use actions consistently
2. **Don't forget to check error returns** - Always handle action errors
3. **Don't assume capabilities** - Actions check automatically, but plan for fallbacks
4. **Don't use generic names** - Use descriptive action names for clarity

## Examples

Complete examples are available in `examples/`:

- `oauth_setup_example.go` - Full OAuth setup flow
- `validation_flow_example.go` - Multi-step validation with AI feedback
- `error_recovery_example.go` - Graceful error handling
- `progressive_disclosure_example.go` - Conditional multi-step forms

## API Reference

See inline documentation in each action file:

- `sampling_actions.go` - Sampling action details
- `elicitation_actions.go` - Elicitation action details
- `validation_actions.go` - Validation action details
- `composite_actions.go` - Composite action details

## Contributing

When adding new actions:

1. Follow the existing patterns and naming conventions
2. Include comprehensive inline documentation with examples
3. Add tests in corresponding `*_test.go` file
4. Update this README with the new action
5. Add example usage if introducing a new pattern

## License

Part of the Ampersand builder-mcp project.
