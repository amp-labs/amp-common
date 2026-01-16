# State Machine Validator

The validator package provides comprehensive validation for state machine configurations with detailed error messages, warnings, and automatic fix suggestions.

## Features

- **Comprehensive validation** - Multiple rule types covering structural, semantic, and best practice issues
- **Detailed error messages** - Clear descriptions of what's wrong and where
- **Auto-fix suggestions** - Automatic fixes for common issues
- **Custom rules** - Extensible rule system for domain-specific validation
- **Build-time checking** - Catch errors before runtime
- **CI/CD integration** - Easy to integrate into automated workflows

## Installation

```go
import "github.com/amp-labs/server/builder-mcp/statemachine/validator"
```

## Quick Start

### Basic Validation

```go
// Validate a config object
result := validator.Validate(config)

if !result.Valid {
    fmt.Println(result.String())
    // Prints formatted errors and suggestions
}
```

### Validate from File

```go
result, err := validator.ValidateFile("workflow.yaml")
if err != nil {
    log.Fatal("Failed to load config:", err)
}

if result.HasErrors() {
    fmt.Println("Validation failed:")
    for _, err := range result.Errors {
        fmt.Printf("  [%s] %s\n", err.Code, err.Message)
        if err.Fix != nil {
            fmt.Printf("    Fix: %s\n", err.Fix.Description)
        }
    }
}
```

## Validation Rules

### Built-in Rules

The validator includes several built-in rules:

#### 1. Unreachable State Rule

Detects states that cannot be reached from the initial state.

```yaml
# ❌ Invalid - "orphan" state is unreachable
initialState: start
states:
  - name: start
    type: action
  - name: orphan    # Cannot be reached!
    type: action
  - name: end
    type: final
transitions:
  - from: start
    to: end

# ✓ Fixed
transitions:
  - from: start
    to: orphan
  - from: orphan
    to: end
```

**Error Code**: `UNREACHABLE_STATE`

**Fix**: Add a transition to the state or remove it

#### 2. Missing Transition Rule

Detects non-final states without outgoing transitions.

```yaml
# ❌ Invalid - "dead_end" has no outgoing transition
states:
  - name: start
    type: action
  - name: dead_end
    type: action    # Not final but no way out!
  - name: end
    type: final
transitions:
  - from: start
    to: dead_end
  # Missing transition from dead_end

# ✓ Fixed
transitions:
  - from: start
    to: dead_end
  - from: dead_end
    to: end
```

**Error Code**: `MISSING_TRANSITION`

**Fix**: Add a transition or mark as final state

#### 3. Duplicate Transition Rule

Detects duplicate transitions with the same from/to/condition.

```yaml
# ❌ Invalid - duplicate transitions
transitions:
  - from: start
    to: end
    condition: always
  - from: start
    to: end
    condition: always  # Duplicate!

# ✓ Fixed
transitions:
  - from: start
    to: end
    condition: always
```

**Error Code**: `DUPLICATE_TRANSITION`

**Fix**: Remove duplicate transition

#### 4. Naming Convention Rule

Warns about non-snake_case state names.

```yaml
# ⚠ Warning - non-standard naming
states:
  - name: StartState    # Should be start_state
    type: action
  - name: ProcessData   # Should be process_data
    type: action

# ✓ Fixed
states:
  - name: start_state
    type: action
  - name: process_data
    type: action
```

**Error Code**: `NAMING_CONVENTION`

**Fix**: Rename to snake_case

#### 5. Cyclic Transition Rule

Detects potential infinite loops without exit conditions.

```yaml
# ⚠ Warning - potential infinite loop
states:
  - name: retry
    type: action
transitions:
  - from: retry
    to: retry
    condition: always  # Loops forever!

# ✓ Fixed - loop has exit condition
transitions:
  - from: retry
    to: retry
    condition: attempts < 3
  - from: retry
    to: complete
    condition: attempts >= 3
```

**Error Code**: `POTENTIAL_INFINITE_LOOP`

**Fix**: Ensure cycle has path to final state

## Auto-Fixes

The validator can automatically fix many common issues:

### Apply Fixes Manually

```go
result := validator.Validate(config)

// Collect all auto-fixes
var fixes []*validator.Fix
for _, err := range result.Errors {
    if err.Fix != nil {
        fixes = append(fixes, err.Fix)
    }
}

// Apply fixes
if err := validator.ApplyFixes(config, fixes); err != nil {
    log.Fatal("Failed to apply fixes:", err)
}

// Re-validate
result = validator.Validate(config)
```

### Available Fix Functions

```go
// Add missing transition
fix := validator.AddMissingTransition("state_a", "state_b")

// Remove unreachable state
fix := validator.RemoveUnreachableState("orphan_state")

// Rename state to follow conventions
fix := validator.RenameState("OldName", "new_name")

// Mark state as final
fix := validator.MarkAsFinalState("end_state")

// Remove duplicate transition
fix := validator.RemoveDuplicateTransition("from", "to", "condition")
```

## Custom Validation Rules

Create custom rules for domain-specific validation:

### Define a Custom Rule

```go
type myCustomRule struct{}

func (r *myCustomRule) Name() string {
    return "MyCustomRule"
}

func (r *myCustomRule) Check(config *statemachine.Config) []validator.ValidationError {
    var errors []validator.ValidationError

    // Your validation logic
    for _, state := range config.States {
        if !isValid(state) {
            errors = append(errors, validator.ValidationError{
                Code:    "CUSTOM_ERROR",
                Message: fmt.Sprintf("State %s violates custom rule", state.Name),
                Location: validator.Location{State: state.Name},
            })
        }
    }

    return errors
}
```

