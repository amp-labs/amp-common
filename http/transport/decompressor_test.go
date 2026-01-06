package transport

import (
	"bytes"
	"compress/flate"
	"compress/gzip"
	"compress/zlib"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/klauspost/compress/snappy"
	"github.com/klauspost/compress/zstd"
	"github.com/pierrec/lz4/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testData = "This is test data that will be compressed using various algorithms " +
	"to verify decompression works correctly."

func TestNewDecompressor(t *testing.T) {
	t.Parallel()

	t.Run("creates decompressor with valid round tripper", func(t *testing.T) {
		t.Parallel()

		baseTransport := http.DefaultTransport
		decompressor := NewDecompressor(baseTransport)

		require.NotNil(t, decompressor)
		assert.Implements(t, (*http.RoundTripper)(nil), decompressor)
	})

	t.Run("panics with nil round tripper", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			NewDecompressor(nil)
		})
	})
}

func TestDecompressor_AllCompressionAlgorithms(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		contentEncoding string
		compressFunc    func([]byte) ([]byte, error)
	}{
		{
			name:            "gzip compression",
			contentEncoding: "gzip",
			compressFunc:    compressGzip,
		},
		{
			name:            "deflate compression",
			contentEncoding: "deflate",
			compressFunc:    compressDeflate,
		},
		{
			name:            "zlib compression",
			contentEncoding: "zlib",
			compressFunc:    compressZlib,
		},
		{
			name:            "brotli compression",
			contentEncoding: "br",
			compressFunc:    compressBrotli,
		},
		{
			name:            "zstd compression",
			contentEncoding: "zstd",
			compressFunc:    compressZstd,
		},
		{
			name:            "snappy compression",
			contentEncoding: "snappy",
			compressFunc:    compressSnappy,
		},
		{
			name:            "lz4 compression",
			contentEncoding: "lz4",
			compressFunc:    compressLz4,
		},
		{
			name:            "no compression",
			contentEncoding: "",
			compressFunc:    uncompressed,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Compress the test data
			compressedData, err := tc.compressFunc([]byte(testData))
			require.NoError(t, err)

			// Create a test server that returns compressed data
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Encoding", tc.contentEncoding)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(compressedData)
			}))
			defer server.Close()

			// Create HTTP client with decompressor
			client := &http.Client{
				Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
			}

			// Make request
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			// Read and verify decompressed data
			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			assert.Equal(t, testData, string(body))
		})
	}
}

func TestDecompressor_UncompressedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(testData))
	}))
	defer server.Close()

	client := &http.Client{
		Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)
	t.Cleanup(func() { _ = resp.Body.Close() })

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Equal(t, testData, string(body))
}

func TestDecompressor_EmptyBody(t *testing.T) {
	t.Parallel()

	t.Run("uncompressed empty body", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		client := &http.Client{
			Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Empty(t, body)
	})

	t.Run("gzip compressed empty body", func(t *testing.T) {
		t.Parallel()

		compressedData, err := compressGzip([]byte(""))
		require.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(compressedData)
		}))
		defer server.Close()

		client := &http.Client{
			Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		t.Cleanup(func() { _ = resp.Body.Close() })

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Empty(t, body)
	})
}

func TestDecompressor_MultipleRequests(t *testing.T) {
	t.Parallel()

	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		data := []byte("Request " + string(rune('0'+callCount)))

		compressedData, err := compressGzip(data)
		if err != nil {
			http.Error(w, "compression failed", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(compressedData)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
	}

	// Make multiple requests
	for i := 1; i <= 3; i++ {
		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		_ = resp.Body.Close()

		expected := "Request " + string(rune('0'+i))
		assert.Equal(t, expected, string(body))
	}
}

func TestDecompressor_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("underlying transport error", func(t *testing.T) {
		t.Parallel()

		// Create a custom transport that always returns an error
		errorTransport := NewCustom(func(*http.Request) (*http.Response, error) {
			return nil, io.ErrUnexpectedEOF
		})

		client := &http.Client{
			Transport: NewDecompressor(errorTransport),
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://example.com", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.Error(t, err)
		assert.Nil(t, resp)

		if resp != nil {
			_ = resp.Body.Close()
		}
	})

	t.Run("invalid compressed data", func(t *testing.T) {
		t.Parallel()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Encoding", "gzip")
			w.WriteHeader(http.StatusOK)
			// Write invalid gzip data
			_, _ = w.Write([]byte("this is not valid gzip data"))
		}))
		defer server.Close()

		client := &http.Client{
			Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
		}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		rsp, err := client.Do(req)

		defer func() {
			if rsp != nil && rsp.Body != nil {
				_ = rsp.Body.Close()
			}
		}()

		require.Error(t, err)
		require.Nil(t, rsp)
	})
}

