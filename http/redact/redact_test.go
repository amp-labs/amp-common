package redact_test

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/amp-labs/amp-common/http/printable"
	"github.com/amp-labs/amp-common/http/redact"
	"github.com/stretchr/testify/assert"
)

func TestPartiallyRedactString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        string
		visibleRunes int
		expected     string
	}{
		{
			name:         "redact API key",
			value:        "sk_live_abc123def456",
			visibleRunes: 8,
			expected:     "sk_live_************",
		},
		{
			name:         "short string unchanged",
			value:        "short",
			visibleRunes: 10,
			expected:     "short",
		},
		{
			name:         "exact length unchanged",
			value:        "exact",
			visibleRunes: 5,
			expected:     "exact",
		},
		{
			name:         "empty string",
			value:        "",
			visibleRunes: 5,
			expected:     "",
		},
		{
			name:         "zero visible runes",
			value:        "secret",
			visibleRunes: 0,
			expected:     "******",
		},
		{
			name:         "show one character",
			value:        "password123",
			visibleRunes: 1,
			expected:     "p**********",
		},
		{
			name:         "unicode characters",
			value:        "helloworld",
			visibleRunes: 5,
			expected:     "hello*****",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := redact.PartiallyRedactString(tt.value, tt.visibleRunes, false)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHeaders_NilHeaders(t *testing.T) {
	t.Parallel()

	result := redact.Headers(t.Context(), nil, nil)
	assert.Nil(t, result)
}

func TestHeaders_NilRedactFunc(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token123"},
	}

	result := redact.Headers(t.Context(), headers, nil)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Equal(t, "Bearer token123", result.Get("Authorization"))

	// Verify it's a clone, not the same instance
	assert.NotSame(t, &headers, &result)
}

func TestHeaders_ActionKeep(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"User-Agent":   []string{"test-client"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Equal(t, "test-client", result.Get("User-Agent"))
}

func TestHeaders_ActionRedact(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": []string{"Bearer secret_token"},
		"X-Api-Key":     []string{"api_key_12345"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "auth") || strings.Contains(strings.ToLower(key), "key") {
			return redact.ActionRedactFully, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	assert.Equal(t, "[redacted]", result.Get("Authorization"))
	assert.Equal(t, "[redacted]", result.Get("X-Api-Key"))
}

func TestHeaders_ActionPartial(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": []string{"Bearer token123456789"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "Authorization") {
			return redact.ActionRedactPartialWithMask, 7 // Show "Bearer "
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	assert.Equal(t, "Bearer **************", result.Get("Authorization"))
}

func TestHeaders_ActionDelete(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Secret":     []string{"should_be_deleted"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "secret") {
			return redact.ActionDelete, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Empty(t, result.Get("X-Secret"))
	assert.NotContains(t, result, "X-Secret")
}

func TestHeaders_MultipleValues(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Set-Cookie": []string{"session=abc123", "tracking=xyz789"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "Set-Cookie") {
			return redact.ActionRedactPartialWithMask, 8
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	cookies := result.Values("Set-Cookie")
	assert.Len(t, cookies, 2)
	assert.Equal(t, "session=******", cookies[0])
	assert.Equal(t, "tracking*******", cookies[1])
}

func TestHeaders_DefaultActionOnUnknown(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type": []string{"application/json"},
	}

	// Return an invalid action value (100)
	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		return redact.Action(100), 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	// Should default to ActionKeep
	assert.Equal(t, "application/json", result.Get("Content-Type"))
}

func TestUrlValues_NilValues(t *testing.T) {
	t.Parallel()

	result := redact.URLValues(t.Context(), nil, nil)
	assert.Nil(t, result)
}

func TestUrlValues_NilRedactFunc(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"api_key": []string{"secret123"},
		"page":    []string{"1"},
	}

	result := redact.URLValues(t.Context(), values, nil)

	assert.Equal(t, "secret123", result.Get("api_key"))
	assert.Equal(t, "1", result.Get("page"))

	// Verify it's a clone, not the same instance
	assert.NotSame(t, &values, &result)
}

func TestUrlValues_ActionKeep(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"page":  []string{"1"},
		"limit": []string{"10"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	assert.Equal(t, "1", result.Get("page"))
	assert.Equal(t, "10", result.Get("limit"))
}

func TestUrlValues_ActionRedact(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"api_key": []string{"secret123"},
		"token":   []string{"bearer_token"},
		"page":    []string{"1"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "token") {
			return redact.ActionRedactFully, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	assert.Equal(t, "[redacted]", result.Get("api_key"))
	assert.Equal(t, "[redacted]", result.Get("token"))
	assert.Equal(t, "1", result.Get("page"))
}

func TestUrlValues_ActionPartial(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"api_key": []string{"sk_live_1234567890"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "api_key") {
			return redact.ActionRedactPartialWithMask, 8 // Show "sk_live_"
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	assert.Equal(t, "sk_live_**********", result.Get("api_key"))
}

func TestUrlValues_ActionDelete(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"page":   []string{"1"},
		"secret": []string{"should_be_deleted"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "secret") {
			return redact.ActionDelete, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	assert.Equal(t, "1", result.Get("page"))
	assert.Empty(t, result.Get("secret"))
	assert.NotContains(t, result, "secret")
}

func TestUrlValues_MultipleValues(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"id": []string{"123", "456", "789"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "id") {
			return redact.ActionRedactPartialWithMask, 1
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	ids := result["id"]
	assert.Len(t, ids, 3)
	assert.Equal(t, "1**", ids[0])
	assert.Equal(t, "4**", ids[1])
	assert.Equal(t, "7**", ids[2])
}

func TestUrlValues_DefaultActionOnUnknown(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"page": []string{"1"},
	}

	// Return an invalid action value (100)
	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		return redact.Action(100), 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	// Should default to ActionKeep
	assert.Equal(t, "1", result.Get("page"))
}

// Test realistic scenarios.
func TestHeaders_RealisticScenario_LoggingSafeHeaders(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type":     []string{"application/json"},
		"User-Agent":       []string{"MyApp/1.0"},
		"Authorization":    []string{"Bearer sk_live_abc123def456ghi789"},
		"X-Api-Key":        []string{"api_key_secret123"},
		"X-Request-Id":     []string{"req-12345"},
		"X-Internal-Token": []string{"internal_secret"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		lowerKey := strings.ToLower(key)

		// Fully redact API keys
		if strings.Contains(lowerKey, "key") {
			return redact.ActionRedactFully, 0
		}

		// Partially redact authorization (show Bearer prefix)
		if strings.Contains(lowerKey, "authorization") {
			return redact.ActionRedactPartialWithMask, 7
		}

		// Delete internal tokens
		if strings.HasPrefix(lowerKey, "x-internal") {
			return redact.ActionDelete, 0
		}

		// Keep everything else
		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Equal(t, "MyApp/1.0", result.Get("User-Agent"))
	assert.Equal(t, "Bearer **************************", result.Get("Authorization"))
	assert.Equal(t, "[redacted]", result.Get("X-Api-Key"))
	assert.Equal(t, "req-12345", result.Get("X-Request-Id"))
	assert.Empty(t, result.Get("X-Internal-Token"))
}

func TestUrlValues_RealisticScenario_LoggingSafeQueryParams(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"page":        []string{"1"},
		"limit":       []string{"10"},
		"api_key":     []string{"sk_live_1234567890abcdef"},
		"access_code": []string{"secret_access_code"},
		"user_id":     []string{"user123"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		lowerKey := strings.ToLower(key)

		// Partially redact API keys (show prefix)
		if strings.Contains(lowerKey, "api_key") {
			return redact.ActionRedactPartialWithMask, 8
		}

		// Fully delete access codes from logs
		if strings.Contains(lowerKey, "access_code") || strings.Contains(lowerKey, "secret") {
			return redact.ActionDelete, 0
		}

		// Keep everything else
		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	assert.Equal(t, "1", result.Get("page"))
	assert.Equal(t, "10", result.Get("limit"))
	assert.Equal(t, "sk_live_****************", result.Get("api_key"))
	assert.Empty(t, result.Get("access_code"))
	assert.Equal(t, "user123", result.Get("user_id"))
}

func TestPartiallyRedactString_WithTruncate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        string
		visibleRunes int
		expected     string
	}{
		{
			name:         "redact API key with truncate",
			value:        "sk_live_abc123def456",
			visibleRunes: 8,
			expected:     "sk_live_[redacted]",
		},
		{
			name:         "short string unchanged",
			value:        "short",
			visibleRunes: 10,
			expected:     "short",
		},
		{
			name:         "exact length unchanged",
			value:        "exact",
			visibleRunes: 5,
			expected:     "exact",
		},
		{
			name:         "empty string",
			value:        "",
			visibleRunes: 5,
			expected:     "",
		},
		{
			name:         "zero visible runes with truncate",
			value:        "secret",
			visibleRunes: 0,
			expected:     "[redacted]",
		},
		{
			name:         "show one character with truncate",
			value:        "password123",
			visibleRunes: 1,
			expected:     "p[redacted]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := redact.PartiallyRedactString(tt.value, tt.visibleRunes, true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHeaders_ActionPartialTruncate(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": []string{"Bearer token123456789"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "Authorization") {
			return redact.ActionRedactPartialTruncate, 7 // Show "Bearer "
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(t.Context(), headers, redactFunc)

	assert.Equal(t, "Bearer [redacted]", result.Get("Authorization"))
}

func TestUrlValues_ActionPartialTruncate(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"api_key": []string{"sk_live_1234567890"},
	}

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "api_key") {
			return redact.ActionRedactPartialTruncate, 8 // Show "sk_live_"
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(t.Context(), values, redactFunc)

	assert.Equal(t, "sk_live_[redacted]", result.Get("api_key"))
}

func TestBody_NilBody(t *testing.T) {
	t.Parallel()

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), nil, redactFunc)
	assert.Nil(t, result)
}

func TestBody_NilRedactFunc(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"key":"value"}`,
		Length:  15,
	}

	result := redact.Body(t.Context(), payload, nil)

	assert.JSONEq(t, `{"key":"value"}`, result.Content)
	assert.Equal(t, int64(15), result.Length)
	// Verify it's a clone, not the same instance
	assert.NotSame(t, payload, result)
}

func TestBody_ActionKeep(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"user":"alice","email":"alice@example.com"}`,
		Length:  43,
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	assert.Equal(t, payload.Content, result.Content)
	assert.Equal(t, payload.Length, result.Length)
	assert.NotSame(t, payload, result) // Should be a clone
}

func TestBody_ActionRedactFully(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"password":"secret123","api_key":"sk_live_xyz"}`,
		Length:  48,
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		if strings.Contains(body.Content, "password") || strings.Contains(body.Content, "api_key") {
			return redact.ActionRedactFully, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	assert.Equal(t, "[redacted]", result.Content)
	assert.Equal(t, int64(48), result.Length) // Original length preserved
	assert.Equal(t, int64(len("[redacted]")), result.TruncatedLength)
}

func TestBody_ActionRedactPartialWithMask(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"data":"sensitive_information_here"}`,
		Length:  38,
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		return redact.ActionRedactPartialWithMask, 10 // Show first 10 characters
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	// Should show first 10 chars and mask the remaining 28
	// nolint:testifylint // Content is intentionally malformed JSON (redacted)
	assert.Equal(t, `{"data":"s***************************`, result.Content)
	assert.Equal(t, int64(38), result.Length)
	assert.Equal(t, int64(len(result.Content)), result.TruncatedLength)
}

func TestBody_ActionRedactPartialTruncate(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"large":"data_payload_with_lots_of_content"}`,
		Length:  46,
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		return redact.ActionRedactPartialTruncate, 15 // Show first 15 characters
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	// nolint:testifylint // Content is intentionally malformed JSON (redacted)
	assert.Equal(t, `{"large":"data_[redacted]`, result.Content)
	assert.Equal(t, int64(46), result.Length)
	assert.Equal(t, int64(len(result.Content)), result.TruncatedLength)
}

func TestBody_ActionDelete(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"secret":"top_secret_data"}`,
		Length:  28,
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		if strings.Contains(body.Content, "secret") {
			return redact.ActionDelete, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	assert.Nil(t, result) // Body should be completely removed
}

func TestBody_DefaultActionOnUnknown(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"test":"data"}`,
		Length:  15,
	}

	// Return an invalid action value (100)
	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		return redact.Action(100), 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	// Should default to ActionKeep (clone the payload)
	assert.Equal(t, payload.Content, result.Content)
	assert.Equal(t, payload.Length, result.Length)
}

func TestBody_WithBase64Content(t *testing.T) {
	t.Parallel()

	binaryContent := "SGVsbG8gV29ybGQ=" // "Hello World" in base64 (16 chars)

	payload := &printable.Payload{
		Base64:  true,
		Content: binaryContent,
		Length:  11, // Original decoded length
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		if body.Base64 {
			return redact.ActionRedactPartialWithMask, 5 // Show first 5 characters
		}

		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	// Base64 content is 16 chars, show first 5, mask remaining 11
	assert.Equal(t, "SGVsb***********", result.Content)
	assert.Equal(t, int64(11), result.Length)
}

func TestBody_PreservesTruncatedLength(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content:         `{"truncated":"data"}`,
		Length:          1000, // Original was 1000 bytes
		TruncatedLength: 20,   // But truncated to 20
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		return redact.ActionRedactPartialWithMask, 10
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	assert.Equal(t, int64(1000), result.Length) // Original length preserved
	// TruncatedLength should reflect the new content length
	assert.Equal(t, int64(len(result.Content)), result.TruncatedLength)
}

func TestBody_RealisticScenario_PasswordInJSON(t *testing.T) {
	t.Parallel()

	payload := &printable.Payload{
		Content: `{"username":"alice","password":"super_secret_123","email":"alice@example.com"}`,
		Length:  79,
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		// Fully redact if body contains passwords
		if strings.Contains(body.Content, `"password"`) {
			return redact.ActionRedactFully, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	assert.Equal(t, "[redacted]", result.Content)
	assert.Equal(t, int64(79), result.Length)
	assert.NotContains(t, result.Content, "super_secret_123")
}

func TestBody_RealisticScenario_LargePayload(t *testing.T) {
	t.Parallel()

	largeContent := strings.Repeat("x", 10000)

	payload := &printable.Payload{
		Content: largeContent,
		Length:  int64(len(largeContent)),
	}

	redactFunc := func(ctx context.Context, body *printable.Payload) (redact.Action, int) {
		// Show first 100 characters for large payloads
		if body.Length > 1000 {
			return redact.ActionRedactPartialTruncate, 100
		}

		return redact.ActionKeep, 0
	}

	result := redact.Body(t.Context(), payload, redactFunc)

	assert.Len(t, result.Content, 100+len("[redacted]"))
	assert.Equal(t, strings.Repeat("x", 100)+"[redacted]", result.Content)
	assert.Equal(t, int64(10000), result.Length)
}
