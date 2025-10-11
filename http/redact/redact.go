// Package redact provides utilities for redacting sensitive information from HTTP headers
// and URL query parameters. This is useful for logging HTTP requests/responses without
// exposing secrets, tokens, or other sensitive data.
package redact

import (
	"net/http"
	"net/url"
	"strings"
)

// PartiallyRedactString shows the first visibleRunes characters and replaces the rest
// with asterisks. If the string is shorter than or equal to visibleRunes,
// it is returned unchanged.
//
// Example:
//
//	PartiallyRedactString("sk_live_abc123def456", 8) // Returns "sk_live_************"
//	PartiallyRedactString("short", 10)                // Returns "short"
func PartiallyRedactString(value string, visibleRunes int) string {
	if len(value) <= visibleRunes {
		return value
	}

	show := value[:visibleRunes]
	hide := strings.Map(func(r rune) rune {
		return '*'
	}, value[visibleRunes:])

	return show + hide
}

// Action represents how a header or query parameter value should be handled during redaction.
type Action int

const (
	// ActionKeep indicates that the header value should be kept as-is.
	ActionKeep Action = iota
	// ActionRedact indicates that the header value should be fully redacted (replaced with "<redacted>").
	ActionRedact Action = iota
	// ActionPartial indicates that the header value should be partially redacted
	// (show first N characters, replace rest with asterisks).
	ActionPartial
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
//	func redactSecrets(key, value string) (Action, int) {
//	    if strings.Contains(strings.ToLower(key), "authorization") {
//	        return ActionPartial, 6  // Show "Bearer" prefix
//	    }
//	    if strings.Contains(strings.ToLower(key), "password") {
//	        return ActionRedact, 0   // Fully redact
//	    }
//	    return ActionKeep, 0         // Keep everything else
//	}
type Func func(key, value string) (action Action, partialLength int)

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
func Headers(headers http.Header, redact Func) http.Header {
	if headers == nil {
		return nil
	}

	if redact == nil {
		return headers.Clone()
	}

	redacted := make(http.Header, len(headers))

	for key, hdrs := range headers {
		for _, val := range hdrs {
			action, partialLen := redact(key, val)

			switch action {
			case ActionKeep:
				redacted.Add(key, val)
			case ActionRedact:
				redacted.Add(key, "<redacted>")
			case ActionPartial:
				redacted.Add(key, PartiallyRedactString(val, partialLen))
			case ActionDelete:
				// Do not add this header
			default:
				redacted.Add(key, val) // Default to keeping the header
			}
		}
	}

	return redacted
}

// UrlValues creates a redacted copy of URL query parameters based on the provided redaction function.
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
//	redactedParams := UrlValues(req.URL.Query(), redactFunc)
func UrlValues(values url.Values, redact Func) url.Values {
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
			action, partialLen := redact(key, val)

			switch action {
			case ActionKeep:
				redacted.Add(key, val)
			case ActionRedact:
				redacted.Add(key, "<redacted>")
			case ActionPartial:
				redacted.Add(key, PartiallyRedactString(val, partialLen))
			case ActionDelete:
				// Do not add this value
			default:
				redacted.Add(key, val) // Default to keeping the value
			}
		}
	}

	return redacted
}
