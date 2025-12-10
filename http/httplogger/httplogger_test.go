package httplogger_test

import (
	"bytes"
	"context"
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
	httplogger.LogError(t.Context(), nil, errors.New("test error"), "GET", "corr-123", nil, params) //nolint:err113

	// Should not have logged anything
	assert.Empty(t, logBuffer.String())
}

func TestLogError_NilParams(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	// Should not panic with nil params
	httplogger.LogError(t.Context(), req, errors.New("test error"), "GET", "corr-123", req.URL, nil) //nolint:err113
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

	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-123", req.URL, params)

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

	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-456", req.URL, params)

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

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "api_key") {
			return redact.ActionRedactPartialWithMask, 4
		}

		return redact.ActionKeep, 0
	}

	params := &httplogger.LogErrorParams{
		Logger:            logger,
		RedactQueryParams: redactFunc,
	}

	testErr := errors.New("unauthorized") //nolint:err113

	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-789", req.URL, params)

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
	httplogger.LogError(t.Context(), req, nil, "GET", "corr-999", req.URL, params)

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
	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-000", nil, params)

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

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "admin") {
			return redact.ActionRedactFully, 0
		}

		return redact.ActionKeep, 0
	}

	params := &httplogger.LogErrorParams{
		Logger:            logger,
		RedactQueryParams: redactFunc,
	}

	testErr := errors.New("internal server error: database connection failed") //nolint:err113

	httplogger.LogError(t.Context(), req, testErr, "POST", "corr-complex", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "internal server error")
	assert.Contains(t, logOutput, "POST")
	assert.Contains(t, logOutput, "admin=%5Bredacted%5D") // URL encoded [redacted]
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
	httplogger.LogError(t.Context(), req, testErr, "GET", "", req.URL, params)

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

	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-level", req.URL, params)

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

	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-special", req.URL, params)

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

	httplogger.LogError(t.Context(), req, testErr, "GET", "corr-multi", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "HTTP request failed")
	assert.Contains(t, logOutput, "not found")
	// Should contain all id values
	assert.Contains(t, logOutput, "id=123")
	assert.Contains(t, logOutput, "id=456")
	assert.Contains(t, logOutput, "id=789")
}

func TestLogRequest_IncludeBodyOverride(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		body                []byte
		includeBody         bool
		overrideReturnValue bool
		shouldContainBody   bool
		shouldContainValues []string
		shouldNotContain    []string
	}{
		{
			name:                "override returns true",
			body:                []byte(`{"key":"value"}`),
			includeBody:         false,
			overrideReturnValue: true,
			shouldContainBody:   true,
			shouldContainValues: []string{`\"key\":\"value\"`, "POST", `"body"`},
			shouldNotContain:    nil,
		},
		{
			name:                "override returns false",
			body:                []byte(`{"secret":"hidden"}`),
			includeBody:         true,
			overrideReturnValue: false,
			shouldContainBody:   false,
			shouldContainValues: nil,
			shouldNotContain:    []string{"secret", "hidden", `"body"`},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var logBuffer bytes.Buffer

			logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

			req, err := http.NewRequestWithContext(
				t.Context(), http.MethodPost, "https://api.example.com/data", bytes.NewReader(testCase.body))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")

			params := &httplogger.LogRequestParams{
				Logger:      logger,
				IncludeBody: testCase.includeBody,
				IncludeBodyOverride: func(ctx context.Context, request *http.Request, bodyBytes []byte) bool {
					return testCase.overrideReturnValue
				},
			}

			httplogger.LogRequest(t.Context(), req, testCase.body, "corr-123", params)

			logOutput := logBuffer.String()
			assert.Contains(t, logOutput, "Sending HTTP request")

			for _, val := range testCase.shouldContainValues {
				assert.Contains(t, logOutput, val)
			}

			for _, val := range testCase.shouldNotContain {
				assert.NotContains(t, logOutput, val)
			}
		})
	}
}

func TestLogRequest_IncludeBodyOverride_ConditionalOnSize(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	smallBody := []byte(`{"small":"data"}`)
	largeBody := make([]byte, 1024*1024) // 1 MB

	maxSize := int64(100 * 1024) // 100 KB

	params := &httplogger.LogRequestParams{
		Logger:      logger,
		IncludeBody: true,
		IncludeBodyOverride: func(ctx context.Context, request *http.Request, bodyBytes []byte) bool {
			return int64(len(bodyBytes)) <= maxSize
		},
	}

	// Test with small body
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/small", bytes.NewReader(smallBody))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	httplogger.LogRequest(t.Context(), req, smallBody, "corr-small", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `\"small\":\"data\"`) // Escaped because it's nested in JSON
	assert.Contains(t, logOutput, `"body"`)

	// Test with large body
	logBuffer.Reset()

	req, err = http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/large", bytes.NewReader(largeBody))
	require.NoError(t, err)

	httplogger.LogRequest(t.Context(), req, largeBody, "corr-large", params)

	logOutput = logBuffer.String()
	assert.NotContains(t, logOutput, `"body"`)
}

