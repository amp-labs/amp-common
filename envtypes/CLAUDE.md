# Package: envtypes

Common types for parsing environment variables (HostPort, Path).

## Usage

```go
// HostPort for network addresses
hostPort := envtypes.HostPort{Host: "localhost", Port: 8080}
addr := hostPort.String()  // "localhost:8080"

// Convert to/from tuples
tuple := hostPort.AsTuple()
hostPort = envtypes.TupleToHostPort(tuple)

// Path type (see path.go)
path := envtypes.Path{Value: "/etc/config"}
```

## Common Patterns

- Used with `envutil` for parsing environment variables
- `HostPort` provides String() for network addresses
- `Path` type for file system paths

## Gotchas

- Simple data types, no validation logic
- Use with `xform` package for validation

## Related

- `envutil` - Environment variable parsing
- `xform` - Transformers for validation
