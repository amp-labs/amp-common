// Package simultaneously provides utilities for running functions concurrently with controlled parallelism.
// It handles context cancellation, panic recovery, and error aggregation automatically.
package simultaneously

import (
	"context"
	"sync"

	"github.com/amp-labs/amp-common/errors"
)

// Job is a function that performs a unit of work and returns an error if it fails.
//
// This is an alias rather than a defined type so that a []Job and a
// []func(ctx context.Context) error remain the same type: callers can build a
// slice under either spelling and spread it into any of the Do* functions.
type Job = func(ctx context.Context) error

// Do runs the given functions in parallel and returns the first error encountered.
// See SimultaneouslyCtx for more information.
func Do(maxConcurrent int, f ...Job) error {
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
//
// If ctx carries an executor (see WithExecutor), the jobs run on that executor, and
// maxConcurrent still caps this call on top of the shared executor's own limit. That executor
// belongs to whoever put it on the context, so it is left open for reuse here. Beware
// that a DoCtx nested inside a job inherits the same executor and can deadlock against
// it; WithExecutor documents the conditions.
func DoCtx(ctx context.Context, maxConcurrent int, callback ...Job) error {
	// An executor on the context (see WithExecutor) wins over a throwaway one, in
	// which case maxConcurrent caps only this call and closeExec is a no-op.
	exec, closeExec := resolveExecutor(ctx, maxConcurrent, len(callback))

	errs := errors.Collection{}

	// Both the work and the shutdown can fail, and a Close error (e.g. a leaked
	// in-flight job) is worth surfacing even when every job succeeded, so collect
	// the two rather than letting either shadow the other.
	errs.Add(DoCtxWithExecutor(ctx, exec, callback...))
	errs.Add(closeExec())

	return errs.GetError()
}

// DoWithExecutor runs the given functions in parallel using a custom executor.
// See DoCtxWithExecutor for more information.
func DoWithExecutor(exec Executor, callback ...Job) error {
	return DoCtxWithExecutor(context.Background(), exec, callback...)
}

// DoCtxWithExecutor runs the given functions in parallel using a custom executor.
// This is useful when you want to reuse an executor across multiple batches of work
// or when you need custom execution behavior. The executor is not closed by this function,
// allowing it to be reused. All other behavior matches DoCtx including context cancellation,
// panic recovery, and error handling.
func DoCtxWithExecutor(ctx context.Context, exec Executor, callback ...Job) error {
	ctx, cancel := context.WithCancel(ctx)

	var cancelOnce sync.Once
	defer cancelOnce.Do(cancel)

	coll := newCollector(exec, len(callback), &cancelOnce, cancel)

	defer coll.cleanup()

	coll.launchAll(ctx, callback)

	errs := coll.collectResults(len(callback))

	return combineErrors(errs)
}
