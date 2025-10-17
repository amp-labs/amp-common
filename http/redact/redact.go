// Package redact provides utilities for redacting sensitive information from HTTP headers
// and URL query parameters. This is useful for logging HTTP requests/responses without
// exposing secrets, tokens, or other sensitive data.
package redact

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"

	"github.com/amp-labs/amp-common/http/printable"
)

// PartiallyRedactString shows the first visibleRunes characters and replaces the rest
// with asterisks. If the string is shorter than or equal to visibleRunes,
// it is returned unchanged.
//
// Example:
//
//	PartiallyRedactString("sk_live_abc123def456", 8, false) // Returns "sk_live_************"
//	PartiallyRedactString("sk_live_abc123def456", 8, true)  // Returns "sk_live_[redacted]"
//	PartiallyRedactString("short", 10, false)               // Returns "short"
func PartiallyRedactString(value string, visibleRunes int, truncate bool) string {
	if len(value) <= visibleRunes {
		return value
	}

	show := value[:visibleRunes]

	if truncate {
		return show + "[redacted]"
	}

	hide := strings.Map(func(r rune) rune {
		return '*'
	}, value[visibleRunes:])

	return show + hide
}

// PartiallyRedactBytes shows the first visibleBytes and replaces the rest
// with asterisks. If the byte slice is shorter than or equal to visibleBytes,
// it is returned unchanged.
//
// Example:
//
//	PartiallyRedactBytes([]byte("sk_live_abc123def456"), 8, false) // Returns []byte("sk_live_************")
//	PartiallyRedactBytes([]byte("sk_live_abc123def456"), 8, true)  // Returns []byte("sk_live_[redacted]")
//	PartiallyRedactBytes([]byte("short"), 10, false)               // Returns []byte("short")
func PartiallyRedactBytes(value []byte, visibleBytes int, truncate bool) []byte {
	if len(value) <= visibleBytes {
		return value
	}

	show := value[:visibleBytes]

	if truncate {
		result := make([]byte, visibleBytes+len("[redacted]"))
		copy(result, show)
		copy(result[visibleBytes:], "[redacted]")

		return result
	}

	// Replace each byte in the hidden part with '*'
	result := make([]byte, len(value))
	copy(result, show)

	for i := visibleBytes; i < len(value); i++ {
		result[i] = '*'
	}

	return result
}

// PartiallyRedactPayload redacts a printable.Payload by showing the first visibleBytes
// and replacing the rest with asterisks (or truncating with "[redacted]" marker).
// It handles both base64-encoded binary data and regular text content appropriately.
//
// For base64-encoded payloads (Base64=true), the function:
//   - Decodes the base64 content to raw bytes
//   - Applies byte-level redaction using PartiallyRedactBytes
//   - Re-encodes the redacted bytes to base64
//   - Sets TruncatedLength to the length of the redacted decoded data
//
// For regular text payloads (Base64=false), the function:
//   - Applies string-level redaction using PartiallyRedactString
//   - Sets TruncatedLength to the length of the redacted string
//
// The original payload.Length is always preserved to indicate the original size.
// Returns nil if the input payload is nil.
// Returns an error if base64 decoding fails.
//
// Example (text payload):
//
//	payload := &printable.Payload{Content: "secret_api_key_12345", Length: 20}
//	redacted, _ := PartiallyRedactPayload(payload, 10, false)
//	// redacted.Content = "secret_api**********"
//	// redacted.Length = 20 (original)
//	// redacted.TruncatedLength = 20 (redacted string length)
//
// Example (base64 payload):
//
//	payload := &printable.Payload{
//	    Content: "c2VjcmV0X2FwaV9rZXlfMTIzNDU=", // base64("secret_api_key_12345")
//	    Length:  20,
//	    Base64:  true,
//	}
//	redacted, _ := PartiallyRedactPayload(payload, 10, false)
//	// redacted.Content = base64("secret_api**********")
//	// redacted.Length = 20 (original)
//	// redacted.TruncatedLength = 20 (decoded redacted bytes length)
//	// redacted.Base64 = true
func PartiallyRedactPayload(payload *printable.Payload, visibleBytes int, truncate bool) (*printable.Payload, error) {
	if payload == nil {
		return nil, nil
	}

	if payload.Base64 {
		data, err := base64.StdEncoding.DecodeString(payload.Content)
		if err != nil {
			return nil, err
		}

		redactedData := PartiallyRedactBytes(data, visibleBytes, truncate)
		redactedContent := base64.StdEncoding.EncodeToString(redactedData)

		return &printable.Payload{
			Content:         redactedContent,
			Length:          payload.Length,
			TruncatedLength: int64(len(redactedData)),
			Base64:          true,
		}, nil
	} else {
		redactedContent := PartiallyRedactString(payload.Content, visibleBytes, truncate)

		return &printable.Payload{
			Content:         redactedContent,
			Length:          payload.Length,
			TruncatedLength: int64(len(redactedContent)),
		}, nil
	}
}

