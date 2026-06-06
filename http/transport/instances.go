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
	disableConnectionPooling, enableDNSCache, insecureTLS, disableCompression, ampersandDNS, publicOnly bool,
) *http.Transport {
	return create(ctx, &config{
		DisableConnectionPooling: disableConnectionPooling,
		EnableDNSCache:           enableDNSCache,
		InsecureTLS:              insecureTLS,
		DisableCompression:       disableCompression,
		AmpersandDNS:             ampersandDNS,
		PublicOnly:               publicOnly,
	})
}

// Singleton transport instances for common configurations.
// These are lazily initialized and reused to avoid creating duplicate transports.

// pooledTransportNoDNSCache is a transport with connection pooling enabled and no DNS caching.
var pooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, false, false, false)
	},
)

// unpooledTransportNoDNSCache is a transport with connection pooling disabled and no DNS caching.
var unpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, false, false, false)
	},
)

// pooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled.
var pooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, false, false, false)
	},
)

// unpooledTransportWithDNSCache is a transport with connection pooling disabled and DNS caching enabled.
var unpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, false, false, false)
	},
)

// insecurePooledTransportNoDNSCache is a transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, false, false, false)
	},
)

// insecureUnpooledTransportNoDNSCache is a transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, false, false, false)
	},
)

// insecurePooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled,
// and TLS verification disabled.
var insecurePooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, false, false, false)
	},
)

// insecureUnpooledTransportWithDNSCache is a transport with connection pooling disabled,
// DNS caching enabled, and TLS verification disabled.
var insecureUnpooledTransportWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, false, false, false)
	},
)

// pooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling enabled,
// no DNS caching, and compression disabled.
var pooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, true, false, false)
	},
)

// unpooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// no DNS caching, and compression disabled.
var unpooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, true, false, false)
	},
)

// pooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling and DNS caching enabled,
// and compression disabled.
var pooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, true, false, false)
	},
)

// unpooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// DNS caching enabled, and compression disabled.
var unpooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, true, false, false)
	},
)

// insecurePooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling enabled,
// no DNS caching, TLS verification disabled, and compression disabled.
var insecurePooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, true, false, false)
	},
)

// insecureUnpooledTransportNoDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// no DNS caching, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, true, false, false)
	},
)

// insecurePooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling and DNS caching enabled,
// TLS verification disabled, and compression disabled.
var insecurePooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, true, false, false)
	},
)

// insecureUnpooledTransportWithDNSCacheIgnoreCompression is a transport with connection pooling disabled,
// DNS caching enabled, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, true, false, false)
	},
)

// Ampersand DNS variants mirror the transports above but route DNS through the
// Ampersand DNS dialer, blocking private DNS names and RFC 1918 addresses. They form
// the same pooling/DNS-cache/TLS/compression matrix as the standard transports.

// pooledTransportAmpersandDNSNoDNSCache is an Ampersand DNS transport with connection pooling
// enabled and no DNS caching.
var pooledTransportAmpersandDNSNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, false, true, false)
	},
)

// unpooledTransportAmpersandDNSNoDNSCache is an Ampersand DNS transport with connection pooling
// disabled and no DNS caching.
var unpooledTransportAmpersandDNSNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, false, true, false)
	},
)

// pooledTransportAmpersandDNSWithDNSCache is an Ampersand DNS transport with connection pooling
// and DNS caching enabled.
var pooledTransportAmpersandDNSWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, false, true, false)
	},
)

// unpooledTransportAmpersandDNSWithDNSCache is an Ampersand DNS transport with connection pooling
// disabled and DNS caching enabled.
var unpooledTransportAmpersandDNSWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, false, true, false)
	},
)

// insecurePooledTransportAmpersandDNSNoDNSCache is an Ampersand DNS transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportAmpersandDNSNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, false, true, false)
	},
)

// insecureUnpooledTransportAmpersandDNSNoDNSCache is an Ampersand DNS transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportAmpersandDNSNoDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, false, true, false)
	},
)

// insecurePooledTransportAmpersandDNSWithDNSCache is an Ampersand DNS transport with connection pooling
// and DNS caching enabled, and TLS verification disabled.
var insecurePooledTransportAmpersandDNSWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, false, true, false)
	},
)

// insecureUnpooledTransportAmpersandDNSWithDNSCache is an Ampersand DNS transport with connection pooling
// disabled, DNS caching enabled, and TLS verification disabled.
var insecureUnpooledTransportAmpersandDNSWithDNSCache = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, false, true, false)
	},
)

// pooledTransportAmpersandDNSNoDNSCacheIgnoreCompression is an Ampersand DNS transport with connection pooling enabled,
// no DNS caching, and compression disabled.
var pooledTransportAmpersandDNSNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, true, true, false)
	},
)

// unpooledTransportAmpersandDNSNoDNSCacheIgnoreCompression is an Ampersand DNS transport with
// connection pooling disabled,
// no DNS caching, and compression disabled.
var unpooledTransportAmpersandDNSNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, true, true, false)
	},
)

// pooledTransportAmpersandDNSWithDNSCacheIgnoreCompression is an Ampersand DNS transport with connection pooling
// and DNS caching enabled, and compression disabled.
var pooledTransportAmpersandDNSWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, true, true, false)
	},
)

// unpooledTransportAmpersandDNSWithDNSCacheIgnoreCompression is an Ampersand DNS transport with connection pooling
// disabled, DNS caching enabled, and compression disabled.
var unpooledTransportAmpersandDNSWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, true, true, false)
	},
)