### Register and Use Custom Rule

```go
// Register custom rule
validator.RegisterRule(&myCustomRule{})

// Validate with all rules (including custom)
rules := append(validator.DefaultRules(), validator.RegisteredRules...)
result := validator.ValidateWithRules(config, rules)
```

## Validation Result

The `ValidationResult` struct provides comprehensive information:

```go
type ValidationResult struct {
    Valid       bool                    // Overall validation status
    Errors      []ValidationError       // Critical errors
    Warnings    []ValidationWarning     // Non-critical warnings
    Suggestions []Suggestion            // Improvement suggestions
}
```

### Check Results

```go
result := validator.Validate(config)

// Quick checks
if result.HasErrors() {
    // Handle errors
}

if result.HasWarnings() {
    // Handle warnings
}

// Detailed inspection
for _, err := range result.Errors {
    fmt.Printf("Error [%s]: %s\n", err.Code, err.Message)
    fmt.Printf("  Location: %s\n", err.Location.State)
    if err.Fix != nil {
        fmt.Printf("  Fix: %s\n", err.Fix.Description)
    }
}

// Suggestions
for _, sug := range result.Suggestions {
    fmt.Printf("💡 %s\n", sug.Message)
    if sug.Example != "" {
        fmt.Printf("Example:\n%s\n", sug.Example)
    }
}
```

### Formatted Output

```go
// Print human-readable summary
fmt.Println(result.String())

// Output:
// ✗ Configuration has 2 error(s)
//   [UNREACHABLE_STATE] State 'orphan' cannot be reached from initial state 'start'
//     Fix: Add a transition to 'orphan' or remove the state
//   [MISSING_TRANSITION] Non-final state 'dead_end' has no outgoing transitions
//     Fix: Add a transition or mark as final state
//
// ⚠ 1 warning(s):
//   [NAMING_CONVENTION] State 'ProcessData' should use snake_case naming
//
// 💡 2 suggestion(s) for improvement
```

## Integration Examples

### In Tests

```go
func TestWorkflowConfig(t *testing.T) {
    config := loadTestConfig()

    result := validator.Validate(config)
    require.True(t, result.Valid, "Config should be valid:\n%s", result.String())
}
```

### In CI/CD

```go
func main() {
    result, err := validator.ValidateFile(os.Args[1])
    if err != nil {
        log.Fatal(err)
    }

    if !result.Valid {
        fmt.Println(result.String())
        os.Exit(1)
    }

    fmt.Println("✓ Configuration is valid")
}
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit

for config in statemachine/configs/*.yaml; do
    if ! statemachine-cli validate "$config"; then
        echo "Validation failed for $config"
        exit 1
    fi
done
```

### GitHub Actions

```yaml
name: Validate Configs

on: [push, pull_request]

jobs:
  validate:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Validate State Machine Configs
        run: |
          go run ./scripts/statemachine-cli validate configs/*.yaml
```

## Advanced Usage

### Strict Mode

Treat warnings as errors:

```go
result := validator.Validate(config)

if result.HasWarnings() {
    // In strict mode, warnings fail the build
    log.Fatal("Strict mode: warnings present")
}
```

### Incremental Validation

Validate specific aspects:

```go
// Only check reachability
rule := &validator.unreachableStateRule{}
errors := rule.Check(config)

// Only check naming
rule := &validator.namingConventionRule{}
errors := rule.Check(config)
```

### Validation Reports

Generate detailed reports:

```go
result := validator.Validate(config)

report := generateReport(result)
saveToFile("validation-report.html", report)
```

## Best Practices

1. **Validate early** - Run validation during development, not just in CI/CD
2. **Fix errors first** - Address critical errors before warnings
3. **Use auto-fixes cautiously** - Review auto-fixes before applying
4. **Add custom rules** - Encode domain knowledge in custom validation rules
5. **Document exceptions** - If you must violate a rule, document why
6. **Version control configs** - Track validation results over time
7. **Integrate with editor** - Use CLI tool for real-time feedback

## Troubleshooting

### False Positives

If a rule reports false positives:

```go
// Create custom rules list without problematic rule
rules := []validator.Rule{
    &validator.unreachableStateRule{},
    &validator.missingTransitionRule{},
    // Exclude: &validator.cyclicTransitionRule{},
}

result := validator.ValidateWithRules(config, rules)
```

### Performance

For large configs:

```go
// Validate only critical rules
rules := []validator.Rule{
    &validator.unreachableStateRule{},
    &validator.missingTransitionRule{},
}

result := validator.ValidateWithRules(config, rules)
```

## Error Codes Reference

| Code | Severity | Description | Auto-Fix |
|------|----------|-------------|----------|
| `UNREACHABLE_STATE` | Error | State cannot be reached from initial state | Yes |
| `MISSING_TRANSITION` | Error | Non-final state has no outgoing transitions | Yes |
| `DUPLICATE_TRANSITION` | Error | Duplicate transition exists | Yes |
| `NAMING_CONVENTION` | Error | State name violates naming convention | Yes |
| `POTENTIAL_INFINITE_LOOP` | Warning | Cycle detected without clear exit | No |
| `CONFIG_LOAD_FAILED` | Error | Failed to load config file | No |

## See Also

- [State Machine README](../README.md)
- [Visualizer Package](../visualizer/README.md)
- [CLI Tool](../../../scripts/statemachine-cli/README.md)
- [Testing Helpers](../testing/README.md)
