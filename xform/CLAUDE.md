# Package: xform

Type-safe transformation and validation functions for data conversion.

## Usage

```go
// Composable transformers
val := envutil.String("PORT",
    envutil.Transform(xform.Int64),
    envutil.Transform(xform.Positive[int64]),
).Value()

// Validators
xform.Positive[int](42)        // Returns 42, nil
xform.OneOf("dev", "dev", "prod")  // Returns "dev", nil
xform.Port(8080)               // Validates port range
```

## Common Patterns

- Designed for use with `envutil` Transform()
- Validators: Positive, NonZero, Port, FileExists, DirExists
- Converters: CastNumeric, HostAndPort, ExpandPath
- Choices: OneOf, OneOfCaseInsensitive

## Gotchas

- All transformers return (T, error) for composability
- Use with envutil for fluent environment variable parsing

## Related

- `envutil` - Uses xform transformers