// Action represents how a header or query parameter value should be handled during redaction.
type Action int

const (
	// ActionKeep indicates that the header value should be kept as-is.
	ActionKeep Action = iota
	// ActionRedactFully indicates that the header value should be fully redacted (replaced with "[redacted]").
	ActionRedactFully
	// ActionRedactPartialWithMask indicates that the header value should be partially redacted
	// (show first N characters, replace rest with asterisks).
	ActionRedactPartialWithMask
	// ActionRedactPartialTruncate indicates that the header value should be partially redacted
	// (show first N characters, truncate the rest and add "[redacted]" to the end).
	ActionRedactPartialTruncate
	// ActionDelete indicates that the header should be removed entirely from the output.
	ActionDelete
)

// Func is a callback function that determines how to redact a given key-value pair.
// It receives the key and value, and returns:
//   - action: what to do with this value (keep, redact, partial redact, or delete)
//   - partialLength: if action is ActionPartial, how many characters to show before redacting
//
// Example:
//
//	func redactSecrets(ctx context.Context, key, value string) (Action, int) {
//	    if strings.Contains(strings.ToLower(key), "authorization") {
//	        return ActionPartial, 6  // Show "Bearer" prefix
//	    }
//	    if strings.Contains(strings.ToLower(key), "password") {
//	        return ActionRedact, 0   // Fully redact
//	    }
//	    return ActionKeep, 0         // Keep everything else
//	}
type Func func(ctx context.Context, key, value string) (action Action, partialLength int)

// BodyFunc is a callback function that determines how to redact an HTTP request/response body.
// It receives the body payload, and returns:
//   - action: what to do with this body (keep, redact, partial redact, or delete)
//   - partialLength: if action is ActionPartial, how many characters to show before redacting
//
// Example:
//
//	func redactBody(ctx context.Context, body *printable.Payload) (Action, int) {
//	    if strings.Contains(body.Content, "password") {
//	        return ActionRedactFully, 0  // Fully redact bodies containing passwords
//	    }
//	    if len(body.Content) > 1000 {
//	        return ActionRedactPartialTruncate, 100  // Show first 100 chars of large bodies
//	    }
//	    return ActionKeep, 0  // Keep everything else
//	}
type BodyFunc func(ctx context.Context, body *printable.Payload) (action Action, partialLength int)

// Headers creates a redacted copy of HTTP headers based on the provided redaction function.
// It processes each header key-value pair through the redact callback to determine how to
// handle sensitive data.
//
// Parameters:
//   - headers: the original HTTP headers to redact
//   - redact: callback function that determines how to redact each header (nil means clone without redaction)
//
// Returns a new http.Header with redacted values. The original headers are not modified.
//
// Example:
//
//	redactFunc := func(key, value string) (Action, int) {
//	    if strings.EqualFold(key, "Authorization") {
//	        return ActionPartial, 7  // Show "Bearer " prefix
//	    }
//	    return ActionKeep, 0
//	}
//	redactedHeaders := Headers(req.Header, redactFunc)
func Headers(ctx context.Context, headers http.Header, redact Func) http.Header {
	if headers == nil {
		return nil
	}

	if redact == nil {
		return headers.Clone()
	}

	redacted := make(http.Header, len(headers))

	for key, hdrs := range headers {
		for _, val := range hdrs {
			action, partialLen := redact(ctx, key, val)

			switch action {
			case ActionKeep:
				redacted.Add(key, val)
			case ActionRedactFully:
				redacted.Add(key, "[redacted]")
			case ActionRedactPartialWithMask:
				redacted.Add(key, PartiallyRedactString(val, partialLen, false))
			case ActionRedactPartialTruncate:
				redacted.Add(key, PartiallyRedactString(val, partialLen, true))
			case ActionDelete:
				// Do not add this header
			default:
				redacted.Add(key, val) // Default to keeping the header
			}
		}
	}

	return redacted
}

