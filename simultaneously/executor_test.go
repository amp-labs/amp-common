package simultaneously //nolint:testpackage // Testing internal functions

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/should"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errExecutorTest  = errors.New("executor test error")
	errExistingError = errors.New("existing error")
	errSingleError   = errors.New("single error")
	errError3        = errors.New("error 3")
	errFirst         = errors.New("first")
	errSecond        = errors.New("second")
)

func TestNewDefaultExecutor_PublicAPI(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                string
		maxConcurrent       int
		expectedConcurrency int
	}{
		{
			name:                "positive concurrency",
			maxConcurrent:       5,
			expectedConcurrency: 5,
		},
		{
			name:                "zero defaults to 1",
			maxConcurrent:       0,
			expectedConcurrency: 1,
		},
		{
			name:                "negative defaults to 1",
			maxConcurrent:       -1,
			expectedConcurrency: 1,
		},
		{
			name:                "single concurrent",
			maxConcurrent:       1,
			expectedConcurrency: 1,
		},
		{
			name:                "high concurrency",
			maxConcurrent:       100,
			expectedConcurrency: 100,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			exec := NewDefaultExecutor(testCase.maxConcurrent)
			defer should.Close(exec, "closing executor")

			require.NotNil(t, exec)

			// Verify it's the expected implementation
			defaultExec, ok := exec.(*defaultExecutor)
			require.True(t, ok, "should return *defaultExecutor")
			assert.Equal(t, testCase.expectedConcurrency, defaultExec.maxConcurrent)
			assert.NotNil(t, defaultExec.sem)
			assert.Equal(t, testCase.expectedConcurrency, cap(defaultExec.sem))
			assert.NotNil(t, defaultExec.closed)
			assert.False(t, defaultExec.closed.Load())
		})
	}
}

func TestNewDefaultExecutor_CanExecuteWork(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(3)
	defer should.Close(exec, "closing executor")

	// Test that it can actually execute work
	var executed atomic.Int32

	done := make(chan error, 3)

	for range 3 {
		exec.Go(func(ctx context.Context) error {
			executed.Add(1)

			return nil
		}, func(err error) {
			done <- err
		})
	}

	// Wait for all executions
	for range 3 {
		err := <-done
		require.NoError(t, err)
	}

	assert.Equal(t, int32(3), executed.Load())
}

func TestNewDefaultExecutor_CloseWorks(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(5)

	// Should close without error
	err := exec.Close()
	require.NoError(t, err)

	// Second close should return error
	err = exec.Close()
	assert.ErrorIs(t, err, ErrExecutorClosed)
}

func TestNewDefaultExecutor_RejectsWorkAfterClose(t *testing.T) {
	t.Parallel()

	exec := NewDefaultExecutor(3)
	should.Close(exec, "closing executor")

	// Try to execute after close
	done := make(chan error, 1)

	exec.Go(func(ctx context.Context) error {
		return nil
	}, func(err error) {
		done <- err
	})

	// Should receive error immediately
	err := <-done
	assert.ErrorIs(t, err, ErrExecutorClosed)
}

func TestNewDefaultExecutor_Internal(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(5, 5)
	defer should.Close(exec, "closing executor")

	require.NotNil(t, exec)
	assert.Equal(t, 5, exec.maxConcurrent)
	assert.NotNil(t, exec.sem)
	assert.NotNil(t, exec.closed)
	assert.False(t, exec.closed.Load())
}

