package transport

import (
	"context"
	"net/http"

	"github.com/amp-labs/amp-common/lazy"
)

// transportFactory creates a *http.Transport with the given config booleans.
// This is a helper function used to initialize the singleton transport instances.
func transportFactory(ctx context.Context, disableConnectionPooling, enableDNSCache, insecureTLS bool) *http.Transport {
	return create(ctx, &config{
		DisableConnectionPooling: disableConnectionPooling,
		EnableDNSCache:           enableDNSCache,
		InsecureTLS:              insecureTLS,
	})
}

// Singleton transport instances for common configurations.
// These are lazily initialized and reused to avoid creating duplicate transports.

// pooledTransportNoDNSCache is a transport with connection pooling enabled and no DNS caching.
var pooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, false, false)
})

// unpooledTransportNoDNSCache is a transport with connection pooling disabled and no DNS caching.
var unpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, false, false)
})

// pooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled.
var pooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, true, false)
})

// unpooledTransportWithDNSCache is a transport with connection pooling disabled and DNS caching enabled.
var unpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, true, false)
})

// insecurePooledTransportNoDNSCache is a transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, false, true)
})

// insecureUnpooledTransportNoDNSCache is a transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, false, true)
})

// insecurePooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled,
// and TLS verification disabled.
var insecurePooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, false, true, true)
})

// insecureUnpooledTransportWithDNSCache is a transport with connection pooling disabled,
// DNS caching enabled, and TLS verification disabled.
var insecureUnpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](func(ctx context.Context) *http.Transport {
	return transportFactory(ctx, true, true, true)
})

// getTransportInstance returns the appropriate singleton transport instance based on the config.
// If a custom transport override is provided, it returns that instead.
func getTransportInstance(ctx context.Context, cfg *config) http.RoundTripper {
	for _, tr := range cfg.TransportOverrides {
		if tr != nil {
			return tr
		}
	}

	switch {
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
