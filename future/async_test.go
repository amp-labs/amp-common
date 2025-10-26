package future

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errAsyncTest  = errors.New("test error")
	errAsyncTest1 = errors.New("error 1")
	errAsyncTest2 = errors.New("error 2")
)

// TestAsync_Success verifies that Async executes a function successfully.
func TestAsync_Success(t *testing.T) {
	t.Parallel()

	executed := make(chan struct{}, 1)

	Async(func() {
		executed <- struct{}{}
	})

	// Function should execute asynchronously
	select {
	case <-executed:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}
}

// TestAsync_NonBlocking verifies that Async returns immediately without blocking.
func TestAsync_NonBlocking(t *testing.T) {
	t.Parallel()

	start := time.Now()

	Async(func() {
		time.Sleep(50 * time.Millisecond)
	})

	elapsed := time.Since(start)

	// Should return immediately (much less than 50ms)
	assert.Less(t, elapsed, 20*time.Millisecond, "Async should not block")
}

// TestAsync_MultipleCalls verifies that multiple Async calls can run concurrently.
func TestAsync_MultipleCalls(t *testing.T) {
	t.Parallel()

	const numCalls = 10
	executed := make(chan int, numCalls)
	start := time.Now()

	// Launch multiple async operations that each sleep 50ms
	for i := range numCalls {
		Async(func() {
			time.Sleep(50 * time.Millisecond)
			executed <- i
		})
	}

	// Collect all results
	results := make(map[int]bool)

	for range numCalls {
		select {
		case val := <-executed:
			results[val] = true
		case <-time.After(200 * time.Millisecond):
			t.Fatal("not all async functions completed")
		}
	}

	elapsed := time.Since(start)

	// All functions should have executed
	assert.Len(t, results, numCalls)

	// Should complete in ~50ms (concurrent), not ~500ms (sequential)
	assert.Less(t, elapsed, 150*time.Millisecond, "async calls should run concurrently")
}

// TestAsync_Panic verifies that Async recovers from panics without crashing.
func TestAsync_Panic(t *testing.T) {
	t.Parallel()

	// Async should recover from panic and log the error
	// The test verifies that the panic doesn't crash the program
	require.NotPanics(t, func() {
		Async(func() {
			panic("test panic")
		})

		// Give async function time to panic and recover
		time.Sleep(50 * time.Millisecond)
	})
}

// TestAsync_PanicDoesNotAffectOtherCalls verifies that a panic in one Async
// call doesn't affect other concurrent calls.
func TestAsync_PanicDoesNotAffectOtherCalls(t *testing.T) {
	t.Parallel()

	executed := make(chan int, 2)

	// First call panics
	Async(func() {
		panic("test panic")
	})

	// Second call should still execute normally
	Async(func() {
		executed <- 42
	})

	// Third call should also execute normally
	Async(func() {
		executed <- 43
	})

	// Both non-panicking calls should complete
	results := make([]int, 0, 2)

	for range 2 {
		select {
		case val := <-executed:
			results = append(results, val)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("async functions did not complete")
		}
	}

	assert.Contains(t, results, 42)
	assert.Contains(t, results, 43)
}

// TestAsync_StateModification verifies that Async can safely modify shared state
// when proper synchronization is used.
func TestAsync_StateModification(t *testing.T) {
	t.Parallel()

	var counter int

	var mutex sync.Mutex

	done := make(chan struct{})

	const numIncrements = 100

	for range numIncrements {
		Async(func() {
			mutex.Lock()
			counter++

			if counter == numIncrements {
				close(done)
			}

			mutex.Unlock()
		})
	}

	// Wait for all increments to complete
	select {
	case <-done:
		mutex.Lock()
		assert.Equal(t, numIncrements, counter)
		mutex.Unlock()
	case <-time.After(1 * time.Second):
		t.Fatal("not all async functions completed")
	}
}

// TestAsyncContext_Success verifies that AsyncContext executes a function successfully.
func TestAsyncContext_Success(t *testing.T) {
	t.Parallel()

	executed := make(chan context.Context, 1)

	AsyncContext(t.Context(), func(ctx context.Context) {
		executed <- ctx
	})

	// Function should execute asynchronously
	select {
	case ctx := <-executed:
		assert.NotNil(t, ctx)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}
}

