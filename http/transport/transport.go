package transport

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"

	"github.com/amp-labs/amp-common/envutil"
)

// New returns a new http.Transport with sane defaults that can be overridden
// with environment variables. The defaults are based on the default Transport
// in net/http. Try to use a single instance of the Transport and reuse it
// for all requests to take advantage of connection pooling.
func New(options ...Option) *http.Transport {
	return create(readOptions(options...))
}

func create(cfg *config) *http.Transport {
	maxIdleConns := envutil.Int("HTTP_TRANSPORT_MAX_IDLE_CONNS",
		envutil.Default(defaultMaxIdleConns)).
		ValueOrElse(defaultMaxIdleConns)

	idleConnTimeout := envutil.Duration("HTTP_TRANSPORT_IDLE_CONN_TIMEOUT",
		envutil.Default(defaultIdleConnTimeout)).
		ValueOrElse(defaultIdleConnTimeout)

	tlsHandshakeTimeout := envutil.Duration("HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT",
		envutil.Default(defaultTLSHandshakeTimeout)).
		ValueOrElse(defaultTLSHandshakeTimeout)

	expectContinueTimeout := envutil.Duration("HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT",
		envutil.Default(defaultExpectContinueTimeout)).
		ValueOrElse(defaultExpectContinueTimeout)

	forceAttemptHTTP2 := envutil.Bool("HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2",
		envutil.Default(defaultForceAttemptHTTP2)).
		ValueOrElse(defaultForceAttemptHTTP2)

	dialTimeout := envutil.Duration("HTTP_TRANSPORT_DIAL_TIMEOUT",
		envutil.Default(defaultTransportDialTimeout)).
		ValueOrElse(defaultTransportDialTimeout)

	keepAlive := envutil.Duration("HTTP_TRANSPORT_DIAL_KEEPALIVE",
		envutil.Default(defaultKeepAlive)).
		ValueOrElse(defaultKeepAlive)

	transport := &http.Transport{
		DialContext: defaultTransportDialContext(&net.Dialer{
			Timeout:   dialTimeout,
			KeepAlive: keepAlive,
		}),
		ForceAttemptHTTP2:     forceAttemptHTTP2,
		MaxIdleConns:          maxIdleConns,
		IdleConnTimeout:       idleConnTimeout,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ExpectContinueTimeout: expectContinueTimeout,
	}

	if cfg.DisableConnectionPooling {
		transport.DisableKeepAlives = true
	}

	if cfg.EnableDNSCache {
		useDNSCacheDialer(transport, dialTimeout, keepAlive)
	}

	if cfg.InsecureTLS {
		transport.TLSClientConfig = &tls.Config{
			InsecureSkipVerify: true, //nolint:gosec
		}
	}

	return transport
}

// Get returns a http.RoundTripper based on the provided options.
// If no options are provided, it returns a default transport instance.
func Get(opts ...Option) http.RoundTripper {
	return getTransportInstance(readOptions(opts...))
}

// GetContext returns a http.RoundTripper based on the provided options or
// from the context if one is set. If no options are provided and no transport
// is set in the context, it returns a default transport instance.
func GetContext(ctx context.Context, opts ...Option) http.RoundTripper {
	if tr := getTransportFromContext(ctx); tr != nil {
		return tr
	}

	return Get(opts...)
}

func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}