func TestDecompressor_ResourceCleanup(t *testing.T) {
	t.Parallel()

	compressedData, err := compressGzip([]byte(testData))
	require.NoError(t, err)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Encoding", "gzip")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(compressedData)
	}))
	defer server.Close()

	client := &http.Client{
		Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := client.Do(req)
	require.NoError(t, err)

	// Read some data
	buf := make([]byte, 10)
	_, err = resp.Body.Read(buf)
	require.NoError(t, err)

	// Close the body (should close both decompressor and underlying body)
	err = resp.Body.Close()
	require.NoError(t, err)

	// Reading after close should fail
	_, err = resp.Body.Read(buf)
	require.Error(t, err)
}

func TestDecompressor_HTTPStatusCodes(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		statusCode int
	}{
		{"200 OK", http.StatusOK},
		{"201 Created", http.StatusCreated},
		{"204 No Content", http.StatusNoContent},
		{"400 Bad Request", http.StatusBadRequest},
		{"404 Not Found", http.StatusNotFound},
		{"500 Internal Server Error", http.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Only compress if there's a body to compress
				if testCase.statusCode != http.StatusNoContent {
					compressedData, err := compressGzip([]byte(testData))
					if err != nil {
						http.Error(w, "compression failed", http.StatusInternalServerError)

						return
					}

					w.Header().Set("Content-Encoding", "gzip")
					w.WriteHeader(testCase.statusCode)
					_, _ = w.Write(compressedData)
				} else {
					// For 204, don't set Content-Encoding or write a body
					w.WriteHeader(testCase.statusCode)
				}
			}))
			defer server.Close()

			client := &http.Client{
				Transport: Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling),
			}

			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL, nil)
			require.NoError(t, err)

			resp, err := client.Do(req)
			require.NoError(t, err)
			t.Cleanup(func() { _ = resp.Body.Close() })

			assert.Equal(t, testCase.statusCode, resp.StatusCode)

			if testCase.statusCode != http.StatusNoContent {
				body, err := io.ReadAll(resp.Body)
				require.NoError(t, err)
				assert.Equal(t, testData, string(body))
			}
		})
	}
}

func TestDecompressor_InterfaceCompliance(t *testing.T) {
	t.Parallel()

	// Verify that NewDecompressor returns an http.RoundTripper
	_ = Get(t.Context(), DisableCompression, EnableEnhancedDecompression, DisableConnectionPooling)
}

// Compression helper functions

func compressGzip(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	gw := gzip.NewWriter(&buf)

	gw.Name = "data.bin"

	_, err := gw.Write(data)
	if err != nil {
		return nil, err
	}

	err = gw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compressDeflate(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	fw, err := flate.NewWriter(&buf, flate.DefaultCompression)
	if err != nil {
		return nil, err
	}

	_, err = fw.Write(data)
	if err != nil {
		return nil, err
	}

	err = fw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compressZlib(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	zw := zlib.NewWriter(&buf)

	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compressBrotli(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	bw := brotli.NewWriter(&buf)

	_, err := bw.Write(data)
	if err != nil {
		return nil, err
	}

	err = bw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compressZstd(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	zw, err := zstd.NewWriter(&buf)
	if err != nil {
		return nil, err
	}

	_, err = zw.Write(data)
	if err != nil {
		return nil, err
	}

	err = zw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compressSnappy(data []byte) ([]byte, error) {
	// Use framed snappy format for HTTP (not raw snappy)
	var buf bytes.Buffer

	sw := snappy.NewBufferedWriter(&buf)

	_, err := sw.Write(data)
	if err != nil {
		return nil, err
	}

	err = sw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func compressLz4(data []byte) ([]byte, error) {
	var buf bytes.Buffer

	lw := lz4.NewWriter(&buf)

	_, err := lw.Write(data)
	if err != nil {
		return nil, err
	}

	err = lw.Close()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func uncompressed(data []byte) ([]byte, error) {
	return data, nil
}
