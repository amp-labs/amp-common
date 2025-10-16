package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/amp-labs/amp-common/http/httplogger"
	"github.com/amp-labs/amp-common/http/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTransport is a mock http.RoundTripper for testing.
type mockTransport struct {
	response *http.Response
	err      error
}

func (m *mockTransport) RoundTrip(_ *http.Request) (*http.Response, error) {
	return m.response, m.err
}

// captureLogger captures log output for testing.
type captureLogger struct {
	logs []logEntry
}

type logEntry struct {
	level   slog.Level
	message string
	attrs   map[string]any
}

func newCaptureLogger() *captureLogger {
	return &captureLogger{
		logs: make([]logEntry, 0),
	}
}

func (c *captureLogger) handler() slog.Handler {
	return &captureHandler{logger: c}
}

type captureHandler struct {
	logger *captureLogger
	attrs  []slog.Attr
}

func (h *captureHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *captureHandler) Handle(_ context.Context, r slog.Record) error {
	entry := logEntry{
		level:   r.Level,
		message: r.Message,
		attrs:   make(map[string]any),
	}

	// Collect attributes from the record
	r.Attrs(func(a slog.Attr) bool {
		entry.attrs[a.Key] = a.Value.Any()

		return true
	})

	// Add handler-level attributes
	for _, attr := range h.attrs {
		if _, exists := entry.attrs[attr.Key]; !exists {
			entry.attrs[attr.Key] = attr.Value.Any()
		}
	}

	h.logger.logs = append(h.logger.logs, entry)

	return nil
}

func (h *captureHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)

	return &captureHandler{
		logger: h.logger,
		attrs:  newAttrs,
	}
}

func (h *captureHandler) WithGroup(_ string) slog.Handler {
	return h
}

func TestNewLoggingTransport(t *testing.T) {
	t.Parallel()

	t.Run("creates transport with default parameters", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		trans := NewLoggingTransport(ctx, nil, nil, nil, nil)

		require.NotNil(t, trans)
		lt, ok := trans.(*loggingTransport)
		require.True(t, ok)

		assert.NotNil(t, lt.transport)
		assert.NotNil(t, lt.requestParams)
		assert.NotNil(t, lt.responseParams)
		assert.NotNil(t, lt.errorParams)
		assert.True(t, lt.requestParams.IncludeBody)
		assert.True(t, lt.responseParams.IncludeBody)
	})

	t.Run("uses provided transport", func(t *testing.T) {
		t.Parallel()

		customTransport := &mockTransport{}
		ctx := t.Context()
		trans := NewLoggingTransport(ctx, customTransport, nil, nil, nil)

		lt, ok := trans.(*loggingTransport)
		require.True(t, ok)
		assert.Same(t, customTransport, lt.transport)
	})

	t.Run("uses provided request params", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		requestParams := &httplogger.LogRequestParams{
			Logger:      customLogger,
			IncludeBody: false,
		}

		trans := NewLoggingTransport(ctx, nil, requestParams, nil, nil)

		lt, ok := trans.(*loggingTransport)
		require.True(t, ok)
		assert.Same(t, requestParams, lt.requestParams)
		assert.False(t, lt.requestParams.IncludeBody)
	})

	t.Run("uses provided response params", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		responseParams := &httplogger.LogResponseParams{
			Logger:      customLogger,
			IncludeBody: false,
		}

		trans := NewLoggingTransport(ctx, nil, nil, responseParams, nil)

		lt, ok := trans.(*loggingTransport)
		require.True(t, ok)
		assert.Same(t, responseParams, lt.responseParams)
		assert.False(t, lt.responseParams.IncludeBody)
	})

	t.Run("uses provided error params", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		errorParams := &httplogger.LogErrorParams{
			Logger:         customLogger,
			DefaultMessage: "Custom error message",
		}

		trans := NewLoggingTransport(ctx, nil, nil, nil, errorParams)

		lt, ok := trans.(*loggingTransport)
		require.True(t, ok)
		assert.Same(t, errorParams, lt.errorParams)
	})

	t.Run("sets logger from context when param logger is nil", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		requestParams := &httplogger.LogRequestParams{
			Logger: nil,
		}

		trans := NewLoggingTransport(ctx, nil, requestParams, nil, nil)

		lt, ok := trans.(*loggingTransport)
		require.True(t, ok)
		assert.NotNil(t, lt.requestParams.Logger)
	})
}

