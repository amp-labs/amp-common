package simultaneously

import (
	"context"
	"errors"
	"runtime/debug"
	"sync"
	"sync/atomic"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/utils"
)

type Executor interface {
	GoContext(ctx context.Context, fn func(context.Context) error, done func(error))
	Go(fn func(context.Context) error, done func(error))
	Close() error
}

type defaultExecutor struct {
	sem    chan struct{}
	closed atomic.Bool
}

func newDefaultExecutor(maxConcurrent int) *defaultExecutor {
	sem := make(chan struct{}, maxConcurrent)

	// Fill the semaphore with maxConcurrent empty structs
	for range maxConcurrent {
		sem <- struct{}{}
	}

	return &defaultExecutor{
		sem: sem,
	}
}

func (d *defaultExecutor) Go(fn func(context.Context) error, done func(error)) {
	d.GoContext(context.Background(), fn, done)
}

func (d *defaultExecutor) GoContext(ctx context.Context, callback func(context.Context) error, done func(error)) {
	select {
	case <-ctx.Done():
		done(ctx.Err())

		return
	case <-d.sem: // take one out (will block if empty)
	}

	go func() {
		defer func() {
			// Return the token to the semaphore if the executor is not closed
			if !d.closed.Load() {
				d.sem <- struct{}{} // put it back
			}
		}()

		done(d.executeCallback(ctx, callback))
	}()
}

func (d *defaultExecutor) Close() error {
	// Use CompareAndSwap to atomically check and set closed flag
	// This ensures Close() is idempotent and thread-safe
	if d.closed.CompareAndSwap(false, true) {
		close(d.sem)
	}

	return nil
}

// executeCallback runs the callback function and sends the result to the appropriate channel.
//
//nolint:contextcheck
func (d *defaultExecutor) executeCallback(ctx context.Context, fn func(context.Context) error) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}

	if !contexts.IsContextAlive(ctx) {
		return ctx.Err()
	}

	defer d.recoverPanic(&err)

	err = fn(ctx)

	return
}

// recoverPanic recovers from panics and converts them to errors.
func (d *defaultExecutor) recoverPanic(err *error) {
	if r := recover(); r != nil {
		if panicErr := utils.GetPanicRecoveryError(r, debug.Stack()); panicErr != nil {
			if *err != nil {
				*err = combineErrors([]error{panicErr, *err})
			} else {
				*err = panicErr
			}
		}
	}
}

// collector manages the concurrent execution of callback functions.
type collector struct {
	exec       Executor
	cancelOnce *sync.Once
	cancel     context.CancelFunc
	errorChan  chan error
	doneChan   chan struct{}
	waitGroup  sync.WaitGroup
}

// newCollector creates a new executor with the given concurrency limit.
func newCollector(exec Executor, size int, cancelOnce *sync.Once, cancel context.CancelFunc) *collector {
	return &collector{
		exec:       exec,
		cancelOnce: cancelOnce,
		cancel:     cancel,
		errorChan:  make(chan error, size),
		doneChan:   make(chan struct{}, size),
	}
}

// cleanup closes all channels after waiting for goroutines to finish.
func (e *collector) cleanup() {
	e.waitGroup.Wait()

	close(e.errorChan)
	close(e.doneChan)
}

// launchAll starts all callback functions in separate goroutines.
func (e *collector) launchAll(ctx context.Context, callbacks []func(context.Context) error) {
	for _, fn := range callbacks {
		e.waitGroup.Add(1)
		e.exec.GoContext(ctx, fn, func(err error) {
			defer e.waitGroup.Done()

			if err != nil {
				e.cancelOnce.Do(e.cancel)
				e.errorChan <- err
			} else {
				e.doneChan <- struct{}{}
			}
		})
	}
}

// collectResults waits for all goroutines to complete and collects errors.
func (e *collector) collectResults(count int) []error {
	var errs []error

	for range count {
		select {
		case err := <-e.errorChan:
			errs = append(errs, err)
		case <-e.doneChan: // Function completed successfully
		}
	}

	return errs
}

// combineErrors returns a single error from a slice of errors.
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