func TestNewDefaultExecutor_EdgeCases(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		maxConcurrent int
		itemCount     int
		expectedMax   int
	}{
		{
			name:          "both zero",
			maxConcurrent: 0,
			itemCount:     0,
			expectedMax:   1,
		},
		{
			name:          "negative max with positive items",
			maxConcurrent: -1,
			itemCount:     5,
			expectedMax:   5,
		},
		{
			name:          "max exceeds items",
			maxConcurrent: 10,
			itemCount:     5,
			expectedMax:   5,
		},
		{
			name:          "negative items with positive max",
			maxConcurrent: 5,
			itemCount:     -1,
			expectedMax:   1,
		},
		{
			name:          "both negative",
			maxConcurrent: -1,
			itemCount:     -1,
			expectedMax:   1,
		},
		{
			name:          "max equals items",
			maxConcurrent: 5,
			itemCount:     5,
			expectedMax:   5,
		},
		{
			name:          "max less than items",
			maxConcurrent: 3,
			itemCount:     10,
			expectedMax:   3,
		},
		{
			name:          "zero max with positive items",
			maxConcurrent: 0,
			itemCount:     10,
			expectedMax:   10,
		},
		{
			name:          "positive max with zero items",
			maxConcurrent: 5,
			itemCount:     0,
			expectedMax:   1,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			exec := newDefaultExecutor(testCase.maxConcurrent, testCase.itemCount)
			defer should.Close(exec, "closing executor")

			require.NotNil(t, exec)
			assert.Equal(t, testCase.expectedMax, exec.maxConcurrent, "maxConcurrent should match expected value")
			assert.Equal(t, testCase.expectedMax, cap(exec.sem), "semaphore capacity should match maxConcurrent")

			// Verify semaphore is pre-filled with correct number of tokens
			tokensAvailable := 0

			for range testCase.expectedMax {
				select {
				case <-exec.sem:
					tokensAvailable++
				default:
					break
				}
			}

			assert.Equal(t, testCase.expectedMax, tokensAvailable, "semaphore should be pre-filled with expected tokens")

			// Return tokens for cleanup

			for range tokensAvailable {
				exec.sem <- struct{}{}
			}
		})
	}
}

func TestDefaultExecutor_Go(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	var executed atomic.Bool

	done := make(chan error, 1)

	exec.Go(func(ctx context.Context) error {
		executed.Store(true)

		return nil
	}, func(err error) {
		done <- err
	})

	err := <-done
	require.NoError(t, err)
	assert.True(t, executed.Load())
}

func TestDefaultExecutor_GoContext_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3, 3)
	defer should.Close(exec, "closing executor")

	var executedCount atomic.Int32

	done := make(chan error, 1)

	exec.GoContext(t.Context(), func(ctx context.Context) error {
		executedCount.Add(1)

		return nil
	}, func(err error) {
		done <- err
	})

	err := <-done
	require.NoError(t, err)
	assert.Equal(t, int32(1), executedCount.Load())
}

func TestDefaultExecutor_GoContext_WithError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	done := make(chan error, 1)

	exec.GoContext(t.Context(), func(ctx context.Context) error {
		return errExecutorTest
	}, func(err error) {
		done <- err
	})

	err := <-done
	assert.ErrorIs(t, err, errExecutorTest)
}

func TestDefaultExecutor_GoContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	done := make(chan error, 1)

	exec.GoContext(ctx, func(ctx context.Context) error {
		return nil
	}, func(err error) {
		done <- err
	})

	err := <-done
	assert.ErrorIs(t, err, context.Canceled)
}

func TestDefaultExecutor_GoContext_ClosedExecutor(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	should.Close(exec, "closing executor")

	done := make(chan error, 1)

	exec.GoContext(t.Context(), func(ctx context.Context) error {
		return nil
	}, func(err error) {
		done <- err
	})

	err := <-done
	assert.ErrorIs(t, err, ErrExecutorClosed)
}

func TestDefaultExecutor_GoContext_ClosedWhileWaiting(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)

	// Block the only available slot
	blocker := make(chan struct{})
	done1 := make(chan error, 1)

	exec.GoContext(t.Context(), func(ctx context.Context) error {
		<-blocker // Wait for signal

		return nil
	}, func(err error) {
		done1 <- err
	})

	// Give the first goroutine time to start
	time.Sleep(50 * time.Millisecond)

	// Try to execute another function - it should block waiting for a slot
	done2 := make(chan error, 1)

	go func() {
		exec.GoContext(t.Context(), func(ctx context.Context) error {
			return nil
		}, func(err error) {
			done2 <- err
		})
	}()

	// Give it time to start waiting on the semaphore
	time.Sleep(50 * time.Millisecond)

	// Unblock the first goroutine before closing
	close(blocker)

	// Wait for first to complete
	err1 := <-done1
	require.NoError(t, err1)

	// Close the executor
	err := exec.Close()
	require.NoError(t, err)

	// Second should either complete successfully or get ErrExecutorClosed depending on timing
	err2 := <-done2
	if err2 != nil {
		assert.ErrorIs(t, err2, ErrExecutorClosed)
	}
}

