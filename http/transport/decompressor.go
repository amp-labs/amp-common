package transport

import (
	"net/http"

	"github.com/amp-labs/amp-common/assert"
	"github.com/amp-labs/amp-common/closer"
	"github.com/fereidani/httpdecompressor"
)

// NewDecompressor creates a new http.RoundTripper that automatically decompresses
// HTTP response bodies based on the Content-Encoding header.
//
// The returned RoundTripper wraps the provided roundTripper and transparently handles
// decompression of responses compressed with gzip, deflate, br (Brotli), or zstd.
// If a response is not compressed, it passes through unchanged.
//
// Important: This wrapper handles proper resource cleanup by ensuring both the
// decompressor and the underlying response body are closed in the correct order
// when the response body is closed.
//
// Parameters:
//   - roundTripper: The underlying http.RoundTripper to wrap. Must not be nil.
//
// Returns:
//   - An http.RoundTripper that provides transparent decompression.
//
// Panics:
//   - If roundTripper is nil.
//
// Example usage:
//
//	baseTransport := &http.Transport{}
//	decompressingTransport := NewDecompressor(baseTransport)
//	client := &http.Client{Transport: decompressingTransport}
func NewDecompressor(roundTripper http.RoundTripper) http.RoundTripper {
	assert.NotNil(roundTripper, "NewDecompressor: roundTripper is nil")

	return &decompressor{
		roundTripper: roundTripper,
	}
}

// decompressor is an http.RoundTripper wrapper that automatically decompresses
// HTTP response bodies based on the Content-Encoding header.
//
// It supports common compression algorithms (gzip, deflate, br, zstd) and handles
// proper resource cleanup by ensuring both the decompressor and the underlying
// response body are closed correctly.
type decompressor struct {
	roundTripper http.RoundTripper
}

// RoundTrip implements http.RoundTripper by performing the HTTP request and
// automatically decompressing the response body if it's compressed.
//
// The decompression process:
//  1. Executes the underlying round trip to get the response
//  2. Creates a decompressor reader based on the Content-Encoding header
//  3. If decompression is needed, wraps the response body with a multi-closer
//  4. Returns the response with the decompressed body reader
//
// Resource Management:
// When decompression occurs, both the decompressor and the original response body
// need to be closed. The multi-closer ensures they're closed in the correct order:
//  1. Close the decompressor first (flushes any buffered data)
//  2. Close the original body second (releases the underlying connection)
//
// If the response is not compressed (bodyReader == origBody), the response is
// returned as-is without additional wrapping.
func (d *decompressor) RoundTrip(request *http.Request) (*http.Response, error) {
	rsp, err := d.roundTripper.RoundTrip(request)
	if err != nil {
		return rsp, err
	}

	origBody := rsp.Body

	// Create a decompressor reader based on Content-Encoding header.
	// Returns origBody unchanged if no decompression is needed.
	bodyReader, err := httpdecompressor.Reader(rsp)
	if err != nil {
		return nil, err
	}

	// If no decompression was needed, return the original response
	if bodyReader == origBody {
		return rsp, nil
	}

	// Set up multi-closer to ensure both the decompressor and original body are closed.
	// The order matters: decoder first, then the underlying body.
	multiCloser := &closer.Closer{}
	multiCloser.Add(bodyReader) // Close the decoder first
	multiCloser.Add(origBody)   // Close the actual body second

	// Replace the response body with a reader that closes both resources
	rsp.Body = closer.ForReader(bodyReader, multiCloser)

	return rsp, nil
}

// Compile-time assertion that decompressor implements http.RoundTripper.
var _ http.RoundTripper = (*decompressor)(nil)
