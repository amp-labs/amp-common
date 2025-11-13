// Package retry provides a flexible and configurable retry mechanism for operations that may fail
// transiently. It supports exponential backoff, jitter strategies, retry budgets, timeouts, and
// attempt tracking.
//
// The package offers both simple one-shot functions (Do, DoValue) and reusable Runner interfaces
// for operations that need consistent retry behavior.
//
// Basic usage:
//
//	err := retry.Do(ctx, func(ctx context.Context) error {
//	    return makeAPICall()
//	})
//
// With custom options:
//
//	err := retry.Do(ctx, operation,
//	    retry.WithAttempts(5),
//	    retry.WithBackoff(retry.ExpBackoff{Base: 100*time.Millisecond, Max: 5*time.Second, Factor: 2}),
//	    retry.WithJitter(retry.FullJitter),
//	)
//
// For operations that return values:
//
//	result, err := retry.DoValue(ctx, func(ctx context.Context) (string, error) {
//	    return fetchData()
//	})
package retry

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/amp-labs/amp-common/zero"
	"go.uber.org/atomic"
)

const (
	defaultAttempts      = 4
	defaultBaseDelay     = 100 // milliseconds
	defaultMaxDelay      = 2   // seconds
	defaultBackoffFactor = 2.0
)

// Runner is an interface for executing operations with retry logic.
// It handles errors and automatically retries based on the configured strategy.
type Runner interface {
	Do(ctx context.Context, f func(ctx context.Context) error) error
}

// ValueRunner is a generic interface for executing operations that return a value with retry logic.
// It handles errors and automatically retries based on the configured strategy, returning the
// successful result or an error.
type ValueRunner[T any] interface {
	Do(ctx context.Context, f func(ctx context.Context) (T, error)) (T, error)
}

// NewRunner creates a new Runner with the specified options.
// If no options are provided, it uses sensible defaults:
//   - 4 attempts (initial call + 3 retries)
//   - Exponential backoff: 100ms base, 2s max, factor of 2
//   - Full jitter to prevent thundering herd
//
// Example:
//
//	runner := retry.NewRunner(
//	    retry.WithAttempts(5),
//	    retry.WithTimeout(30 * time.Second),
//	)
//	err := runner.Do(ctx, operation)
func NewRunner(opts ...Option) Runner {
	intOpts := &options{
		attempts: Attempts(defaultAttempts),
		backoff: ExpBackoff{
			Base:   defaultBaseDelay * time.Millisecond,
			Max:    defaultMaxDelay * time.Second,
			Factor: defaultBackoffFactor,
		},
		jitter: FullJitter,
	}

	for _, option := range opts {
		option(intOpts)
	}

	return &runnerImpl{
		opts: intOpts,
	}
}

// NewValueRunner creates a new ValueRunner for operations that return a value.
// If no options are provided, it uses sensible defaults:
//   - 4 attempts (initial call + 3 retries)
//   - Exponential backoff: 100ms base, 2s max, factor of 2
//   - Full jitter to prevent thundering herd
//
// Example:
//
//	runner := retry.NewValueRunner[string](
//	    retry.WithAttempts(5),
//	    retry.WithTimeout(30 * time.Second),
//	)
//	result, err := runner.Do(ctx, operation)
func NewValueRunner[T any](opts ...Option) ValueRunner[T] {
	intOpts := &options{
		attempts: Attempts(defaultAttempts),
		backoff: ExpBackoff{
			Base:   defaultBaseDelay * time.Millisecond,
			Max:    defaultMaxDelay * time.Second,
			Factor: defaultBackoffFactor,
		},
		jitter: FullJitter,
	}

	for _, option := range opts {
		option(intOpts)
	}

	return &valueRunnerImpl[T]{
		opts: intOpts,
	}
}

// runnerImpl is the concrete implementation of the Runner interface.
type runnerImpl struct {
	opts *options
}

// Do executes the provided function with retry logic according to the runner's configuration.
func (r *runnerImpl) Do(ctx context.Context, f func(ctx context.Context) error) error {
	return do(ctx, r.opts, f)
}

// valueRunnerImpl is the concrete implementation of the ValueRunner interface.
type valueRunnerImpl[T any] struct {
	opts *options
}

