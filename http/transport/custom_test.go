package transport

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	amperrors "github.com/amp-labs/amp-common/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errNetworkFailure = errors.New("network failure")
	errCustom         = errors.New("custom error")
)

func TestNewCustom(t *testing.T) {
	t.Parallel()

	t.Run("creates transport with custom function", func(t *testing.T) {
		t.Parallel()

		called := false
		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			called = true

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"status":"ok"}`)),
			}, nil
		})

		require.NotNil(t, transport)
		assert.Implements(t, (*http.RoundTripper)(nil), transport)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.True(t, called, "custom function should be called")
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("creates transport with nil function", func(t *testing.T) {
		t.Parallel()

		transport := NewCustom(nil)

		require.NotNil(t, transport)
		assert.Implements(t, (*http.RoundTripper)(nil), transport)

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req) //nolint:bodyclose // resp is nil on error
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.ErrorIs(t, err, amperrors.ErrNotImplemented)
	})
}

func TestCustomTransport_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("returns successful response", func(t *testing.T) {
		t.Parallel()

		expectedBody := `{"data":"test"}`
		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(expectedBody)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		})

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com/test", nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, expectedBody, string(body))
	})

	t.Run("returns error from custom function", func(t *testing.T) {
		t.Parallel()

		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			return nil, errNetworkFailure
		})

		req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "http://example.com/api", nil)
		require.NoError(t, err)

		resp, err := transport.RoundTrip(req) //nolint:bodyclose // resp is nil on error
		assert.Nil(t, resp)
		require.Error(t, err)
		assert.ErrorIs(t, err, errNetworkFailure)
	})

	t.Run("passes request to custom function", func(t *testing.T) {
		t.Parallel()

		var capturedReq *http.Request

		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			capturedReq = req

			return &http.Response{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
			}, nil
		})

		req, err := http.NewRequestWithContext(t.Context(), http.MethodDelete, "http://example.com/resource/123", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer token")

		resp, err := transport.RoundTrip(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		require.NotNil(t, capturedReq)
		assert.Equal(t, http.MethodDelete, capturedReq.Method)
		assert.Equal(t, "http://example.com/resource/123", capturedReq.URL.String())
		assert.Equal(t, "Bearer token", capturedReq.Header.Get("Authorization"))
	})

	t.Run("can be used with http.Client", func(t *testing.T) {
		t.Parallel()

		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"id":"123"}`)),
			}, nil
		})

		client := &http.Client{Transport: transport}

		req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer func() { _ = resp.Body.Close() }()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.JSONEq(t, `{"id":"123"}`, string(body))
	})
}

func TestCustomTransport_MultipleCalls(t *testing.T) {
	t.Parallel()

	t.Run("handles multiple sequential requests", func(t *testing.T) {
		t.Parallel()

		callCount := 0
		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			callCount++

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}, nil
		})

		for range 5 {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req) //nolint:bodyclose // body closed in checkResponse when needed
			require.NoError(t, err)
			_ = resp.Body.Close()

			assert.Equal(t, http.StatusOK, resp.StatusCode)
		}

		assert.Equal(t, 5, callCount, "custom function should be called for each request")
	})

	t.Run("maintains state across calls", func(t *testing.T) {
		t.Parallel()

		requestPaths := []string{}
		transport := NewCustom(func(req *http.Request) (*http.Response, error) {
			requestPaths = append(requestPaths, req.URL.Path)

			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`ok`)),
			}, nil
		})

		paths := []string{"/users", "/posts", "/comments"}
		for _, path := range paths {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com"+path, nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req) //nolint:bodyclose // body closed in checkResponse when needed
			require.NoError(t, err)
			_ = resp.Body.Close()
		}

		assert.Equal(t, paths, requestPaths)
	})
}

func TestCustomTransport_ErrorScenarios(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		roundTrip     func(req *http.Request) (*http.Response, error)
		expectedError error
		checkResponse func(t *testing.T, resp *http.Response)
	}{
		{
			name:          "nil transport returns ErrNotImplemented",
			roundTrip:     nil,
			expectedError: amperrors.ErrNotImplemented,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()

				assert.Nil(t, resp)
			},
		},
		{
			name: "custom error",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, errCustom
			},
			expectedError: errCustom,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()

				assert.Nil(t, resp)
			},
		},
		{
			name: "http error status",
			roundTrip: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`not found`)),
				}, nil
			},
			expectedError: nil,
			checkResponse: func(t *testing.T, resp *http.Response) {
				t.Helper()

				require.NotNil(t, resp)
				defer func() { _ = resp.Body.Close() }()

				assert.Equal(t, http.StatusNotFound, resp.StatusCode)
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			transport := NewCustom(testCase.roundTrip)
			req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://example.com", nil)
			require.NoError(t, err)

			resp, err := transport.RoundTrip(req) //nolint:bodyclose // body closed in checkResponse when needed

			if testCase.expectedError != nil {
				require.Error(t, err)

				if errors.Is(testCase.expectedError, amperrors.ErrNotImplemented) {
					assert.ErrorIs(t, err, amperrors.ErrNotImplemented)
				}
			} else {
				require.NoError(t, err)
			}

			testCase.checkResponse(t, resp)
		})
	}
}
