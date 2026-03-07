# Package: should

Utilities for cleanup operations that log failures instead of returning errors.

## Usage

```go
// Close resources in defer
defer should.Close(file)
defer should.Close(conn, "failed to close connection")
defer should.Close(file, "failed to close %s", filename)

// Remove temporary files
defer should.Remove("/tmp/file", "failed to remove temp file")
```

## Common Patterns

- Use in defer statements for cleanup
- Logs errors but doesn't return them
- Simplifies cleanup code that shouldn't fail but might

## Gotchas

- Errors are logged, not returned - use regular error handling if you need errors
- Accepts optional message formatting args
