// Package transport provides HTTP transport configuration with DNS caching and connection pooling.
//
// This package creates reusable http.Transport instances with configurable options for
// connection pooling, DNS caching, and TLS settings. It provides singleton instances for
// common configurations to avoid creating duplicate transports.
//
// # Basic Usage
//
//	// Create a new transport with defaults
//	transport := transport.New()
//
//	// Get a singleton instance with specific options
//	rt := transport.Get(transport.EnableDNSCache)
//
//	// Use context to override transport
//	ctx := transport.WithTransport(ctx, customTransport)
//	rt := transport.GetContext(ctx)
//
// # Configuration Options
//
//   - DisableConnectionPooling: Disable HTTP keep-alive and connection reuse
//   - EnableDNSCache: Use cached DNS lookups to reduce DNS traffic
//   - InsecureTLS: Skip TLS certificate verification (use only for testing)
//   - WithTransportOverride: Provide a custom transport implementation
//
// # Environment Variables
//
// The following environment variables can be used to configure transport behavior:
//
//   - HTTP_TRANSPORT_PREFER_POOLED: Enable connection pooling by default (default: true)
//   - HTTP_TRANSPORT_MAX_IDLE_CONNS: Maximum idle connections (default: 100)
//   - HTTP_TRANSPORT_IDLE_CONN_TIMEOUT: Idle connection timeout (default: 90s)
//   - HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT: TLS handshake timeout (default: 10s)
//   - HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT: Expect-Continue timeout (default: 1s)
//   - HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2: Force HTTP/2 attempts (default: false)
//   - HTTP_TRANSPORT_DIAL_TIMEOUT: Connection dial timeout (default: 30s)
//   - HTTP_TRANSPORT_DIAL_KEEPALIVE: TCP keep-alive duration (default: 30s)
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
func New(ctx context.Context, options ...Option) *http.Transport {
	return create(ctx, readOptions(ctx, options...))
}

// create builds a new http.Transport from the given config.
// It reads environment variables for fine-tuning transport parameters and applies
// the configuration options (connection pooling, DNS caching, TLS settings).
func create(ctx context.Context, cfg *config) *http.Transport {
	maxIdleConns := envutil.Int(ctx, "HTTP_TRANSPORT_MAX_IDLE_CONNS",
		envutil.Default(defaultMaxIdleConns)).
		ValueOrElse(defaultMaxIdleConns)

	idleConnTimeout := envutil.Duration(ctx, "HTTP_TRANSPORT_IDLE_CONN_TIMEOUT",
		envutil.Default(defaultIdleConnTimeout)).
		ValueOrElse(defaultIdleConnTimeout)

	tlsHandshakeTimeout := envutil.Duration(ctx, "HTTP_TRANSPORT_TLS_HANDSHAKE_TIMEOUT",
		envutil.Default(defaultTLSHandshakeTimeout)).
		ValueOrElse(defaultTLSHandshakeTimeout)

	expectContinueTimeout := envutil.Duration(ctx, "HTTP_TRANSPORT_EXPECT_CONTINUE_TIMEOUT",
		envutil.Default(defaultExpectContinueTimeout)).
		ValueOrElse(defaultExpectContinueTimeout)

	disableHTTP2 := envutil.Bool(ctx, "HTTP_TRANSPORT_DISABLE_HTTP2",
		envutil.Default(true)).
		ValueOrElse(true)

	forceAttemptHTTP2 := envutil.Bool(ctx, "HTTP_TRANSPORT_FORCE_ATTEMPT_HTTP2",
		envutil.Default(defaultForceAttemptHTTP2)).
		ValueOrElse(defaultForceAttemptHTTP2)

	dialTimeout := envutil.Duration(ctx, "HTTP_TRANSPORT_DIAL_TIMEOUT",
		envutil.Default(defaultTransportDialTimeout)).
		ValueOrElse(defaultTransportDialTimeout)

	keepAlive := envutil.Duration(ctx, "HTTP_TRANSPORT_DIAL_KEEPALIVE",
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

	if disableHTTP2 {
		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
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

// Get returns a http.RoundTripper based on the provided options or
// from the context if one is set. If no options are provided and no transport
// is set in the context, it returns a default transport instance.
func Get(ctx context.Context, opts ...Option) http.RoundTripper {
	if tr := getTransportFromContext(ctx); tr != nil {
		return tr
	}

	return getTransportInstance(ctx, readOptions(ctx, opts...))
}

// defaultTransportDialContext returns a DialContext function from the given dialer.
// This is used as the default dial function for transports that don't use DNS caching.
func defaultTransportDialContext(dialer *net.Dialer) func(context.Context, string, string) (net.Conn, error) {
	return dialer.DialContext
}
