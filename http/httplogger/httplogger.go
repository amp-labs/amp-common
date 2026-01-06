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

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/http/printable"
	"github.com/amp-labs/amp-common/http/redact"
	"github.com/amp-labs/amp-common/logger"
)

type contextKey string

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

	// contextKeyArchive is the context key for marking HTTP logs for archival.
	// This is used internally to tag logs that should be archived for long-term storage.
	contextKeyArchive contextKey = "archive"
)

// WithArchive returns a new context with the archive flag set to the specified value.
// When archive is true, HTTP logs will include an "archive": "1" field to indicate
// they should be preserved for long-term storage or analysis.
//
// This is typically used for important requests that need to be auditable or
// debuggable after the standard log retention period.
//
// Example:
//
//	ctx := httplogger.WithArchive(context.Background(), true)
//	// Logs made with this context will be marked for archival
func WithArchive(ctx context.Context, archive bool) context.Context {
	return contexts.WithValue[contextKey, bool](ctx, contextKeyArchive, archive)
}

// SetArchive configures the archive flag using a callback setter function.
// This is used with lazy value overrides to set the archive flag without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms (e.g., lazy.SetValueOverride) to store the value for later retrieval.
//
// Parameters:
//   - archive: Whether to mark logs for archival storage
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
//
// This function is typically used in conjunction with lazy value override systems
// where context values need to be configured before a context is created.
func SetArchive(archive bool, set func(any, any)) {
	if set == nil {
		return
	}

	set(contextKeyArchive, archive)
}

// shouldArchive checks if the context has been marked for log archival.
// Returns true if the archive flag is set and true, false otherwise.
func shouldArchive(ctx context.Context) bool {
	value, found := contexts.GetValue[contextKey, bool](ctx, contextKeyArchive)

	return found && value
}

// LogRequestParams configures how HTTP requests are logged.
type LogRequestParams struct {
	// Logger is the slog.Logger instance to use for logging.
	Logger *slog.Logger

	// GetLogger is an optional function to dynamically obtain a logger from a context.
	// If nil, the Logger field is used directly. If GetLogger returns nil,
	// Logger is used as fallback.
	GetLogger func(ctx context.Context) *slog.Logger

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

	// RedactBody is an optional function to redact the request body.
	RedactBody redact.BodyFunc

	// IncludeBody determines whether to include the request body in logs.
	// Set to false for requests with sensitive or large bodies.
	IncludeBody bool

	// IncludeBodyOverride is an optional function that dynamically determines whether to include
	// the request body in logs. If non-nil, this function is called instead of using the IncludeBody field.
	// This allows for conditional body logging based on context, request properties, or body content.
	// Example use cases: skip logging bodies over a certain size, exclude specific endpoints, etc.
	IncludeBodyOverride func(ctx context.Context, request *http.Request, body []byte) bool

	// TransformBody is an optional function to transform the payload before logging.
	// This can be used to format, redact, or modify the body content.
	// If the function returns nil, the original payload is used.
	TransformBody func(ctx context.Context, payload *printable.Payload) *printable.Payload

	// BodyTruncationLength sets the maximum body size in bytes.
	// Bodies larger than this will be truncated. If <= 0, uses DefaultTruncationLength.
	BodyTruncationLength int64
}

// getLogger returns the slog.Logger to use for logging.
// Priority: GetLogger(ctx) > Logger > slog.Default().
func (p *LogRequestParams) getLogger(ctx context.Context) *slog.Logger {
	if p == nil {
		return logger.Get(ctx)
	}

	if p.GetLogger != nil {
		log := p.GetLogger(ctx)
		if log != nil {
			return log
		}
	}

	if p.Logger != nil {
		return p.Logger
	}

	return logger.Get(ctx)
}

// getHeaders returns the request headers, applying redaction if configured.
func (p *LogRequestParams) getHeaders(ctx context.Context, req *http.Request) http.Header {
	if p.RedactHeaders != nil {
		return redact.Headers(ctx, req.Header, p.RedactHeaders)
	}

	return req.Header
}

// shouldIncludeBody determines whether the request body should be included in logs.
// Priority: IncludeBodyOverride function > IncludeBody boolean field.
func (p *LogRequestParams) shouldIncludeBody(ctx context.Context, req *http.Request, body []byte) bool {
	if p.IncludeBodyOverride != nil {
		return p.IncludeBodyOverride(ctx, req, body)
	}

	return p.IncludeBody
}

