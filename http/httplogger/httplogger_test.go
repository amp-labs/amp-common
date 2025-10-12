package httplogger_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/amp-labs/amp-common/http/httplogger"
	"github.com/amp-labs/amp-common/http/redact"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogError_NilRequest(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	// Should not panic with nil request
	httplogger.LogError(nil, errors.New("test error"), "GET", "corr-123", nil, params) //nolint:err113

	// Should not have logged anything
	assert.Empty(t, logBuffer.String())
}

func TestLogError_NilParams(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	// Should not panic with nil params
	httplogger.LogError(req, errors.New("test error"), "GET", "corr-123", req.URL, nil) //nolint:err113
}

func TestLogError_BasicError(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/path", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	testErr := errors.New("connection timeout") //nolint:err113

	httplogger.LogError(req, testErr, "GET", "corr-123", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "connection timeout")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "https://api.example.com/path")
	assert.Contains(t, logOutput, "corr-123")
}

func TestLogError_WithQueryParams(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, "https://api.example.com/path?page=1&limit=10", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	testErr := errors.New("bad request") //nolint:err113

	httplogger.LogError(req, testErr, "GET", "corr-456", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "page=1")
	assert.Contains(t, logOutput, "limit=10")
}

func TestLogError_WithRedactedQueryParams(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet, "https://api.example.com/path?api_key=secret123&page=1", nil)
	require.NoError(t, err)

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "api_key") {
			return redact.ActionPartial, 4
		}

		return redact.ActionKeep, 0
	}

	params := &httplogger.LogErrorParams{
		Logger:            logger,
		RedactQueryParams: redactFunc,
	}

	testErr := errors.New("unauthorized") //nolint:err113

	httplogger.LogError(req, testErr, "GET", "corr-789", req.URL, params)

	logOutput := logBuffer.String()
	// Asterisks are URL encoded as %2A
	assert.Contains(t, logOutput, "api_key=secr%2A%2A%2A%2A%2A") // Redacted with URL encoding
	assert.Contains(t, logOutput, "page=1")                      // Not redacted
	assert.Contains(t, logOutput, "unauthorized")
}

func TestLogError_NilError(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	// Should still log even with nil error
	httplogger.LogError(req, nil, "GET", "corr-999", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "GET")
	// Error field should not be present when err is nil
	assert.NotContains(t, logOutput, `"error":`)
}

func TestLogError_NilURL(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	testErr := errors.New("network error") //nolint:err113

	// Should handle nil URL gracefully
	httplogger.LogError(req, testErr, "GET", "corr-000", nil, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "network error")
}

func TestLogError_ComplexError(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/users?admin=true", nil)
	require.NoError(t, err)

	redactFunc := func(key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "admin") {
			return redact.ActionRedact, 0
		}

		return redact.ActionKeep, 0
	}

	params := &httplogger.LogErrorParams{
		Logger:            logger,
		RedactQueryParams: redactFunc,
	}

	testErr := errors.New("internal server error: database connection failed") //nolint:err113

	httplogger.LogError(req, testErr, "POST", "corr-complex", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "internal server error")
	assert.Contains(t, logOutput, "POST")
	assert.Contains(t, logOutput, "admin=%3Credacted%3E") // URL encoded <redacted>
	assert.Contains(t, logOutput, "corr-complex")
}

func TestLogError_EmptyCorrelationID(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	testErr := errors.New("timeout") //nolint:err113

	// Should handle empty correlation ID
	httplogger.LogError(req, testErr, "GET", "", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "timeout")
	assert.Contains(t, logOutput, `"correlationId":""`)
}

func TestLogError_LogLevel(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger:       logger,
		DefaultLevel: slog.LevelError,
	}

	testErr := errors.New("critical error") //nolint:err113

	httplogger.LogError(req, testErr, "GET", "corr-level", req.URL, params)

	logOutput := logBuffer.String()
	// Should log at ERROR level
	assert.Contains(t, logOutput, `"level":"ERROR"`)
	assert.Contains(t, logOutput, "critical error")
}

func TestLogError_SpecialCharactersInURL(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet,
		"https://api.example.com/search?q=hello%20world&filter=a%26b", nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	testErr := errors.New("parse error") //nolint:err113

	httplogger.LogError(req, testErr, "GET", "corr-special", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "parse error")
	// URL should be properly encoded
	assert.Contains(t, logOutput, "api.example.com/search")
}

func TestLogError_MultipleQueryParamsWithSameKey(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	u, err := url.Parse("https://api.example.com/items")
	require.NoError(t, err)

	// Add multiple values for the same key
	q := u.Query()
	q.Add("id", "123")
	q.Add("id", "456")
	q.Add("id", "789")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, u.String(), nil)
	require.NoError(t, err)

	params := &httplogger.LogErrorParams{
		Logger: logger,
	}

	testErr := errors.New("not found") //nolint:err113

	httplogger.LogError(req, testErr, "GET", "corr-multi", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "not found")
	// Should contain all id values
	assert.Contains(t, logOutput, "id=123")
	assert.Contains(t, logOutput, "id=456")
	assert.Contains(t, logOutput, "id=789")
}
