package simultaneously

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"sync"

	"github.com/amp-labs/amp-common/utils"
)

// ErrPanicRecovered is the base error for panic recovery.
var ErrPanicRecovered = errors.New("panic recovered")

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
	// We'll wrap the context for this function so we can cancel it
	ctx, cancel := context.WithCancel(ctx)

	// We only want to cancel the context once, so we'll use a sync.Once
	var cancelOnce sync.Once
	defer cancelOnce.Do(cancel)

	// If maxConcurrent is less than 1, we'll just run everything at once.
	if maxConcurrent < 1 {
		maxConcurrent = len(callback)
	}

	// We'll use a buffered channel as a semaphore to limit
	// the number of concurrent goroutines. This is just
	// a simple way to permit parallelism without overwhelming
	// the system. A little burstiness is fine, but a lot
	// can cause problems, so this just helps smooth things out.
	sem := make(chan struct{}, maxConcurrent)

	// This is how we avoid racing with the channels themselves
	// being closed. Without this, occasional panics can happen,
	// especially with respect to the sem channel.
	var waitGroup sync.WaitGroup

	// Fill the semaphore with maxConcurrent empty structs
	for range maxConcurrent {
		sem <- struct{}{}
	}

	// We'll use two channels to communicate with the goroutines.
	errorChan := make(chan error)
	doneChan := make(chan struct{})

	// Make sure we close the channels when we're done.
	defer func() {
		waitGroup.Wait()
		close(sem)
		close(errorChan)
		close(doneChan)
	}()

	// Invoker will do the following:
	// 1. Take a semaphore (blocking if none are available)
	// 2. Call the function (only if the context is still alive)
	// 3. If the function returns an error, send it to the error channel
	// 4. If the function returns nil, send a signal to the done channel
	// 5. Put the semaphore back
	// 6. Recover from any panics and convert them to errors
	invoker := func(callerFn func(context.Context) error) {
		<-sem // take one out (will block if empty)

		defer func() {
			sem <- struct{}{} // put it back

			waitGroup.Done()
		}()

		// Recover from panics and convert them to errors
		defer func() {
			if r := recover(); r != nil {
				var err error
				if e, ok := r.(error); ok {
					err = fmt.Errorf("%w: %w\n%s", ErrPanicRecovered, e, debug.Stack())
				} else {
					err = fmt.Errorf("%w: %v\n%s", ErrPanicRecovered, r, debug.Stack())
				}

				// Cancel the context to stop other functions
				cancelOnce.Do(cancel)

				// Send the panic as an error
				errorChan <- err
			}
		}()

		// If the context is already canceled, don't bother running the function
		// since it will error out anyway.
		if !utils.IsContextAlive(ctx) {
			errorChan <- ctx.Err()

			return
		}

		if e := callerFn(ctx); e != nil {
			// Cancel the context as soon as we know there was an error,
			// to try to save on CPU cycles. Other functions should
			// check their context and return early if they are canceled.
			// Anything blocked on the semaphore will be unblocked and then
			// will check the context and return early.
			cancelOnce.Do(cancel)

			// Send the error to the error channel
			errorChan <- e
		} else {
			// All good, send a signal to the done channel
			doneChan <- struct{}{}
		}
	}

	// Start all the goroutines at once.
	// Ideally, most of these will block on the semaphore
	// and only run when there's room. So the fact that there
	// are a lot of goroutines here is not necessarily a problem.
	running := 0

	for _, callerFn := range callback {
		waitGroup.Add(1)

		running++

		go invoker(callerFn)
	}

	// Keep track of all the errors we encounter.
	var errs []error

	// Wait for all the goroutines to finish.
	for running > 0 {
		select {
		case e := <-errorChan:
			// Keep track of the error
			errs = append(errs, e)

			break
		case <-doneChan:
			// The function finished successfully
			break
		}

		running--
	}

	switch len(errs) {
	case 0:
		// Everything succeeded
		return nil
	case 1:
		// Only one error
		return errs[0]
	default:
		// Multiple errors
		return errors.Join(errs...)
	}
}
