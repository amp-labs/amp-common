# Package: stage

Environment detection utilities (local, test, dev, staging, prod).

## Usage

```go
// Detect current environment
stage := stage.Current(ctx)

// Check specific stages
if stage.IsLocal(ctx) {
    // Enable debug features
}
if stage.IsProd(ctx) {
    // Enable production monitoring
}
```

## Common Patterns

- Determined by RUNNING_ENV environment variable
- Auto-detects test environment from test flags
- Use `WithStage(ctx, stage.Test)` to override in tests
- Value is cached after first detection

## Gotchas

- Stage is determined once and cached
- Use context override for unit tests
- Supports: Unknown, Local, Test, Dev, Staging, Prod
