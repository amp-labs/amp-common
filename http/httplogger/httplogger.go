// Package httplogger provides utilities for logging HTTP requests and responses with structured logging.
//
// This package integrates with Go's slog package to provide detailed HTTP logging capabilities.
// It supports both request and response logging with configurable options for redacting sensitive
// information, truncating large bodies, and transforming payloads before logging.
//
// # Basic Usage
//
//	logger := slog.Default()
//	req, _ := http.NewRequest("GET", "https://api.example.com?api_key=secret", nil)
//	req.Header.Set("Authorization", "Bearer token123")
//
//	// Define redaction function for sensitive headers
//	redactFunc := func(key, value string) (redact.Action, int) {
//	    if strings.Contains(strings.ToLower(key), "authorization") {
//	        return redact.ActionPartial, 7 // Show "Bearer " prefix
//	    }
//	    return redact.ActionKeep, 0
//	}
//
//	// Log request with redaction
//	params := &httplogger.LogRequestParams{
//	    Logger:        logger,
//	    RedactHeaders: redactFunc,
//	    IncludeBody:   true,
//	}
//	httplogger.LogRequest(req, nil, "correlation-123", params)
//
// # Features
//
//   - Automatic redaction of sensitive headers and query parameters
//   - Body truncation to prevent excessive log sizes (default 256 KiB)
//   - Optional body transformation for custom formatting
//   - Structured logging with method, URL, correlation ID, headers, and bodies
//   - Support for printable and base64-encoded content
package httplogger

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/amp-labs/amp-common/http/printable"
	"github.com/amp-labs/amp-common/http/redact"
)

const (
	// DefaultTruncationLength defines the maximum size (in bytes) for HTTP request/response bodies in logs.
	// Bodies larger than this will be truncated to prevent excessive log sizes.
	DefaultTruncationLength = 256 * 1024 // 256 KiB

	// DefaultLogRequestMessage is the default log message used when logging HTTP requests.
	// This can be overridden using LogRequestParams.DefaultMessage or LogRequestParams.MessageOverride.
	DefaultLogRequestMessage = "Sending HTTP request"

	// DefaultLogResponseMessage is the default log message used when logging HTTP responses.
	// This can be overridden using LogResponseParams.DefaultMessage or LogResponseParams.MessageOverride.
	DefaultLogResponseMessage = "Received HTTP response"

	// DefaultLogErrorMessage is the default log message used when logging HTTP errors.
	// This can be overridden using LogErrorParams.DefaultMessage or LogErrorParams.MessageOverride.
	DefaultLogErrorMessage = "HTTP request failed"
)

// LogRequestParams configures how HTTP requests are logged.
type LogRequestParams struct {
	// Logger is the slog.Logger instance to use for logging.
	Logger *slog.Logger

	// DefaultLevel is the default log level to use for requests.
	// If not set, defaults to slog.LevelDebug. Can be overridden per-request using LevelOverride.
	DefaultLevel slog.Level

	// LevelOverride is an optional function that allows dynamic log level selection based on the request.
	// This is useful for logging certain requests at different levels (e.g., health checks at DEBUG, errors at WARN).
	// If nil or returns zero value, DefaultLevel is used.
	LevelOverride func(request *http.Request) slog.Level

	// DefaultMessage is the default log message to use for requests.
	// If empty, DefaultLogRequestMessage is used. Can be overridden per-request using MessageOverride.
	DefaultMessage string

	// MessageOverride is an optional function that allows dynamic message selection based on the request.
	// This is useful for customizing log messages per endpoint (e.g., "API request to /users" vs "Health check").
	// If nil or returns empty string, DefaultMessage or DefaultLogRequestMessage is used.
	MessageOverride func(request *http.Request) string

	// RedactHeaders is an optional function to redact sensitive header values.
	// If nil, headers are logged as-is without redaction.
	RedactHeaders redact.Func

	// RedactQueryParams is an optional function to redact sensitive query parameters.
	// If nil, query parameters are logged as-is without redaction.
	RedactQueryParams redact.Func

	// IncludeBody determines whether to include the request body in logs.
	// Set to false for requests with sensitive or large bodies.
	IncludeBody bool

	// TransformBody is an optional function to transform the payload before logging.
	// This can be used to format, redact, or modify the body content.
	// If the function returns nil, the original payload is used.
	TransformBody func(payload *printable.Payload) *printable.Payload

	// BodyTruncationLength sets the maximum body size in bytes.
	// Bodies larger than this will be truncated. If <= 0, uses DefaultTruncationLength.
	BodyTruncationLength int64
}

