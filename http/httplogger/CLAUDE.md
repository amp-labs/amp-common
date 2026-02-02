# Package: http/httplogger

Structured logging for HTTP requests and responses with automatic redaction and body handling.

## Usage

```go
import "github.com/amp-labs/amp-common/http/httplogger"

// Define redaction for sensitive headers
redactFunc := func(key, value string) (redact.Action, int) {
    if strings.Contains(strings.ToLower(key), "authorization") {
        return redact.ActionPartial, 7  // Show "Bearer " prefix
    }
    return redact.ActionKeep, 0
}

// Log request
params := &httplogger.LogRequestParams{
    Logger:        slog.Default(),
    RedactHeaders: redactFunc,
    IncludeBody:   true,
}
httplogger.LogRequest(req, nil, correlationID, params)
```

## Common Patterns

- `LogRequest()` - Log HTTP requests with redaction
- `LogResponse()` - Log HTTP responses
- `LogError()` - Log HTTP errors
- Auto-redacts sensitive headers and query params
- Body truncation (default 256 KiB)
- `WithArchive()` - Mark logs for long-term storage

## Gotchas

- Bodies truncated at 256 KiB by default
- Uses `printable` package for body formatting
- Integrates with slog for structured logging

## Related

- `http/printable` - Body formatting
- `http/redact` - Redaction utilities
