// Package simultaneously provides utilities for running functions concurrently with controlled parallelism.
// It handles context cancellation, panic recovery, and error aggregation automatically.
package simultaneously

import (
	"context"
	"sync"

	"github.com/amp-labs/amp-common/errors"
)

// Do runs the given functions in parallel and returns the first error encountered.
// See SimultaneouslyCtx for more information.
func Do(maxConcurrent int, f ...func(ctx context.Context) error) error {
	return DoCtx(context.Background(), maxConcurrent, f...)
}

// DoCtx runs the given functions in parallel and returns the first error encountered.
// If no error is encountered, it returns nil. In the event that an error happens, all other functions
// are canceled (via their context) to hopefully save on CPU cycles. It's up to the individual functions
// to check their context and return early if they are canceled.
//
// The maxConcurrent parameter is used to limit the number of functions that run at the same time.
// If maxConcurrent is less than 1, all functions will run at the same time.
//
// Panics that occur within the callback functions are automatically recovered and converted to errors.
// This prevents a single panicking function from crashing the entire process.
func DoCtx(ctx context.Context, maxConcurrent int, callback ...func(ctx context.Context) error) error {
	if maxConcurrent < 1 {
		maxConcurrent = len(callback)
	}

	de := newDefaultExecutor(maxConcurrent)

	errs := errors.Collection{}

	errs.Add(DoCtxWithExecutor(ctx, de, callback...))
	errs.Add(de.Close())

	return errs.GetError()
}

// DoWithExecutor runs the given functions in parallel using a custom executor.
// See DoCtxWithExecutor for more information.
func DoWithExecutor(exec Executor, callback ...func(ctx context.Context) error) error {
	return DoCtxWithExecutor(context.Background(), exec, callback...)
}

// DoCtxWithExecutor runs the given functions in parallel using a custom executor.
// This is useful when you want to reuse an executor across multiple batches of work
// or when you need custom execution behavior. The executor is not closed by this function,
// allowing it to be reused. All other behavior matches DoCtx including context cancellation,
// panic recovery, and error handling.
func DoCtxWithExecutor(ctx context.Context, exec Executor, callback ...func(ctx context.Context) error) error {
	ctx, cancel := context.WithCancel(ctx)

	var cancelOnce sync.Once
	defer cancelOnce.Do(cancel)

	coll := newCollector(exec, len(callback), &cancelOnce, cancel)

	defer coll.cleanup()

	coll.launchAll(ctx, callback)

	errs := coll.collectResults(len(callback))

	return combineErrors(errs)
}
