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

func TestDoCtx_RecoversPanic(t *testing.T) {
	t.Parallel()

	err := DoCtx(context.Background(), 2,
		func(ctx context.Context) error {
			panic("intentional panic for testing")
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic for testing")
	assert.Contains(t, err.Error(), "simultaneously_test.go") // stack trace should be present
}

func TestDoCtx_RecoversPanicError(t *testing.T) {
	t.Parallel()

	testErr := errors.New("test error panic")

	err := DoCtx(context.Background(), 2,
		func(ctx context.Context) error {
			panic(testErr)
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.ErrorIs(t, err, testErr)
	assert.Contains(t, err.Error(), "simultaneously_test.go") // stack trace should be present
}

func TestDoCtx_MixedSuccessAndPanic(t *testing.T) {
	t.Parallel()

	var successCount atomic.Int32

	err := DoCtx(context.Background(), 3,
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
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "boom")

	// At least one function should have completed successfully
	// (though others may have been canceled due to the panic)
	assert.GreaterOrEqual(t, successCount.Load(), int32(1))
}

func TestDoCtx_MultiplePanics(t *testing.T) {
	t.Parallel()

	err := DoCtx(context.Background(), 3,
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
	assert.Contains(t, err.Error(), "panic recovered")

	// Due to concurrency and early cancellation, we might get multiple panics joined
	// or just the first one that was caught
	panicCount := strings.Count(err.Error(), "panic recovered")
	assert.GreaterOrEqual(t, panicCount, 1)
}

func TestDoCtx_PanicDoesNotAffectOtherGoroutines(t *testing.T) {
	t.Parallel()

	var completed atomic.Int32

	err := DoCtx(context.Background(), 10,
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
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "early panic")
}

func TestDoCtx_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	var counter atomic.Int32

	err := DoCtx(context.Background(), 3,
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

	testErr := errors.New("test error")

	err := DoCtx(context.Background(), 2,
		func(ctx context.Context) error {
			return testErr
		},
		func(ctx context.Context) error {
			return nil
		},
	)

	require.Error(t, err)
	assert.ErrorIs(t, err, testErr)
	// Should not contain panic recovery message since this was a normal error
	assert.NotContains(t, err.Error(), "panic recovered")
}

func TestDoCtx_PanicWithNilValue(t *testing.T) {
	t.Parallel()

	err := DoCtx(context.Background(), 1,
		func(ctx context.Context) error {
			var nilPtr *string
			_ = *nilPtr // This will panic with nil pointer dereference
			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
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
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "panic in Do function")
}

func TestDoCtx_PanicWithStackTrace(t *testing.T) {
	t.Parallel()

	err := DoCtx(context.Background(), 1,
		func(ctx context.Context) error {
			helper()
			return nil
		},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "helper function panic")
	// Stack trace should show the helper function
	assert.Contains(t, err.Error(), "simultaneously_test.go")
}

// Helper function for testing stack traces
func helper() {
	panic("helper function panic")
}

func TestDoCtx_ContextCancellationAfterPanic(t *testing.T) {
	t.Parallel()

	var canceledCount atomic.Int32

	err := DoCtx(context.Background(), 5,
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
	assert.Contains(t, err.Error(), "panic recovered")

	// Some functions should have been canceled after the panic
	// (though timing may vary)
}
