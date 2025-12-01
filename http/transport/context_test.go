package transport

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithTransport(t *testing.T) {
	t.Parallel()

	t.Run("stores transport in context", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{MaxIdleConns: 42}
		ctx := WithTransport(t.Context(), customTransport)

		require.NotNil(t, ctx)

		// Verify it can be retrieved
		retrieved := getTransportFromContext(ctx)
		assert.Same(t, customTransport, retrieved)
	})

	t.Run("works with background context", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{MaxIdleConns: 42}
		ctx := WithTransport(t.Context(), customTransport)

		require.NotNil(t, ctx)

		retrieved := getTransportFromContext(ctx)
		assert.Same(t, customTransport, retrieved)
	})

	t.Run("allows chaining context values", func(t *testing.T) {
		t.Parallel()

		type testKey string

		ctx := context.WithValue(t.Context(), testKey("key"), "value")
		customTransport := &http.Transport{MaxIdleConns: 42}
		ctx = WithTransport(ctx, customTransport)

		// Both values should be retrievable
		assert.Equal(t, "value", ctx.Value(testKey("key")))
		assert.Same(t, customTransport, getTransportFromContext(ctx))
	})

	t.Run("overwrites previous transport in context", func(t *testing.T) {
		t.Parallel()

		transport1 := &http.Transport{MaxIdleConns: 1}
		transport2 := &http.Transport{MaxIdleConns: 2}

		ctx := WithTransport(t.Context(), transport1)
		ctx = WithTransport(ctx, transport2)

		retrieved := getTransportFromContext(ctx)
		assert.Same(t, transport2, retrieved)
	})
}

func TestGetTransportFromContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil context", func(t *testing.T) {
		t.Parallel()

		result := getTransportFromContext(t.Context())

		assert.Nil(t, result)
	})

	t.Run("returns nil for context without transport", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		result := getTransportFromContext(ctx)

		assert.Nil(t, result)
	})

	t.Run("returns transport from context", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{MaxIdleConns: 99}
		ctx := WithTransport(t.Context(), customTransport)

		result := getTransportFromContext(ctx)

		assert.Same(t, customTransport, result)
	})

	t.Run("returns nil for wrong type in context", func(t *testing.T) {
		t.Parallel()

		// Manually set wrong type in context
		ctx := context.WithValue(t.Context(), contextKeyTransport, "not a transport")

		result := getTransportFromContext(ctx)

		assert.Nil(t, result)
	})

	t.Run("handles various RoundTripper implementations", func(t *testing.T) {
		t.Parallel()

		// Test with a custom RoundTripper
		type customRoundTripper struct {
			http.RoundTripper
		}

		custom := &customRoundTripper{}
		ctx := WithTransport(t.Context(), custom)

		result := getTransportFromContext(ctx)

		assert.Same(t, custom, result)
	})
}

func TestContextIntegrationWithGetContext(t *testing.T) {
	t.Parallel()

	t.Run("GetContext uses context transport", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{MaxIdleConns: 123}
		ctx := WithTransport(t.Context(), customTransport)

		rt := Get(ctx)

		assert.Same(t, customTransport, rt)
	})

	t.Run("GetContext falls back to default when no context transport", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		rt := Get(ctx)

		require.NotNil(t, rt)
		assert.IsType(t, &http.Transport{}, rt)
	})

	t.Run("GetContext with nil context uses default", func(t *testing.T) {
		t.Parallel()

		rt := Get(t.Context())

		require.NotNil(t, rt)
		assert.IsType(t, &http.Transport{}, rt)
	})
}
