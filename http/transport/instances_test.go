package transport

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransportFactory(t *testing.T) {
	t.Parallel()

	t.Run("creates transport with all options disabled", func(t *testing.T) {
		t.Parallel()

		trans := transportFactory(t.Context(), false, false, false, false)

		require.NotNil(t, trans)
		assert.False(t, trans.DisableKeepAlives)
		assert.Nil(t, trans.TLSClientConfig)
	})

	t.Run("creates transport with connection pooling disabled", func(t *testing.T) {
		t.Parallel()

		trans := transportFactory(t.Context(), true, false, false, false)

		require.NotNil(t, trans)
		assert.True(t, trans.DisableKeepAlives)
	})

	t.Run("creates transport with DNS cache enabled", func(t *testing.T) {
		t.Parallel()

		trans := transportFactory(t.Context(), false, true, false, false)

		require.NotNil(t, trans)
		assert.NotNil(t, trans.DialContext)
	})

	t.Run("creates transport with insecure TLS", func(t *testing.T) {
		t.Parallel()

		trans := transportFactory(t.Context(), false, false, true, false)

		require.NotNil(t, trans)
		require.NotNil(t, trans.TLSClientConfig)
		assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("creates transport with all options enabled", func(t *testing.T) {
		t.Parallel()

		trans := transportFactory(t.Context(), true, true, true, false)

		require.NotNil(t, trans)
		assert.True(t, trans.DisableKeepAlives)
		assert.NotNil(t, trans.DialContext)
		require.NotNil(t, trans.TLSClientConfig)
		assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
	})
}

func TestGetTransportInstance(t *testing.T) {
	t.Parallel()

	t.Run("returns pooled transport with no options", func(t *testing.T) {
		t.Parallel()

		cfg := &config{}
		rt := getTransportInstance(t.Context(), cfg)

		require.NotNil(t, rt)
		assert.IsType(t, &http.Transport{}, rt)
	})

	t.Run("returns unpooled transport when connection pooling disabled", func(t *testing.T) {
		t.Parallel()

		cfg := &config{DisableConnectionPooling: true}
		rt := getTransportInstance(t.Context(), cfg)

		trans, ok := rt.(*http.Transport)
		require.True(t, ok)
		assert.True(t, trans.DisableKeepAlives)
	})

	t.Run("returns transport with DNS cache", func(t *testing.T) {
		t.Parallel()

		cfg := &config{EnableDNSCache: true}
		rt := getTransportInstance(t.Context(), cfg)

		trans, ok := rt.(*http.Transport)
		require.True(t, ok)
		assert.NotNil(t, trans.DialContext)
	})

	t.Run("returns insecure transport", func(t *testing.T) {
		t.Parallel()

		cfg := &config{InsecureTLS: true}
		rt := getTransportInstance(t.Context(), cfg)

		trans, ok := rt.(*http.Transport)
		require.True(t, ok)
		require.NotNil(t, trans.TLSClientConfig)
		assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
	})

	t.Run("returns transport override when provided", func(t *testing.T) {
		t.Parallel()

		customTransport := &http.Transport{MaxIdleConns: 555}
		cfg := &config{
			TransportOverrides: []http.RoundTripper{customTransport},
		}

		rt := getTransportInstance(t.Context(), cfg)

		assert.Same(t, customTransport, rt)
	})

	t.Run("returns first non-nil transport override", func(t *testing.T) {
		t.Parallel()

		trans1 := &http.Transport{MaxIdleConns: 1}
		trans2 := &http.Transport{MaxIdleConns: 2}
		cfg := &config{
			TransportOverrides: []http.RoundTripper{nil, trans1, trans2},
		}

		rt := getTransportInstance(t.Context(), cfg)

		assert.Same(t, trans1, rt)
	})

	t.Run("uses default transport when all overrides are nil", func(t *testing.T) {
		t.Parallel()

		cfg := &config{
			TransportOverrides: []http.RoundTripper{nil, nil},
		}

		rt := getTransportInstance(t.Context(), cfg)

		require.NotNil(t, rt)
		assert.IsType(t, &http.Transport{}, rt)
	})
}

func TestSingletonInstances(t *testing.T) {
	t.Parallel()

	t.Run("pooledTransportNoDNSCache returns same instance", func(t *testing.T) {
		t.Parallel()

		trans1 := pooledTransportNoDNSCache.Get(t.Context())
		trans2 := pooledTransportNoDNSCache.Get(t.Context())

		assert.Same(t, trans1, trans2)
	})

	t.Run("unpooledTransportNoDNSCache returns same instance", func(t *testing.T) {
		t.Parallel()

		trans1 := unpooledTransportNoDNSCache.Get(t.Context())
		trans2 := unpooledTransportNoDNSCache.Get(t.Context())

		assert.Same(t, trans1, trans2)
	})

	t.Run("pooledTransportWithDNSCache returns same instance", func(t *testing.T) {
		t.Parallel()

		trans1 := pooledTransportWithDNSCache.Get(t.Context())
		trans2 := pooledTransportWithDNSCache.Get(t.Context())

		assert.Same(t, trans1, trans2)
	})

	t.Run("unpooledTransportWithDNSCache returns same instance", func(t *testing.T) {
		t.Parallel()

		trans1 := unpooledTransportWithDNSCache.Get(t.Context())
		trans2 := unpooledTransportWithDNSCache.Get(t.Context())

		assert.Same(t, trans1, trans2)
	})

	t.Run("different configurations return different instances", func(t *testing.T) {
		t.Parallel()

		pooled := pooledTransportNoDNSCache.Get(t.Context())
		unpooled := unpooledTransportNoDNSCache.Get(t.Context())

		assert.NotSame(t, pooled, unpooled)
	})

	t.Run("all singleton instances have correct configuration", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name              string
			instance          *http.Transport
			expectKeepAlives  bool
			expectDialContext bool
			expectInsecureTLS bool
		}{
			{
				name:              "pooledTransportNoDNSCache",
				instance:          pooledTransportNoDNSCache.Get(t.Context()),
				expectKeepAlives:  false,
				expectDialContext: false,
				expectInsecureTLS: false,
			},
			{
				name:              "unpooledTransportNoDNSCache",
				instance:          unpooledTransportNoDNSCache.Get(t.Context()),
				expectKeepAlives:  true,
				expectDialContext: false,
				expectInsecureTLS: false,
			},
			{
				name:              "pooledTransportWithDNSCache",
				instance:          pooledTransportWithDNSCache.Get(t.Context()),
				expectKeepAlives:  false,
				expectDialContext: true,
				expectInsecureTLS: false,
			},
			{
				name:              "unpooledTransportWithDNSCache",
				instance:          unpooledTransportWithDNSCache.Get(t.Context()),
				expectKeepAlives:  true,
				expectDialContext: true,
				expectInsecureTLS: false,
			},
			{
				name:              "insecurePooledTransportNoDNSCache",
				instance:          insecurePooledTransportNoDNSCache.Get(t.Context()),
				expectKeepAlives:  false,
				expectDialContext: false,
				expectInsecureTLS: true,
			},
			{
				name:              "insecureUnpooledTransportNoDNSCache",
				instance:          insecureUnpooledTransportNoDNSCache.Get(t.Context()),
				expectKeepAlives:  true,
				expectDialContext: false,
				expectInsecureTLS: true,
			},
			{
				name:              "insecurePooledTransportWithDNSCache",
				instance:          insecurePooledTransportWithDNSCache.Get(t.Context()),
				expectKeepAlives:  false,
				expectDialContext: true,
				expectInsecureTLS: true,
			},
			{
				name:              "insecureUnpooledTransportWithDNSCache",
				instance:          insecureUnpooledTransportWithDNSCache.Get(t.Context()),
				expectKeepAlives:  true,
				expectDialContext: true,
				expectInsecureTLS: true,
			},
		}

		for _, testCase := range testCases {
			t.Run(testCase.name, func(t *testing.T) {
				t.Parallel()

				assert.Equal(t, testCase.expectKeepAlives, testCase.instance.DisableKeepAlives,
					"DisableKeepAlives mismatch for %s", testCase.name)

				if testCase.expectDialContext {
					assert.NotNil(t, testCase.instance.DialContext,
						"Expected DialContext to be set for %s", testCase.name)
				}

				if testCase.expectInsecureTLS {
					require.NotNil(t, testCase.instance.TLSClientConfig,
						"Expected TLSClientConfig to be set for %s", testCase.name)
					assert.True(t, testCase.instance.TLSClientConfig.InsecureSkipVerify,
						"Expected InsecureSkipVerify to be true for %s", testCase.name)
				}
			})
		}
	})
}

