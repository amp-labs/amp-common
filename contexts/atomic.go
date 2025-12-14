package contexts

import (
	"context"
	"time"

	"go.uber.org/atomic"
)

// atomicContext is a thread-safe context.Context implementation that allows
// atomic swapping of the underlying context. This is useful for long-running
// operations that need to dynamically replace their context, such as:
//   - Replacing a canceled context with a fresh one
//   - Updating context values without creating a new context chain
//   - Coordinating context changes across multiple goroutines
//
// All context.Context interface methods safely delegate to the current
// underlying context, with nil-safe fallbacks.
type atomicContext struct {
	// ctx stores a pointer to the current context.Context.
	// The pointer-to-pointer indirection enables atomic swapping.
	ctx *atomic.Pointer[context.Context]
}

// Compile-time assertion that atomicContext implements context.Context.
var _ context.Context = (*atomicContext)(nil)

// NewAtomic creates a new atomic context initialized with the given context.
// It returns:
//  1. The atomic context itself, which implements context.Context
//  2. A swap function that atomically replaces the underlying context
//
// The swap function is thread-safe and can be called from multiple goroutines.
// It returns the previous context that was replaced. If the previous context
// was nil (which shouldn't happen in normal use), it returns context.Background().
//
// Example usage:
//
//	ctx, swap := contexts.NewAtomic(context.Background())
//	go worker(ctx)
//
//	// Later, replace the context atomically
//	newCtx := context.WithValue(context.Background(), key, value)
//	oldCtx := swap(newCtx)
func NewAtomic(ctx context.Context) (context.Context, func(context.Context) context.Context) {
	ac := &atomicContext{
		ctx: atomic.NewPointer[context.Context](&ctx),
	}

	// Return the atomic context and a closure that captures the atomic pointer
	// to enable swapping the underlying context.
	return ac, func(ctx context.Context) context.Context {
		prev := ac.ctx.Swap(&ctx)

		// Defensive: if prev is nil, return a safe default
		if prev == nil {
			return context.Background()
		}

		return *prev
	}
}

// Deadline returns the deadline of the current underlying context.
// If the context has been swapped, this returns the deadline of the new context.
// Returns zero time and false if the underlying context is nil or has no deadline.
func (a *atomicContext) Deadline() (deadline time.Time, ok bool) {
	value := a.ctx.Load()

	// If no context is stored, there's no deadline
	if value == nil {
		return time.Time{}, false
	}

	return (*value).Deadline()
}

// Done returns the cancellation channel of the current underlying context.
// If the context has been swapped, this returns the Done channel of the new context.
// Returns a nil channel (never ready) if the underlying context is nil, similar to
// context.Background().Done().
func (a *atomicContext) Done() <-chan struct{} {
	value := a.ctx.Load()

	// If no context is stored, return a channel that will never be closed
	if value == nil {
		return context.Background().Done()
	}

	return (*value).Done()
}

// Err returns the cancellation error of the current underlying context.
// If the context has been swapped, this returns the error of the new context.
// Returns nil if the underlying context is nil or not canceled.
func (a *atomicContext) Err() error {
	value := a.ctx.Load()

	// If no context is stored, there's no error
	if value == nil {
		return nil
	}

	return (*value).Err()
}

// Value returns the value associated with the given key in the current underlying context.
// If the context has been swapped, this looks up the value in the new context.
// Returns nil if the underlying context is nil or doesn't contain the key.
func (a *atomicContext) Value(key any) any {
	value := a.ctx.Load()

	// If no context is stored, there are no values
	if value == nil {
		return nil
	}

	return (*value).Value(key)
}
