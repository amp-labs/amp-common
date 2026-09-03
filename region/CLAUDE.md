# Package: region

Deployment region detection (us, eu).

## Usage

```go
// Detect the current region
current := region.Current(ctx)

// Fall back to a default when no region is configured
current := region.Current(ctx).OrElse(region.Us)

// Branch on region
if region.Current(ctx) == region.Eu {
    // EU-specific behaviour
}
```

## Common Patterns

- Determined by the `REGION` environment variable (`us` or `eu`)
- Use `WithRegion(ctx, region.Eu)` to override in tests or for request-scoped
  work that must act on behalf of a specific region
- `SetRegion(region, setter)` is the callback-based counterpart of `WithRegion`
  for context builders and lazy override mechanisms (same convention as
  `stage.SetStage` and `envutil.SetEnvOverrides`)
- `Valid()` reports whether a region is `us` or `eu`; `OrElse(dfl)` substitutes
  a default otherwise
- Value is cached after first detection
- Mirrors the `stage` package; read that first if the structure is unfamiliar

## Gotchas

- Region is determined once and cached; later changes to `REGION` are not seen
- Unset or unrecognized `REGION` yields `Unknown`. An invalid value logs a
  warning (twice, once from parsing and once from the fallback)
- There is no test-mode fallback like `stage` has: tests that do not set
  `REGION` or use `WithRegion` see `Unknown`
- `REGION=unknown` is treated as a misconfiguration, not an explicit choice
- A context override is returned as-is by `Current`, even if it is not `Valid()`
- Supports: Unknown, Us, Eu
