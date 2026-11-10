package ratelimit

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/utils"
)

// rateLimitContextKey is the type of this package's context keys. A
// package-private type keeps them from colliding with keys set by other
// packages.
type rateLimitContextKey string

// limiterContextKey is the context key under which WithRateLimiter stores
// the Limiter.
const limiterContextKey rateLimitContextKey = "rate-limiter"

// WithRateLimiter returns a copy of ctx that carries l. Code that receives
// the returned context, or a context derived from it, can retrieve l with
// GetRateLimiter. If ctx is nil, the copy is derived from
// context.Background().
//
// Passing a nil l, or a typed nil such as a nil *T, hides any Limiter that
// ctx already carries: GetRateLimiter and HasRateLimiter report false for
// the returned context.
func WithRateLimiter(ctx context.Context, l Limiter) context.Context {
	return contexts.WithValue[rateLimitContextKey, Limiter](ctx, limiterContextKey, l)
}

// SetRateLimiter stores l in a mutable context: one that is changed in place
// rather than derived with context.WithValue, as WithRateLimiter does. It
// calls f with this package's context key and l; f must store the pair so
// that the context's Value method returns l for that key, and GetRateLimiter
// then finds l in that context. SetRateLimiter does nothing if f is nil.
//
// For example, in a Fiber handler, pass c.Context().SetUserValue as f (or a
// closure that calls c.Locals) and read the Limiter back with
// GetRateLimiter(c.Context()). It is not visible through c.UserContext(),
// which is a separate context.
//
// Storing a nil or typed-nil l makes GetRateLimiter and HasRateLimiter report
// false for that context.
func SetRateLimiter(l Limiter, f func(key any, value any)) {
	if f != nil {
		f(limiterContextKey, l)
	}
}

// GetRateLimiter returns the Limiter that WithRateLimiter or SetRateLimiter
// stored in ctx, and whether one was found. It returns nil and false if ctx
// is nil, carries no Limiter, or carries a typed nil such as a nil *T, so a
// true result is always safe to call.
func GetRateLimiter(ctx context.Context) (Limiter, bool) {
	lim, exists := contexts.GetValue[rateLimitContextKey, Limiter](ctx, limiterContextKey)
	if !exists || utils.IsNilish(lim) {
		return nil, false
	}

	return lim, true
}

// HasRateLimiter reports whether ctx carries a usable Limiter, that is,
// whether GetRateLimiter would return true.
func HasRateLimiter(ctx context.Context) bool {
	_, ok := GetRateLimiter(ctx)

	return ok
}