// insecurePooledTransportAmpersandDNSNoDNSCacheIgnoreCompression is an Ampersand DNS transport with connection pooling
// enabled, no DNS caching, TLS verification disabled, and compression disabled.
var insecurePooledTransportAmpersandDNSNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, true, true, false)
	},
)

// insecureUnpooledTransportAmpersandDNSNoDNSCacheIgnoreCompression is an Ampersand DNS transport
// with connection pooling
// disabled, no DNS caching, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportAmpersandDNSNoDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, true, true, false)
	},
)

// insecurePooledTransportAmpersandDNSWithDNSCacheIgnoreCompression is an Ampersand DNS transport
// with connection pooling
// and DNS caching enabled, TLS verification disabled, and compression disabled.
var insecurePooledTransportAmpersandDNSWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, true, true, false)
	},
)

// insecureUnpooledTransportAmpersandDNSWithDNSCacheIgnoreCompression is an Ampersand DNS transport
// with connection pooling
// disabled, DNS caching enabled, TLS verification disabled, and compression disabled.
var insecureUnpooledTransportAmpersandDNSWithDNSCacheIgnoreCompression = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, true, true, false)
	},
)

var pooledTransportAmpersandDNSNoDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, false, true, true)
	},
)

var unpooledTransportAmpersandDNSNoDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, false, true, true)
	},
)

var pooledTransportAmpersandDNSWithDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, false, true, true)
	},
)

var unpooledTransportAmpersandDNSWithDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, false, true, true)
	},
)

var insecurePooledTransportAmpersandDNSNoDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, false, true, true)
	},
)

var insecureUnpooledTransportAmpersandDNSNoDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, false, true, true)
	},
)

var insecurePooledTransportAmpersandDNSWithDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, false, true, true)
	},
)

var insecureUnpooledTransportAmpersandDNSWithDNSCachePubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, false, true, true)
	},
)

var pooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, false, true, true, true)
	},
)

var unpooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, false, true, true, true)
	},
)

var pooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, false, true, true, true)
	},
)

var unpooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, false, true, true, true)
	},
)

var insecurePooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, false, true, true, true, true)
	},
)

var insecureUnpooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, false, true, true, true, true)
	},
)

var insecurePooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, false, true, true, true, true, true)
	},
)

var insecureUnpooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly = lazy.NewCtx[*http.Transport](
	func(ctx context.Context) *http.Transport {
		return transportFactory(ctx, true, true, true, true, true, true)
	},
)

// getTransportInstance returns the appropriate singleton transport instance based on the config.
// If a custom transport override is provided, it returns that instead.
//
//nolint:gocyclo,dupl // exhaustive flag dispatch; mirrors getTransportInstanceAmpersandDNS by design
func getTransportInstance(ctx context.Context, cfg *config) http.RoundTripper {
	for _, tr := range cfg.TransportOverrides {
		if tr != nil {
			return tr
		}
	}

	if cfg.AmpersandDNS {
		if cfg.PublicOnly {
			return getTransportInstanceAmpersandDNSPubOnly(ctx, cfg)
		} else {
			return getTransportInstanceAmpersandDNS(ctx, cfg)
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

// getTransportInstanceAmpersandDNS returns the Ampersand DNS singleton transport matching the
// config. It is the AmpersandDNS counterpart of getTransportInstance, selecting from the
// Ampersand DNS matrix.
//
//nolint:gocyclo,dupl // exhaustive flag dispatch; mirrors getTransportInstance by design
func getTransportInstanceAmpersandDNS(ctx context.Context, cfg *config) http.RoundTripper {
	switch {
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportAmpersandDNSWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS:
		return pooledTransportAmpersandDNSNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportAmpersandDNSWithDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSNoDNSCacheIgnoreCompression.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS:
		return insecurePooledTransportAmpersandDNSNoDNSCacheIgnoreCompression.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSWithDNSCache.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportAmpersandDNSWithDNSCache.Get(ctx)
	case !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSNoDNSCache.Get(ctx)
	case !cfg.InsecureTLS:
		return pooledTransportAmpersandDNSNoDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSWithDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportAmpersandDNSWithDNSCache.Get(ctx)
	case cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSNoDNSCache.Get(ctx)
	case cfg.InsecureTLS:
		return insecurePooledTransportAmpersandDNSNoDNSCache.Get(ctx)
	default:
		return pooledTransportAmpersandDNSNoDNSCache.Get(ctx)
	}
}

//nolint:gocyclo,dupl
func getTransportInstanceAmpersandDNSPubOnly(ctx context.Context, cfg *config) http.RoundTripper {
	switch {
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && !cfg.InsecureTLS:
		return pooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportAmpersandDNSWithDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case cfg.DisableCompression && cfg.InsecureTLS:
		return insecurePooledTransportAmpersandDNSNoDNSCacheIgnoreCompressionPubOnly.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSWithDNSCachePubOnly.Get(ctx)
	case !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportAmpersandDNSWithDNSCachePubOnly.Get(ctx)
	case !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportAmpersandDNSNoDNSCachePubOnly.Get(ctx)
	case !cfg.InsecureTLS:
		return pooledTransportAmpersandDNSNoDNSCachePubOnly.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSWithDNSCachePubOnly.Get(ctx)
	case cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportAmpersandDNSWithDNSCachePubOnly.Get(ctx)
	case cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportAmpersandDNSNoDNSCachePubOnly.Get(ctx)
	case cfg.InsecureTLS:
		return insecurePooledTransportAmpersandDNSNoDNSCachePubOnly.Get(ctx)
	default:
		return pooledTransportAmpersandDNSNoDNSCachePubOnly.Get(ctx)
	}
}
