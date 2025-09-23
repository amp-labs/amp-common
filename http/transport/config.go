package transport

import (
	"net/http"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/lazy"
)

type Option func(*config)

type config struct {
	TransportOverrides       []http.RoundTripper
	DisableConnectionPooling bool
	EnableDNSCache           bool
	InsecureTLS              bool
}

func DisableConnectionPooling(c *config) {
	c.DisableConnectionPooling = true
}

func EnableDNSCache(c *config) {
	c.EnableDNSCache = true
}

func InsecureTLS(c *config) {
	c.InsecureTLS = true
}

func WithTransportOverride(transport ...http.RoundTripper) Option {
	return func(c *config) {
		c.TransportOverrides = append(c.TransportOverrides, transport...)
	}
}

var preferPooledForDefault = lazy.New[bool](func() bool {
	return envutil.Bool("HTTP_TRANSPORT_PREFER_POOLED",
		envutil.Default(true)).ValueOrElse(true)
})

func readOptions(opts ...Option) *config {
	cfg := &config{}

	if !preferPooledForDefault.Get() {
		cfg.DisableConnectionPooling = true
	}

	for _, c := range opts {
		if c != nil {
			c(cfg)
		}
	}

	return cfg
}
