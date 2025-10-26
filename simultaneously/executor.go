package simultaneously

import (
	"context"
	"errors"
	"runtime/debug"
	"sync/atomic"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/utils"
)

// ErrExecutorClosed is returned when attempting to execute functions on a closed executor.
var ErrExecutorClosed = errors.New("executor is closed")

// Executor manages concurrent execution of functions with a configurable concurrency limit.
// It provides methods to execute functions asynchronously while respecting resource constraints.
type Executor interface {
	// GoContext executes fn asynchronously using the provided context, calling done with the result.
	// If the executor is closed or the context is canceled, done is called with the appropriate error.
	GoContext(ctx context.Context, fn func(context.Context) error, done func(error))

	// Go executes fn asynchronously using a background context, calling done with the result.
	// This is a convenience wrapper around GoContext that uses context.Background().
	Go(fn func(context.Context) error, done func(error))

	// Close shuts down the executor, preventing new executions and waiting for in-flight operations.
	// Returns ErrExecutorClosed if the executor is already closed.
	Close() error
}

// defaultExecutor implements the Executor interface using a semaphore pattern for concurrency control.
// It uses a buffered channel (sem) as a counting semaphore to limit concurrent executions.
type defaultExecutor struct {
	maxConcurrent int           // Maximum number of concurrent executions allowed
	sem           chan struct{} // Semaphore channel for concurrency control
	closed        *atomic.Bool  // Thread-safe flag indicating if executor is closed
}

// NewDefaultExecutor creates a new executor with the specified concurrency limit.
//
// The executor manages parallel execution of functions while respecting the maxConcurrent limit.
// It uses a semaphore-based approach to control how many functions can run simultaneously.
//
// Parameters:
//   - maxConcurrent: Maximum number of functions that can execute concurrently.
//     If less than 1, defaults to 1 (sequential execution).
//
// Returns:
//   - An Executor that can be used with DoWithExecutor, MapSliceWithExecutor,
//     and other *WithExecutor variant functions.
//
// The executor must be closed when no longer needed to release resources:
//
//	exec := NewDefaultExecutor(5)
//	defer exec.Close()
//
// Example usage:
//
//	// Create executor with max 3 concurrent operations
//	exec := NewDefaultExecutor(3)
//	defer exec.Close()
//
//	// Use with DoWithExecutor to reuse across multiple batches
//	batch1 := []func(context.Context) error{...}
//	batch2 := []func(context.Context) error{...}
//
//	if err := DoWithExecutor(exec, batch1...); err != nil {
//	    return err
//	}
//	if err := DoWithExecutor(exec, batch2...); err != nil {
//	    return err
//	}
//
// Executor reuse is beneficial when processing multiple batches of work
// as it avoids the overhead of creating and destroying executors repeatedly.
func NewDefaultExecutor(maxConcurrent int) Executor {
	// If maxConcurrent not specified (< 1), use 1 as the limit
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	sem := make(chan struct{}, maxConcurrent)

	// Fill the semaphore with maxConcurrent empty structs (tokens).
	// Each token represents an available execution slot.
	for range maxConcurrent {
		sem <- struct{}{}
	}

	return &defaultExecutor{
		maxConcurrent: maxConcurrent,
		sem:           sem,
		closed:        &atomic.Bool{},
	}
}

// newDefaultExecutor creates a new executor with the specified concurrency limit.
// The semaphore is pre-filled with tokens, allowing up to maxConcurrent operations.
// If maxConcurrent is less than 1, itemCount is used to determine the buffer size.
func newDefaultExecutor(maxConcurrent, itemCount int) *defaultExecutor {
	// If maxConcurrent not specified (< 1), use itemCount as the limit
	if maxConcurrent < 1 {
		maxConcurrent = itemCount
	}

	if maxConcurrent > itemCount {
		maxConcurrent = itemCount
	}

	if maxConcurrent < 1 {
		maxConcurrent = 1
	}

	sem := make(chan struct{}, maxConcurrent)

	// Fill the semaphore with maxConcurrent empty structs (tokens).
	// Each token represents an available execution slot.
	for range maxConcurrent {
		sem <- struct{}{}
	}

	return &defaultExecutor{
		maxConcurrent: maxConcurrent,
		sem:           sem,
		closed:        &atomic.Bool{},
	}
}

