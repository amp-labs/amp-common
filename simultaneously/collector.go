package simultaneously

import (
	"context"
	"sync"
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
		e.exec.GoContext(ctx, fn, func(err error) {
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