func TestGetTransportInstance_AllCombinations(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name                     string
		disableConnectionPooling bool
		enableDNSCache           bool
		insecureTLS              bool
	}{
		{"secure_pooled_no_dns", false, false, false},
		{"secure_unpooled_no_dns", true, false, false},
		{"secure_pooled_dns", false, true, false},
		{"secure_unpooled_dns", true, true, false},
		{"insecure_pooled_no_dns", false, false, true},
		{"insecure_unpooled_no_dns", true, false, true},
		{"insecure_pooled_dns", false, true, true},
		{"insecure_unpooled_dns", true, true, true},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cfg := &config{
				DisableConnectionPooling: testCase.disableConnectionPooling,
				EnableDNSCache:           testCase.enableDNSCache,
				InsecureTLS:              testCase.insecureTLS,
			}

			rt := getTransportInstance(t.Context(), cfg)

			require.NotNil(t, rt)
			trans, ok := rt.(*http.Transport)
			require.True(t, ok)

			assert.Equal(t, testCase.disableConnectionPooling, trans.DisableKeepAlives)

			if testCase.insecureTLS {
				require.NotNil(t, trans.TLSClientConfig)
				assert.True(t, trans.TLSClientConfig.InsecureSkipVerify)
			}

			// Verify same config returns same instance
			rt2 := getTransportInstance(t.Context(), cfg)
			assert.Same(t, rt, rt2, "should return same singleton instance")
		})
	}
}