func TestLoggingTransport_RoundTrip(t *testing.T) {
	t.Parallel()

	t.Run("successful request logs request and response", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		// Create a mock successful response
		mockResp := &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("success")),
			Header:     make(http.Header),
		}

		mockTransport := &mockTransport{response: mockResp}

		requestParams := &httplogger.LogRequestParams{
			Logger:         customLogger,
			DefaultMessage: httplogger.DefaultLogRequestMessage,
			IncludeBody:    false,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:         customLogger,
			DefaultMessage: httplogger.DefaultLogResponseMessage,
			IncludeBody:    false,
		}
		errorParams := &httplogger.LogErrorParams{
			Logger:         customLogger,
			DefaultMessage: httplogger.DefaultLogErrorMessage,
		}

		trans := NewLoggingTransport(ctx, mockTransport, requestParams, responseParams, errorParams)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com/test", nil)
		require.NoError(t, err)

		resp, err := trans.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Same(t, mockResp, resp)

		// Should have 2 log entries: request and response
		assert.Len(t, captureLog.logs, 2)

		// Check request log
		requestLog := captureLog.logs[0]
		assert.Equal(t, httplogger.DefaultLogRequestMessage, requestLog.message)

		details, ok := requestLog.attrs["details"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, http.MethodGet, details["method"])
		assert.Equal(t, "https://api.example.com/test", details["url"])
		assert.NotEmpty(t, details["correlationId"])

		// Check response log
		responseLog := captureLog.logs[1]
		assert.Equal(t, httplogger.DefaultLogResponseMessage, responseLog.message)

		details, ok = responseLog.attrs["details"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, http.MethodGet, details["method"])
		assert.Equal(t, "https://api.example.com/test", details["url"])
		assert.Equal(t, http.StatusOK, details["statusCode"])
		assert.Equal(t, "200 OK", details["status"])
	})

	t.Run("failed request logs request and error", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		mockErr := errors.New("connection timeout") //nolint:err113
		mockTransport := &mockTransport{err: mockErr}

		requestParams := &httplogger.LogRequestParams{
			Logger:         customLogger,
			DefaultMessage: httplogger.DefaultLogRequestMessage,
			IncludeBody:    false,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:         customLogger,
			DefaultMessage: httplogger.DefaultLogResponseMessage,
			IncludeBody:    false,
		}
		errorParams := &httplogger.LogErrorParams{
			Logger:         customLogger,
			DefaultMessage: httplogger.DefaultLogErrorMessage,
		}

		trans := NewLoggingTransport(ctx, mockTransport, requestParams, responseParams, errorParams)

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.example.com/data", nil)
		require.NoError(t, err)

		resp, err := trans.RoundTrip(req)
		if resp != nil {
			defer resp.Body.Close()
		}

		require.Error(t, err)
		assert.Same(t, mockErr, err)
		assert.Nil(t, resp)

		// Should have 2 log entries: request and error
		assert.Len(t, captureLog.logs, 2)

		// Check request log
		requestLog := captureLog.logs[0]
		assert.Equal(t, httplogger.DefaultLogRequestMessage, requestLog.message)

		// Check error log
		errorLog := captureLog.logs[1]
		assert.Equal(t, httplogger.DefaultLogErrorMessage, errorLog.message)

		details, ok := errorLog.attrs["details"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, http.MethodPost, details["method"])
		assert.Equal(t, "https://api.example.com/data", details["url"])
		assert.Equal(t, "connection timeout", details["error"])
	})

	t.Run("correlation ID is consistent across request and response", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		mockResp := &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}

		mockTransport := &mockTransport{response: mockResp}

		requestParams := &httplogger.LogRequestParams{
			Logger:      customLogger,
			IncludeBody: false,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:      customLogger,
			IncludeBody: false,
		}

		trans := NewLoggingTransport(ctx, mockTransport, requestParams, responseParams, nil)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com", nil)
		require.NoError(t, err)

		resp, err := trans.RoundTrip(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		require.Len(t, captureLog.logs, 2)

		requestDetails, ok := captureLog.logs[0].attrs["details"].(map[string]any)
		require.True(t, ok)

		requestCorrelationID := requestDetails["correlationId"]

		responseDetails, ok := captureLog.logs[1].attrs["details"].(map[string]any)
		require.True(t, ok)

		responseCorrelationID := responseDetails["correlationId"]

		assert.NotEmpty(t, requestCorrelationID)
		assert.Equal(t, requestCorrelationID, responseCorrelationID)
	})

	t.Run("logs request with redacted headers", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		mockResp := &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}

		mockTransport := &mockTransport{response: mockResp}

		redactFunc := func(key, value string) (redact.Action, int) {
			if strings.ToLower(key) == "authorization" {
				return redact.ActionRedactFully, 0
			}

			return redact.ActionKeep, 0
		}

		requestParams := &httplogger.LogRequestParams{
			Logger:        customLogger,
			RedactHeaders: redactFunc,
			IncludeBody:   false,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:      customLogger,
			IncludeBody: false,
		}

		trans := NewLoggingTransport(ctx, mockTransport, requestParams, responseParams, nil)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com", nil)
		require.NoError(t, err)
		req.Header.Set("Authorization", "Bearer secret-token")
		req.Header.Set("Content-Type", "application/json")

		resp, err := trans.RoundTrip(req)
		require.NoError(t, err)

		if resp != nil {
			defer resp.Body.Close()
		}

		require.Len(t, captureLog.logs, 2)

		requestDetails, ok := captureLog.logs[0].attrs["details"].(map[string]any)
		require.True(t, ok)

		headers, ok := requestDetails["headers"].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "[redacted]", headers["Authorization"])
		assert.Equal(t, "application/json", headers["Content-Type"])
	})

	t.Run("works with real HTTP server", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		// Create a test server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test response"))
		}))
		defer server.Close()

		requestParams := &httplogger.LogRequestParams{
			Logger:      customLogger,
			IncludeBody: false,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:      customLogger,
			IncludeBody: false,
		}

		trans := NewLoggingTransport(ctx, http.DefaultTransport, requestParams, responseParams, nil)

		client := &http.Client{
			Transport: trans,
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify logs were captured
		require.GreaterOrEqual(t, len(captureLog.logs), 2)

		requestLog := captureLog.logs[0]
		assert.Equal(t, httplogger.DefaultLogRequestMessage, requestLog.message)

		responseLog := captureLog.logs[1]
		assert.Equal(t, httplogger.DefaultLogResponseMessage, responseLog.message)
	})

	t.Run("handles request with body", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		mockResp := &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}

		mockTransport := &mockTransport{response: mockResp}

		requestParams := &httplogger.LogRequestParams{
			Logger:      customLogger,
			IncludeBody: true,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:      customLogger,
			IncludeBody: false,
		}

		trans := NewLoggingTransport(ctx, mockTransport, requestParams, responseParams, nil)

		bodyContent := `{"key": "value"}`
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.example.com/data",
			bytes.NewBufferString(bodyContent))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")

		resp, err := trans.RoundTrip(req)
		require.NoError(t, err)

		if resp != nil {
			defer resp.Body.Close()
		}

		require.Len(t, captureLog.logs, 2)

		requestDetails, ok := captureLog.logs[0].attrs["details"].(map[string]any)
		require.True(t, ok)

		// Body should be present (though the exact format depends on httplogger implementation)
		assert.Contains(t, requestDetails, "body")
	})

	t.Run("handles redacted query parameters", func(t *testing.T) {
		t.Parallel()

		captureLog := newCaptureLogger()
		customLogger := slog.New(captureLog.handler())
		ctx := t.Context()

		mockResp := &http.Response{
			StatusCode: http.StatusOK,
			Status:     "200 OK",
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		}

		mockTransport := &mockTransport{response: mockResp}

		redactFunc := func(key, _ string) (redact.Action, int) {
			if key == "api_key" {
				return redact.ActionRedactFully, 0
			}

			return redact.ActionKeep, 0
		}

		requestParams := &httplogger.LogRequestParams{
			Logger:            customLogger,
			RedactQueryParams: redactFunc,
			IncludeBody:       false,
		}
		responseParams := &httplogger.LogResponseParams{
			Logger:            customLogger,
			RedactQueryParams: redactFunc,
			IncludeBody:       false,
		}

		trans := NewLoggingTransport(ctx, mockTransport, requestParams, responseParams, nil)

		req, err := http.NewRequestWithContext(ctx, http.MethodGet,
			"https://api.example.com/data?api_key=secret&limit=10", nil)
		require.NoError(t, err)

		resp, err := trans.RoundTrip(req)
		require.NoError(t, err)

		if resp != nil {
			defer resp.Body.Close()
		}

		require.Len(t, captureLog.logs, 2)

		requestDetails, ok := captureLog.logs[0].attrs["details"].(map[string]any)
		require.True(t, ok)

		urlStr, ok := requestDetails["url"].(string)
		require.True(t, ok)
		assert.Contains(t, urlStr, "api_key=%5Bredacted%5D") // URL encoded [redacted]
		assert.Contains(t, urlStr, "limit=10")
	})
}

func TestLoggingTransport_Interface(t *testing.T) {
	t.Parallel()

	t.Run("implements http.RoundTripper", func(t *testing.T) {
		t.Parallel()

		var _ http.RoundTripper = (*loggingTransport)(nil)
	})
}