func (d *defaultExecutor) Go(fn func(context.Context) error, done func(error)) {
	d.GoContext(context.Background(), fn, done)
}

// GoContext executes the callback asynchronously while respecting concurrency limits.
// It implements a double-check pattern to prevent race conditions during executor closure:
// 1. Check if closed before acquiring semaphore token
// 2. Check again after acquiring token (while blocking, executor may have closed)
// This ensures proper cleanup and prevents goroutine leaks during shutdown.
func (d *defaultExecutor) GoContext(ctx context.Context, callback func(context.Context) error, done func(error)) {
	// First check: Fail fast if executor is already closed
	if d.closed.Load() {
		done(ErrExecutorClosed)

		return
	}

	// Wait for either a chance to run or the context to be canceled.
	// This blocks until a semaphore token is available or context expires.
	select {
	case <-ctx.Done():
		done(ctx.Err())

		return
	case <-d.sem: // Acquire token (blocks if no slots available)
	}

	// Second check: Verify executor wasn't closed while we were waiting.
	// If it was closed, return the token immediately to prevent resource leak.
	if d.closed.Load() {
		d.sem <- struct{}{} // Return token to semaphore

		done(ErrExecutorClosed)

		return
	}

	// Launch goroutine to execute the callback.
	// The semaphore token is held for the duration of execution.
	go func() {
		defer func() {
			d.sem <- struct{}{} // Always return the token when done
		}()

		done(d.executeCallback(ctx, callback))
	}()
}

// Close gracefully shuts down the executor and waits for all in-flight operations to complete.
// It performs three steps:
// 1. Atomically marks the executor as closed (prevents new executions)
// 2. Drains all tokens from the semaphore (waits for all goroutines to finish)
// 3. Closes the semaphore channel
// Returns ErrExecutorClosed if already closed.
func (d *defaultExecutor) Close() error {
	// Atomically set closed flag from false to true.
	// If already true, another goroutine closed it first.
	if !d.closed.CompareAndSwap(false, true) {
		return ErrExecutorClosed
	}

	// Wait for all in-flight operations to complete by draining the semaphore.
	// Each goroutine returns its token when done, so this blocks until all tokens are back.
	for range d.maxConcurrent {
		<-d.sem
	}

	// Now that all goroutines are done, it's safe to close the channel
	close(d.sem)

	return nil
}

// executeCallback runs the callback function with panic recovery and context validation.
// It ensures the context is valid before execution and converts any panics to errors.
// The nolint directive is necessary because the function intentionally passes through
// the context parameter without modification for maximum flexibility.
//
//nolint:contextcheck
func (d *defaultExecutor) executeCallback(ctx context.Context, fn func(context.Context) error) (err error) {
	// Ensure we have a valid context
	if ctx == nil {
		ctx = context.Background()
	}

	// Short-circuit if context is already canceled/expired
	if !contexts.IsContextAlive(ctx) {
		return ctx.Err()
	}

	// Set up panic recovery to convert panics into errors
	defer d.recoverPanic(&err)

	// Execute the callback
	err = fn(ctx)

	return
}

// recoverPanic recovers from panics in callback functions and converts them to errors.
// If the callback already returned an error AND panicked, both errors are combined.
// This ensures panics don't crash the executor and are reported through normal error channels.
func (d *defaultExecutor) recoverPanic(err *error) {
	if r := recover(); r != nil {
		// Convert panic to structured error with stack trace
		if panicErr := utils.GetPanicRecoveryError(r, debug.Stack()); panicErr != nil {
			if *err != nil {
				// Both panic and error occurred - combine them
				*err = combineErrors([]error{panicErr, *err})
			} else {
				// Only panic occurred
				*err = panicErr
			}
		}
	}
}

// combineErrors consolidates multiple errors into a single error value.
// Returns nil for empty slice, the single error if only one exists,
// or a combined error using errors.Join for multiple errors.
func combineErrors(errs []error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return errors.Join(errs...)
	}
}
