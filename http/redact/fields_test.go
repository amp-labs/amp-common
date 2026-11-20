package redact_test

import (
	"context"
	"testing"

	"github.com/amp-labs/amp-common/http/printable"
	"github.com/amp-labs/amp-common/http/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tokenFields is a redaction policy like the one for OAuth token endpoints: it keeps a few
// known fields, shows only that a password was sent, deletes one field, and fingerprints the
// rest.
func tokenFields(_ context.Context, key, _ string) (redact.Action, int) {
	switch key {
	case "grant_type", "error", "error_description", "token_type", "scope":
		return redact.ActionKeep, 0
	case "password":
		return redact.ActionRedactFully, 0
	case "drop_me":
		return redact.ActionDelete, 0
	default:
		return redact.ActionFingerprint, 0
	}
}

func payload(content string) *printable.Payload {
	return &printable.Payload{Content: content, Length: int64(len(content))}
}

func TestBodyFieldsForm(t *testing.T) {
	t.Parallel()

	ctx := redact.WithFingerprintKey(t.Context(), []byte("key-one"))

	t.Run("keeps known fields and replaces secrets", func(t *testing.T) {
		t.Parallel()

		body := payload("grant_type=refresh_token&refresh_token=rt-secret%2Fwith%2Fslashes&client_secret=cs-secret" +
			"&password=hunter2&drop_me=x&scope=read+write")

		got, err := redact.BodyFields(ctx, body, tokenFields)
		require.NoError(t, err)

		assert.Equal(t,
			"grant_type=refresh_token"+
				"&refresh_token="+redact.Fingerprint(ctx, "rt-secret/with/slashes")+
				"&client_secret="+redact.Fingerprint(ctx, "cs-secret")+
				"&password=[redacted]"+
				"&scope=read+write",
			got.Content)
		assert.Equal(t, body.Length, got.Length, "the original size is kept")
		assert.NotContains(t, got.Content, "rt-secret")
		assert.NotContains(t, got.Content, "hunter2")
	})

	t.Run("the fingerprint is taken of the decoded value", func(t *testing.T) {
		t.Parallel()

		encoded, err := redact.BodyFields(ctx, payload("code=a%2Bb"), tokenFields)
		require.NoError(t, err)

		decoded, err := redact.BodyFields(ctx, payload(`{"code":"a+b"}`), tokenFields)
		require.NoError(t, err)

		assert.Equal(t, "code="+redact.Fingerprint(ctx, "a+b"), encoded.Content)
		assert.Equal(t, `{"code":"`+redact.Fingerprint(ctx, "a+b")+`"}`, decoded.Content)
	})

	t.Run("an empty value stays empty", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(ctx, payload("grant_type=refresh_token&refresh_token="), tokenFields)
		require.NoError(t, err)

		assert.Equal(t, "grant_type=refresh_token&refresh_token=", got.Content)
	})

	for name, content := range map[string]string{ //nolint:gosec // test data, not real credentials
		"a bare token":                     "a1b2c3d4e5f6",
		"a padded base64 token":            "QWxhZGRpbjpvcGVuIHNlc2FtZQ==",
		"a token after a field":            "grant_type=refresh_token&a1b2c3d4",
		"an upper-case field name":         "Refresh_Token=abc",
		"a field name with a space":        "refresh token=abc",
		"a field name that is too long":    "a234567890123456789012345678901234567890123456789012345678901234x=1",
		"a value with a broken % escape":   "code=%zz",
		"plain text":                       "invalid grant: the token has expired",
		"a JSON array":                     `["a","b"]`,
		"invalid JSON":                     `{"refresh_token": "abc"`,
		"JSON followed by more data":       `{"a":"b"} trailing`,
		"a field name escaped into upper":  "%41=1",
		"a field name with a percent sign": "a%25b=1",
	} {
		t.Run("rejects "+name, func(t *testing.T) {
			t.Parallel()

			got, err := redact.BodyFields(ctx, payload(content), tokenFields)

			require.ErrorIs(t, err, redact.ErrUnstructuredBody)
			assert.Nil(t, got)
		})
	}
}

func TestBodyFieldsJSON(t *testing.T) {
	t.Parallel()

	ctx := redact.WithFingerprintKey(t.Context(), []byte("key-one"))

	t.Run("redacts nested strings and keeps other values", func(t *testing.T) {
		t.Parallel()

		body := payload(`{
			"access_token": "at-secret",
			"refresh_token": "rt-secret",
			"token_type": "Bearer",
			"expires_in": 3600,
			"ok": true,
			"nothing": null,
			"drop_me": "x",
			"authed_user": {"id": "U1", "access_token": "user-secret"},
			"tokens": ["t1", "t2"],
			"error_description": "a <b> & c"
		}`)

		got, err := redact.BodyFields(ctx, body, tokenFields)
		require.NoError(t, err)

		assert.JSONEq(t, `{
			"access_token": "`+redact.Fingerprint(ctx, "at-secret")+`",
			"refresh_token": "`+redact.Fingerprint(ctx, "rt-secret")+`",
			"token_type": "Bearer",
			"expires_in": 3600,
			"ok": true,
			"nothing": null,
			"authed_user": {"id": "`+redact.Fingerprint(ctx, "U1")+`", "access_token": "`+
			redact.Fingerprint(ctx, "user-secret")+`"},
			"tokens": ["`+redact.Fingerprint(ctx, "t1")+`", "`+redact.Fingerprint(ctx, "t2")+`"],
			"error_description": "a <b> & c"
		}`, got.Content)
		assert.Contains(t, got.Content, "a <b> & c", "HTML characters are not escaped")
		assert.Equal(t, body.Length, got.Length, "the original size is kept")
	})

	t.Run("keeps large numbers exactly", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(ctx, payload(`{"expires_in": 12345678901234567890}`), tokenFields)
		require.NoError(t, err)

		assert.JSONEq(t, `{"expires_in": 12345678901234567890}`, got.Content)
	})

	t.Run("deletes array elements the policy deletes", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(ctx, payload(`{"drop_me": ["a", "b"], "scope": ["c"]}`), tokenFields)
		require.NoError(t, err)

		assert.JSONEq(t, `{"drop_me": [], "scope": ["c"]}`, got.Content)
	})

	t.Run("without a key, secrets are fully redacted", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(t.Context(), payload(`{"refresh_token": "rt-secret"}`), tokenFields)
		require.NoError(t, err)

		assert.JSONEq(t, `{"refresh_token": "[redacted]"}`, got.Content)
	})
}

func TestBodyFieldsEdgeCases(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	t.Run("nil body", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(ctx, nil, tokenFields)
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("empty body", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(ctx, payload(""), tokenFields)
		require.NoError(t, err)
		assert.Empty(t, got.Content)
	})

	t.Run("no policy keeps the body", func(t *testing.T) {
		t.Parallel()

		got, err := redact.BodyFields(ctx, payload("refresh_token=abc"), nil)
		require.NoError(t, err)
		assert.Equal(t, "refresh_token=abc", got.Content)
	})

	t.Run("binary body", func(t *testing.T) {
		t.Parallel()

		body := &printable.Payload{Content: "AAEC", Length: 3, Base64: true}

		_, err := redact.BodyFields(ctx, body, tokenFields)
		require.ErrorIs(t, err, redact.ErrUnstructuredBody)
	})
}