// URLValues creates a redacted copy of URL query parameters based on the provided redaction function.
// It processes each query parameter key-value pair through the redact callback to determine how to
// handle sensitive data.
//
// Parameters:
//   - values: the original URL query parameters to redact
//   - redact: callback function that determines how to redact each parameter (nil means clone without redaction)
//
// Returns a new url.Values with redacted values. The original values are not modified.
//
// Example:
//
//	redactFunc := func(key, value string) (Action, int) {
//	    if strings.EqualFold(key, "api_key") {
//	        return ActionPartial, 4  // Show first 4 characters
//	    }
//	    if strings.EqualFold(key, "secret") {
//	        return ActionDelete, 0   // Remove entirely from logs
//	    }
//	    return ActionKeep, 0
//	}
//	redactedParams := URLValues(req.URL.Query(), redactFunc)
func URLValues(ctx context.Context, values url.Values, redact Func) url.Values {
	if values == nil {
		return nil
	}

	if redact == nil {
		cloned := make(url.Values, len(values))

		for key, vals := range values {
			cloned[key] = append([]string(nil), vals...)
		}

		return cloned
	}

	redacted := make(url.Values, len(values))

	for key, vals := range values {
		for _, val := range vals {
			action, partialLen := redact(ctx, key, val)

			switch action {
			case ActionKeep:
				redacted.Add(key, val)
			case ActionRedactFully:
				redacted.Add(key, "[redacted]")
			case ActionRedactPartialWithMask:
				redacted.Add(key, PartiallyRedactString(val, partialLen, false))
			case ActionRedactPartialTruncate:
				redacted.Add(key, PartiallyRedactString(val, partialLen, true))
			case ActionDelete:
				// Do not add this value
			default:
				redacted.Add(key, val) // Default to keeping the value
			}
		}
	}

	return redacted
}

// Body creates a redacted copy of an HTTP body based on the provided redaction function.
// It processes the body payload through the redact callback to determine how to handle
// sensitive data.
//
// Parameters:
//   - body: the original HTTP body payload to redact
//   - redact: callback function that determines how to redact the body (nil means clone without redaction)
//
// Returns a new *printable.Payload with redacted content. The original body is not modified.
// If the action is ActionDelete, returns nil.
//
// Example:
//
//	redactFunc := func(ctx context.Context, body *printable.Payload) (Action, int) {
//	    if strings.Contains(body.Content, `"password"`) {
//	        return ActionRedactFully, 0  // Fully redact bodies with passwords
//	    }
//	    return ActionKeep, 0
//	}
//	redactedBody := Body(ctx, requestBody, redactFunc)
func Body(ctx context.Context, body *printable.Payload, redact BodyFunc) (*printable.Payload, error) {
	if body == nil {
		return nil, nil
	}

	if redact == nil {
		return body.Clone(), nil
	}

	action, partialLen := redact(ctx, body)

	switch action {
	case ActionKeep:
		return body.Clone(), nil
	case ActionRedactFully:
		// Replace entire content with redaction marker
		// Preserve original Length but update TruncatedLength to reflect new content size
		redactedText := "[redacted]"

		return &printable.Payload{
			Content:         redactedText,
			Length:          body.Length,
			TruncatedLength: int64(len(redactedText)),
		}, nil
	case ActionRedactPartialWithMask:
		// Show first N characters, replace rest with asterisks
		return PartiallyRedactPayload(body, partialLen, false)
	case ActionRedactPartialTruncate:
		// Show first N characters, truncate rest with "[redacted]" marker
		return PartiallyRedactPayload(body, partialLen, true)
	case ActionDelete:
		// Remove body entirely from output (returns nil)
		return nil, nil
	default:
		// Default to keeping the body if action is unknown
		return body.Clone(), nil
	}
}
