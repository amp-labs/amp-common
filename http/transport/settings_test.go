package transport

import (
	"net/http"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		provider string
		want     string
	}{
		{provider: "accuLynx", want: "HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_ACCULYNX"},
		{provider: "hubspot", want: "HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_HUBSPOT"},
		{provider: "netsuiteM2M", want: "HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_NETSUITEM2M"},
		{provider: "foo-bar.baz", want: "HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_FOO_BAR_BAZ"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.want, providerKey(envMaxIdleConnsPerHost, tt.provider))
		})
	}
}

func TestNew_ProviderSettings(t *testing.T) {
	t.Parallel()

	t.Run("idle connections per host default to net/http's", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, http.DefaultMaxIdleConnsPerHost, New(t.Context()).MaxIdleConnsPerHost)
		assert.Equal(t, http.DefaultMaxIdleConnsPerHost,
			New(t.Context(), WithProvider("accuLynx")).MaxIdleConnsPerHost)
	})

	t.Run("the shared variable applies to every provider without its own", func(t *testing.T) {
		t.Parallel()

		ctx := envutil.WithEnvOverrides(t.Context(), map[string]string{
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST": "8",
		})

		assert.Equal(t, 8, New(ctx).MaxIdleConnsPerHost)
		assert.Equal(t, 8, New(ctx, WithProvider("hubspot")).MaxIdleConnsPerHost)
	})

	t.Run("the provider's variable wins over the shared one", func(t *testing.T) {
		t.Parallel()

		ctx := envutil.WithEnvOverrides(t.Context(), map[string]string{
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST":          "8",
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_ACCULYNX": "32",
		})

		assert.Equal(t, 32, New(ctx, WithProvider("accuLynx")).MaxIdleConnsPerHost)
		assert.Equal(t, 8, New(ctx, WithProvider("hubspot")).MaxIdleConnsPerHost)
		assert.Equal(t, 8, New(ctx).MaxIdleConnsPerHost)
	})

	t.Run("every setting reads the provider's variable", func(t *testing.T) {
		t.Parallel()

		ctx := envutil.WithEnvOverrides(t.Context(), map[string]string{
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_ACCULYNX":           "7",
			"HTTP_TRANSPORT_IDLE_CONN_TIMEOUT_ACCULYNX":        "5s",
			"HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT_ACCULYNX":    "3s",
			"HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT_ACCULYNX":  "2s",
			"HTTP_TRANSPORT_DISABLE_HTTP2_ACCULYNX":            "false",
			"HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2_ACCULYNX":      "true",
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_ACCULYNX":  "32",
			"HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT":           "9s",
			"HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT_SOMETHING": "8s",
		})

		trans := New(ctx, WithProvider("accuLynx"))

		assert.Equal(t, 7, trans.MaxIdleConns)
		assert.Equal(t, 32, trans.MaxIdleConnsPerHost)
		assert.Equal(t, 5*time.Second, trans.IdleConnTimeout)
		assert.Equal(t, 3*time.Second, trans.TLSHandshakeTimeout)
		assert.Equal(t, 2*time.Second, trans.ExpectContinueTimeout)
		assert.Nil(t, trans.TLSNextProto, "HTTP/2 is left on")
		assert.True(t, trans.ForceAttemptHTTP2)
	})
}

func TestGet_WithProvider(t *testing.T) {
	t.Parallel()

	t.Run("a provider without its own variables shares the transport", func(t *testing.T) {
		t.Parallel()

		assert.Same(t, Get(t.Context()), Get(t.Context(), WithProvider("providerWithoutSettings")))
	})

	t.Run("a provider with its own variables gets its own transport", func(t *testing.T) {
		t.Parallel()

		ctx := envutil.WithEnvOverrides(t.Context(), map[string]string{
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_PROVIDERWITHSETTINGS": "32",
		})

		first := Get(ctx, WithProvider("providerWithSettings"))
		second := Get(ctx, WithProvider("providerWithSettings"))

		assert.Same(t, first, second, "the provider's transport is created once")
		assert.NotSame(t, Get(t.Context()), first)

		trans, ok := first.(*http.Transport)
		require.True(t, ok)
		assert.Equal(t, 32, trans.MaxIdleConnsPerHost)
		assert.False(t, trans.DisableKeepAlives)
	})

	t.Run("options still pick the provider's transport", func(t *testing.T) {
		t.Parallel()

		ctx := envutil.WithEnvOverrides(t.Context(), map[string]string{
			"HTTP_TRANSPORT_MAX_IDLE_CONNS_PER_HOST_PROVIDERWITHOPTIONS": "16",
		})

		pooled := Get(ctx, WithProvider("providerWithOptions"))
		unpooled := Get(ctx, WithProvider("providerWithOptions"), DisableConnectionPooling)

		assert.NotSame(t, pooled, unpooled)

		trans, ok := unpooled.(*http.Transport)
		require.True(t, ok)
		assert.True(t, trans.DisableKeepAlives)
		assert.Equal(t, 16, trans.MaxIdleConnsPerHost)
	})

	t.Run("HTTP_TRANSPORT_PREFER_POOLED can be turned off for one provider", func(t *testing.T) {
		t.Parallel()

		ctx := envutil.WithEnvOverrides(t.Context(), map[string]string{
			"HTTP_TRANSPORT_PREFER_POOLED_PROVIDERWITHOUTPOOLING": "false",
		})

		rt := Get(ctx, WithProvider("providerWithoutPooling"))

		assert.Same(t, Get(t.Context(), DisableConnectionPooling), rt, "it uses the shared unpooled transport")

		trans, ok := rt.(*http.Transport)
		require.True(t, ok)
		assert.True(t, trans.DisableKeepAlives)
		assert.Same(t, Get(t.Context()), Get(ctx, WithProvider("anotherProvider")),
			"other providers stay pooled")
	})
}
