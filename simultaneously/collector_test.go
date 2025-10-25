package simultaneously //nolint:testpackage // Testing internal functions

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errCollectorTest   = errors.New("collector test error")
	errSecondError     = errors.New("second error")
	errError1          = errors.New("error 1")
	errError2          = errors.New("error 2")
	errError3Collector = errors.New("error 3")
	errTestError       = errors.New("test error")
)

func TestNewCollector(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	_, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 5, cancelOnce, cancel)

	require.NotNil(t, collector)
	assert.NotNil(t, collector.exec)
	assert.NotNil(t, collector.cancelOnce)
	assert.NotNil(t, collector.cancel)
	assert.NotNil(t, collector.errorChan)
	assert.NotNil(t, collector.doneChan)
}

func TestCollector_Cleanup(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 2, cancelOnce, cancel)

	// Launch some goroutines
	collector.launchAll(ctx, []func(context.Context) error{
		func(ctx context.Context) error {
			return nil
		},
		func(ctx context.Context) error {
			return nil
		},
	})

	// Collect results before cleanup
	errs := collector.collectResults(2)
	assert.Empty(t, errs)

	// Cleanup should wait for all goroutines and close channels
	collector.cleanup()

	// Verify channels are closed
	_, errorChanOpen := <-collector.errorChan
	assert.False(t, errorChanOpen, "errorChan should be closed")

	_, doneChanOpen := <-collector.doneChan
	assert.False(t, doneChanOpen, "doneChan should be closed")
}

func TestCollector_LaunchAll_AllSuccess(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 3, cancelOnce, cancel)

	var executedCount atomic.Int32

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			executedCount.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			executedCount.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			executedCount.Add(1)

			return nil
		},
	}

	collector.launchAll(ctx, callbacks)
	collector.cleanup()

	assert.Equal(t, int32(3), executedCount.Load())
}

func TestCollector_LaunchAll_WithErrors(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 3, cancelOnce, cancel)

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			return errCollectorTest
		},
		func(ctx context.Context) error {
			return errSecondError
		},
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)

			return nil
		},
	}

	collector.launchAll(ctx, callbacks)

	// Collect results before cleanup
	errs := collector.collectResults(3)
	assert.NotEmpty(t, errs)

	collector.cleanup()

	// Channels should be closed after cleanup
	_, errorChanOpen := <-collector.errorChan
	assert.False(t, errorChanOpen)
}

func TestCollector_LaunchAll_CancelsContextOnError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 3, cancelOnce, cancel)

	var cancelledCount atomic.Int32

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			// Return error immediately
			return errCollectorTest
		},
		func(ctx context.Context) error {
			// This should see context cancellation
			time.Sleep(50 * time.Millisecond)
			if ctx.Err() != nil {
				cancelledCount.Add(1)
			}

			return ctx.Err()
		},
		func(ctx context.Context) error {
			// This should see context cancellation
			time.Sleep(50 * time.Millisecond)
			if ctx.Err() != nil {
				cancelledCount.Add(1)
			}

			return ctx.Err()
		},
	}

	collector.launchAll(ctx, callbacks)
	collector.cleanup()

	// Context should have been canceled due to the error
	assert.Error(t, ctx.Err())
}

func TestCollector_LaunchAll_EmptyCallbacks(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 0, cancelOnce, cancel)

	// Launch with empty slice
	collector.launchAll(ctx, []func(context.Context) error{})
	collector.cleanup()

	// Should complete without issues
	assert.NoError(t, ctx.Err())
}

func TestCollector_CollectResults_AllSuccess(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 3, cancelOnce, cancel)

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
		func(ctx context.Context) error { return nil },
	}

	collector.launchAll(ctx, callbacks)

	errs := collector.collectResults(3)
	collector.cleanup()

	assert.Empty(t, errs)
}

