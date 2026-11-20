package ratelimit

import (
	"fmt"
	"time"
)

// ThrottledError is returned when a request is throttled and must wait before retrying.
type ThrottledError struct {
	Reason    string // Optional reason for throttling
	Delay     time.Duration
	Err       error // Optional, if something other than a 429 caused this
	Retryable *bool // Optional, only set if you have a specific reason to prefer one or the other
}

// Ensure that ThrottledError adheres to the error interface.
var _ error = (*ThrottledError)(nil)

// IsThrottled returns true if there is a non-zero delay.
func (e *ThrottledError) IsThrottled() bool {
	return e.Delay > 0
}

// GetReason returns the throttling reason.
func (e *ThrottledError) GetReason() string {
	return e.Reason
}

// Unwrap returns the underlying error, if there is one.
func (e *ThrottledError) Unwrap() error {
	return e.Err
}

// Error returns the error message including throttling reason and delay.
func (e *ThrottledError) Error() string {
	if e.Err != nil {
		if e.Reason != "" {
			return fmt.Sprintf("%s: throttled (%s), try again in %s", e.Err.Error(), e.Reason, e.Delay)
		}

		return fmt.Sprintf("%s: throttled, try again in %s", e.Err.Error(), e.Delay)
	}

	if e.Reason != "" {
		return fmt.Sprintf("throttled (%s), try again in %s", e.Reason, e.Delay)
	}

	return fmt.Sprintf("throttled, try again in %s", e.Delay)
}

// IsRetryable returns true since throttled requests can be retried after the delay.
func (e *ThrottledError) IsRetryable() bool {
	if e == nil {
		return true
	}

	if e.Retryable == nil {
		return true
	}

	return *e.Retryable
}

// GetDelay returns the duration to wait before retrying.
func (e *ThrottledError) GetDelay() time.Duration {
	return e.Delay
}
