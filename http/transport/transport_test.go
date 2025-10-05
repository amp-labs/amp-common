package transport

import (
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("creates transport with defaults", func(t *testing.T) {
		t.Parallel()

		trans := New()

		require.NotNil(t, trans)
		assert.Equal(t, defaultMaxIdleConns, trans.MaxIdleConns)
		assert.Equal(t, defaultIdleConnTimeout, trans.IdleConnTimeout)
		assert.Equal(t, defaultTLSHandshakeTimeout, trans.TLSHandshakeTimeout)
		assert.Equal(t, defaultExpectContinueTimeout, trans.ExpectContinueTimeout)
		assert.Equal(t, defaultForceAttemptHTTP2, trans.ForceAttemptHTTP2)
		assert.False(t, trans.DisableKeepAlives)
		assert.Nil(t, trans.TLSClientConfig)
	})

	t.Run("creates transport with disabled connection pooling", func(t *testing.T) {
		t.Parallel()

		trans := New(DisableConnectionPooling)

		require.NotNil(t, trans)
		assert.True(t, trans.DisableKeepAlives)
	})

	t.Run("creates transport with DNS cache", func(t *testing.T) {
		t.Parallel()

		trans := New(EnableDNSCache)

		require.NotNil(t, trans)
		assert.NotNil(t, trans.DialContext)
	})

	t.Run("creates transport with insecure TLS", func(t *testing.T) {
		t.Parallel()

		trans := New(InsecureTLS)

		require.NotNil(t, trans)
		require.NotNil(t, trans.TLSClientConfig)
		assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("creates transport with multiple options", func(t *testing.T) {
		t.Parallel()

		trans := New(DisableConnectionPooling, EnableDNSCache, InsecureTLS)

		require.NotNil(t, trans)
		assert.True(t, trans.DisableKeepAlives)
		assert.NotNil(t, trans.DialContext)
		require.NotNil(t, trans.TLSClientConfig)
		assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
	})
}

func TestNew_EnvironmentVariables(t *testing.T) {
	t.Run("respects HTTP_TRANSPORT_MAX_IDLE_CONNS", func(t *testing.T) {
		t.Setenv("HTTP_TRANSPORT_MAX_IDLE_CONNS", "50")

		trans := New()

		assert.Equal(t, 50, trans.MaxIdleConns)
	})

	t.Run("respects HTTP_TRANSPORT_IDLE_CONN_TIMEOUT", func(t *testing.T) {
		t.Setenv("HTTP_TRANSPORT_IDLE_CONN_TIMEOUT", "60s")

		trans := New()

		assert.Equal(t, 60*time.Second, trans.IdleConnTimeout)
	})

	t.Run("respects HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT", func(t *testing.T) {
		t.Setenv("HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT", "5s")

		trans := New()

		assert.Equal(t, 5*time.Second, trans.TLSHandshakeTimeout)
	})

	t.Run("respects HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2", func(t *testing.T) {
		t.Setenv("HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2", "true")

		trans := New()

		assert.True(t, trans.ForceAttemptHTTP2)
	})
}

func TestGet(t *testing.T) {
	t.Parallel()

	t.Run("returns default pooled transport", func(t *testing.T) {
		t.Parallel()

		rt := Get()

		require.NotNil(t, rt)
		assert.IsType(t, &http.Transport{}, rt)
	})

	t.Run("returns unpooled transport when connection pooling disabled", func(t *testing.T) {
		t.Parallel()

		rt := Get(DisableConnectionPooling)

		require.NotNil(t, rt)
		trans, ok := rt.(*http.Transport)
		require.True(t, ok)
		assert.True(t, trans.DisableKeepAlives)
	})

	t.Run("returns same instance for same config", func(t *testing.T) {
		t.Parallel()

		rt1 := Get()
		rt2 := Get()

		assert.Same(t, rt1, rt2, "should return same singleton instance")
	})

	t.Run("returns different instances for different configs", func(t *testing.T) {
		t.Parallel()

		rt1 := Get()
		rt2 := Get(DisableConnectionPooling)

		assert.NotSame(t, rt1, rt2, "should return different instances for different configs")
	})

	t.Run("returns custom transport override", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{
			MaxIdleConns: 999,
		}

		rt := Get(WithTransportOverride(customTransport))

		assert.Same(t, customTransport, rt)
	})

	t.Run("returns first non-nil transport override", func(t *testing.T) {
		t.Parallel()

		customTransport1 := &http.Transport{MaxIdleConns: 111}
		customTransport2 := &http.Transport{MaxIdleConns: 222}

		rt := Get(WithTransportOverride(nil, customTransport1, customTransport2))

		assert.Same(t, customTransport1, rt)
	})
}

func TestGet_AllCombinations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                     string
		disableConnectionPooling bool
		enableDNSCache           bool
		insecureTLS              bool
	}{
		{"pooled_no_dns_secure", false, false, false},
		{"unpooled_no_dns_secure", true, false, false},
		{"pooled_dns_secure", false, true, false},
		{"unpooled_dns_secure", true, true, false},
		{"pooled_no_dns_insecure", false, false, true},
		{"unpooled_no_dns_insecure", true, false, true},
		{"pooled_dns_insecure", false, true, true},
		{"unpooled_dns_insecure", true, true, true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var opts []Option
			if testCase.disableConnectionPooling {
				opts = append(opts, DisableConnectionPooling)
			}

			if testCase.enableDNSCache {
				opts = append(opts, EnableDNSCache)
			}

			if testCase.insecureTLS {
				opts = append(opts, InsecureTLS)
			}

			rt := Get(opts...)

			require.NotNil(t, rt)
			trans, ok := rt.(*http.Transport)
			require.True(t, ok)

			assert.Equal(t, testCase.disableConnectionPooling, trans.DisableKeepAlives)

			if testCase.insecureTLS {
				require.NotNil(t, trans.TLSClientConfig)
				assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
			}
		})
	}
}

func TestGetContext(t *testing.T) {
	t.Parallel()

	t.Run("returns default transport when no context transport", func(t *testing.T) {
		t.Parallel()

		rt := GetContext(t.Context())

		require.NotNil(t, rt)
		assert.IsType(t, &http.Transport{}, rt)
	})

	t.Run("returns transport from context", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ServerName: "test",
			},
		}
		ctx := WithTransport(t.Context(), customTransport)

		rt := GetContext(ctx)

		assert.Same(t, customTransport, rt)
	})

	t.Run("context transport takes precedence over options", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{MaxIdleConns: 777}
		ctx := WithTransport(t.Context(), customTransport)

		rt := GetContext(ctx, DisableConnectionPooling)

		assert.Same(t, customTransport, rt)
	})
}