func TestCollector_CollectResults_WithErrors(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 3, cancelOnce, cancel)

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			time.Sleep(5 * time.Millisecond)

			return errError1
		},
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			// Check if context was canceled by previous error
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(15 * time.Millisecond)
			// Check if context was canceled
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return errError2
		},
	}

	collector.launchAll(ctx, callbacks)

	errs := collector.collectResults(3)
	collector.cleanup()

	// At least errError1 should be present. Others may be errors or context.Canceled
	assert.NotEmpty(t, errs)
	// Check that we got at least one of the expected errors
	hasErr1 := false

	for _, e := range errs {
		if errors.Is(e, errError1) {
			hasErr1 = true

			break
		}
	}

	assert.True(t, hasErr1, "should contain error 1")
}

func TestCollector_CollectResults_MixedSuccessAndErrors(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(5)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 5, cancelOnce, cancel)

	var successCount atomic.Int32

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			successCount.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(5 * time.Millisecond)

			return errError1
		},
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)
			// May be canceled by previous error
			if ctx.Err() == nil {
				successCount.Add(1)
			}

			return ctx.Err()
		},
		func(ctx context.Context) error {
			time.Sleep(15 * time.Millisecond)
			// May be canceled by previous error
			if ctx.Err() != nil {
				return ctx.Err()
			}

			return errError2
		},
		func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			// May be canceled by previous error
			if ctx.Err() == nil {
				successCount.Add(1)
			}

			return ctx.Err()
		},
	}

	collector.launchAll(ctx, callbacks)

	errs := collector.collectResults(5)
	collector.cleanup()

	// Should have at least one error (error 1)
	assert.NotEmpty(t, errs)
	// At least one callback should have succeeded (the first one with no sleep)
	assert.GreaterOrEqual(t, successCount.Load(), int32(1))
}

func TestCollector_CollectResults_NoResults(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 0, cancelOnce, cancel)

	// Launch no callbacks
	collector.launchAll(ctx, []func(context.Context) error{})

	errs := collector.collectResults(0)
	collector.cleanup()

	assert.Empty(t, errs)
}

func TestCollector_CancelOnceEnsuresSingleCancellation(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	var cancelCallCount atomic.Int32

	wrappedCancel := func() {
		cancelCallCount.Add(1)
		cancel()
	}

	collector := newCollector(exec, 3, cancelOnce, wrappedCancel)

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			return errError1
		},
		func(ctx context.Context) error {
			return errError2
		},
		func(ctx context.Context) error {
			return errError3Collector
		},
	}

	collector.launchAll(ctx, callbacks)
	collector.cleanup()

	// Even though multiple errors occurred, cancel should only be called once
	assert.Equal(t, int32(1), cancelCallCount.Load())
}

func TestCollector_BufferedChannelsPreventBlocking(t *testing.T) {
	t.Parallel()

	// Create executor with low concurrency to test buffering
	exec := newDefaultExecutor(1)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	size := 100
	collector := newCollector(exec, size, cancelOnce, cancel)

	// Create many callbacks that will complete quickly
	callbacks := make([]func(context.Context) error, size)

	for i := range size {
		if i%2 == 0 {
			callbacks[i] = func(ctx context.Context) error {
				return nil
			}
		} else {
			callbacks[i] = func(ctx context.Context) error {
				return errTestError
			}
		}
	}

	// This should not block even though we're not collecting results yet
	collector.launchAll(ctx, callbacks)

	// Now collect results
	errs := collector.collectResults(size)
	collector.cleanup()

	// Should have collected errors without blocking
	assert.NotEmpty(t, errs)
}

func TestCollector_LaunchAll_WithSlowCallbacks(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	cancelOnce := &sync.Once{}

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	collector := newCollector(exec, 3, cancelOnce, cancel)

	var completedCount atomic.Int32

	callbacks := []func(context.Context) error{
		func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			completedCount.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			completedCount.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			completedCount.Add(1)

			return nil
		},
	}

	startTime := time.Now()

	collector.launchAll(ctx, callbacks)

	errs := collector.collectResults(3)
	collector.cleanup()

	duration := time.Since(startTime)

	assert.Empty(t, errs)
	assert.Equal(t, int32(3), completedCount.Load())
	// With maxConcurrent=3, all should run in parallel, so total time should be ~100ms, not 300ms
	assert.Less(t, duration, 200*time.Millisecond)
}
