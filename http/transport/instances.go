package transport

import (
	"context"
	"net/http"

	"github.com/amp-labs/amp-common/lazy"
)

// transportFactory creates a *http.Transport with the given config booleans.
// This is a helper function used to initialize the singleton transport instances.
func transportFactory(
	ctx context.Context,
	disableConnectionPooling, enableDNSCache, insecureTLS, disableCompression bool,
) *http.Transport {
	return create(ctx, &config{
		DisableConnectionPooling: disableConnectionPooling,
		EnableDNSCache:           enableDNSCache,
		InsecureTLS:              insecureTLS,
		DisableCompression:       disableCompression,
	})
}

// Singleton transport instances for common configurations.
// These are lazily initialized and reused to avoid creating duplicate transports.

// pooledTransportNoDNSCache is a transport with connection pooling enabled and no DNS caching.
var pooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, false, false, false)
})

// unpooledTransportNoDNSCache is a transport with connection pooling disabled and no DNS caching.
var unpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, false, false, false)
})

// pooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled.
var pooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, true, false, false)
})

// unpooledTransportWithDNSCache is a transport with connection pooling disabled and DNS caching enabled.
var unpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, true, false, false)
})

// insecurePooledTransportNoDNSCache is a transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, false, true, false)
})

// insecureUnpooledTransportNoDNSCache is a transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, false, true, false)
})

// insecurePooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled,
// and TLS verification disabled.
var insecurePooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, true, true, false)
})

// insecureUnpooledTransportWithDNSCache is a transport with connection pooling disabled,
// DNS caching enabled, and TLS verification disabled.
var insecureUnpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, true, true, false)
})

// pooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling enabled,
// no DNS caching, and compression disabled.
var pooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, true)
	},
)

// unpooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// no DNS caching, and compression disabled.
var unpooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, true)
	},
)

// pooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling and DNS caching enabled,
// and compression disabled.
var pooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, true)
	},
)

// unpooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// DNS caching enabled, and compression disabled.
var unpooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, true)
	},
)

// insecurePooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling enabled,
// no DNS caching, TLS verification disabled, and compression disabled.
var insecurePooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, true)
	},
)

// insecureUnpooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// no DNS caching, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, true)
	},
)

// insecurePooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling and DNS caching enabled,
// TLS verification disabled, and compression disabled.
var insecurePooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, true)
	},
)

// insecureUnpooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// DNS caching enabled, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, true)
	},
)

// getTransportInstance returns the appropriate singleton transport instance based on the config.
// If a custom transport override is provided, it returns that instead.
//
//nolint:gocyclo
func getTransportInstance(ctx context.Context, cfg *config) http.RoundTripper {
	for _, tr := range cfg.TransportOverrides {
		if tr != nil {
			return tr
		}
	}

	switch {
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS:
		return pooledTransportNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS:
		return insecurePooledTransportNoDNSCacheIgnoreCompression.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportWithDNSCache.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportWithDNSCache.Get(ctx)
	case !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportNoDNSCache.Get(ctx)
	case !cfg.InsecureTLS:
		return pooledTransportNoDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportWithDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportWithDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportNoDNSCache.Get(ctx)
	case cfg.InsecureTLS:
		return insecurePooledTransportNoDNSCache.Get(ctx)
	default:
		return pooledTransportNoDNSCache.Get(ctx)
	}
}
