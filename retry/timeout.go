package retry

import "time"

// Timeout represents the maximum duration for a single retry attempt.
// If an attempt takes longer than this duration, it will be canceled
// and counted as a failure, triggering a retry.
//
// A zero Timeout means no timeout - attempts can run indefinitely.
//
// Example:
//
//	runner := retry.NewRunner(
//	    retry.WithTimeout(retry.Timeout(30 * time.Second)),
//	)
type Timeout time.Duration
