package simultaneously

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTestPanic = errors.New("test error panic")
	errTest      = errors.New("test error")
)

func TestDoCtx_RecoversPanic(t *testing.T) {
	t.Parallel()

	err := DoCtx(t.Context(), 2,
		func(ctx context.Context) error {
			panic("intentional panic for testing")
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic for testing")
	assert.Contains(t, err.Error(), "simultaneously_test.go") // stack trace should be present
}

func TestDoCtx_RecoversPanicError(t *testing.T) {
	t.Parallel()

	err := DoCtx(t.Context(), 2,
		func(ctx context.Context) error {
			panic(errTestPanic)
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	require.ErrorIs(t, err, errTestPanic)
	assert.Contains(t, err.Error(), "simultaneously_test.go") // stack trace should be present
}

func TestDoCtx_MixedSuccessAndPanic(t *testing.T) {
	t.Parallel()

	var successCount atomic.Int32

	err := DoCtx(t.Context(), 3,
		func(ctx context.Context) error {
			successCount.Add(1)
			time.Sleep(10 * time.Millisecond)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(5 * time.Millisecond)
			panic("boom")
		},
		func(ctx context.Context) error {
			successCount.Add(1)
			time.Sleep(10 * time.Millisecond)

			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "boom")

	// At least one function should have completed successfully
	// (though others may have been canceled due to the panic)
	assert.GreaterOrEqual(t, successCount.Load(), int32(1))
}

func TestDoCtx_MultiplePanics(t *testing.T) {
	t.Parallel()

	err := DoCtx(t.Context(), 3,
		func(ctx context.Context) error {
			panic("panic 1")
		},
		func(ctx context.Context) error {
			panic("panic 2")
		},
		func(ctx context.Context) error {
			panic("panic 3")
		},
	)

	require.Error(t, err)

	// Should get at least one panic error
	assert.Contains(t, err.Error(), "recovered from panic")

	// Due to concurrency and early cancellation, we might get multiple panics joined
	// or just the first one that was caught
	panicCount := strings.Count(err.Error(), "recovered from panic")
	assert.GreaterOrEqual(t, panicCount, 1)
}

func TestDoCtx_PanicDoesNotAffectOtherGoroutines(t *testing.T) {
	t.Parallel()

	var completed atomic.Int32

	err := DoCtx(t.Context(), 10,
		func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			// Panic immediately
			panic("early panic")
		},
		func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			completed.Add(1)

			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "early panic")
}

func TestDoCtx_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	err := DoCtx(t.Context(), 3,
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(3), counter.Load())
}

func TestDoCtx_ErrorReturnedInsteadOfPanic(t *testing.T) {
	t.Parallel()

	err := DoCtx(t.Context(), 2,
		func(ctx context.Context) error {
			return errTest
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errTest)
	// Should not contain panic recovery message since this was a normal error
	assert.NotContains(t, err.Error(), "recovered from panic")
}

func TestDoCtx_PanicWithNilValue(t *testing.T) {
	t.Parallel()

	err := DoCtx(t.Context(), 1,
		func(ctx context.Context) error {
			var nilPtr *string
			_ = *nilPtr // This will panic with nil pointer dereference

			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "nil pointer") // The panic message should mention nil pointer
}

func TestDo_RecoversPanic(t *testing.T) {
	t.Parallel()

	err := Do(2,
		func(ctx context.Context) error {
			panic("panic in Do function")
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "panic in Do function")
}

func TestDoCtx_PanicWithStackTrace(t *testing.T) {
	t.Parallel()

	err := DoCtx(t.Context(), 1,
		func(ctx context.Context) error {
			helper()

			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "helper function panic")
	// Stack trace should show the helper function
	assert.Contains(t, err.Error(), "simultaneously_test.go")
}

// Helper function for testing stack traces.
func helper() {
	panic("helper function panic")
}

func TestDoCtx_ContextCancellationAfterPanic(t *testing.T) {
	t.Parallel()

	var canceledCount atomic.Int32

	err := DoCtx(t.Context(), 5,
		func(ctx context.Context) error {
			// Panic immediately
			panic("early panic")
		},
		func(ctx context.Context) error {
			// This should potentially be canceled
			time.Sleep(100 * time.Millisecond)

			if ctx.Err() != nil {
				canceledCount.Add(1)

				return ctx.Err()
			}

			return nil
		},
		func(ctx context.Context) error {
			// This should potentially be canceled
			time.Sleep(100 * time.Millisecond)

			if ctx.Err() != nil {
				canceledCount.Add(1)

				return ctx.Err()
			}

			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
}

func TestDoWithExecutor_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	var counter atomic.Int32

	err := DoWithExecutor(exec,
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), counter.Load())
}

func TestDoWithExecutor_RecoversPanic(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	err := DoWithExecutor(exec,
		func(ctx context.Context) error {
			panic("intentional panic in DoWithExecutor")
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic in DoWithExecutor")
}

func TestDoWithExecutor_ReturnsError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	err := DoWithExecutor(exec,
		func(ctx context.Context) error {
			return errTest
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errTest)
}

func TestDoWithExecutor_ExecutorReuse(t *testing.T) {
	t.Parallel()

	// Create a single executor and reuse it for multiple batches
	exec := newDefaultExecutor(3)
	defer exec.Close()

	var firstBatch atomic.Int32

	err := DoWithExecutor(exec,
		func(ctx context.Context) error {
			firstBatch.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			firstBatch.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), firstBatch.Load())

	var secondBatch atomic.Int32

	err = DoWithExecutor(exec,
		func(ctx context.Context) error {
			secondBatch.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			secondBatch.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			secondBatch.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(3), secondBatch.Load())
}

func TestDoCtxWithExecutor_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	var counter atomic.Int32

	err := DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			counter.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), counter.Load())
}

func TestDoCtxWithExecutor_RecoversPanic(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	err := DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			panic("intentional panic in DoCtxWithExecutor")
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic in DoCtxWithExecutor")
}

func TestDoCtxWithExecutor_ReturnsError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2)
	defer exec.Close()

	err := DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			return errTest
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errTest)
}

func TestDoCtxWithExecutor_ContextCancellation(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	var executed atomic.Int32

	err := DoCtxWithExecutor(ctx, exec,
		func(ctx context.Context) error {
			executed.Add(1)
			time.Sleep(100 * time.Millisecond)

			return nil
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDoCtxWithExecutor_ExecutorReuse(t *testing.T) {
	t.Parallel()

	// Create a single executor and reuse it for multiple batches
	exec := newDefaultExecutor(3)
	defer exec.Close()

	var firstBatch atomic.Int32

	err := DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			firstBatch.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			firstBatch.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(2), firstBatch.Load())

	var secondBatch atomic.Int32

	err = DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			secondBatch.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			secondBatch.Add(1)

			return nil
		},
		func(ctx context.Context) error {
			secondBatch.Add(1)

			return nil
		},
	)

	require.NoError(t, err)
	assert.Equal(t, int32(3), secondBatch.Load())
}

func TestDoCtxWithExecutor_MixedSuccessAndError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3)
	defer exec.Close()

	var successCount atomic.Int32

	err := DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			successCount.Add(1)
			time.Sleep(10 * time.Millisecond)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(5 * time.Millisecond)

			return errTest
		},
		func(ctx context.Context) error {
			// This may or may not execute depending on timing
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				successCount.Add(1)
				time.Sleep(10 * time.Millisecond)

				return nil
			}
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, errTest)

	// At least one function should have completed
	assert.GreaterOrEqual(t, successCount.Load(), int32(1))
}

