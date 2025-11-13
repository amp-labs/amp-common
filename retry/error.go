package retry

import "errors"

// ErrExhausted is returned when the retry budget is exhausted and no more
// retry attempts are allowed. This prevents cascading failures by limiting
// retries under high load.
var ErrExhausted = errors.New("retry budget exhausted")

// Error is an interface for errors that can indicate whether they are temporary
// (retryable) or permanent (non-retryable). Operations can return errors that
// implement this interface to control retry behavior.
//
// If an error implements this interface and Temporary() returns false, the retry
// loop will stop immediately and return the error without further attempts.
type Error interface {
	// Temporary returns true if the error is temporary and the operation should be retried.
	// Returns false if the error is permanent and retries should stop.
	Temporary() bool
	error
}

// permanentError wraps an error to mark it as permanent (non-retryable).
// This is used internally by the Abort function.
type permanentError struct {
	error
}

// Temporary returns false to indicate this error should not be retried.
func (e *permanentError) Temporary() bool { return false }

// Unwrap returns the underlying error for error chain unwrapping.
func (e *permanentError) Unwrap() error {
	return e.error
}

// Abort wraps an error to mark it as permanent, causing the retry loop to stop
// immediately without further attempts. Use this when you know an error is not
// transient and retrying would not help.
//
// Example:
//
//	if err := validateInput(data); err != nil {
//	    return retry.Abort(err)  // Don't retry validation errors
//	}
//	if err := makeAPICall(data); err != nil {
//	    return err  // Do retry API errors
//	}
func Abort(err error) Error {
	return &permanentError{err}
}