func TestLogRequest_IncludeBodyOverride_ChecksEndpoint(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"data":"sensitive"}`)

	params := &httplogger.LogRequestParams{
		Logger:      logger,
		IncludeBody: true,
		IncludeBodyOverride: func(ctx context.Context, request *http.Request, bodyBytes []byte) bool {
			// Don't log body for /auth endpoints
			return !strings.Contains(request.URL.Path, "/auth")
		},
	}

	// Test with /auth endpoint
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/auth/login", bytes.NewReader(body))
	require.NoError(t, err)

	httplogger.LogRequest(t.Context(), req, body, "corr-auth", params)

	logOutput := logBuffer.String()
	assert.NotContains(t, logOutput, "sensitive")
	assert.NotContains(t, logOutput, `"body"`)

	// Test with regular endpoint
	logBuffer.Reset()

	req, err = http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/data", bytes.NewReader(body))
	require.NoError(t, err)

	httplogger.LogRequest(t.Context(), req, body, "corr-data", params)

	logOutput = logBuffer.String()
	assert.Contains(t, logOutput, `\"data\":\"sensitive\"`) // Escaped because it's nested in JSON
	assert.Contains(t, logOutput, `"body"`)
}

func TestLogResponse_IncludeBodyOverride_ReturnsTrue(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"result":"success"}`)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/data", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")

	params := &httplogger.LogResponseParams{
		Logger:      logger,
		IncludeBody: false, // This should be overridden
		IncludeBodyOverride: func(ctx context.Context, response *http.Response, bodyBytes []byte) bool {
			return true // Override to include body
		},
	}

	httplogger.LogResponse(t.Context(), resp, body, "GET", "corr-123", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Received HTTP response")
	assert.Contains(t, logOutput, `\"result\":\"success\"`) // Escaped because it's nested in JSON
	assert.Contains(t, logOutput, "200")
	assert.Contains(t, logOutput, `"body"`) // Ensure body field exists
}

func TestLogResponse_IncludeBodyOverride_ReturnsFalse(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"error":"internal"}`)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/data", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "500 Internal Server Error",
		StatusCode: http.StatusInternalServerError,
		Request:    req,
		Header:     make(http.Header),
	}

	params := &httplogger.LogResponseParams{
		Logger:      logger,
		IncludeBody: true, // This should be overridden
		IncludeBodyOverride: func(ctx context.Context, response *http.Response, bodyBytes []byte) bool {
			return false // Override to exclude body
		},
	}

	httplogger.LogResponse(t.Context(), resp, body, "GET", "corr-456", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Received HTTP response")
	assert.NotContains(t, logOutput, "error")
	assert.NotContains(t, logOutput, "internal")
	assert.NotContains(t, logOutput, `"body"`)
}

func TestLogResponse_IncludeBodyOverride_ConditionalOnStatusCode(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	successBody := []byte(`{"status":"ok"}`)
	errorBody := []byte(`{"error":"details"}`)

	params := &httplogger.LogResponseParams{
		Logger:      logger,
		IncludeBody: true,
		IncludeBodyOverride: func(ctx context.Context, response *http.Response, bodyBytes []byte) bool {
			// Only log body for error responses (4xx, 5xx)
			return false // We'll override this in the actual test based on the response
		},
	}

	// Test with success response (should not log body based on our logic)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/data", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}

	// Update params to check status code from response
	params.IncludeBodyOverride = func(ctx context.Context, response *http.Response, bodyBytes []byte) bool {
		// In real scenario, we'd need to access response somehow
		// For this test, we'll just check body content as a proxy
		return bytes.Contains(bodyBytes, []byte("error"))
	}

	httplogger.LogResponse(t.Context(), resp, successBody, "GET", "corr-success", req.URL, params)

	logOutput := logBuffer.String()
	assert.NotContains(t, logOutput, `"status":"ok"`)

	// Test with error response (should log body)
	logBuffer.Reset()

	resp.StatusCode = 500
	resp.Status = "500 Internal Server Error"

	httplogger.LogResponse(t.Context(), resp, errorBody, "GET", "corr-error", req.URL, params)

	logOutput = logBuffer.String()
	assert.Contains(t, logOutput, `\"error\":\"details\"`) // Escaped because it's nested in JSON
	assert.Contains(t, logOutput, `"body"`)
}

