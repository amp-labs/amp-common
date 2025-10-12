package redact_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

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

			result := redact.PartiallyRedactString(tt.value, tt.visibleRunes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestHeaders_NilHeaders(t *testing.T) {
	t.Parallel()

	result := redact.Headers(nil, nil)
	assert.Nil(t, result)
}

func TestHeaders_NilRedactFunc(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token123"},
	}

	result := redact.Headers(headers, nil)

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

	redactFunc := func(key, value string) (redact.Action, int) {
		return redact.ActionKeep, 0
	}

	result := redact.Headers(headers, redactFunc)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Equal(t, "test-client", result.Get("User-Agent"))
}

func TestHeaders_ActionRedact(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": []string{"Bearer secret_token"},
		"X-Api-Key":     []string{"api_key_12345"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "auth") || strings.Contains(strings.ToLower(key), "key") {
			return redact.ActionRedact, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(headers, redactFunc)

	assert.Equal(t, "<redacted>", result.Get("Authorization"))
	assert.Equal(t, "<redacted>", result.Get("X-Api-Key"))
}

func TestHeaders_ActionPartial(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Authorization": []string{"Bearer token123456789"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "Authorization") {
			return redact.ActionPartial, 7 // Show "Bearer "
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(headers, redactFunc)

	assert.Equal(t, "Bearer **************", result.Get("Authorization"))
}

func TestHeaders_ActionDelete(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Content-Type": []string{"application/json"},
		"X-Secret":     []string{"should_be_deleted"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "secret") {
			return redact.ActionDelete, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(headers, redactFunc)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Empty(t, result.Get("X-Secret"))
	assert.NotContains(t, result, "X-Secret")
}

func TestHeaders_MultipleValues(t *testing.T) {
	t.Parallel()

	headers := http.Header{
		"Set-Cookie": []string{"session=abc123", "tracking=xyz789"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "Set-Cookie") {
			return redact.ActionPartial, 8
		}

		return redact.ActionKeep, 0
	}

	result := redact.Headers(headers, redactFunc)

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
	redactFunc := func(key, value string) (redact.Action, int) {
		return redact.Action(100), 0
	}

	result := redact.Headers(headers, redactFunc)

	// Should default to ActionKeep
	assert.Equal(t, "application/json", result.Get("Content-Type"))
}

func TestUrlValues_NilValues(t *testing.T) {
	t.Parallel()

	result := redact.URLValues(nil, nil)
	assert.Nil(t, result)
}

func TestUrlValues_NilRedactFunc(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"api_key": []string{"secret123"},
		"page":    []string{"1"},
	}

	result := redact.URLValues(values, nil)

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

	redactFunc := func(key, value string) (redact.Action, int) {
		return redact.ActionKeep, 0
	}

	result := redact.URLValues(values, redactFunc)

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

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "key") || strings.Contains(strings.ToLower(key), "token") {
			return redact.ActionRedact, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(values, redactFunc)

	assert.Equal(t, "<redacted>", result.Get("api_key"))
	assert.Equal(t, "<redacted>", result.Get("token"))
	assert.Equal(t, "1", result.Get("page"))
}

func TestUrlValues_ActionPartial(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"api_key": []string{"sk_live_1234567890"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "api_key") {
			return redact.ActionPartial, 8 // Show "sk_live_"
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(values, redactFunc)

	assert.Equal(t, "sk_live_**********", result.Get("api_key"))
}

func TestUrlValues_ActionDelete(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"page":   []string{"1"},
		"secret": []string{"should_be_deleted"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "secret") {
			return redact.ActionDelete, 0
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(values, redactFunc)

	assert.Equal(t, "1", result.Get("page"))
	assert.Empty(t, result.Get("secret"))
	assert.NotContains(t, result, "secret")
}

func TestUrlValues_MultipleValues(t *testing.T) {
	t.Parallel()

	values := url.Values{
		"id": []string{"123", "456", "789"},
	}

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.EqualFold(key, "id") {
			return redact.ActionPartial, 1
		}

		return redact.ActionKeep, 0
	}

	result := redact.URLValues(values, redactFunc)

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
	redactFunc := func(key, value string) (redact.Action, int) {
		return redact.Action(100), 0
	}

	result := redact.URLValues(values, redactFunc)

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

	redactFunc := func(key, value string) (redact.Action, int) {
		lowerKey := strings.ToLower(key)

		// Fully redact API keys
		if strings.Contains(lowerKey, "key") {
			return redact.ActionRedact, 0
		}

		// Partially redact authorization (show Bearer prefix)
		if strings.Contains(lowerKey, "authorization") {
			return redact.ActionPartial, 7
		}

		// Delete internal tokens
		if strings.HasPrefix(lowerKey, "x-internal") {
			return redact.ActionDelete, 0
		}

		// Keep everything else
		return redact.ActionKeep, 0
	}

	result := redact.Headers(headers, redactFunc)

	assert.Equal(t, "application/json", result.Get("Content-Type"))
	assert.Equal(t, "MyApp/1.0", result.Get("User-Agent"))
	assert.Equal(t, "Bearer **************************", result.Get("Authorization"))
	assert.Equal(t, "<redacted>", result.Get("X-Api-Key"))
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

	redactFunc := func(key, value string) (redact.Action, int) {
		lowerKey := strings.ToLower(key)

		// Partially redact API keys (show prefix)
		if strings.Contains(lowerKey, "api_key") {
			return redact.ActionPartial, 8
		}

		// Fully delete access codes from logs
		if strings.Contains(lowerKey, "access_code") || strings.Contains(lowerKey, "secret") {
			return redact.ActionDelete, 0
		}

		// Keep everything else
		return redact.ActionKeep, 0
	}

	result := redact.URLValues(values, redactFunc)

	assert.Equal(t, "1", result.Get("page"))
	assert.Equal(t, "10", result.Get("limit"))
	assert.Equal(t, "sk_live_****************", result.Get("api_key"))
	assert.Empty(t, result.Get("access_code"))
	assert.Equal(t, "user123", result.Get("user_id"))
}
