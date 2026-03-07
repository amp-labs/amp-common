# Package: envutil

Type-safe environment variable parsing with fluent API and extensive built-in type support.

## Usage

```go
// Basic usage with defaults
port := envutil.Int(ctx, "PORT", envutil.Default(8080)).Value()

// Chain transformations
port := envutil.String(ctx, "PORT",
    envutil.Transform(xform.Int64),
    envutil.Transform(xform.Positive[int64]),
).Value()

// Custom types
url := envutil.URL(ctx, "API_URL", envutil.Required()).Value()
dur := envutil.Duration(ctx, "TIMEOUT").ValueOrElse(30*time.Second)

// Context overrides (for testing)
ctx = envutil.WithEnvOverride(ctx, "PORT", "9000")
```

## Common Patterns

- Fluent API: `Reader[T]` with options (Default, Required, Validate, Transform)
- Built-in types: String, Int, Bool, Duration, URL, UUID, FilePath, DirPath, HostAndPort
- `ValueOrElse()` - Return value or fallback
- Context overrides for testing/multi-tenancy
- Recording/observation for debugging

## Gotchas

- Options applied in order (Default before Required, etc.)
- Context overrides take precedence over OS environment
- Recording/observers available for auditing env var reads

## Related

- `xform` - Transformers for validation
- `envtypes` - Types like HostPort, Path
