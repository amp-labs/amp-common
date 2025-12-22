package transport

import (
	"context"
	"net/http"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
)

// Option is a functional option for configuring transport behavior.
type Option func(*config)

// config holds the configuration options for creating an HTTP transport.
type config struct {
	// TransportOverrides allows providing custom RoundTrippers to use instead of the default transport.
	// The first non-nil transport in this slice will be used.
	TransportOverrides []http.RoundTripper

	// DisableConnectionPooling disables HTTP keep-alive and forces each request to use a new connection.
	DisableConnectionPooling bool

	// EnableDNSCache enables DNS result caching to reduce DNS lookup overhead.
	EnableDNSCache bool

	// InsecureTLS disables TLS certificate verification. Use only for testing.
	InsecureTLS bool

	// DisableCompression will disable normal compression handling.
	DisableCompression bool

	// EnableEnhancedDecompression is only considered if DisableCompression is true.
	// If the value is false, the transport will have zero awareness of
	// compression and the values just pass through. If this value is
	// true, we use NewDecompressor to create a decompressing transport which
	// knows how to handle more modern compression formats, like brotli, zstd,
	// and snappy).
	EnableEnhancedDecompression bool
}

// DisableConnectionPooling returns an Option that disables connection pooling.
// When enabled, each HTTP request will use a new connection instead of reusing existing ones.
func DisableConnectionPooling(c *config) {
	c.DisableConnectionPooling = true
}

// DisableCompression returns an Option that disables normal compression handling.
// When enabled, responses won't be transparently decoded, bodies will be returned as received.
func DisableCompression(c *config) {
	c.DisableCompression = true
}

// EnableEnhancedDecompression returns an Option that enables the EnableEnhancedDecompression flag.
// Note that this implies DisableCompression automatically. See the EnableEnhancedDecompression
// definition for further details.
func EnableEnhancedDecompression(c *config) {
	c.DisableCompression = true          // disable legacy compression handling
	c.EnableEnhancedDecompression = true // use new compression handling
}

// EnableDNSCache returns an Option that enables DNS caching.
// This reduces DNS lookup overhead by caching resolved IP addresses.
func EnableDNSCache(c *config) {
	c.EnableDNSCache = true
}

// InsecureTLS returns an Option that disables TLS certificate verification.
// WARNING: This should only be used for testing purposes.
func InsecureTLS(c *config) {
	c.InsecureTLS = true
}

// WithTransportOverride returns an Option that sets custom RoundTripper implementations.
// The first non-nil transport provided will be used instead of creating a default transport.
func WithTransportOverride(transport ...http.RoundTripper) Option {
	return func(c *config) {
		c.TransportOverrides = append(c.TransportOverrides, transport...)
	}
}

// preferPooledForDefault is a lazily-initialized flag that determines whether connection pooling
// is enabled by default. It can be configured via the HTTP_TRANSPORT_PREFER_POOLED environment variable.
var preferPooledForDefault = lazy.NewCtx[bool](func(ctx context.Context) bool {
	return envutil.Bool(ctx, "HTTP_TRANSPORT_PREFER_POOLED",
		envutil.Default(true)).ValueOrElse(true)
})

// readOptions processes the provided options and returns a config struct.
// It applies the HTTP_TRANSPORT_PREFER_POOLED environment variable as a default,
// then applies each provided option in order.
func readOptions(ctx context.Context, opts ...Option) *config {
	cfg := &config{}

	if !preferPooledForDefault.Get(ctx) {
		cfg.DisableConnectionPooling = true
	}

	for _, c := range opts {
		if c != nil {
			c(cfg)
		}
	}

	return cfg
}