func TestDefaultExecutor_GoContext_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	maxConcurrent := 2

	exec := newDefaultExecutor(maxConcurrent, 5) // 5 tasks to run
	defer should.Close(exec, "closing executor")

	var activeCount atomic.Int32

	var maxActive atomic.Int32

	done := make(chan error, 5)

	for i := range 5 {
		_ = i

		exec.GoContext(t.Context(), func(ctx context.Context) error {
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
		}, func(err error) {
			done <- err
		})
	}

	// Wait for all to complete
	for range 5 {
		err := <-done
		require.NoError(t, err)
	}

	// Should never exceed maxConcurrent
	assert.LessOrEqual(t, maxActive.Load(), int32(maxConcurrent))
}

func TestDefaultExecutor_Close(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3, 3)

	err := exec.Close()
	require.NoError(t, err)
	assert.True(t, exec.closed.Load())
}

func TestDefaultExecutor_Close_AlreadyClosed(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)

	err := exec.Close()
	require.NoError(t, err)

	// Try to close again
	err = exec.Close()
	assert.ErrorIs(t, err, ErrExecutorClosed)
}

func TestDefaultExecutor_Close_WaitsForInFlight(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(3, 3)

	var completed atomic.Int32

	done := make(chan error, 3)

	// Start some work
	for range 3 {
		exec.GoContext(t.Context(), func(ctx context.Context) error {
			time.Sleep(100 * time.Millisecond)
			completed.Add(1)

			return nil
		}, func(err error) {
			done <- err
		})
	}

	// Give goroutines time to start
	time.Sleep(20 * time.Millisecond)

	// Close should wait for all in-flight operations
	startTime := time.Now()
	err := exec.Close()
	duration := time.Since(startTime)

	require.NoError(t, err)
	assert.GreaterOrEqual(t, duration, 70*time.Millisecond) // Should have waited

	// All operations should have completed
	for range 3 {
		err := <-done
		require.NoError(t, err)
	}

	assert.Equal(t, int32(3), completed.Load())
}

func TestDefaultExecutor_ExecuteCallback_NilContext(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	var executed atomic.Bool

	err := exec.executeCallback(t.Context(), func(ctx context.Context) error {
		assert.NotNil(t, ctx) // Should be replaced with background context
		executed.Store(true)

		return nil
	})

	require.NoError(t, err)
	assert.True(t, executed.Load())
}

func TestDefaultExecutor_ExecuteCallback_CanceledContext(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err := exec.executeCallback(ctx, func(ctx context.Context) error {
		t.Fatal("callback should not be executed")

		return nil
	})

	assert.ErrorIs(t, err, context.Canceled)
}

