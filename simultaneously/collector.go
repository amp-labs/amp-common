package simultaneously

import (
	"context"
	"runtime/debug"
	"sync"

	"github.com/amp-labs/amp-common/utils"
)

// collector orchestrates the concurrent execution of multiple callback functions.
// It manages error collection, context cancellation, and synchronization across goroutines.
// When any callback fails, it cancels the shared context to stop remaining work.
type collector struct {
	exec       Executor           // Executor to run callbacks with concurrency control
	cancelOnce *sync.Once         // Ensures cancel is called exactly once on first error
	cancel     context.CancelFunc // Cancels shared context to stop remaining callbacks
	errorChan  chan error         // Buffered channel for collecting errors from callbacks
	doneChan   chan struct{}      // Buffered channel signaling successful completions
	waitGroup  sync.WaitGroup     // Tracks completion of all launched goroutines
}

// newCollector creates a collector for executing multiple callbacks concurrently.
// The size parameter determines the buffer size for error and done channels,
// which should match the number of callbacks to prevent blocking.
func newCollector(exec Executor, size int, cancelOnce *sync.Once, cancel context.CancelFunc) *collector {
	return &collector{
		exec:       exec,
		cancelOnce: cancelOnce,
		cancel:     cancel,
		errorChan:  make(chan error, size),    // Buffered to prevent goroutine blocking
		doneChan:   make(chan struct{}, size), // Buffered to prevent goroutine blocking
	}
}

// cleanup waits for all goroutines to finish and then closes communication channels.
// This must be called after launchAll to ensure proper resource cleanup.
func (e *collector) cleanup() {
	// Wait for all callbacks to complete (either success or error)
	e.waitGroup.Wait()

	// Safe to close channels now that all goroutines are done
	close(e.errorChan)
	close(e.doneChan)
}

// launchAll starts all callback functions concurrently using the executor.
// Each callback completion (success or error) is tracked via the wait group.
// On first error, the shared context is canceled to signal remaining callbacks to stop.
func (e *collector) launchAll(ctx context.Context, callbacks []func(context.Context) error) {
	for _, fn := range callbacks {
		e.waitGroup.Add(1)
		e.launch(ctx, fn)
	}
}

// launch dispatches a single callback and guarantees that exactly one result
// reaches the collector for it, whatever the executor does.
//
// That guarantee is what the rest of the type is built on: cleanup blocks until
// the wait group drains, and collectResults blocks until it has seen one result
// per callback. An Executor that neither delivers a result nor returns normally
// would strand both. A panic out of GoContext is the realistic way that happens
// -- a custom Executor with a bug, or a nil one -- and it used to hang the whole
// call: the Add(1) above was never matched by a Done, so the deferred cleanup
// waited forever on a wait group that could not drain, swallowing the panic that
// was unwinding through it. Reporting it as this callback's error instead keeps
// the counts balanced and matches how the package treats a panicking job.
func (e *collector) launch(ctx context.Context, callback func(context.Context) error) {
	var reportOnce sync.Once

	// Exactly-once, so a late result from an executor that already panicked is
	// dropped rather than double-counted or sent on a channel cleanup has closed.
	report := func(err error) {
		reportOnce.Do(func() {
			defer e.waitGroup.Done()

			if err != nil {
				// Cancel context on first error (sync.Once ensures this happens exactly once)
				e.cancelOnce.Do(e.cancel)

				e.errorChan <- err
			} else {
				// Signal successful completion
				e.doneChan <- struct{}{}
			}
		})
	}

	defer func() {
		if r := recover(); r != nil {
			report(utils.GetPanicRecoveryError(r, debug.Stack()))
		}
	}()

	e.exec.GoContext(ctx, callback, report)
}

// collectResults gathers results from all callbacks, collecting errors or success signals.
// This blocks until exactly 'count' results are received (one per launched callback).
// Returns a slice of all errors encountered, or empty slice if all succeeded.
func (e *collector) collectResults(count int) []error {
	var errs []error

	// Wait for exactly count results (one per callback)
	for range count {
		select {
		case err := <-e.errorChan:
			errs = append(errs, err)
		case <-e.doneChan: // Callback completed successfully
		}
	}

	return errs
}
