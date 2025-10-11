// Package transport provides HTTP transport utilities including logging middleware.
//
// This package contains implementations of http.RoundTripper that add functionality
// like request/response logging with automatic redaction and error handling.
package transport

import (
	"context"
	"fmt"
	"net/http"

	"github.com/amp-labs/amp-common/http/httplogger"
	"github.com/amp-labs/amp-common/logger"
	"github.com/google/uuid"
)

// NewLoggingTransport creates an http.RoundTripper that logs all HTTP requests, responses, and errors.
// It wraps an existing transport (or http.DefaultTransport if nil) and automatically logs:
//   - Outgoing requests (method, URL, headers, optional body)
//   - Incoming responses (status, headers, optional body)
//   - Errors that occur during the request (network errors, timeouts, etc.)
//
// Each request is assigned a unique correlation ID (UUID v7) that appears in all related log entries,
// making it easy to trace a request through the system.
//
// Parameters:
//   - ctx: Context used to retrieve the logger via logger.Get(ctx)
//   - transport: The underlying http.RoundTripper to wrap (uses http.DefaultTransport if nil)
//   - requestParams: Configuration for request logging (uses default with body logging if nil)
//   - responseParams: Configuration for response logging (uses default with body logging if nil)
//   - errorParams: Configuration for error logging (uses default if nil)
//
// Returns an http.RoundTripper that can be used with http.Client or anywhere an http.RoundTripper is expected.
//
// Example:
//
//	// Create a logging transport with custom redaction
//	redactFunc := func(key, value string) (redact.Action, int) {
//	    if strings.Contains(strings.ToLower(key), "authorization") {
//	        return redact.ActionPartial, 7
//	    }
//	    return redact.ActionKeep, 0
//	}
//
//	requestParams := &httplogger.LogRequestParams{
//	    Logger:        logger.Get(ctx),
//	    RedactHeaders: redactFunc,
//	    IncludeBody:   true,
//	}
//
//	client := &http.Client{
//	    Transport: transport.NewLoggingTransport(ctx, nil, requestParams, nil, nil),
//	}
//
//	// All requests made with this client will be automatically logged
//	resp, err := client.Get("https://api.example.com/data")
func NewLoggingTransport(
	ctx context.Context,
	transport http.RoundTripper,
	requestParams *httplogger.LogRequestParams,
	responseParams *httplogger.LogResponseParams,
	errorParams *httplogger.LogErrorParams,
) http.RoundTripper {
	if transport == nil {
		transport = http.DefaultTransport
	}

	if requestParams == nil {
		requestParams = &httplogger.LogRequestParams{
			Logger:         logger.Get(ctx),
			DefaultMessage: httplogger.DefaultLogRequestMessage,
			IncludeBody:    true,
		}
	}

	if responseParams == nil {
		responseParams = &httplogger.LogResponseParams{
			Logger:         logger.Get(ctx),
			DefaultMessage: httplogger.DefaultLogResponseMessage,
			IncludeBody:    true,
		}
	}

	if errorParams == nil {
		errorParams = &httplogger.LogErrorParams{
			Logger:         logger.Get(ctx),
			DefaultMessage: httplogger.DefaultLogErrorMessage,
		}
	}

	if requestParams.Logger == nil {
		requestParams.Logger = logger.Get(ctx)
	}

	if responseParams.Logger == nil {
		responseParams.Logger = logger.Get(ctx)
	}

	if errorParams.Logger == nil {
		errorParams.Logger = logger.Get(ctx)
	}

	return &loggingTransport{
		requestParams:  requestParams,
		responseParams: responseParams,
		errorParams:    errorParams,
		transport:      transport,
	}
}

// loggingTransport is an http.RoundTripper implementation that logs HTTP requests, responses, and errors.
// It delegates the actual HTTP transport to an underlying http.RoundTripper while capturing and logging
// all relevant information.
type loggingTransport struct {
	// requestParams configures how outgoing requests are logged
	requestParams *httplogger.LogRequestParams

	// responseParams configures how incoming responses are logged
	responseParams *httplogger.LogResponseParams

	// errorParams configures how errors are logged
	errorParams *httplogger.LogErrorParams

	// transport is the underlying http.RoundTripper that performs the actual HTTP request
	transport http.RoundTripper
}

// Compile-time check to ensure loggingTransport implements http.RoundTripper.
var _ http.RoundTripper = (*loggingTransport)(nil)

// RoundTrip implements the http.RoundTripper interface.
// It logs the outgoing request, performs the HTTP round trip, and logs either the response or error.
//
// The method follows this sequence:
//  1. Generate a unique correlation ID (UUID v7) for this request/response pair
//  2. Log the outgoing request with the correlation ID
//  3. Perform the actual HTTP request using the underlying transport
//  4. If an error occurs (network error, timeout, etc.), log the error and return
//  5. If successful, log the response and return
//
// The correlation ID allows tracking a request through logs, making debugging easier.
// All three log entries (request, response/error) will share the same correlation ID.
//
// Error handling:
//   - If UUID generation fails, returns an error without attempting the request
//   - If the underlying transport returns an error, logs it at ERROR level and returns the error
//   - If the underlying transport succeeds, logs the response at DEBUG level
func (l *loggingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	// Generate unique correlation ID for this request/response pair
	uuid7, err := uuid.NewV7()
	if err != nil {
		return nil, fmt.Errorf("error generating UUID: %w", err)
	}

	correlationID := uuid7.String()

	// Log the outgoing request
	httplogger.LogRequest(request, nil, correlationID, l.requestParams)

	// Perform the actual HTTP request
	response, err := l.transport.RoundTrip(request)
	if err != nil {
		// Log the error at ERROR level
		httplogger.LogError(request, err, request.Method, correlationID, request.URL, l.errorParams)

		return response, err
	}

	// Log the successful response at DEBUG level
	httplogger.LogResponse(response, nil, request.Method, correlationID, request.URL, l.responseParams)

	return response, err
}
