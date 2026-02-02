# Package: http/printable

Convert HTTP request/response bodies into human-readable or loggable formats.

## Usage

```go
import "github.com/amp-labs/amp-common/http/printable"

// Convert request body
payload, err := printable.Request(req, bodyBytes)

// Check if JSON
isJSON, _ := payload.IsJSON()

// Truncate large bodies
truncated, _ := payload.Truncate(1024)

// Use with slog (implements slog.LogValuer)
logger.Info("request", "body", payload)
```

## Common Patterns

- `Request()` / `Response()` - Convert HTTP bodies to Payload
- Automatic MIME type detection
- Character encoding detection (UTF-8 conversion)
- Base64 encoding for binary content
- Printability heuristics (95% printable threshold)
- `Payload` type integrates with slog

## Gotchas

- Binary content auto-detected and base64-encoded
- Only first 1024 bytes checked for printability
- Non-UTF-8 content converted using chardet

## Related

- `http/httplogger` - Uses printable for logging
- `http/redact` - Redaction after formatting
