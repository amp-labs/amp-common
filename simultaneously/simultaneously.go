package simultaneously

import (
	"context"
	"sync"
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
func DoCtx(ctx context.Context, maxConcurrent int, callback ...func(ctx context.Context) error) (errOut error) {
	ctx, cancel := context.WithCancel(ctx)

	var cancelOnce sync.Once
	defer cancelOnce.Do(cancel)

	if maxConcurrent < 1 {
		maxConcurrent = len(callback)
	}

	de := newDefaultExecutor(maxConcurrent)

	coll := newCollector(de, len(callback), &cancelOnce, cancel)

	defer func() {
		cleanupErr := coll.cleanup()
		if cleanupErr != nil {
			if errOut == nil {
				errOut = cleanupErr
			} else {
				errOut = combineErrors([]error{errOut, cleanupErr})
			}
		}
	}()

	coll.launchAll(ctx, callback)

	errs := coll.collectResults(len(callback))

	return combineErrors(errs)
}
