# Helper Library

High-level convenience functions for common patterns with automatic capability checking, fallback handling, and standardized error messages.

## Overview

The helper library wraps the sampling and elicitation APIs with opinionated, production-ready patterns:

- **Automatic capability checking** - Detect if sampling/elicitation is available
- **Graceful fallback** - Use defaults when capabilities unavailable
- **Standardized error handling** - Consistent error messages and types
- **Logging and observability** - Built-in structured logging
- **Best practice patterns** - Proven patterns for common scenarios

## Quick Start

```go
import "github.com/amp-labs/server/builder-mcp/statemachine/helpers"

// Sampling with fallback
explanation, err := helpers.SampleWithFallback(
    ctx,
    samplingClient,
    "Explain why the error occurred: " + errorMsg,
    "The operation failed. Please check the logs for details.",
)

// Elicitation with defaults
answers, err := helpers.ElicitWithDefaults(
    ctx,
    elicitationClient,
    question,
    map[string]string{
        "provider": "salesforce", // Default if user can't answer
    },
)

// Validation
err := helpers.ValidateRequired(input, []string{
    "projectId",
    "integrationId",
})
```

## Sampling Helpers

### Core Functions

- `SampleWithFallback` - Generate content with automatic fallback
- `SampleJSON` - Generate structured JSON with schema validation
- `SampleWithRetry` - Generate content with automatic retries
- `CanSample` - Check if sampling is available

### Usage Patterns

**Basic Sampling:**

```go
content, err := helpers.SampleWithFallback(
    ctx,
    client,
    "Generate recommendations for {{provider}}",
    "Use standard best practices for this provider",
)
```

**JSON Generation:**

```go
type Recommendation struct {
    Action   string   `json:"action"`
    Priority string   `json:"priority"`
    Reasons  []string `json:"reasons"`
}

var rec Recommendation
err := helpers.SampleJSON(
    ctx,
    client,
    "Generate recommendations for improving integration health",
    &rec,
)
```

**With Retries:**

```go
content, err := helpers.SampleWithRetry(
    ctx,
    client,
    "Generate config for " + provider,
    3, // Max retries
)
```

**Conditional Sampling:**

```go
if helpers.CanSample(ctx, client) {
    content, _ := helpers.SampleWithFallback(ctx, client, prompt, fallback)
} else {
    content = fallback
}
```

## Elicitation Helpers

### Core Functions

- `ElicitWithDefaults` - Collect input with automatic defaults
- `ElicitConfirmation` - Ask yes/no question
- `ElicitMultipleChoice` - Collect multiple selections
- `ElicitText` - Collect free-form text with validation
- `CanElicit` - Check if elicitation is available

### Usage Patterns

**Single Choice:**

```go
question := ElicitationQuestion{
    Text: "Which provider?",
    Type: "single_choice",
    Options: []Option{
        {Label: "Salesforce", Value: "salesforce"},
        {Label: "HubSpot", Value: "hubspot"},
    },
}

answers, err := helpers.ElicitWithDefaults(
    ctx,
    client,
    question,
    map[string]string{"selection": "salesforce"},
)

provider := answers["selection"]
```

**Confirmation:**

```go
confirmed, err := helpers.ElicitConfirmation(
    ctx,
    client,
    "Enable webhooks?",
    true, // Default to yes
)
```

**Multiple Choice:**

```go
objects, err := helpers.ElicitMultipleChoice(
    ctx,
    client,
    "Which objects?",
    []string{"Account", "Contact", "Lead", "Opportunity"},
    []string{"Account", "Contact"}, // Defaults
)
```

**Text with Validation:**

```go
projectName, err := helpers.ElicitText(
    ctx,
    client,
    "Enter project name:",
    "my-integration",
    func(s string) error {
        if len(s) < 3 {
            return errors.New("name too short")
        }
        return nil
    },
)
```

## Validation Helpers

### Core Functions

- `ValidateRequired` - Check required fields present
- `ValidateEnum` - Check value in allowed list
- `ValidateRegex` - Check value matches pattern
- `ValidateStruct` - Validate struct with tags

### Usage Patterns

**Required Fields:**

```go
err := helpers.ValidateRequired(input, []string{
    "projectId",
    "integrationId",
    "provider",
})
```

**Enum Validation:**

```go
err := helpers.ValidateEnum(provider, []string{
    "salesforce",
    "hubspot",
    "notion",
})
```

**Regex Validation:**

```go
err := helpers.ValidateRegex(email, `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
```

**Struct Validation:**

```go
type Config struct {
    Provider string   `validate:"required,enum=salesforce|hubspot"`
    Objects  []string `validate:"required,min=1"`
    Email    string   `validate:"email"`
}