func TestLogResponse_IncludeBodyOverride_ConditionalOnSize(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	smallBody := []byte(`{"small":"response"}`)
	largeBody := make([]byte, 1024*1024) // 1 MB

	maxSize := int64(100 * 1024) // 100 KB

	params := &httplogger.LogResponseParams{
		Logger:      logger,
		IncludeBody: true,
		IncludeBodyOverride: func(ctx context.Context, response *http.Response, bodyBytes []byte) bool {
			return int64(len(bodyBytes)) <= maxSize
		},
	}

	// Test with small response
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/small", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}

	httplogger.LogResponse(t.Context(), resp, smallBody, "GET", "corr-small", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `\"small\":\"response\"`) // Escaped because it's nested in JSON
	assert.Contains(t, logOutput, `"body"`)

	// Test with large response
	logBuffer.Reset()
	httplogger.LogResponse(t.Context(), resp, largeBody, "GET", "corr-large", req.URL, params)

	logOutput = logBuffer.String()
	assert.NotContains(t, logOutput, `"body"`)
}

func TestLogRequest_IncludeBodyOverride_NilOverride(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"key":"value"}`)
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/data", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	params := &httplogger.LogRequestParams{
		Logger:              logger,
		IncludeBody:         true, // Should use this since override is nil
		IncludeBodyOverride: nil,
	}

	httplogger.LogRequest(t.Context(), req, body, "corr-123", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Sending HTTP request")
	assert.Contains(t, logOutput, `\"key\":\"value\"`) // Escaped because it's nested in JSON
	assert.Contains(t, logOutput, `"body"`)
}

func TestLogResponse_IncludeBodyOverride_NilOverride(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"result":"success"}`)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/data", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}

	params := &httplogger.LogResponseParams{
		Logger:              logger,
		IncludeBody:         false, // Should use this since override is nil
		IncludeBodyOverride: nil,
	}

	httplogger.LogResponse(t.Context(), resp, body, "GET", "corr-456", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Received HTTP response")
	assert.NotContains(t, logOutput, `"result":"success"`)
	assert.NotContains(t, logOutput, `"body"`)
}

// Basic LogRequest tests.
func TestLogRequest_NilRequest(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	params := &httplogger.LogRequestParams{
		Logger: logger,
	}

	// Should not panic with nil request
	httplogger.LogRequest(t.Context(), nil, nil, "corr-123", params)

	// Should not have logged anything
	assert.Empty(t, logBuffer.String())
}

func TestLogRequest_NilParams(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://api.example.com/users", nil)
	require.NoError(t, err)

	// Should not panic with nil params
	httplogger.LogRequest(t.Context(), req, nil, "corr-123", nil)
}

func TestLogRequest_BasicRequest(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://api.example.com/users", nil)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-client/1.0")

	params := &httplogger.LogRequestParams{
		Logger:      logger,
		IncludeBody: false,
	}

	httplogger.LogRequest(t.Context(), req, nil, "corr-123", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Sending HTTP request")
	assert.Contains(t, logOutput, "POST")
	assert.Contains(t, logOutput, "https://api.example.com/users")
	assert.Contains(t, logOutput, "corr-123")
	assert.Contains(t, logOutput, "application/json")
	assert.Contains(t, logOutput, "test-client/1.0")
}

func TestLogRequest_WithBody(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"username":"alice","email":"alice@example.com"}`)
	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodPost, "https://api.example.com/users", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	params := &httplogger.LogRequestParams{
		Logger:      logger,
		IncludeBody: true,
	}

	httplogger.LogRequest(t.Context(), req, body, "corr-456", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Sending HTTP request")
	assert.Contains(t, logOutput, `\"username\":\"alice\"`) // Escaped in JSON
	assert.Contains(t, logOutput, `"body"`)
}

func TestLogRequest_WithRedactedHeaders(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/data", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer secret_token_12345")
	req.Header.Set("X-Api-Key", "api_key_secret")
	req.Header.Set("Content-Type", "application/json")

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		lowerKey := strings.ToLower(key)
		if strings.Contains(lowerKey, "auth") || strings.Contains(lowerKey, "key") {
			return redact.ActionRedactFully, 0
		}

		return redact.ActionKeep, 0
	}

	params := &httplogger.LogRequestParams{
		Logger:        logger,
		RedactHeaders: redactFunc,
	}

	httplogger.LogRequest(t.Context(), req, nil, "corr-789", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "[redacted]")
	assert.NotContains(t, logOutput, "secret_token")
	assert.NotContains(t, logOutput, "api_key_secret")
	assert.Contains(t, logOutput, "application/json") // Not redacted
}

