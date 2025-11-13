package retry

import (
	"math"
	"time"
)

// Backoff is an interface for calculating the delay between retry attempts.
// Different backoff strategies can be implemented to control retry behavior.
type Backoff interface {
	// Delay calculates the duration to wait before the next retry attempt.
	// The attempt parameter is zero-indexed (0 for first retry).
	Delay(attempt uint) time.Duration
}

// ExpBackoff implements exponential backoff with configurable parameters.
// The delay grows exponentially with each attempt: Base * Factor^attempt.
// The delay is capped at Max to prevent excessive wait times.
//
// Example:
//
//	backoff := retry.ExpBackoff{
//	    Base:   100 * time.Millisecond,  // Start with 100ms
//	    Max:    10 * time.Second,         // Cap at 10s
//	    Factor: 2.0,                      // Double each time
//	}
//	// Delays: 100ms, 200ms, 400ms, 800ms, 1.6s, 3.2s, 6.4s, 10s, 10s, ...
type ExpBackoff struct {
	// Base is the initial delay duration.
	Base time.Duration
	// Max is the maximum delay duration (cap).
	Max time.Duration
	// Factor is the multiplier applied to each successive delay (e.g., 2.0 for doubling).
	Factor float64
}

// Delay calculates the exponential backoff delay for the given attempt.
// The formula is: Base * Factor^attempt, clamped between Base and Max.
func (b ExpBackoff) Delay(attempt uint) time.Duration {
	f := float64(b.Base) * math.Pow(b.Factor, float64(attempt))

	d := time.Duration(f)
	if d < b.Base {
		return b.Base
	} else if d > b.Max {
		return b.Max
	}

	return d
}
