package contexts

import (
	"context"
	"time"
)

// WithIgnoreLifecycle wraps a context to ignore all lifecycle signals (cancellation,
// deadlines, and timeouts) while preserving access to context values.
//
// This is useful in scenarios where you need to perform cleanup operations or finalize
// work that should continue even after the parent context has been canceled. For example:
//   - Flushing buffers to disk after a request is canceled
//   - Sending final metrics or logs during shutdown
//   - Completing database transactions that shouldn't be interrupted
//
// The returned context:
//   - Never reports as done (Done() returns a channel that never closes)
//   - Never has a deadline (Deadline() returns zero time and false)
//   - Never reports an error (Err() always returns nil)
//   - Preserves value lookups from the wrapped context (Value() delegates to inner context)
//
// WARNING: Use this carefully. Code using this context will not respond to cancellation
// signals, which can lead to goroutine leaks or hung operations if not properly bounded
// by other mechanisms (e.g., timeouts, iteration limits).
//
// Example:
//
//	func cleanup(ctx context.Context) {
//	    // Create a context that ignores cancellation for cleanup work
//	    cleanupCtx := contexts.WithIgnoreLifecycle(ctx)
//
//	    // This operation will complete even if ctx is canceled
//	    flushBuffers(cleanupCtx)
//	}
//
// If ctx is nil, returns nil.
func WithIgnoreLifecycle(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}

	return &lifecycleInsensitiveContext{
		inner: ctx,
	}
}

// neverClosed is a channel that never closes, used to signal that the context
// will never be done. This is shared across all lifecycle-insensitive contexts
// to avoid allocating a new channel for each wrapper.
var neverClosed = make(chan struct{})

// lifecycleInsensitiveContext is a context.Context implementation that ignores
// all lifecycle signals from its wrapped context while preserving value lookups.
//
// This type implements the context.Context interface by:
//   - Returning a channel that never closes for Done()
//   - Returning zero time and false for Deadline()
//   - Returning nil for Err()
//   - Delegating Value() calls to the wrapped context
type lifecycleInsensitiveContext struct {
	inner context.Context //nolint:containedctx // This is a context wrapper by design
}

// Deadline returns a zero time and false, indicating that this context has no deadline.
// This ignores any deadline that may be set on the wrapped context.
func (l *lifecycleInsensitiveContext) Deadline() (deadline time.Time, ok bool) {
	return time.Time{}, false
}

// Done returns a channel that will never close, indicating that this context will never
// be canceled or time out. This allows operations to continue regardless of the state
// of the wrapped context.
func (l *lifecycleInsensitiveContext) Done() <-chan struct{} {
	return neverClosed
}

// Err always returns nil, indicating that this context is never canceled or timed out.
// This ignores any error state from the wrapped context.
func (l *lifecycleInsensitiveContext) Err() error {
	return nil
}

// Value retrieves values from the wrapped context. This is the only method that delegates
// to the inner context, allowing access to context values while ignoring lifecycle signals.
//
// This enables use cases like accessing request IDs, trace spans, or other metadata even
// during cleanup operations that should continue after the parent context is canceled.
func (l *lifecycleInsensitiveContext) Value(key any) any {
	return l.inner.Value(key)
}

// Compile-time assertion that lifecycleInsensitiveContext implements context.Context.
var _ context.Context = (*lifecycleInsensitiveContext)(nil)