// TestAsyncContext_NonBlocking verifies that AsyncContext returns immediately.
func TestAsyncContext_NonBlocking(t *testing.T) {
	t.Parallel()

	start := time.Now()

	AsyncContext(t.Context(), func(ctx context.Context) {
		time.Sleep(50 * time.Millisecond)
	})

	elapsed := time.Since(start)

	// Should return immediately (much less than 50ms)
	assert.Less(t, elapsed, 20*time.Millisecond, "AsyncContext should not block")
}

// TestAsyncContext_ContextPropagation verifies that context is properly propagated.
func TestAsyncContext_ContextPropagation(t *testing.T) {
	t.Parallel()

	type contextKey string

	const key contextKey = "test-key"

	const value = "test-value"

	ctx := context.WithValue(t.Context(), key, value)
	received := make(chan string, 1)

	AsyncContext(ctx, func(ctx context.Context) {
		val := ctx.Value(key)
		if val != nil {
			if str, ok := val.(string); ok {
				received <- str
			}
		}
	})

	// Context value should be propagated
	select {
	case val := <-received:
		assert.Equal(t, value, val)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not receive context value")
	}
}

// TestAsyncContext_ContextCancellation verifies that context cancellation
// is respected by the async function.
func TestAsyncContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	started := make(chan struct{})
	cancelled := make(chan struct{})

	AsyncContext(ctx, func(ctx context.Context) {
		close(started)

		// Wait for cancellation
		<-ctx.Done()
		close(cancelled)
	})

	// Wait for function to start
	select {
	case <-started:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not start")
	}

	// Cancel the context
	cancel()

	// Function should detect cancellation
	select {
	case <-cancelled:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not detect cancellation")
	}
}

// TestAsyncContext_MultipleCalls verifies that multiple AsyncContext calls
// can run concurrently.
func TestAsyncContext_MultipleCalls(t *testing.T) {
	t.Parallel()

	const numCalls = 10
	executed := make(chan int, numCalls)
	start := time.Now()

	// Launch multiple async operations that each sleep 50ms
	for i := range numCalls {
		AsyncContext(t.Context(), func(ctx context.Context) {
			time.Sleep(50 * time.Millisecond)
			executed <- i
		})
	}

	// Collect all results
	results := make(map[int]bool)

	for range numCalls {
		select {
		case val := <-executed:
			results[val] = true
		case <-time.After(200 * time.Millisecond):
			t.Fatal("not all async functions completed")
		}
	}

	elapsed := time.Since(start)

	// All functions should have executed
	assert.Len(t, results, numCalls)

	// Should complete in ~50ms (concurrent), not ~500ms (sequential)
	assert.Less(t, elapsed, 150*time.Millisecond, "async calls should run concurrently")
}

// TestAsyncContext_Panic verifies that AsyncContext recovers from panics.
func TestAsyncContext_Panic(t *testing.T) {
	t.Parallel()

	// AsyncContext should recover from panic and log the error
	require.NotPanics(t, func() {
		AsyncContext(t.Context(), func(ctx context.Context) {
			panic("test panic")
		})

		// Give async function time to panic and recover
		time.Sleep(50 * time.Millisecond)
	})
}

// TestAsyncContext_PanicDoesNotAffectOtherCalls verifies that a panic in one
// AsyncContext call doesn't affect other concurrent calls.
func TestAsyncContext_PanicDoesNotAffectOtherCalls(t *testing.T) {
	t.Parallel()

	executed := make(chan int, 2)

	// First call panics
	AsyncContext(t.Context(), func(ctx context.Context) {
		panic("test panic")
	})

	// Second call should still execute normally
	AsyncContext(t.Context(), func(ctx context.Context) {
		executed <- 42
	})

	// Third call should also execute normally
	AsyncContext(t.Context(), func(ctx context.Context) {
		executed <- 43
	})

	// Both non-panicking calls should complete
	results := make([]int, 0, 2)

	for range 2 {
		select {
		case val := <-executed:
			results = append(results, val)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("async functions did not complete")
		}
	}

	assert.Contains(t, results, 42)
	assert.Contains(t, results, 43)
}