// getHeaders returns the request headers, applying redaction if configured.
func (p *LogRequestParams) getHeaders(req *http.Request) http.Header {
	if p.RedactHeaders != nil {
		return redact.Headers(req.Header, p.RedactHeaders)
	}

	return req.Header
}

// getBody returns the request body as a printable payload, applying transformation and truncation.
// Returns (payload, true) if successful, (nil/payload, false) if body should not be logged.
func (p *LogRequestParams) getBody(req *http.Request, body []byte) (*printable.Payload, bool) {
	if !p.IncludeBody {
		return nil, false
	}

	payload, err := printable.Request(req, body)
	if err != nil {
		p.Logger.Error("Error creating printable request", "error", err)

		return nil, false
	}

	if p.TransformBody != nil {
		transformed := p.TransformBody(payload)
		if transformed != nil {
			payload = transformed
		}
	}

	truncationLength := p.BodyTruncationLength
	if truncationLength <= 0 {
		truncationLength = DefaultTruncationLength
	}

	truncatedBody, err := payload.Truncate(truncationLength)
	if err != nil {
		p.Logger.Error("Error truncating payload", "error", err)

		return payload, true
	}

	return truncatedBody, true
}

// Priority: LevelOverride > DefaultLevel > slog.LevelDebug.
func (p *LogRequestParams) getLevel(req *http.Request) slog.Level {
	if p == nil {
		return slog.LevelDebug
	}

	if req == nil || p.LevelOverride == nil {
		return p.DefaultLevel
	}

	return p.LevelOverride(req)
}

// Priority: MessageOverride (if returns non-empty) > DefaultMessage > DefaultLogRequestMessage.
func (p *LogRequestParams) getLogMessage(request *http.Request) string {
	if p == nil {
		return DefaultLogRequestMessage
	}

	if p.MessageOverride == nil {
		if p.DefaultMessage == "" {
			return DefaultLogRequestMessage
		}

		return p.DefaultMessage
	}

	msg := p.MessageOverride(request)
	if len(msg) > 0 {
		return msg
	}

	if p.DefaultMessage == "" {
		return DefaultLogRequestMessage
	}

	return p.DefaultMessage
}

// log writes the log entry using the configured logger, level, and message.
func (p *LogRequestParams) log(request *http.Request, details map[string]any) {
	p.Logger.Log(context.Background(), p.getLevel(request),
		p.getLogMessage(request), "details", details)
}