// Do executes the provided function with retry logic according to the runner's configuration,
// returning the successful result or an error. If all retries are exhausted, it returns the
// zero value of type T and the last error encountered.
func (v valueRunnerImpl[T]) Do(ctx context.Context, f func(ctx context.Context) (T, error)) (T, error) {
	var out T

	err := do(ctx, v.opts, func(ctx context.Context) error {
		var err error

		out, err = f(ctx)

		return err
	})
	if err != nil {
		return zero.Value[T](), err
	}

	return out, nil
}

// do is the core retry loop that executes the provided function with retry logic.
// It handles:
//   - Attempt tracking via context
//   - Budget enforcement to prevent cascading failures
//   - Timeout handling for each attempt
//   - Backoff and jitter between retries
//   - Context cancellation
//   - Permanent vs temporary error handling
//
// The function returns:
//   - nil if the operation succeeds
//   - ctx.Err() if the context is canceled
//   - ErrExhausted if the retry budget is exhausted
//   - The permanent error if one is returned
//   - The last error if all retries are exhausted
func do(ctx context.Context, opts *options, operation func(ctx context.Context) error) error {
	var err error

	var mut sync.Mutex

	running := atomic.NewBool(true)
	defer running.Store(false)

	// Loop until we reach the maximum attempts or attempts is 0 (infinite retries)
	for attemptIndex := uint(0); Attempts(attemptIndex) < opts.attempts || opts.attempts == 0; attemptIndex++ {
		// Add attempt number to context for tracking
		ctx := withAttempt(ctx, attemptIndex)

		// Check if retry budget allows this attempt (prevents cascading failures)
		if !opts.budget.sendOK(attemptIndex != 0) {
			return ErrExhausted
		}

		// Create a new channel for each attempt to avoid race conditions
		// with goroutines from previous attempts
		errChan := make(chan error, 1)

		// Execute the operation in a goroutine to support timeout handling
		go func(ctx context.Context) {
			defer close(errChan)

			if opts.timeout != 0 {
				errChan <- callWithTimeout(ctx, operation, opts.timeout, &mut, running)
			} else {
				mut.Lock()
				defer mut.Unlock()

				if !running.Load() {
					return
				}

				errChan <- operation(ctx)
			}
		}(ctx)

		// Wait for either the operation to complete or context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err = <-errChan:
			if err == nil {
				return nil
			}

			// Check if the error is permanent (non-retryable)
			var retryErr Error
			if errors.As(err, &retryErr) && !retryErr.Temporary() {
				var p permanentError
				if errors.As(err, &p) {
					return p.error
				}

				return err
			}
		}

		// Calculate backoff delay with jitter
		delay := opts.backoff.Delay(attemptIndex)
		delay = opts.jitter.jitter(delay)

		// Wait for the delay period, respecting context cancellation
		ticker := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			ticker.Stop()

			return ctx.Err()
		case <-ticker.C:
			ticker.Stop()
		}
	}

	return err
}

// callWithTimeout wraps a function call with a timeout. If the function does not complete
// within the specified timeout, it returns context.DeadlineExceeded.
func callWithTimeout(
	ctx context.Context,
	callback func(context.Context) error,
	timeout Timeout,
	mut *sync.Mutex,
	running *atomic.Bool,
) error {
	// Brief lock/unlock provides a memory barrier to ensure visibility of running flag
	mut.Lock()
	mut.Unlock() //nolint:staticcheck

	if !running.Load() {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout))
	defer cancel()

	errChan := make(chan error, 1)

	go func(ctx context.Context) {
		defer close(errChan)

		mut.Lock()
		defer mut.Unlock()

		if !running.Load() {
			return
		}

		errChan <- callback(ctx)
	}(ctx)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
	}
}

// Do is a convenience function that creates a Runner and executes the provided function
// with retry logic in a single call. It uses the default configuration unless options are provided.
//
// Example:
//
//	err := retry.Do(ctx, func(ctx context.Context) error {
//	    return makeAPICall()
//	}, retry.WithAttempts(5))
func Do(ctx context.Context, f func(ctx context.Context) error, opts ...Option) error {
	return NewRunner(opts...).Do(ctx, f)
}

// DoValue is a convenience function that creates a ValueRunner and executes the provided function
// with retry logic in a single call. It uses the default configuration unless options are provided.
//
// Example:
//
//	result, err := retry.DoValue(ctx, func(ctx context.Context) (string, error) {
//	    return fetchData()
//	}, retry.WithAttempts(5))
func DoValue[T any](ctx context.Context, f func(ctx context.Context) (T, error), opts ...Option) (T, error) {
	return NewValueRunner[T](opts...).Do(ctx, f)
}