// TestAsyncContext_NilContext verifies that AsyncContext handles nil context
// gracefully by using a default context.
func TestAsyncContext_NilContext(t *testing.T) {
	t.Parallel()

	executed := make(chan context.Context, 1)

	AsyncContext(nil, func(ctx context.Context) { //nolint:staticcheck
		executed <- ctx
	})

	// Function should execute with a valid context
	select {
	case ctx := <-executed:
		assert.NotNil(t, ctx, "context should not be nil")
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}
}

// TestAsyncContext_CanceledContextBeforeExecution verifies behavior when
// context is already canceled before AsyncContext is called.
func TestAsyncContext_CanceledContextBeforeExecution(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately

	executed := make(chan struct{}, 1)

	AsyncContext(ctx, func(ctx context.Context) {
		// Check if context is already canceled
		select {
		case <-ctx.Done():
			executed <- struct{}{}
		default:
			executed <- struct{}{}
		}
	})

	// Function should still execute (AsyncContext is fire-and-forget)
	select {
	case <-executed:
		// Expected - function executed and detected canceled context
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}
}

// TestAsyncContext_StateModification verifies that AsyncContext can safely
// modify shared state when proper synchronization is used.
func TestAsyncContext_StateModification(t *testing.T) {
	t.Parallel()

	var counter int

	var mutex sync.Mutex

	done := make(chan struct{})

	const numIncrements = 100

	for range numIncrements {
		AsyncContext(t.Context(), func(ctx context.Context) {
			mutex.Lock()
			counter++

			if counter == numIncrements {
				close(done)
			}

			mutex.Unlock()
		})
	}

	// Wait for all increments to complete
	select {
	case <-done:
		mutex.Lock()
		assert.Equal(t, numIncrements, counter)
		mutex.Unlock()
	case <-time.After(1 * time.Second):
		t.Fatal("not all async functions completed")
	}
}

// TestAsyncContext_ContextDeadline verifies that AsyncContext respects
// context deadlines.
func TestAsyncContext_ContextDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	started := make(chan struct{})
	deadlineExceeded := make(chan struct{})

	AsyncContext(ctx, func(ctx context.Context) {
		close(started)

		// Wait for deadline to be exceeded
		<-ctx.Done()

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			close(deadlineExceeded)
		}
	})

	// Wait for function to start
	select {
	case <-started:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not start")
	}

	// Function should detect deadline exceeded
	select {
	case <-deadlineExceeded:
		// Expected
	case <-time.After(200 * time.Millisecond):
		t.Fatal("async function did not detect deadline exceeded")
	}
}

// TestAsync_vs_AsyncContext verifies that both functions work correctly
// side by side.
func TestAsync_vs_AsyncContext(t *testing.T) {
	t.Parallel()

	asyncExecuted := make(chan struct{}, 1)
	asyncContextExecuted := make(chan struct{}, 1)

	Async(func() {
		asyncExecuted <- struct{}{}
	})

	AsyncContext(t.Context(), func(ctx context.Context) {
		asyncContextExecuted <- struct{}{}
	})

	// Both should execute
	select {
	case <-asyncExecuted:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Async function was not executed")
	}

	select {
	case <-asyncContextExecuted:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("AsyncContext function was not executed")
	}
}

// TestAsyncWithError_Success verifies that AsyncWithError executes a function
// successfully when no error is returned.
func TestAsyncWithError_Success(t *testing.T) {
	t.Parallel()

	executed := make(chan struct{}, 1)

	AsyncWithError(func() error {
		executed <- struct{}{}

		return nil
	})

	// Function should execute asynchronously
	select {
	case <-executed:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}
}