// LogResponseParams configures how HTTP responses are logged.
type LogResponseParams struct {
	// Logger is the slog.Logger instance to use for logging.
	Logger *slog.Logger

	// DefaultLevel is the default log level to use for responses.
	// If not set, defaults to slog.LevelDebug. Can be overridden per-response using LevelOverride.
	DefaultLevel slog.Level

	// LevelOverride is an optional function that allows dynamic log level selection based on the response.
	// This is useful for logging responses at different levels based on status code (e.g., 5xx at ERROR, 2xx at DEBUG).
	// If nil or returns zero value, DefaultLevel is used.
	LevelOverride func(response *http.Response) slog.Level

	// DefaultMessage is the default log message to use for responses.
	// If empty, DefaultLogResponseMessage is used. Can be overridden per-response using MessageOverride.
	DefaultMessage string

	// MessageOverride is an optional function that allows dynamic message selection based on the response.
	// This is useful for customizing log messages based on status code or other response properties.
	// If nil or returns empty string, DefaultMessage or DefaultLogResponseMessage is used.
	MessageOverride func(response *http.Response) string

	// RedactHeaders is an optional function to redact sensitive header values.
	// If nil, headers are logged as-is without redaction.
	RedactHeaders redact.Func

	// RedactQueryParams is an optional function to redact sensitive query parameters from the request URL.
	// If nil, query parameters are logged as-is without redaction.
	RedactQueryParams redact.Func

	// IncludeBody determines whether to include the response body in logs.
	// Set to false for responses with sensitive or large bodies.
	IncludeBody bool

	// TransformBody is an optional function to transform the payload before logging.
	// This can be used to format, redact, or modify the body content.
	// If the function returns nil, the original payload is used.
	TransformBody func(payload *printable.Payload) *printable.Payload

	// BodyTruncationLength sets the maximum body size in bytes.
	// Bodies larger than this will be truncated. If <= 0, uses DefaultTruncationLength.
	BodyTruncationLength int64
}

// getHeaders returns the response headers, applying redaction if configured.
func (p *LogResponseParams) getHeaders(resp *http.Response) http.Header {
	if p.RedactHeaders != nil {
		return redact.Headers(resp.Header, p.RedactHeaders)
	}

	return resp.Header
}

// getBody returns the response body as a printable payload, applying transformation and truncation.
// Returns (payload, true) if successful, (nil/payload, false) if body should not be logged.
func (p *LogResponseParams) getBody(resp *http.Response, body []byte) (*printable.Payload, bool) {
	if !p.IncludeBody {
		return nil, false
	}

	payload, err := printable.Response(resp, body)
	if err != nil {
		p.Logger.Error("Error creating printable response", "error", err)

		return nil, false
	}

	if p.TransformBody != nil {
		transformed := p.TransformBody(payload)
		if transformed != nil {
			payload = transformed
		}
	}

	truncationLength := p.BodyTruncationLength
	if truncationLength <= 0 {
		truncationLength = DefaultTruncationLength
	}

	truncatedBody, err := payload.Truncate(truncationLength)
	if err != nil {
		p.Logger.Error("Error truncating payload", "error", err)

		return payload, true
	}

	return truncatedBody, true
}

// Priority: LevelOverride > DefaultLevel > slog.LevelDebug.
func (p *LogResponseParams) getLevel(resp *http.Response) slog.Level {
	if p == nil {
		return slog.LevelDebug
	}

	if resp == nil || p.LevelOverride == nil {
		return p.DefaultLevel
	}

	return p.LevelOverride(resp)
}

// Priority: MessageOverride (if returns non-empty) > DefaultMessage > DefaultLogResponseMessage.
func (p *LogResponseParams) getLogMessage(resp *http.Response) string {
	if p == nil {
		return DefaultLogResponseMessage
	}

	if p.MessageOverride == nil {
		if p.DefaultMessage == "" {
			return DefaultLogResponseMessage
		}

		return p.DefaultMessage
	}

	msg := p.MessageOverride(resp)
	if len(msg) > 0 {
		return msg
	}

	if p.DefaultMessage == "" {
		return DefaultLogResponseMessage
	}

	return p.DefaultMessage
}

// log writes the log entry using the configured logger, level, and message.
func (p *LogResponseParams) log(resp *http.Response, details map[string]any) {
	p.Logger.Log(context.Background(), p.getLevel(resp),
		p.getLogMessage(resp), "details", details)
}

