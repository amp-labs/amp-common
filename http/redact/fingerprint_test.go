package redact_test

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"testing"

	"github.com/amp-labs/amp-common/http/printable"
	"github.com/amp-labs/amp-common/http/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var fingerprintPattern = regexp.MustCompile(`^\[fingerprint [0-9a-f]{12}\]$`)

func TestFingerprint(t *testing.T) {
	t.Parallel()

	ctx := redact.WithFingerprintKey(t.Context(), []byte("key-one"))
	otherKeyCtx := redact.WithFingerprintKey(t.Context(), []byte("key-two"))

	t.Run("same value and key give the same fingerprint", func(t *testing.T) {
		t.Parallel()

		first := redact.Fingerprint(ctx, "refresh-token-1")

		assert.Regexp(t, fingerprintPattern, first)
		assert.Equal(t, first, redact.Fingerprint(ctx, "refresh-token-1"))
	})

	t.Run("different values give different fingerprints", func(t *testing.T) {
		t.Parallel()

		assert.NotEqual(t, redact.Fingerprint(ctx, "refresh-token-1"), redact.Fingerprint(ctx, "refresh-token-2"))
	})

	t.Run("a different key gives a different fingerprint", func(t *testing.T) {
		t.Parallel()

		assert.NotEqual(t, redact.Fingerprint(ctx, "refresh-token-1"), redact.Fingerprint(otherKeyCtx, "refresh-token-1"))
	})

	t.Run("the fingerprint does not contain the value", func(t *testing.T) {
		t.Parallel()

		assert.NotContains(t, redact.Fingerprint(ctx, "abc"), "abc")
	})

	t.Run("no key redacts fully", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "[redacted]", redact.Fingerprint(t.Context(), "refresh-token-1"))
	})

	t.Run("an empty key is ignored", func(t *testing.T) {
		t.Parallel()

		emptyKeyCtx := redact.WithFingerprintKey(t.Context(), nil)

		assert.Equal(t, "[redacted]", redact.Fingerprint(emptyKeyCtx, "refresh-token-1"))
	})

	t.Run("an empty value stays empty", func(t *testing.T) {
		t.Parallel()

		assert.Empty(t, redact.Fingerprint(ctx, ""))
	})
}

func TestActionFingerprint(t *testing.T) {
	t.Parallel()

	ctx := redact.WithFingerprintKey(t.Context(), []byte("key-one"))
	fingerprintAll := func(_ context.Context, _, _ string) (redact.Action, int) {
		return redact.ActionFingerprint, 0
	}

	t.Run("headers", func(t *testing.T) {
		t.Parallel()

		headers := http.Header{"X-Secret": {"secret-value"}}

		got := redact.Headers(ctx, headers, fingerprintAll)

		assert.Equal(t, redact.Fingerprint(ctx, "secret-value"), got.Get("X-Secret"))
		assert.Equal(t, "secret-value", headers.Get("X-Secret"), "the original is not modified")
	})

	t.Run("query parameters", func(t *testing.T) {
		t.Parallel()

		values := url.Values{"code": {"secret-code"}}

		got := redact.URLValues(ctx, values, fingerprintAll)

		assert.Equal(t, redact.Fingerprint(ctx, "secret-code"), got.Get("code"))
	})

	t.Run("whole body", func(t *testing.T) {
		t.Parallel()

		body := &printable.Payload{Content: "secret-body", Length: 11}

		got, err := redact.Body(ctx, body, func(_ context.Context, _ *printable.Payload) (redact.Action, int) {
			return redact.ActionFingerprint, 0
		})
		require.NoError(t, err)

		assert.Equal(t, redact.Fingerprint(ctx, "secret-body"), got.Content)
		assert.Equal(t, int64(11), got.Length)
	})
}
