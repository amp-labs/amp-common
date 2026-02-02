# Package: http/redact

Redact sensitive information from HTTP headers, query parameters, and payloads.

## Usage

```go
import "github.com/amp-labs/amp-common/http/redact"

// Partial redaction of strings
redacted := redact.PartiallyRedactString("sk_live_abc123", 8, false)
// Returns: "sk_live_******"

// Redact payloads
redacted, err := redact.PartiallyRedactPayload(payload, 10, false)

// Redaction actions
type RedactFunc func(key, value string) (Action, int)
// Action: ActionKeep, ActionRedact, ActionPartial
```

## Common Patterns

- `PartiallyRedactString()` - Show prefix, hide rest
- `PartiallyRedactBytes()` - Byte-level redaction
- `PartiallyRedactPayload()` - Redact printable.Payload
- Use with httplogger for automatic sensitive data protection
- Actions: Keep, Redact fully, or Partial (show N chars)

## Gotchas

- Partial redaction shows first N chars, replaces rest with '*'
- Handles base64-encoded payloads correctly
- Truncate option uses "[redacted]" instead of asterisks

## Related

- `http/httplogger` - Uses redact for logging
- `http/printable` - Payload formatting