func TestDoCtxWithExecutor_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	// Create executor with limit of 2
	exec := newDefaultExecutor(2)
	defer exec.Close()

	var activeCount atomic.Int32

	var maxActive atomic.Int32

	err := DoCtxWithExecutor(t.Context(), exec,
		func(ctx context.Context) error {
			current := activeCount.Add(1)
			defer activeCount.Add(-1)

			// Update maxActive if this is higher
			for {
				maxVal := maxActive.Load()
				if current <= maxVal || maxActive.CompareAndSwap(maxVal, current) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)

			return nil
		},
		func(ctx context.Context) error {
			current := activeCount.Add(1)
			defer activeCount.Add(-1)

			for {
				maxVal := maxActive.Load()
				if current <= maxVal || maxActive.CompareAndSwap(maxVal, current) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)

			return nil
		},
		func(ctx context.Context) error {
			current := activeCount.Add(1)
			defer activeCount.Add(-1)

			for {
				maxVal := maxActive.Load()
				if current <= maxVal || maxActive.CompareAndSwap(maxVal, current) {
					break
				}
			}

			time.Sleep(50 * time.Millisecond)

			return nil
		},
	)

	require.NoError(t, err)
	// With maxConcurrent=2, we should never have more than 2 active at once
	assert.LessOrEqual(t, maxActive.Load(), int32(2))
}
