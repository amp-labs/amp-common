package contexts

import (
	"context"
	"errors"
	"time"
)

// NewBatmanContext creates a context that only becomes Done when BOTH parent contexts are done.
//
// This is the opposite of standard Go context behavior, where a child context is canceled
// when ANY parent is canceled. Batman context waits for ALL parents to be canceled.
//
// The metaphor: Batman's origin story - he only becomes Batman when both his parents die.
// This context only becomes "done" when both parent contexts have been canceled/completed.
//
// Use cases:
//   - Waiting for multiple independent operations to complete
//   - Coordinating shutdown when multiple resources must be released
//   - Fan-in scenarios where all inputs must finish before proceeding
//
// Example:
//
//	httpCtx := httpServer.Context()  // Done when HTTP server shuts down
//	grpcCtx := grpcServer.Context()  // Done when gRPC server shuts down
//	batman := NewBatmanContext(httpCtx, grpcCtx)
//	<-batman.Done()  // Waits until BOTH servers have shut down
func NewBatmanContext(mom, dad context.Context) context.Context {
	mom = EnsureContext(mom)
	dad = EnsureContext(dad)

	// Channel that closes when both parents are done (buffered to prevent goroutine leak)
	done := make(chan struct{}, 1)

	// Launch background goroutine to monitor both parents
	go func() {
		defer close(done) // The moment of transformation

		momDone := mom.Done()
		dadDone := dad.Done()
		remaining := 2 // Both parents must fall

		for remaining > 0 {
			select {
			case <-momDone: // Martha! NOOOOOOOOOOOOOOOOOOOOOOO
				momDone = nil // Prevent repeated selection of closed channel
				remaining--
			case <-dadDone: // Thomas! ...OOOOOOOOOOOOOOOOOOOO0
				dadDone = nil // Prevent repeated selection of closed channel
				remaining--
			}
		}
		// At this point, both parents are gone. The Batman is born.
	}()

	return &batmanContext{
		mom:  mom,
		dad:  dad,
		done: done,
	}
}

// batmanContext is a context implementation that waits for both parent contexts to complete.
// It implements the full context.Context interface.
type batmanContext struct {
	mom  context.Context //nolint:containedctx // First parent context
	dad  context.Context //nolint:containedctx // Second parent context
	done chan struct{}   // Closes when both parents are done
}

// Compile-time assertion that batmanContext implements context.Context.
var _ context.Context = (*batmanContext)(nil)

// Deadline returns the time when both parents will be done (the later of the two deadlines).
//
// Since Batman only becomes active when BOTH parents are gone, the deadline is the later
// of the two parent deadlines. If only one parent has a deadline, that deadline is returned.
// If neither has a deadline, returns (zero time, false).
func (b *batmanContext) Deadline() (deadline time.Time, ok bool) {
	momDeadline, momHasDeadline := b.mom.Deadline()
	dadDeadline, dadHasDeadline := b.dad.Deadline()

	if !momHasDeadline && !dadHasDeadline {
		// Neither parent has a deadline
		return time.Time{}, false
	}

	if momHasDeadline && !dadHasDeadline {
		// Only mom has a deadline
		return momDeadline, true
	}

	if !momHasDeadline {
		// Only dad has a deadline
		return dadDeadline, true
	}

	// Both parents have deadlines. Return the later deadline
	// since that's when *both* parents will be dead.
	// And THAT is when this instance becomes THE BATMAN.
	return getLaterTime(dadDeadline, momDeadline), true
}

// Done returns a channel that's closed when both parent contexts are done.
//
// This channel closes only after BOTH mom.Done() and dad.Done() have closed.
// When this channel closes... The Batman rises.
func (b *batmanContext) Done() <-chan struct{} {
	return b.done
}

// Err returns a joined error containing errors from both parent contexts.
//
// This provides the full error context from both parents. If both parents
// have errors, they are combined using errors.Join. If only one has an error,
// that error is returned. If neither has an error, returns nil.
func (b *batmanContext) Err() error {
	return errors.Join(b.mom.Err(), b.dad.Err())
}

// Value returns the value associated with this context for key.
//
// Values are looked up in mom first, then dad. This mirrors the priority
// that Batman gives to his mother's memory (ask mom first, then dad).
// Returns the first non-nil value found, or nil if neither parent has the value.
func (b *batmanContext) Value(key any) any {
	// Ask mom
	value := b.mom.Value(key)
	if value != nil {
		return value
	}

	// Ask dad
	return b.dad.Value(key)
}

// getLaterTime returns the later of two times.
// Used to determine when both parent contexts will be done (the later deadline).
func getLaterTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}

	return b
}