err := helpers.ValidateStruct(config)
```

## Error Handling

All helpers follow consistent error handling:

**Capability Unavailable:**

- Returns fallback/default value
- Never returns error for unavailability
- Logs warning for observability

**Validation Failure:**

- Returns descriptive error
- Includes field name and reason
- Aggregates multiple errors

**Critical Failure:**

- Returns error
- Includes context
- Wrapped for tracing

## Best Practices

### Always Provide Fallbacks

```go
// ✓ Good
content, _ := helpers.SampleWithFallback(ctx, client, prompt, "Fallback content")

// ✗ Bad
content, err := client.Sample(ctx, prompt)
if err != nil {
    // Now what?
}
```

### Validate Early

```go
// ✓ Good
if err := helpers.ValidateRequired(input, requiredFields); err != nil {
    return nil, err
}
// Continue processing

// ✗ Bad
processData(input)
if err := helpers.ValidateRequired(input, requiredFields); err != nil {
    // Already processed invalid data!
}
```

### Check Capabilities

```go
// ✓ Good
if helpers.CanElicit(ctx, client) {
    // Use elicitation
} else {
    // Use defaults
}

// ✗ Bad
answers, _ := client.Elicit(ctx, question) // May fail!
```

### Provide Meaningful Defaults

```go
// ✓ Good
defaults := map[string]string{
    "provider": "salesforce", // Most common
    "sync": "true",           // Sensible default
}

// ✗ Bad
defaults := map[string]string{
    "provider": "",  // Empty
    "sync": "null",  // Confusing
}
```

## Integration with State Machines

Helpers work seamlessly with state machine actions:

**In YAML:**

```yaml
- type: sampling
  name: generate_content
  parameters:
    prompt: "Generate content"
    fallback: "Default content"
```

**In Handler:**

```go
func generateContentHandler(ctx context.Context, smCtx *statemachine.Context) error {
    content, err := helpers.SampleWithFallback(
        ctx,
        samplingClient,
        "Generate content for {{provider}}",
        "Default content",
    )
    if err != nil {
        return err
    }
    smCtx.Set("content", content)
    return nil
}
```

## Common Patterns

### Sampling with Elicitation Fallback

```go
var content string

if helpers.CanSample(ctx, samplingClient) {
    content, _ = helpers.SampleWithFallback(ctx, samplingClient, prompt, fallback)
} else if helpers.CanElicit(ctx, elicitClient) {
    answers, _ := helpers.ElicitText(ctx, elicitClient, "Enter content:", fallback, nil)
    content = answers
} else {
    content = fallback
}
```

### Validated Elicitation

```go
provider, err := helpers.ElicitText(
    ctx,
    client,
    "Enter provider:",
    "salesforce",
    func(p string) error {
        return helpers.ValidateEnum(p, []string{"salesforce", "hubspot", "notion"})
    },
)
```

### Conditional Sampling

```go
func smartGenerate(ctx context.Context, enabled bool) (string, error) {
    if !enabled || !helpers.CanSample(ctx, client) {
        return "Template content", nil
    }

    return helpers.SampleWithFallback(
        ctx,
        client,
        "Generate content",
        "Template content",
    )
}
```

## File Organization

```
helpers/
├── README.md              # This file
├── sampling.go            # Sampling helpers
├── sampling_test.go       # Sampling tests
├── elicitation.go         # Elicitation helpers
├── elicitation_test.go    # Elicitation tests
├── validation.go          # Validation helpers
├── errors.go              # Error handling
└── errors_test.go         # Error tests
```

## Testing

Helpers are fully tested with unit tests:

```bash
cd statemachine/helpers
go test -v
```

Run specific tests:

```bash
go test -v -run TestSampleWithFallback
```

With coverage:

```bash
go test -cover
```

## See Also

- [Action Types Reference](../../docs/reference/action-types.md)
- [Helper Functions Reference](../../docs/reference/helper-functions.md)
- [Patterns Guide](../PATTERNS.md)
- [Developer Guide](../DEVELOPER_GUIDE.md)
- [Examples](../examples/)

## Contributing

When adding new helpers:

1. Follow existing patterns
2. Add comprehensive tests
3. Document with examples
4. Update this README
5. Add to reference docs

**Helper Checklist:**

- [ ] Automatic capability checking
- [ ] Graceful fallback handling
- [ ] Clear error messages
- [ ] Comprehensive tests
- [ ] Usage examples
- [ ] Documentation
