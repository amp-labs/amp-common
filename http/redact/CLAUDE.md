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
// Action: ActionKeep, ActionRedact, ActionPartial, ActionFingerprint

// Fingerprint secrets so logs show whether two values were the same.
// The key is a secret shared by everything whose logs are compared.
ctx = redact.WithFingerprintKey(ctx, key)
redact.Fingerprint(ctx, "token") // "[fingerprint 1a2b3c4d5e6f]"

// Redact a JSON object or form body one field at a time
policy := func(ctx context.Context, key, value string) (redact.Action, int) {
    if key == "grant_type" {
        return redact.ActionKeep, 0
    }
    return redact.ActionFingerprint, 0
}
redacted, err := redact.BodyFields(ctx, payload, policy)
// Any other body fails with redact.ErrUnstructuredBody:
// redact the whole body instead.
```

## Common Patterns

- `PartiallyRedactString()` - Show prefix, hide rest
- `PartiallyRedactBytes()` - Byte-level redaction
- `PartiallyRedactPayload()` - Redact printable.Payload
- `Fingerprint()` / `ActionFingerprint` - Keyed HMAC in place of a secret
- `BodyFields()` - Per-field redaction of JSON object and form bodies
- Use with httplogger for automatic sensitive data protection
- Actions: Keep, Redact fully, or Partial (show N chars)

## Gotchas

- Partial redaction shows first N chars, replaces rest with '*'
- Handles base64-encoded payloads correctly
- Truncate option uses "[redacted]" instead of asterisks
- Without a key from `WithFingerprintKey`, `ActionFingerprint` redacts
  fully. An empty value stays empty.
- `BodyFields` re-encodes JSON (fields sorted) and keeps numbers, booleans
  and nulls. It rejects form bodies whose field names are not lower-case
  `[a-z0-9_.\-\[\]]`, so a bare token is never logged as a field name.

## Related

- `http/httplogger` - Uses redact for logging
- `http/printable` - Payload formatting
