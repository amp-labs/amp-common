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

// SetTransport configures a custom HTTP transport using a callback setter function.
// This is used with lazy value overrides to set the transport without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms (e.g., lazy.SetValueOverride) to store the value for later retrieval.
//
// Parameters:
//   - transport: The http.RoundTripper to use for HTTP requests
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
//
// This function is typically used in conjunction with lazy value override systems
// where context values need to be configured before a context is created.
func SetTransport(transport http.RoundTripper, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKeyTransport, transport)
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
