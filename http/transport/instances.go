package transport

import (
	"net/http"

	"github.com/amp-labs/amp-common/lazy"
)

// transportFactory creates a *http.Transport with the given config booleans.
// This is a helper function used to initialize the singleton transport instances.
func transportFactory(disableConnectionPooling, enableDNSCache, insecureTLS bool) *http.Transport {
	return create(&config{
		DisableConnectionPooling: disableConnectionPooling,
		EnableDNSCache:           enableDNSCache,
		InsecureTLS:              insecureTLS,
	})
}

// Singleton transport instances for common configurations.
// These are lazily initialized and reused to avoid creating duplicate transports.

// pooledTransportNoDNSCache is a transport with connection pooling enabled and no DNS caching.
var pooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, false, false)
})

// unpooledTransportNoDNSCache is a transport with connection pooling disabled and no DNS caching.
var unpooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, false, false)
})

// pooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled.
var pooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, true, false)
})

// unpooledTransportWithDNSCache is a transport with connection pooling disabled and DNS caching enabled.
var unpooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, true, false)
})

// insecurePooledTransportNoDNSCache is a transport with connection pooling enabled,
// no DNS caching, and TLS verification disabled.
var insecurePooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, false, true)
})

// insecureUnpooledTransportNoDNSCache is a transport with connection pooling disabled,
// no DNS caching, and TLS verification disabled.
var insecureUnpooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, false, true)
})

// insecurePooledTransportWithDNSCache is a transport with connection pooling and DNS caching enabled,
// and TLS verification disabled.
var insecurePooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, true, true)
})

// insecureUnpooledTransportWithDNSCache is a transport with connection pooling disabled,
// DNS caching enabled, and TLS verification disabled.
var insecureUnpooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, true, true)
})

// getTransportInstance returns the appropriate singleton transport instance based on the config.
// If a custom transport override is provided, it returns that instead.
func getTransportInstance(cfg *config) http.RoundTripper {
	for _, tr := range cfg.TransportOverrides {
		if tr != nil {
			return tr
		}
	}

	switch {
	case !cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return unpooledTransportWithDNSCache.Get()
	case !cfg.InsecureTLS && cfg.EnableDNSCache:
		return pooledTransportWithDNSCache.Get()
	case !cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return unpooledTransportNoDNSCache.Get()
	case !cfg.InsecureTLS:
		return pooledTransportNoDNSCache.Get()
	case cfg.InsecureTLS && cfg.EnableDNSCache && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportWithDNSCache.Get()
	case cfg.InsecureTLS && cfg.EnableDNSCache:
		return insecurePooledTransportWithDNSCache.Get()
	case cfg.InsecureTLS && cfg.DisableConnectionPooling:
		return insecureUnpooledTransportNoDNSCache.Get()
	case cfg.InsecureTLS:
		return insecurePooledTransportNoDNSCache.Get()
	default:
		return pooledTransportNoDNSCache.Get()
	}
}
