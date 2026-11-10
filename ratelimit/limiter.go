// Package ratelimit defines the interfaces a caller uses to stay within a
// rate limit, independent of how the limit is enforced.
//
// The package enforces no limit itself. A program supplies the Limiter: its
// own implementation, or one assembled from functions with LimiterBuilder and
// ReservationBuilder. A LimiterBuilder with nothing set builds a Limiter that
// allows every call without delay.
//
// Callers receive the Limiter directly or through a context: WithRateLimiter
// stores it (SetRateLimiter, for a context that is changed in place), and
// code further down the call chain retrieves it with GetRateLimiter.
//
// A rate-limited call goes through these steps:
//
//  1. Call Limiter.Reserve with the number of tokens the call will consume.
//  2. If Reservation.OK returns false, don't make the call.
//  3. Wait for the duration that Reservation.Delay returns.
//  4. Make the call, then report its Status with Reservation.Done. Pass
//     ThrottleParams if the service throttled the call. If you decided not
//     to make the call after all, report Canceled so that the limiter
//     returns the unused tokens.
package ratelimit

import "context"

// Limiter contains all of the logic necessary to rate
// limit calls.
type Limiter interface {
	// Reserve will hint to the limiter that a call is about to
	// be made which will consume N tokens. The limiter will
	// immediately return a Reservation instance (or an error).
	// The caller must check the reservation details to know
	// if the call is permissible, and how long to wait. Error
	// will only be returned in circumstances where the limiter
	// itself is having problems, or the input parameters are bad.
	Reserve(ctx context.Context, tokens int) (Reservation, error)
}

// LimiterBuilder builds Limiter instances backed by a reserve function, using
// the builder pattern. The zero value is ready to use, and is what
// NewLimiterBuilder returns.
type LimiterBuilder struct {
	// reserve backs Limiter.Reserve. If it is nil, Build uses a function
	// that allows every call without delay.
	reserve func(ctx context.Context, tokens int) (Reservation, error)
}

// NewLimiterBuilder creates a new limiter builder for constructing rate limiters.
func NewLimiterBuilder() *LimiterBuilder {
	return &LimiterBuilder{}
}

// SetReserve configures the reserve function that will be called when Reserve is invoked on the limiter.
func (b *LimiterBuilder) SetReserve(
	reserve func(ctx context.Context, tokens int) (Reservation, error),
) *LimiterBuilder {
	b.reserve = reserve

	return b
}

// Build constructs a Limiter with the configured reserve function, or a no-op limiter if none is set.
func (b *LimiterBuilder) Build() Limiter { //nolint:ireturn
	limiter := &limiterImpl{
		reserve: b.reserve,
	}

	if limiter.reserve == nil {
		limiter.reserve = func(ctx context.Context, tokens int) (Reservation, error) {
			return (&ReservationBuilder{}).Build(), nil
		}
	}

	return limiter
}

// limiterImpl is the concrete implementation of the Limiter interface. It
// delegates Reserve to the function that LimiterBuilder.Build set, which is
// never nil.
type limiterImpl struct {
	reserve func(ctx context.Context, tokens int) (Reservation, error)
}

// Compile-time check that *limiterImpl implements Limiter.
var _ = (Limiter)((*limiterImpl)(nil))

// Reserve attempts to reserve the specified number of tokens for a rate-limited operation.
func (l *limiterImpl) Reserve(ctx context.Context, tokens int) (Reservation, error) { //nolint:ireturn
	return l.reserve(ctx, tokens)
}
