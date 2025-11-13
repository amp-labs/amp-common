package retry

import "context"

// Attempts represents the maximum number of attempts to make for an operation.
// A value of 0 means unlimited retries (use with caution).
type Attempts uint

// ctxKey is the type for context keys used internally to avoid collisions.
type ctxKey string

// attemptKey is the context key used to store and retrieve the current attempt number.
const attemptKey ctxKey = "attempt"

// withAttempt adds the attempt number to the context. This allows the operation
// being retried to know which attempt it is on.
func withAttempt(ctx context.Context, attempt uint) context.Context {
	return context.WithValue(ctx, attemptKey, attempt)
}

// Attempt retrieves the current attempt number from the context.
// Returns 0 if no attempt number is stored in the context.
//
// Example:
//
//	err := retry.Do(ctx, func(ctx context.Context) error {
//	    attemptNum := retry.Attempt(ctx)
//	    log.Printf("Attempt %d", attemptNum)
//	    return makeAPICall()
//	})
func Attempt(ctx context.Context) uint {
	i := ctx.Value(attemptKey)
	if i == nil {
		return 0
	}

	attemptNum, ok := i.(uint)
	if !ok {
		return 0
	}

	return attemptNum
}
