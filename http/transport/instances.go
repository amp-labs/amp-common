package transport

import (
	"net/http"

	"github.com/amp-labs/amp-common/lazy"
)

// transportFactory creates a *http.Transport with the given config booleans.
func transportFactory(disableConnectionPooling, enableDNSCache, insecureTLS bool) *http.Transport {
	return create(&config{
		DisableConnectionPooling: disableConnectionPooling,
		EnableDNSCache:           enableDNSCache,
		InsecureTLS:              insecureTLS,
	})
}

var pooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, false, false)
})

var unpooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, false, false)
})

var pooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, true, false)
})

var unpooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, true, false)
})

var insecurePooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, false, true)
})

var insecureUnpooledTransportNoDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, false, true)
})

var insecurePooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(false, true, true)
})

var insecureUnpooledTransportWithDNSCache = lazy.New[*http.Transport](func() *http.Transport {
	return transportFactory(true, true, true)
})

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