// getBody returns the request body as a printable payload, applying transformation and truncation.
// Returns (payload, true) if successful, (nil/payload, false) if body should not be logged.
func (p *LogRequestParams) getBody(ctx context.Context, req *http.Request, body []byte) (*printable.Payload, bool) {
	if !p.shouldIncludeBody(ctx, req, body) {
		return nil, false
	}

	payload, err := printable.Request(req, body)
	if err != nil {
		p.getLogger(ctx).Error("Error creating printable request", "error", err)

		return nil, false
	}

	return processPayload(ctx, payload, p.getLogger(ctx), p.TransformBody, p.RedactBody, p.BodyTruncationLength)
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
// If the context is marked for archival (via WithArchive), an "archive": "1" field
// is added to indicate the log should be preserved for long-term storage.
func (p *LogRequestParams) log(ctx context.Context, request *http.Request, details map[string]any) {
	if shouldArchive(ctx) {
		p.getLogger(ctx).Log(ctx, p.getLevel(request),
			p.getLogMessage(request), "details", details, "archive", "1")
	} else {
		p.getLogger(ctx).Log(ctx, p.getLevel(request),
			p.getLogMessage(request), "details", details)
	}
}

// LogResponseParams configures how HTTP responses are logged.
type LogResponseParams struct {
	// Logger is the slog.Logger instance to use for logging.
	Logger *slog.Logger

	// GetLogger is an optional function to dynamically obtain a logger from a context.
	// If nil, the Logger field is used directly. If GetLogger returns nil,
	// Logger is used as fallback.
	GetLogger func(ctx context.Context) *slog.Logger

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

	// RedactBody is an optional function to redact the response body.
	RedactBody redact.BodyFunc

	// IncludeBody determines whether to include the response body in logs.
	// Set to false for responses with sensitive or large bodies.
	IncludeBody bool

	// IncludeBodyOverride is an optional function that dynamically determines whether to include
	// the response body in logs. If non-nil, this function is called instead of using the IncludeBody field.
	// This allows for conditional body logging based on context, response properties, or body content.
	// Example use cases: skip logging bodies over a certain size, exclude specific status codes, etc.
	IncludeBodyOverride func(ctx context.Context, response *http.Response, body []byte) bool

	// TransformBody is an optional function to transform the payload before logging.
	// This can be used to format, redact, or modify the body content.
	// If the function returns nil, the original payload is used.
	TransformBody func(ctx context.Context, payload *printable.Payload) *printable.Payload

	// BodyTruncationLength sets the maximum body size in bytes.
	// Bodies larger than this will be truncated. If <= 0, uses DefaultTruncationLength.
	BodyTruncationLength int64
}

// getLogger returns the slog.Logger to use for logging.
// Priority: GetLogger(ctx) > Logger > slog.Default().
func (p *LogResponseParams) getLogger(ctx context.Context) *slog.Logger {
	if p == nil {
		return logger.Get(ctx)
	}

	if p.GetLogger != nil {
		log := p.GetLogger(ctx)
		if log != nil {
			return log
		}
	}

	if p.Logger != nil {
		return p.Logger
	}

	return logger.Get(ctx)
}

// getHeaders returns the response headers, applying redaction if configured.
func (p *LogResponseParams) getHeaders(ctx context.Context, resp *http.Response) http.Header {
	if p.RedactHeaders != nil {
		return redact.Headers(ctx, resp.Header, p.RedactHeaders)
	}

	return resp.Header
}

// shouldIncludeBody determines whether the response body should be included in logs.
// Priority: IncludeBodyOverride function > IncludeBody boolean field.
func (p *LogResponseParams) shouldIncludeBody(ctx context.Context, resp *http.Response, body []byte) bool {
	if p.IncludeBodyOverride != nil {
		return p.IncludeBodyOverride(ctx, resp, body)
	}

	return p.IncludeBody
}

// getBody returns the response body as a printable payload, applying transformation and truncation.
// Returns (payload, true) if successful, (nil/payload, false) if body should not be logged.
func (p *LogResponseParams) getBody(ctx context.Context, resp *http.Response, body []byte) (*printable.Payload, bool) {
	if !p.shouldIncludeBody(ctx, resp, body) {
		return nil, false
	}

	payload, err := printable.Response(resp, body)
	if err != nil {
		p.getLogger(ctx).Error("Error creating printable response", "error", err)

		return nil, false
	}

	return processPayload(ctx, payload, p.getLogger(ctx), p.TransformBody, p.RedactBody, p.BodyTruncationLength)
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
// If the context is marked for archival (via WithArchive), an "archive": "1" field
// is added to indicate the log should be preserved for long-term storage.
func (p *LogResponseParams) log(ctx context.Context, resp *http.Response, details map[string]any) {
	if shouldArchive(ctx) {
		p.getLogger(ctx).Log(ctx, p.getLevel(resp),
			p.getLogMessage(resp), "details", details, "archive", "1")
	} else {
		p.getLogger(ctx).Log(ctx, p.getLevel(resp),
			p.getLogMessage(resp), "details", details)
	}
}

// LogErrorParams configures how HTTP errors are logged.
// This is used when http.RoundTripper.RoundTrip returns an error instead of a response.
type LogErrorParams struct {
	// Logger is the slog.Logger instance to use for logging.
	Logger *slog.Logger

	// GetLogger is an optional function to dynamically obtain a logger from a context.
	// If nil, the Logger field is used directly. If GetLogger returns nil,
	// Logger is used as fallback.
	GetLogger func(ctx context.Context) *slog.Logger

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

// getLogger returns the slog.Logger to use for logging.
// Priority: GetLogger(ctx) > Logger > slog.Default().
func (p *LogErrorParams) getLogger(ctx context.Context) *slog.Logger {
	if p == nil {
		return logger.Get(ctx)
	}

	if p.GetLogger != nil {
		log := p.GetLogger(ctx)
		if log != nil {
			return log
		}
	}

	if p.Logger != nil {
		return p.Logger
	}

	return logger.Get(ctx)
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
// If the context is marked for archival (via WithArchive), an "archive": "1" field
// is added to indicate the log should be preserved for long-term storage.
func (p *LogErrorParams) log(ctx context.Context, err error, details map[string]any) {
	if shouldArchive(ctx) {
		p.getLogger(ctx).Log(ctx, p.getLevel(err),
			p.getLogMessage(err), "details", details, "archive", "1")
	} else {
		p.getLogger(ctx).Log(ctx, p.getLevel(err),
			p.getLogMessage(err), "details", details)
	}
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
//   - ctx: The context for obtaining a logger
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
func LogRequest(
	ctx context.Context,
	request *http.Request,
	optionalBody []byte,
	correlationID string,
	params *LogRequestParams,
) {
	if request == nil || params == nil {
		return
	}

	u := cloneURL(request.URL)

	if params.RedactQueryParams != nil {
		values := redact.URLValues(ctx, request.URL.Query(), params.RedactQueryParams)

		u.RawQuery = values.Encode()
	}

	details := map[string]any{
		"method":        request.Method,
		"url":           u.String(),
		"correlationId": correlationID,
		"headers":       params.getHeaders(ctx, request),
	}

	body, _ := params.getBody(ctx, request, optionalBody)
	if body != nil {
		details["body"] = body
	}

	params.log(ctx, request, details)
}

// LogResponse logs an HTTP response with optional body content.
// It applies configured redaction to headers and query parameters from the original request URL,
// and includes optional body content if configured. Nil checks are performed to prevent panics.
//
// Parameters:
//   - ctx: The context for obtaining a logger
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
	ctx context.Context,
	response *http.Response,
	optionalBody []byte,
	requestMethod, correlationID string,
	requestURL *url.URL,
	params *LogResponseParams,
) {
	if response == nil || params == nil {
		return
	}

	u := cloneURL(requestURL)

	if params.RedactQueryParams != nil {
		values := redact.URLValues(ctx, requestURL.Query(), params.RedactQueryParams)

		u.RawQuery = values.Encode()
	}

	details := map[string]any{
		"method":        requestMethod,
		"url":           u.String(),
		"correlationId": correlationID,
		"headers":       params.getHeaders(ctx, response),
		"status":        response.Status,
		"statusCode":    response.StatusCode,
	}

	body, _ := params.getBody(ctx, response, optionalBody)
	if body != nil {
		details["body"] = body
	}

	params.log(ctx, response, details)
}

// LogError logs an HTTP request error that occurred during RoundTrip.
// This should be called when http.RoundTripper.RoundTrip returns an error instead of a response.
// It logs the request context (method, URL, correlation ID) along with the error details.
//
// Parameters:
//   - ctx: The context for obtaining a logger
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
	ctx context.Context,
	request *http.Request, err error,
	requestMethod, correlationID string,
	requestURL *url.URL,
	params *LogErrorParams,
) {
	if request == nil || params == nil {
		return
	}

	var urlString string

	if requestURL != nil {
		u := cloneURL(requestURL)

		if params.RedactQueryParams != nil {
			values := redact.URLValues(ctx, requestURL.Query(), params.RedactQueryParams)

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

	params.log(ctx, err, details)
}

// processPayload applies transformation, redaction, and truncation to a payload.
// Returns (payload, true) if successful, (nil/payload, false) if processing indicates body should not be logged.
func processPayload(
	ctx context.Context,
	payload *printable.Payload,
	logger *slog.Logger,
	transformBody func(ctx context.Context, payload *printable.Payload) *printable.Payload,
	redactBody redact.BodyFunc,
	truncationLength int64,
) (*printable.Payload, bool) {
	if transformBody != nil {
		transformed := transformBody(ctx, payload)
		if transformed != nil {
			payload = transformed
		}
	}

	if payload == nil {
		return nil, false
	}

	if redactBody != nil {
		redactedPayload, err := redact.Body(ctx, payload, redactBody)
		if err != nil {
			logger.Error("Error redacting body", "error", err)
		} else {
			payload = redactedPayload
		}
	}

	if payload == nil {
		return nil, false
	}

	if truncationLength <= 0 {
		truncationLength = DefaultTruncationLength
	}

	truncatedBody, err := payload.Truncate(truncationLength)
	if err != nil {
		logger.Error("Error truncating payload", "error", err)

		return payload, true
	}

	return truncatedBody, true
}
