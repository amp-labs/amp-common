package redact

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"

	"github.com/amp-labs/amp-common/contexts"
)

// fingerprintBytes is how many bytes of the HMAC a fingerprint keeps. Six bytes (twelve hex
// characters) are enough to tell apart the few values a log reader compares, such as the
// refresh tokens one connection has held, and short enough to read.
const fingerprintBytes = 6

// fingerprintKeyType is the context key type for the fingerprint key.
type fingerprintKeyType struct{}

// WithFingerprintKey returns a copy of ctx in which Fingerprint, and the ActionFingerprint
// redaction action, fingerprint values with key. An empty key returns ctx unchanged.
//
// The key must be secret and the same everywhere fingerprints are compared (for example, in
// every service of one environment). Anyone who has it can test guesses against a fingerprint.
func WithFingerprintKey(ctx context.Context, key []byte) context.Context {
	if len(key) == 0 {
		return ctx
	}

	return contexts.WithValue[fingerprintKeyType, []byte](ctx, fingerprintKeyType{}, key)
}

// Fingerprint returns the text to log in place of a secret value: "[fingerprint 1a2b3c4d5e6f]",
// a truncated HMAC-SHA256 of value under the key that WithFingerprintKey put in ctx.
//
// The same value and key always give the same fingerprint, so two log lines show whether they
// carried the same secret. Without the key, a fingerprint does not reveal its value, even when
// the value is easy to guess. A plain hash would not do that.
//
// Without a key in ctx it returns "[redacted]". An empty value returns "", because there is
// nothing to hide and an empty value is worth seeing.
func Fingerprint(ctx context.Context, value string) string {
	if value == "" {
		return ""
	}

	key, found := contexts.GetValue[fingerprintKeyType, []byte](ctx, fingerprintKeyType{})
	if !found || len(key) == 0 {
		return redactedMarker
	}

	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value)) // hash.Hash writes never fail

	return "[fingerprint " + hex.EncodeToString(mac.Sum(nil)[:fingerprintBytes]) + "]"
}