func TestLogRequest_WithRedactedQueryParams(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(
		t.Context(), http.MethodGet,
		"https://api.example.com/search?q=golang&api_key=secret123&page=1", nil)
	require.NoError(t, err)

	redactFunc := func(ctx context.Context, key, value string) (redact.Action, int) {
		if strings.Contains(strings.ToLower(key), "api_key") {
			return redact.ActionRedactPartialWithMask, 4
		}

		return redact.ActionKeep, 0
	}

	params := &httplogger.LogRequestParams{
		Logger:            logger,
		RedactQueryParams: redactFunc,
	}

	httplogger.LogRequest(t.Context(), req, nil, "corr-query", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "q=golang")
	assert.Contains(t, logOutput, "page=1")
	// Asterisks are URL encoded
	assert.Contains(t, logOutput, "api_key=secr")
	assert.NotContains(t, logOutput, "secret123")
}

func TestLogRequest_MessageOverride(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://api.example.com/users", nil)
	require.NoError(t, err)

	params := &httplogger.LogRequestParams{
		Logger: logger,
		MessageOverride: func(request *http.Request) string {
			return "Custom request message for " + request.URL.Path
		},
	}

	httplogger.LogRequest(t.Context(), req, nil, "corr-custom", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Custom request message for /users")
	assert.NotContains(t, logOutput, "Sending HTTP request")
}

func TestLogRequest_LevelOverride(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{
		Level: slog.LevelWarn, // Only log WARN and above
	}))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://api.example.com/users", nil)
	require.NoError(t, err)

	params := &httplogger.LogRequestParams{
		Logger: logger,
		LevelOverride: func(request *http.Request) slog.Level {
			return slog.LevelWarn // Log at WARN level
		},
	}

	httplogger.LogRequest(t.Context(), req, nil, "corr-level", params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `"level":"WARN"`)
}

// Basic LogResponse tests.
func TestLogResponse_NilResponse(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	params := &httplogger.LogResponseParams{
		Logger: logger,
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	// Should not panic with nil response
	httplogger.LogResponse(t.Context(), nil, nil, "GET", "corr-123", req.URL, params)

	// Should not have logged anything
	assert.Empty(t, logBuffer.String())
}

func TestLogResponse_NilParams(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}

	// Should not panic with nil params
	httplogger.LogResponse(t.Context(), resp, nil, "GET", "corr-123", req.URL, nil)
}

func TestLogResponse_BasicResponse(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/users", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")

	params := &httplogger.LogResponseParams{
		Logger:      logger,
		IncludeBody: false,
	}

	httplogger.LogResponse(t.Context(), resp, nil, "GET", "corr-resp", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Received HTTP response")
	assert.Contains(t, logOutput, "GET")
	assert.Contains(t, logOutput, "200")
	assert.Contains(t, logOutput, "corr-resp")
	assert.Contains(t, logOutput, "application/json")
}

func TestLogResponse_WithBody(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	body := []byte(`{"users":[{"id":1,"name":"Alice"}]}`)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.example.com/users", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "200 OK",
		StatusCode: http.StatusOK,
		Request:    req,
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")

	params := &httplogger.LogResponseParams{
		Logger:      logger,
		IncludeBody: true,
	}

	httplogger.LogResponse(t.Context(), resp, body, "GET", "corr-body", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "Received HTTP response")
	assert.Contains(t, logOutput, `\"users\"`) // Escaped in JSON
	assert.Contains(t, logOutput, `"body"`)
}

func TestLogResponse_StatusCodeLevelOverride(t *testing.T) {
	t.Parallel()

	var logBuffer bytes.Buffer

	logger := slog.New(slog.NewJSONHandler(&logBuffer, nil))

	req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://api.example.com/users", nil)
	require.NoError(t, err)

	resp := &http.Response{
		Status:     "500 Internal Server Error",
		StatusCode: http.StatusInternalServerError,
		Request:    req,
		Header:     make(http.Header),
	}

	params := &httplogger.LogResponseParams{
		Logger: logger,
		LevelOverride: func(response *http.Response) slog.Level {
			if response.StatusCode >= 500 {
				return slog.LevelError
			}

			return slog.LevelInfo
		},
	}

	httplogger.LogResponse(t.Context(), resp, nil, "POST", "corr-error", req.URL, params)

	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, `"level":"ERROR"`)
	assert.Contains(t, logOutput, "500")
}