// TestAsyncWithError_Error verifies that AsyncWithError logs errors returned
// by the function without crashing.
func TestAsyncWithError_Error(t *testing.T) {
	t.Parallel()

	executed := make(chan error, 1)

	AsyncWithError(func() error {
		executed <- errAsyncTest

		return errAsyncTest
	})

	// Function should execute and return error
	select {
	case err := <-executed:
		assert.Equal(t, errAsyncTest, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}

	// Give time for error logging
	time.Sleep(50 * time.Millisecond)
}

// TestAsyncWithError_NonBlocking verifies that AsyncWithError returns
// immediately without blocking.
func TestAsyncWithError_NonBlocking(t *testing.T) {
	t.Parallel()

	start := time.Now()

	AsyncWithError(func() error {
		time.Sleep(50 * time.Millisecond)

		return nil
	})

	elapsed := time.Since(start)

	// Should return immediately (much less than 50ms)
	assert.Less(t, elapsed, 20*time.Millisecond, "AsyncWithError should not block")
}

// TestAsyncWithError_Panic verifies that AsyncWithError recovers from panics
// without crashing.
func TestAsyncWithError_Panic(t *testing.T) {
	t.Parallel()

	// AsyncWithError should recover from panic and log the error
	require.NotPanics(t, func() {
		AsyncWithError(func() error {
			panic("test panic")
		})

		// Give async function time to panic and recover
		time.Sleep(50 * time.Millisecond)
	})
}

// TestAsyncWithError_MultipleCalls verifies that multiple AsyncWithError calls
// can run concurrently.
func TestAsyncWithError_MultipleCalls(t *testing.T) {
	t.Parallel()

	const numCalls = 10
	executed := make(chan int, numCalls)
	start := time.Now()

	// Launch multiple async operations that each sleep 50ms
	for i := range numCalls {
		AsyncWithError(func() error {
			time.Sleep(50 * time.Millisecond)
			executed <- i

			return nil
		})
	}

	// Collect all results
	results := make(map[int]bool)

	for range numCalls {
		select {
		case val := <-executed:
			results[val] = true
		case <-time.After(200 * time.Millisecond):
			t.Fatal("not all async functions completed")
		}
	}

	elapsed := time.Since(start)

	// All functions should have executed
	assert.Len(t, results, numCalls)

	// Should complete in ~50ms (concurrent), not ~500ms (sequential)
	assert.Less(t, elapsed, 150*time.Millisecond, "async calls should run concurrently")
}

// TestAsyncWithError_MixedSuccessAndErrors verifies that AsyncWithError handles
// a mix of successful and error-returning functions.
func TestAsyncWithError_MixedSuccessAndErrors(t *testing.T) {
	t.Parallel()

	results := make(chan error, 4)

	AsyncWithError(func() error {
		results <- nil

		return nil
	})

	AsyncWithError(func() error {
		results <- errAsyncTest1

		return errAsyncTest1
	})

	AsyncWithError(func() error {
		results <- nil

		return nil
	})

	AsyncWithError(func() error {
		results <- errAsyncTest2

		return errAsyncTest2
	})

	// Collect all results
	var successCount, errorCount int

	for range 4 {
		select {
		case err := <-results:
			if err == nil {
				successCount++
			} else {
				errorCount++
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("not all async functions completed")
		}
	}

	assert.Equal(t, 2, successCount)
	assert.Equal(t, 2, errorCount)
}

// TestAsyncContextWithError_Success verifies that AsyncContextWithError executes
// a function successfully when no error is returned.
func TestAsyncContextWithError_Success(t *testing.T) {
	t.Parallel()

	executed := make(chan context.Context, 1)

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		executed <- ctx

		return nil
	})

	// Function should execute asynchronously
	select {
	case ctx := <-executed:
		assert.NotNil(t, ctx)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}
}

// TestAsyncContextWithError_Error verifies that AsyncContextWithError logs
// errors returned by the function without crashing.
func TestAsyncContextWithError_Error(t *testing.T) {
	t.Parallel()

	executed := make(chan error, 1)

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		executed <- errAsyncTest

		return errAsyncTest
	})

	// Function should execute and return error
	select {
	case err := <-executed:
		assert.Equal(t, errAsyncTest, err)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function was not executed")
	}

	// Give time for error logging
	time.Sleep(50 * time.Millisecond)
}

// TestAsyncContextWithError_NonBlocking verifies that AsyncContextWithError
// returns immediately without blocking.
func TestAsyncContextWithError_NonBlocking(t *testing.T) {
	t.Parallel()

	start := time.Now()

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		time.Sleep(50 * time.Millisecond)

		return nil
	})

	elapsed := time.Since(start)

	// Should return immediately (much less than 50ms)
	assert.Less(t, elapsed, 20*time.Millisecond, "AsyncContextWithError should not block")
}

