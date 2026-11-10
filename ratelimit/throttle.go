package ratelimit

import "time"

// ThrottleParams includes any details returned from the service about a rate limited call:
// why it was throttled, and how long to wait if the service suggested a delay.
// Pass it to Reservation.Done along with the Throttled status.
type ThrottleParams struct {
	// A reason string for the throttling.
	Reason string

	// If the 429 includes a suggested delay in the headers,
	// include it here.
	Delay *time.Duration
}
