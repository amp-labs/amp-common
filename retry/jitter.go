package retry

import (
	"math/rand"
	"time"
)

// Jitter represents a jitter strategy for retry delays. Jitter adds randomness
// to backoff delays to prevent the "thundering herd" problem where many clients
// retry at the same time, overwhelming the server.
//
// The value represents the amount of randomness:
//   - 0.0: No jitter (deterministic delays)
//   - 0.5: Equal jitter (50% random, 50% deterministic)
//   - 1.0: Full jitter (completely random between 0 and delay)
//   - Negative values: Disable jitter (use exact delay)
type Jitter float64

// EqualJitter provides a balanced jitter strategy where the delay is 50% random
// and 50% deterministic. This gives a good balance between avoiding thundering
// herd and maintaining predictable retry timing.
//
// Formula: delay/2 + random(0, delay/2).
const EqualJitter Jitter = 0.5

// FullJitter provides maximum randomness where the delay is completely random
// between 0 and the calculated delay. This provides the best protection against
// thundering herd but makes retry timing unpredictable.
//
// Formula: random(0, delay).
const FullJitter Jitter = 1.0

// WithoutJitter disables jitter entirely, using the exact calculated delay.
// This is useful for testing or when deterministic retry timing is required.
//
// Formula: delay (no randomness).
const WithoutJitter Jitter = -1.0

// jitter applies the jitter strategy to the given delay duration.
// Returns a randomized delay based on the jitter value:
//   - Negative jitter: Returns delay unchanged
//   - Zero jitter: Returns delay unchanged
//   - Full jitter (1.0): Returns random value between 0 and delay
//   - Partial jitter (0.0-1.0): Returns weighted average of random and delay
func (j Jitter) jitter(d time.Duration) time.Duration {
	// Disable jitter for negative values
	if j < 0.0 {
		return d
	}

	// Generate random value between 0 and delay
	//nolint:gosec // G404: math/rand is sufficient for jitter; crypto/rand is unnecessary overhead
	r := rand.Float64() * float64(d)

	// For partial jitter, blend random value with original delay
	// Formula: jitter * random + (1 - jitter) * delay
	if j > 0.0 && j < 1.0 {
		r = float64(j)*r + float64(1.0-j)*float64(d)
	}

	return time.Duration(r)
}
