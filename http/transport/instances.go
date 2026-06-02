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
	disableConnectionPooling, enableDNSCache, insecureTLS, disableCompression, publicOnly bool,
) *http.Transport {
	return create(ctx, &config{
		DisableConnectionPooling: disableConnectionPooling,
		EnableDNSCache:           enableDNSCache,
		InsecureTLS:              insecureTLS,
		DisableCompression:       disableCompression,
		PublicOnly:               publicOnly,
	})
}

// Singleton transport instances for common configurations.
// These are lazily initialized and reused to avoid creating duplicate transports.

// pooledTransportNoDNSCache is a transport with connection pooling enabled and no DNS caching.
var pooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, false, false)
	},
)

// unpooledTransportNoDNSCache is a transport with connection pooling disabled and no DNS caching.
var unpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, false, false)
	},
)

// pooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled.
var pooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, false, false)
	},
)

// unpooledTransportWithDNSCache is a transport with connection pooling disabled and DNS caching enabled.
var unpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, false, false)
	},
)

// insecurePooledTransportNoDNSCache is a transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, false, false)
	},
)

// insecureUnpooledTransportNoDNSCache is a transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, false, false)
	},
)

// insecurePooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled,
// and TLS verification disabled.
var insecurePooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, false, false)
	},
)

// insecureUnpooledTransportWithDNSCache is a transport with connection pooling disabled,
// DNS caching enabled, and TLS verification disabled.
var insecureUnpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, false, false)
	},
)

// pooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling enabled,
// no DNS caching, and compression disabled.
var pooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, true, false)
	},
)

// unpooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// no DNS caching, and compression disabled.
var unpooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, true, false)
	},
)

// pooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling and DNS caching enabled,
// and compression disabled.
var pooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, true, false)
	},
)

// unpooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// DNS caching enabled, and compression disabled.
var unpooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, true, false)
	},
)

// insecurePooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling enabled,
// no DNS caching, TLS verification disabled, and compression disabled.
var insecurePooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, true, false)
	},
)

// insecureUnpooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// no DNS caching, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, true, false)
	},
)

// insecurePooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling and DNS caching enabled,
// TLS verification disabled, and compression disabled.
var insecurePooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, true, false)
	},
)

// insecureUnpooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// DNS caching enabled, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, true, false)
	},
)

// Public-only variants mirror the transports above but route DNS through public
// resolvers only, blocking private DNS names and RFC 1918 addresses. They form the
// same pooling/DNS-cache/TLS/compression matrix as the standard transports.

// pooledTransportPublicOnlyNoDNSCache is a public-only transport with connection pooling enabled and no DNS caching.
var pooledTransportPublicOnlyNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, false, true)
	},
)

// unpooledTransportPublicOnlyNoDNSCache is a public-only transport with connection pooling disabled and no DNS caching.
var unpooledTransportPublicOnlyNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, false, true)
	},
)

// pooledTransportPublicOnlyWithDNSCache is a public-only transport with connection pooling and DNS caching enabled.
var pooledTransportPublicOnlyWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, false, true)
	},
)

// unpooledTransportPublicOnlyWithDNSCache is a public-only transport with connection pooling disabled and DNS caching enabled.
var unpooledTransportPublicOnlyWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, false, true)
	},
)

// insecurePooledTransportPublicOnlyNoDNSCache is a public-only transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportPublicOnlyNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, false, true)
	},
)

// insecureUnpooledTransportPublicOnlyNoDNSCache is a public-only transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportPublicOnlyNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, false, true)
	},
)

// insecurePooledTransportPublicOnlyWithDNSCache is a public-only transport with connection pooling and DNS caching enabled,
// and TLS verification disabled.
var insecurePooledTransportPublicOnlyWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, false, true)
	},
)

var insecureUnpooledTransportPublicOnlyWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, false, true)
	},
)

// pooledTransportPublicOnlyNoDNSCacheIgnoreCompression is a public-only transport with connection pooling enabled,
// no DNS caching, and compression disabled.
var pooledTransportPublicOnlyNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, true, true)
	},
)

// unpooledTransportPublicOnlyNoDNSCacheIgnoreCompression is a public-only transport with connection pooling disabled,
// no DNS caching, and compression disabled.
var unpooledTransportPublicOnlyNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, true, true)
	},
)

// pooledTransportPublicOnlyWithDNSCacheIgnoreCompression is a public-only transport with connection pooling
// and DNS caching enabled, and compression disabled.
var pooledTransportPublicOnlyWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, true, true)
	},
)

// unpooledTransportPublicOnlyWithDNSCacheIgnoreCompression is a public-only transport with connection pooling
// disabled, DNS caching enabled, and compression disabled.
var unpooledTransportPublicOnlyWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, true, true)
	},
)

// insecurePooledTransportPublicOnlyNoDNSCacheIgnoreCompression is a public-only transport with connection pooling
// enabled, no DNS caching, TLS verification disabled, and compression disabled.
var insecurePooledTransportPublicOnlyNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, true, true)
	},
)

// insecureUnpooledTransportPublicOnlyNoDNSCacheIgnoreCompression is a public-only transport with connection pooling
// disabled, no DNS caching, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportPublicOnlyNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, true, true)
	},
)

// insecurePooledTransportPublicOnlyWithDNSCacheIgnoreCompression is a public-only transport with connection pooling
// and DNS caching enabled, TLS verification disabled, and compression disabled.
var insecurePooledTransportPublicOnlyWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, true, true)
	},
)

// insecureUnpooledTransportPublicOnlyWithDNSCacheIgnoreCompression is a public-only transport with connection pooling
// disabled, DNS caching enabled, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportPublicOnlyWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, true, true)
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

	if cfg.PublicOnly {
		return getTransportInstancePublicOnly(ctx, cfg)
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

// getTransportInstancePublicOnly returns the public-only singleton transport matching the config.
// It is the PublicOnly counterpart of getTransportInstance, selecting from the public-only matrix.
//
//nolint:gocyclo
func getTransportInstancePublicOnly(ctx context.Context, cfg *config) http.RoundTripper {
	switch {
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportPublicOnlyWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportPublicOnlyWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportPublicOnlyNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS:
		return pooledTransportPublicOnlyNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportPublicOnlyWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportPublicOnlyWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportPublicOnlyNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS:
		return insecurePooledTransportPublicOnlyNoDNSCacheIgnoreCompression.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportPublicOnlyWithDNSCache.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportPublicOnlyWithDNSCache.Get(ctx)
	case !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportPublicOnlyNoDNSCache.Get(ctx)
	case !cfg.InsecureTLS:
		return pooledTransportPublicOnlyNoDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportPublicOnlyWithDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportPublicOnlyWithDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportPublicOnlyNoDNSCache.Get(ctx)
	case cfg.InsecureTLS:
		return insecurePooledTransportPublicOnlyNoDNSCache.Get(ctx)
	default:
		return pooledTransportPublicOnlyNoDNSCache.Get(ctx)
	}
}
