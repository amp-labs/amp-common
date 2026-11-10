package ratelimit

import "context"

// NewNullLimiter returns a Limiter that does nothing. It will never
// enforce any limits, and will always return a reservation that is
// immediately available.
func NewNullLimiter() Limiter { //nolint:ireturn
	return NewLimiterBuilder().
		SetReserve(func(ctx context.Context, tokens int) (Reservation, error) {
			return NewNullReservation(), nil
		}).Build()
}

// NewNullReservation creates a reservation which always allows the action
// without any delay.
func NewNullReservation() Reservation {
	return NewReservationBuilder().Build()
}
