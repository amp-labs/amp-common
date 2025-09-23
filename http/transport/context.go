package transport

import (
	"context"
	"net/http"
)

// contextKey is a type for context keys defined in this package.
// It is unexported to prevent collisions with context keys defined in
// other packages.
type contextKey string

const contextKeyTransport contextKey = "http-transport"

// WithTransport allows setting a custom http.Transport in the context.
// This can be used to override the default transport used by GetContext.
// Note that the transport should be reused for multiple requests to
// take advantage of connection pooling - if you need pooling.
func WithTransport(ctx context.Context, transport http.RoundTripper) context.Context {
	return context.WithValue(ctx, contextKeyTransport, transport)
}

// getTransportFromContext extracts a custom http.Transport from the context.
func getTransportFromContext(ctx context.Context) http.RoundTripper {
	if ctx == nil {
		return nil
	}

	val := ctx.Value(contextKeyTransport)
	if val == nil {
		return nil
	}

	transport, ok := val.(http.RoundTripper)
	if !ok {
		return nil
	}

	return transport
}
