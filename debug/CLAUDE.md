# Package: debug

Debugging utilities for local development only (NOT for production use).

## Usage

```go
import "github.com/amp-labs/amp-common/debug"

// Dump context hierarchy as JSON
debug.DumpContext(ctx, os.Stdout)

// Dump any value as formatted JSON
debug.DumpJSON(ctx, myStruct, os.Stdout)
```

## Common Patterns

- Inspect context values and hierarchy
- Pretty-print JSON for debugging
- Never import this package in production code

## Gotchas

- FOR LOCAL DEBUGGING ONLY
- Will call logger.Fatal on JSON marshal errors
- Output goes to provided writer (usually os.Stdout)
