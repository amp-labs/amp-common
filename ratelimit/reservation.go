package ratelimit

import "time"

// Status represents the outcome of a rate-limited API call (OK, Throttled, Failed, Canceled, None).
type Status int

const (
	// None means someone didn't initialize an actual status.
	// It's the zero value of Status.
	None Status = iota

	// OK means the call succeeded.
	OK //nolint:varnamelen

	// Throttled means that we got a 429 (or equivalent) response.
	Throttled

	// Failed means the call failed for some other reason.
	Failed

	// Canceled means we never made the call.
	Canceled
)

// String returns the string representation of a rate limiter status.
func (s Status) String() string {
	switch s {
	case None:
		return "None"
	case OK:
		return "OK"
	case Throttled:
		return "Throttled"
	case Failed:
		return "Failed"
	case Canceled:
		return "Canceled"
	default:
		return "Unknown"
	}
}

// Reservation represents an entitlement to make a call.
// Once the holder knows that (1) OK is true, and (2) Delay
// has been fulfilled, they can proceed confidently knowing
// that they're adhering to the rate limit. In the event
// that they decide to not make the call, they call Done
// with Canceled to return the tokens to the rate limiter.
type Reservation interface {
	// OK returns true if the call is allowed to proceed.
	// If OK is false, that means we're load shedding
	// or doing short-circuiting, and the call should not
	// be made. Bail out immediately.
	OK() bool

	// Delay returns the duration the caller should wait
	// before making the call. Only valid if OK() is true.
	// The first return value is a string that describes
	// the reason for the delay, such as "429 error" or
	// "load shedding" or whatever is appropriate. Blank
	// is acceptable but not recommended for clarity.
	Delay() (string, time.Duration)

	// Done informs the rate limiter that the call has been
	// made, and the token(s) has been consumed. Don't call
	// this if OK() is false. In the event that OK() is true
	// but the call was not made, you need to call Done with
	// a value of Canceled - this is how the rate limiter knows
	// to return the unconsumed token(s) to the pool.
	Done(status Status, throttleParams ...ThrottleParams)
}

// ReservationBuilder builds rate limiter Reservation instances using the builder pattern.
// The zero value is ready to use, and is what NewReservationBuilder returns.
type ReservationBuilder struct {
	// ok backs Reservation.OK. If it is nil, Build uses a function that
	// returns true.
	ok func() bool

	// delay backs Reservation.Delay. If it is nil, Build uses a function
	// that returns no reason and no delay.
	delay func() (string, time.Duration)

	// done backs Reservation.Done. If it is nil, Build uses a function that
	// does nothing.
	done func(status Status, throttleParams ...ThrottleParams)
}

// NewReservationBuilder creates a new reservation builder for constructing rate limiter reservations.
// Contributors to the connector repo, please read: you don't need to call this, unless it's for testing.
// Most of the time these will be passed in for you, and the server side will decide what the details
// are going to be. You shouldn't call this from a provider's connector code.
func NewReservationBuilder() *ReservationBuilder {
	return &ReservationBuilder{}
}

// SetOK sets the function that determines if the reservation is valid.
func (b *ReservationBuilder) SetOK(ok func() bool) *ReservationBuilder {
	b.ok = ok

	return b
}

// SetDelay sets the function that returns the delay reason and duration before the operation can proceed.
func (b *ReservationBuilder) SetDelay(delay func() (string, time.Duration)) *ReservationBuilder {
	b.delay = delay

	return b
}

// SetDone sets the callback function to execute when the reservation is released.
func (b *ReservationBuilder) SetDone(
	done func(status Status, throttleParams ...ThrottleParams),
) *ReservationBuilder {
	b.done = done

	return b
}

// Build constructs the Reservation with default implementations for any unset functions.
func (b *ReservationBuilder) Build() Reservation { //nolint:ireturn
	rsrv := &reservationImpl{
		ok:    b.ok,
		delay: b.delay,
		done:  b.done,
	}

	if rsrv.ok == nil {
		rsrv.ok = func() bool { return true }
	}

	if rsrv.delay == nil {
		rsrv.delay = func() (string, time.Duration) { return "", 0 }
	}

	if rsrv.done == nil {
		rsrv.done = func(status Status, throttleParams ...ThrottleParams) {
		}
	}

	return rsrv
}

// reservationImpl is the concrete implementation of the Reservation interface.
// Each method delegates to the matching function that ReservationBuilder.Build
// set; none of them is nil.
type reservationImpl struct {
	ok    func() bool
	delay func() (string, time.Duration)
	done  func(status Status, throttleParams ...ThrottleParams)
}

// Compile-time check that *reservationImpl implements Reservation.
var _ = Reservation((*reservationImpl)(nil))

// OK returns whether the reservation is valid and can proceed.
func (r *reservationImpl) OK() bool {
	return r.ok()
}

// Delay returns the reason and duration to wait before the operation can proceed.
func (r *reservationImpl) Delay() (string, time.Duration) {
	return r.delay()
}

// Done marks the reservation as complete with the given status and optional throttle parameters.
func (r *reservationImpl) Done(status Status, throttleParams ...ThrottleParams) {
	r.done(status, throttleParams...)
}