func TestDefaultExecutor_ExecuteCallback_WithPanic(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	err := exec.executeCallback(t.Context(), func(ctx context.Context) error {
		panic("intentional panic")
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic")
}

func TestDefaultExecutor_ExecuteCallback_PanicWithError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	err := exec.executeCallback(t.Context(), func(ctx context.Context) error {
		panic(errExecutorTest)
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.ErrorIs(t, err, errExecutorTest)
}

func TestDefaultExecutor_RecoverPanic_NoPanic(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	var err error

	// No panic
	func() {
		defer exec.recoverPanic(&err)
	}()

	assert.NoError(t, err)
}

func TestDefaultExecutor_RecoverPanic_WithPanic(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	var err error

	func() {
		defer exec.recoverPanic(&err)
		panic("test panic")
	}()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "test panic")
}

func TestDefaultExecutor_RecoverPanic_PanicWithExistingError(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	err := errExistingError

	func() {
		defer exec.recoverPanic(&err)
		panic("panic error")
	}()

	require.Error(t, err)
	// Should contain both errors
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "panic error")
	assert.Contains(t, err.Error(), "existing error")
}

func TestDefaultExecutor_RecoverPanic_NilPointerPanic(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	var err error

	func() {
		defer exec.recoverPanic(&err)

		var nilPtr *string

		_ = *nilPtr // This will panic
	}()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
}

func TestCombineErrors_NoErrors(t *testing.T) {
	t.Parallel()

	err := combineErrors([]error{})
	assert.NoError(t, err)
}

func TestCombineErrors_SingleError(t *testing.T) {
	t.Parallel()

	err := combineErrors([]error{errSingleError})

	assert.Equal(t, errSingleError, err)
}

func TestCombineErrors_MultipleErrors(t *testing.T) {
	t.Parallel()

	combined := combineErrors([]error{errExecutorTest, errError3, errExistingError})

	require.Error(t, combined)
	assert.Contains(t, combined.Error(), "executor test error")
	assert.Contains(t, combined.Error(), "error 3")
	assert.Contains(t, combined.Error(), "existing error")

	// Should be unwrappable
	require.ErrorIs(t, combined, errExecutorTest)
	require.ErrorIs(t, combined, errError3)
	require.ErrorIs(t, combined, errExistingError)
}

func TestCombineErrors_TwoErrors(t *testing.T) {
	t.Parallel()

	combined := combineErrors([]error{errFirst, errSecond})

	require.Error(t, combined)
	assert.Contains(t, combined.Error(), "first")
	assert.Contains(t, combined.Error(), "second")
}

func TestDefaultExecutor_GoContext_MultipleCallbacks(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(5, 5)
	defer should.Close(exec, "closing executor")

	var completedCount atomic.Int32

	done := make(chan error, 10)

	// Execute multiple callbacks
	for range 10 {
		exec.GoContext(t.Context(), func(ctx context.Context) error {
			completedCount.Add(1)
			time.Sleep(10 * time.Millisecond)

			return nil
		}, func(err error) {
			done <- err
		})
	}

	// Collect all results
	for range 10 {
		err := <-done
		require.NoError(t, err)
	}

	assert.Equal(t, int32(10), completedCount.Load())
}

func TestDefaultExecutor_GoContext_ContextDeadline(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	done := make(chan error, 1)

	exec.GoContext(ctx, func(ctx context.Context) error {
		// Sleep in small increments to check context
		for range 10 {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			time.Sleep(20 * time.Millisecond)
		}

		return nil
	}, func(err error) {
		done <- err
	})

	err := <-done
	// Should get context deadline exceeded since we check ctx.Err() in the callback
	if err != nil {
		assert.ErrorIs(t, err, context.DeadlineExceeded)
	}
}

func TestDefaultExecutor_StressTest(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(10, 10)
	defer should.Close(exec, "closing executor")

	numCallbacks := 100
	done := make(chan error, numCallbacks)

	var successCount atomic.Int32

	for i := range numCallbacks {
		exec.GoContext(t.Context(), func(ctx context.Context) error {
			// Simulate work
			time.Sleep(time.Millisecond * time.Duration(i%10))
			successCount.Add(1)

			return nil
		}, func(err error) {
			done <- err
		})
	}

	// Collect all results
	for range numCallbacks {
		err := <-done
		require.NoError(t, err)
	}

	assert.Equal(t, int32(numCallbacks), successCount.Load())
}

func TestDefaultExecutor_GoContext_SemaphoreTokenReturn(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	done := make(chan error, 4)

	// Execute callbacks that return quickly
	for range 4 {
		exec.GoContext(t.Context(), func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)

			return nil
		}, func(err error) {
			done <- err
		})
	}

	// All should complete successfully, proving tokens are returned
	for range 4 {
		err := <-done
		require.NoError(t, err)
	}
}

func TestDefaultExecutor_ExecuteCallback_ContextPassthrough(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(1, 1)
	defer should.Close(exec, "closing executor")

	type contextKey string

	key := contextKey("test-key")
	expectedValue := "test-value"

	ctx := context.WithValue(t.Context(), key, expectedValue)

	var receivedValue string

	err := exec.executeCallback(ctx, func(ctx context.Context) error {
		val, ok := ctx.Value(key).(string)
		if !ok {
			t.Fatal("context value not found or wrong type")
		}

		receivedValue = val

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, expectedValue, receivedValue)
}
