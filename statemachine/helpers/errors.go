// Package helpers provides utility functions for state machine operations.
package helpers

import (
	"errors"
	"fmt"
)

// Sentinel errors for common failure modes.
var (
	ErrSamplingUnavailable     = errors.New("sampling capability not available")
	ErrElicitationUnavailable  = errors.New("elicitation capability not available")
	ErrUserDeclinedAction      = errors.New("user declined action")
	ErrValidationFailed        = errors.New("validation failed")
	ErrJSONParseFailed         = errors.New("failed to parse JSON from response")
	ErrCapabilityCheckFailed   = errors.New("capability check failed")
	ErrFallbackExecutionFailed = errors.New("fallback execution failed")
	ErrUnsupportedFallbackType = errors.New("unsupported fallback type")
)

// SamplingError wraps sampling-related errors with additional context.
type SamplingError struct {
	Operation string
	Err       error
	Context   map[string]any
}

func (e *SamplingError) Error() string {
	if len(e.Context) == 0 {
		return fmt.Sprintf("sampling error in %s: %v", e.Operation, e.Err)
	}

	return fmt.Sprintf("sampling error in %s: %v (context: %+v)", e.Operation, e.Err, e.Context)
}

func (e *SamplingError) Unwrap() error {
	return e.Err
}

// ElicitationError wraps elicitation-related errors with additional context.
type ElicitationError struct {
	Operation string
	Err       error
	Context   map[string]any
}

func (e *ElicitationError) Error() string {
	if len(e.Context) == 0 {
		return fmt.Sprintf("elicitation error in %s: %v", e.Operation, e.Err)
	}

	return fmt.Sprintf("elicitation error in %s: %v (context: %+v)", e.Operation, e.Err, e.Context)
}

func (e *ElicitationError) Unwrap() error {
	return e.Err
}

// WrapSamplingError wraps an error with sampling context.
func WrapSamplingError(operation string, err error, context map[string]any) error {
	if err == nil {
		return nil
	}

	return &SamplingError{
		Operation: operation,
		Err:       err,
		Context:   context,
	}
}

// WrapElicitationError wraps an error with elicitation context.
func WrapElicitationError(operation string, err error, context map[string]any) error {
	if err == nil {
		return nil
	}

	return &ElicitationError{
		Operation: operation,
		Err:       err,
		Context:   context,
	}
}

// IsUserDeclined checks if an error represents a user declining an action.
func IsUserDeclined(err error) bool {
	return errors.Is(err, ErrUserDeclinedAction)
}

// IsCapabilityMissing checks if an error is due to missing sampling/elicitation capability.
func IsCapabilityMissing(err error) bool {
	return errors.Is(err, ErrSamplingUnavailable) || errors.Is(err, ErrElicitationUnavailable)
}

// IsSamplingError checks if an error is a sampling error.
func IsSamplingError(err error) bool {
	var samplingErr *SamplingError

	return errors.As(err, &samplingErr)
}

// IsElicitationError checks if an error is an elicitation error.
func IsElicitationError(err error) bool {
	var elicitationErr *ElicitationError

	return errors.As(err, &elicitationErr)
}

// HandleGracefulDegradation handles an error with graceful fallback
// If the error is capability-related or user decline, returns the fallback value
// Otherwise, returns the error.
func HandleGracefulDegradation(err error, fallback any) (any, error) {
	if err == nil {
		return nil, nil
	}

	// If capability is missing or user declined, use fallback
	if IsCapabilityMissing(err) || IsUserDeclined(err) {
		return fallback, nil
	}

	// For other errors, propagate them
	return nil, err
}
