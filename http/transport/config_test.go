package transport

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisableConnectionPooling(t *testing.T) {
	t.Parallel()

	t.Run("sets DisableConnectionPooling flag", func(t *testing.T) {
		t.Parallel()

		cfg := &config{}
		DisableConnectionPooling(cfg)

		assert.True(t, cfg.DisableConnectionPooling)
	})
}

func TestEnableDNSCache(t *testing.T) {
	t.Parallel()

	t.Run("sets EnableDNSCache flag", func(t *testing.T) {
		t.Parallel()

		cfg := &config{}
		EnableDNSCache(cfg)

		assert.True(t, cfg.EnableDNSCache)
	})
}

func TestInsecureTLS(t *testing.T) {
	t.Parallel()

	t.Run("sets InsecureTLS flag", func(t *testing.T) {
		t.Parallel()

		cfg := &config{}
		InsecureTLS(cfg)

		assert.True(t, cfg.InsecureTLS)
	})
}

func TestWithTransportOverride(t *testing.T) {
	t.Parallel()

	t.Run("appends single transport", func(t *testing.T) {
		t.Parallel()

		trans := &http.Transport{MaxIdleConns: 42}
		cfg := &config{}

		opt := WithTransportOverride(trans)
		opt(cfg)

		require.Len(t, cfg.TransportOverrides, 1)
		assert.Same(t, trans, cfg.TransportOverrides[0])
	})

	t.Run("appends multiple transports", func(t *testing.T) {
		t.Parallel()

		trans1 := &http.Transport{MaxIdleConns: 1}
		trans2 := &http.Transport{MaxIdleConns: 2}
		trans3 := &http.Transport{MaxIdleConns: 3}
		cfg := &config{}

		opt := WithTransportOverride(trans1, trans2, trans3)
		opt(cfg)

		require.Len(t, cfg.TransportOverrides, 3)
		assert.Same(t, trans1, cfg.TransportOverrides[0])
		assert.Same(t, trans2, cfg.TransportOverrides[1])
		assert.Same(t, trans3, cfg.TransportOverrides[2])
	})

	t.Run("can be called multiple times", func(t *testing.T) {
		t.Parallel()

		trans1 := &http.Transport{MaxIdleConns: 1}
		trans2 := &http.Transport{MaxIdleConns: 2}
		cfg := &config{}

		opt1 := WithTransportOverride(trans1)
		opt2 := WithTransportOverride(trans2)

		opt1(cfg)
		opt2(cfg)

		require.Len(t, cfg.TransportOverrides, 2)
		assert.Same(t, trans1, cfg.TransportOverrides[0])
		assert.Same(t, trans2, cfg.TransportOverrides[1])
	})

	t.Run("accepts nil transports", func(t *testing.T) {
		t.Parallel()

		cfg := &config{}

		opt := WithTransportOverride(nil, nil)
		opt(cfg)

		require.Len(t, cfg.TransportOverrides, 2)
		assert.Nil(t, cfg.TransportOverrides[0])
		assert.Nil(t, cfg.TransportOverrides[1])
	})
}

func TestReadOptions(t *testing.T) {
	t.Parallel()

	t.Run("creates default config", func(t *testing.T) {
		t.Parallel()

		cfg := readOptions(t.Context())

		require.NotNil(t, cfg)
		assert.False(t, cfg.DisableConnectionPooling)
		assert.False(t, cfg.EnableDNSCache)
		assert.False(t, cfg.InsecureTLS)
		assert.Empty(t, cfg.TransportOverrides)
	})

	t.Run("applies single option", func(t *testing.T) {
		t.Parallel()

		cfg := readOptions(t.Context(), DisableConnectionPooling)

		assert.True(t, cfg.DisableConnectionPooling)
	})

	t.Run("applies multiple options", func(t *testing.T) {
		t.Parallel()

		cfg := readOptions(t.Context(), DisableConnectionPooling, EnableDNSCache, InsecureTLS)

		assert.True(t, cfg.DisableConnectionPooling)
		assert.True(t, cfg.EnableDNSCache)
		assert.True(t, cfg.InsecureTLS)
	})

	t.Run("handles nil options", func(t *testing.T) {
		t.Parallel()

		cfg := readOptions(t.Context(), nil, DisableConnectionPooling, nil)

		assert.True(t, cfg.DisableConnectionPooling)
		assert.False(t, cfg.EnableDNSCache)
	})

	t.Run("applies options in order", func(t *testing.T) {
		t.Parallel()

		trans1 := &http.Transport{MaxIdleConns: 1}
		trans2 := &http.Transport{MaxIdleConns: 2}

		cfg := readOptions(
			t.Context(),
			WithTransportOverride(trans1),
			WithTransportOverride(trans2),
		)

		require.Len(t, cfg.TransportOverrides, 2)
		assert.Same(t, trans1, cfg.TransportOverrides[0])
		assert.Same(t, trans2, cfg.TransportOverrides[1])
	})

	t.Run("explicit option can override default pooling behavior", func(t *testing.T) {
		t.Parallel()

		// Create a custom option that enables pooling
		enablePooling := func(c *config) {
			c.DisableConnectionPooling = false
		}

		cfg := readOptions(t.Context(), DisableConnectionPooling, enablePooling)

		assert.False(t, cfg.DisableConnectionPooling)
	})
}

func TestConfig(t *testing.T) {
	t.Parallel()

	t.Run("zero value config has all flags false", func(t *testing.T) {
		t.Parallel()

		cfg := &config{}

		assert.False(t, cfg.DisableConnectionPooling)
		assert.False(t, cfg.EnableDNSCache)
		assert.False(t, cfg.InsecureTLS)
		assert.Nil(t, cfg.TransportOverrides)
	})

	t.Run("config fields can be set independently", func(t *testing.T) {
		t.Parallel()

		cfg := &config{
			DisableConnectionPooling: true,
		}

		assert.True(t, cfg.DisableConnectionPooling)
		assert.False(t, cfg.EnableDNSCache)
		assert.False(t, cfg.InsecureTLS)
	})
}