// TestAsyncContextWithError_ContextPropagation verifies that context is properly
// propagated to AsyncContextWithError functions.
func TestAsyncContextWithError_ContextPropagation(t *testing.T) {
	t.Parallel()

	type contextKey string

	const key contextKey = "test-key"

	const value = "test-value"

	ctx := context.WithValue(t.Context(), key, value)
	received := make(chan string, 1)

	AsyncContextWithError(ctx, func(ctx context.Context) error {
		val := ctx.Value(key)
		if val != nil {
			if str, ok := val.(string); ok {
				received <- str
			}
		}

		return nil
	})

	// Context value should be propagated
	select {
	case val := <-received:
		assert.Equal(t, value, val)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not receive context value")
	}
}

// TestAsyncContextWithError_ContextCancellation verifies that context
// cancellation is respected by AsyncContextWithError functions.
func TestAsyncContextWithError_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	started := make(chan struct{})
	cancelled := make(chan struct{})

	AsyncContextWithError(ctx, func(ctx context.Context) error {
		close(started)

		// Wait for cancellation
		<-ctx.Done()
		close(cancelled)

		return ctx.Err()
	})

	// Wait for function to start
	select {
	case <-started:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not start")
	}

	// Cancel the context
	cancel()

	// Function should detect cancellation
	select {
	case <-cancelled:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not detect cancellation")
	}
}

// TestAsyncContextWithError_Panic verifies that AsyncContextWithError recovers
// from panics without crashing.
func TestAsyncContextWithError_Panic(t *testing.T) {
	t.Parallel()

	// AsyncContextWithError should recover from panic and log the error
	require.NotPanics(t, func() {
		AsyncContextWithError(t.Context(), func(ctx context.Context) error {
			panic("test panic")
		})

		// Give async function time to panic and recover
		time.Sleep(50 * time.Millisecond)
	})
}

// TestAsyncContextWithError_MultipleCalls verifies that multiple
// AsyncContextWithError calls can run concurrently.
func TestAsyncContextWithError_MultipleCalls(t *testing.T) {
	t.Parallel()

	const numCalls = 10
	executed := make(chan int, numCalls)
	start := time.Now()

	// Launch multiple async operations that each sleep 50ms
	for i := range numCalls {
		AsyncContextWithError(t.Context(), func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			executed <- i

			return nil
		})
	}

	// Collect all results
	results := make(map[int]bool)

	for range numCalls {
		select {
		case val := <-executed:
			results[val] = true
		case <-time.After(200 * time.Millisecond):
			t.Fatal("not all async functions completed")
		}
	}

	elapsed := time.Since(start)

	// All functions should have executed
	assert.Len(t, results, numCalls)

	// Should complete in ~50ms (concurrent), not ~500ms (sequential)
	assert.Less(t, elapsed, 150*time.Millisecond, "async calls should run concurrently")
}

// TestAsyncContextWithError_ContextDeadline verifies that AsyncContextWithError
// respects context deadlines.
func TestAsyncContextWithError_ContextDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	started := make(chan struct{})
	deadlineExceeded := make(chan error)

	AsyncContextWithError(ctx, func(ctx context.Context) error {
		close(started)

		// Wait for deadline to be exceeded
		<-ctx.Done()

		err := ctx.Err()
		deadlineExceeded <- err

		return err
	})

	// Wait for function to start
	select {
	case <-started:
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Fatal("async function did not start")
	}

	// Function should detect deadline exceeded
	select {
	case err := <-deadlineExceeded:
		require.ErrorIs(t, err, context.DeadlineExceeded)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("async function did not detect deadline exceeded")
	}
}

// TestAsyncContextWithError_MixedSuccessAndErrors verifies that
// AsyncContextWithError handles a mix of successful and error-returning functions.
func TestAsyncContextWithError_MixedSuccessAndErrors(t *testing.T) {
	t.Parallel()

	results := make(chan error, 4)

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		results <- nil

		return nil
	})

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		results <- errAsyncTest1

		return errAsyncTest1
	})

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		results <- nil

		return nil
	})

	AsyncContextWithError(t.Context(), func(ctx context.Context) error {
		results <- errAsyncTest2

		return errAsyncTest2
	})

	// Collect all results
	var successCount, errorCount int

	for range 4 {
		select {
		case err := <-results:
			if err == nil {
				successCount++
			} else {
				errorCount++
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("not all async functions completed")
		}
	}

	assert.Equal(t, 2, successCount)
	assert.Equal(t, 2, errorCount)
}
