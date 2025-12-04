package transport

import (
	"fmt"
	"net/http"

	"github.com/amp-labs/amp-common/errors"
)

// NewCustom creates an http.RoundTripper that wraps a custom RoundTrip function.
//
// This is useful for testing HTTP clients or implementing custom transport logic
// without creating a full transport type. The provided function will be called
// for every HTTP request made through this transport.
//
// If roundTrip is nil, the transport will return an ErrNotImplemented error for
// all requests. This allows creating placeholder transports that can be configured
// later or used in tests to detect unintended HTTP calls.
//
// Example usage:
//
//	// Create a transport that always returns a 200 OK response
//	transport := NewCustom(func(req *http.Request) (*http.Response, error) {
//	    return &http.Response{
//	        StatusCode: 200,
//	        Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
//	    }, nil
//	})
//	client := &http.Client{Transport: transport}
//
// Example testing usage:
//
//	// Create a transport that fails to detect missing error handling
//	transport := NewCustom(func(req *http.Request) (*http.Response, error) {
//	    return nil, errors.New("network failure")
//	})
func NewCustom(roundTrip func(req *http.Request) (*http.Response, error)) http.RoundTripper {
	if roundTrip == nil {
		roundTrip = func(req *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("%w: RoundTrip", errors.ErrNotImplemented)
		}
	}

	return &customTransport{
		roundTrip: roundTrip,
	}
}

// customTransport is an http.RoundTripper implementation that delegates
// to a custom function. This allows for simple mocking and testing of
// HTTP interactions without creating full transport implementations.
type customTransport struct {
	roundTrip func(req *http.Request) (*http.Response, error)
}

// Compile-time check to ensure customTransport implements http.RoundTripper.
// This will cause a compilation error if the interface is not satisfied,
// catching interface contract violations early.
var _ http.RoundTripper = (*customTransport)(nil)

// RoundTrip executes the custom RoundTrip function for the given HTTP request.
// This implements the http.RoundTripper interface, delegating to the function
// provided during construction.
//
// The caller is responsible for ensuring the response body is closed if a
// non-nil response is returned, following standard http.RoundTripper semantics.
func (c *customTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return c.roundTrip(request)
}