// LogErrorParams configures how HTTP errors are logged.
// This is used when http.RoundTripper.RoundTrip returns an error instead of a response.
type LogErrorParams struct {
	// Logger is the slog.Logger instance to use for logging.
	Logger *slog.Logger

	// DefaultLevel is the default log level to use for errors.
	// If not set, defaults to slog.LevelDebug (though ERROR is more typical).
	// Can be overridden per-error using LevelOverride.
	DefaultLevel slog.Level

	// LevelOverride is an optional function that allows dynamic log level selection based on the error.
	// This is useful for logging different error types at different levels
	// (e.g., context.Canceled at INFO, others at ERROR).
	// If nil or returns zero value, DefaultLevel is used.
	LevelOverride func(err error) slog.Level

	// DefaultMessage is the default log message to use for errors.
	// If empty, DefaultLogErrorMessage is used. Can be overridden per-error using MessageOverride.
	DefaultMessage string

	// MessageOverride is an optional function that allows dynamic message selection based on the error.
	// This is useful for customizing log messages based on error type
	// (e.g., "Connection timeout" vs "DNS resolution failed").
	// If nil or returns empty string, DefaultMessage or DefaultLogErrorMessage is used.
	MessageOverride func(err error) string

	// RedactQueryParams is an optional function to redact sensitive query parameters from the request URL.
	// If nil, query parameters are logged as-is without redaction.
	RedactQueryParams redact.Func
}

// getLevel determines the log level to use for the error.
// Priority: LevelOverride > DefaultLevel > slog.LevelDebug.
func (p *LogErrorParams) getLevel(err error) slog.Level {
	if p == nil {
		return slog.LevelDebug
	}

	if err == nil || p.LevelOverride == nil {
		return p.DefaultLevel
	}

	return p.LevelOverride(err)
}

// Priority: MessageOverride (if returns non-empty) > DefaultMessage > DefaultLogErrorMessage.
func (p *LogErrorParams) getLogMessage(err error) string {
	if p == nil {
		return DefaultLogErrorMessage
	}

	if p.MessageOverride == nil {
		if p.DefaultMessage == "" {
			return DefaultLogErrorMessage
		}

		return p.DefaultMessage
	}

	msg := p.MessageOverride(err)
	if len(msg) > 0 {
		return msg
	}

	if p.DefaultMessage == "" {
		return DefaultLogErrorMessage
	}

	return p.DefaultMessage
}

// log writes the log entry using the configured logger, level, and message.
func (p *LogErrorParams) log(err error, details map[string]any) {
	p.Logger.Log(context.Background(), p.getLevel(err),
		p.getLogMessage(err), "details", details)
}

// cloneURL creates a shallow copy of a URL.
// This is useful when we need to modify the URL (e.g., redact query params) without affecting the original.
func cloneURL(sourceURL *url.URL) *url.URL {
	if sourceURL == nil {
		return nil
	}

	return &url.URL{
		Scheme:      sourceURL.Scheme,
		Opaque:      sourceURL.Opaque,
		User:        sourceURL.User,
		Host:        sourceURL.Host,
		Path:        sourceURL.Path,
		RawPath:     sourceURL.RawPath,
		OmitHost:    sourceURL.OmitHost,
		ForceQuery:  sourceURL.ForceQuery,
		RawQuery:    sourceURL.RawQuery,
		Fragment:    sourceURL.Fragment,
		RawFragment: sourceURL.RawFragment,
	}
}

// LogRequest logs an HTTP request with optional body content.
// It applies configured redaction to headers and query parameters, and includes
// optional body content if configured. Nil checks are performed to prevent panics.
//
// Parameters:
//   - request: The HTTP request to log (required, returns early if nil)
//   - optionalBody: Optional pre-read body bytes. If nil, body won't be logged unless IncludeBody is false.
//   - correlationID: A correlation ID to track the request across systems
//   - params: Configuration for logging behavior (required, returns early if nil)
//
// Example:
//
//	params := &LogRequestParams{
//	    Logger:        slog.Default(),
//	    IncludeBody:   true,
//	    RedactHeaders: myRedactFunc,
//	}
//	LogRequest(req, bodyBytes, "corr-123", params)
func LogRequest(request *http.Request, optionalBody []byte, correlationID string, params *LogRequestParams) {
	if request == nil || params == nil {
		return
	}

	u := cloneURL(request.URL)

	if params.RedactQueryParams != nil {
		values := redact.URLValues(request.URL.Query(), params.RedactQueryParams)

		u.RawQuery = values.Encode()
	}

	details := map[string]any{
		"method":        request.Method,
		"url":           u.String(),
		"correlationId": correlationID,
		"headers":       params.getHeaders(request),
	}

	body, _ := params.getBody(request, optionalBody)
	if body != nil {
		details["body"] = body
	}

	params.log(request, details)
}

// LogResponse logs an HTTP response with optional body content.
// It applies configured redaction to headers and query parameters from the original request URL,
// and includes optional body content if configured. Nil checks are performed to prevent panics.
//
// Parameters:
//   - response: The HTTP response to log (required, returns early if nil)
//   - optionalBody: Optional pre-read body bytes. If nil, body won't be logged unless IncludeBody is false.
//   - requestMethod: The HTTP method from the original request (GET, POST, etc.)
//   - correlationID: A correlation ID to track the request/response pair across systems
//   - requestURL: The URL from the original request (for logging the full context)
//   - params: Configuration for logging behavior (required, returns early if nil)
//
// Example:
//
//	params := &LogResponseParams{
//	    Logger:        slog.Default(),
//	    IncludeBody:   true,
//	    RedactHeaders: myRedactFunc,
//	}
//	LogResponse(resp, bodyBytes, "GET", "corr-123", req.URL, params)
func LogResponse(
	response *http.Response, optionalBody []byte,
	requestMethod, correlationID string, requestURL *url.URL, params *LogResponseParams,
) {
	if response == nil || params == nil {
		return
	}

	u := cloneURL(requestURL)

	if params.RedactQueryParams != nil {
		values := redact.URLValues(requestURL.Query(), params.RedactQueryParams)

		u.RawQuery = values.Encode()
	}

	details := map[string]any{
		"method":        requestMethod,
		"url":           u.String(),
		"correlationId": correlationID,
		"headers":       params.getHeaders(response),
		"status":        response.Status,
		"statusCode":    response.StatusCode,
	}

	body, _ := params.getBody(response, optionalBody)
	if body != nil {
		details["body"] = body
	}

	params.log(response, details)
}

// LogError logs an HTTP request error that occurred during RoundTrip.
// This should be called when http.RoundTripper.RoundTrip returns an error instead of a response.
// It logs the request context (method, URL, correlation ID) along with the error details.
//
// Parameters:
//   - request: The HTTP request that failed (required, returns early if nil)
//   - err: The error that occurred (required for meaningful logging)
//   - requestMethod: The HTTP method from the request (GET, POST, etc.)
//   - correlationID: A correlation ID to track the request across systems
//   - requestURL: The URL from the request (for logging the full context)
//   - params: Configuration for logging behavior (required, returns early if nil)
//
// Example:
//
//	params := &LogErrorParams{
//	    Logger:            slog.Default(),
//	    RedactQueryParams: myRedactFunc,
//	}
//	resp, err := transport.RoundTrip(req)
//	if err != nil {
//	    LogError(req, err, req.Method, "corr-123", req.URL, params)
//	    return nil, err
//	}
func LogError(
	request *http.Request, err error,
	requestMethod, correlationID string, requestURL *url.URL, params *LogErrorParams,
) {
	if request == nil || params == nil {
		return
	}

	var urlString string

	if requestURL != nil {
		u := cloneURL(requestURL)

		if params.RedactQueryParams != nil {
			values := redact.URLValues(requestURL.Query(), params.RedactQueryParams)

			u.RawQuery = values.Encode()
		}

		urlString = u.String()
	}

	details := map[string]any{
		"method":        requestMethod,
		"url":           urlString,
		"correlationId": correlationID,
	}

	if err != nil {
		details["error"] = err.Error()
	}

	params.log(err, details)
}
