package future

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errTest      = errors.New("test error")
	errOriginal  = errors.New("original error")
	errTransform = errors.New("transform error")
	errInner     = errors.New("inner error")
	errSource    = errors.New("source error")
)

func TestNew_Success(t *testing.T) {
	t.Parallel()

	fut, promise := New[int]()

	go func() {
		promise.Success(42)
	}()

	result, err := fut.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestNew_Error(t *testing.T) {
	t.Parallel()

	fut, promise := New[int]()

	go func() {
		promise.Failure(errTest)
	}()

	result, err := fut.Await()

	require.Error(t, err)
	assert.Equal(t, errTest, err)
	assert.Equal(t, 0, result)
}

func TestPromise_Complete(t *testing.T) {
	t.Parallel()

	fut, promise := New[int]()

	go func() {
		promise.Complete(42, nil)
	}()

	result, err := fut.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestGo_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	result, err := fut.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestGo_Error(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 0, errTest
	})

	result, err := fut.Await()

	require.Error(t, err)
	assert.Equal(t, errTest, err)
	assert.Equal(t, 0, result)
}

func TestGo_Panic(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		panic("test panic")
	})

	result, err := fut.Await()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic: test panic")
	assert.Contains(t, err.Error(), "stack trace:")
	assert.Equal(t, 0, result)
}

func TestGoContext_Success(t *testing.T) {
	t.Parallel()

	fut := GoContext(t.Context(), func(_ context.Context) (string, error) {
		return "hello", nil
	})

	result, err := fut.Await()

	require.NoError(t, err)
	assert.Equal(t, "hello", result)
}

func TestGoContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	fut := GoContext(ctx, func(ctx context.Context) (string, error) {
		<-ctx.Done()

		return "", ctx.Err()
	})

	// Cancel the context
	cancel()

	result, err := fut.Await()

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, "", result)
}

func TestAwaitContext_Timeout(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		time.Sleep(100 * time.Millisecond)

		return 42, nil
	})

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	result, err := fut.AwaitContext(ctx)

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	assert.Equal(t, 0, result)
}

func TestAwaitContext_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()

	result, err := fut.AwaitContext(ctx)

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestMap_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 21, nil
	})

	mapped := Map(fut, func(val int) (int, error) {
		return val * 2, nil
	})

	result, err := mapped.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestMap_OriginalError(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 0, errOriginal
	})

	mapped := Map(fut, func(val int) (int, error) {
		return val * 2, nil
	})

	result, err := mapped.Await()

	require.Error(t, err)
	assert.Equal(t, errOriginal, err)
	assert.Equal(t, 0, result)
}

func TestMap_TransformError(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 21, nil
	})

	mapped := Map(fut, func(_ int) (int, error) {
		return 0, errTransform
	})

	result, err := mapped.Await()

	require.Error(t, err)
	assert.Equal(t, errTransform, err)
	assert.Equal(t, 0, result)
}

func TestFlatMap_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 21, nil
	})

	flatMapped := FlatMap(fut, func(val int) *Future[int] {
		return Go(func() (int, error) {
			return val * 2, nil
		})
	})

	result, err := flatMapped.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestFlatMap_OriginalError(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 0, errOriginal
	})

	flatMapped := FlatMap(fut, func(val int) *Future[int] {
		return Go(func() (int, error) {
			return val * 2, nil
		})
	})

	result, err := flatMapped.Await()

	require.Error(t, err)
	assert.Equal(t, errOriginal, err)
	assert.Equal(t, 0, result)
}

func TestFlatMap_InnerError(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 21, nil
	})

	flatMapped := FlatMap(fut, func(_ int) *Future[int] {
		return Go(func() (int, error) {
			return 0, errInner
		})
	})

	result, err := flatMapped.Await()

	require.Error(t, err)
	assert.Equal(t, errInner, err)
	assert.Equal(t, 0, result)
}

func TestCombine_Success(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		return 2, nil
	})

	fut3 := Go(func() (int, error) {
		return 3, nil
	})

	combined := Combine(fut1, fut2, fut3)

	results, err := combined.Await()

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, results)
}

func TestCombine_OneError(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		return 0, errTest
	})

	fut3 := Go(func() (int, error) {
		return 3, nil
	})

	combined := Combine(fut1, fut2, fut3)

	results, err := combined.Await()

	require.Error(t, err)
	assert.Equal(t, errTest, err)
	assert.Nil(t, results)
}

func TestCombineNoShortCircuit_Mixed(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		return 0, errTest
	})

	fut3 := Go(func() (int, error) {
		return 3, nil
	})

	combined := CombineNoShortCircuit(fut1, fut2, fut3)

	results, err := combined.Await()

	// When there are errors, Await returns zero value and the error
	require.Error(t, err)
	assert.Contains(t, err.Error(), errTest.Error())
	assert.Nil(t, results)
}

func TestConcurrency(t *testing.T) {
	t.Parallel()

	// Test that multiple futures can run concurrently
	start := time.Now()

	fut1 := Go(func() (int, error) {
		time.Sleep(50 * time.Millisecond)

		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		time.Sleep(50 * time.Millisecond)

		return 2, nil
	})

	fut3 := Go(func() (int, error) {
		time.Sleep(50 * time.Millisecond)

		return 3, nil
	})

	combined := Combine(fut1, fut2, fut3)

	results, err := combined.Await()

	elapsed := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, results)

	// Should complete in ~50ms (concurrent), not ~150ms (sequential)
	assert.Less(t, elapsed, 100*time.Millisecond, "futures should run concurrently")
}

func TestAwait_Idempotent(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	// Call Await multiple times
	result1, err1 := fut.Await()
	require.NoError(t, err1)
	assert.Equal(t, 42, result1)

	result2, err2 := fut.Await()
	require.NoError(t, err2)
	assert.Equal(t, 42, result2)

	result3, err3 := fut.Await()
	require.NoError(t, err3)
	assert.Equal(t, 42, result3)
}

func TestAwaitContext_Idempotent(t *testing.T) {
	t.Parallel()

	fut := Go(func() (string, error) {
		return "hello", nil
	})

	ctx := t.Context()

	// Call AwaitContext multiple times
	result1, err1 := fut.AwaitContext(ctx)
	require.NoError(t, err1)
	assert.Equal(t, "hello", result1)

	result2, err2 := fut.AwaitContext(ctx)
	require.NoError(t, err2)
	assert.Equal(t, "hello", result2)

	result3, err3 := fut.AwaitContext(ctx)
	require.NoError(t, err3)
	assert.Equal(t, "hello", result3)
}

func TestMixedReads_Idempotent(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 99, nil
	})

	ctx := t.Context()

	// Mix different read operations
	result1, err1 := fut.Await()
	require.NoError(t, err1)
	assert.Equal(t, 99, result1)

	result2, err2 := fut.AwaitContext(ctx)
	require.NoError(t, err2)
	assert.Equal(t, 99, result2)

	result3, err3 := fut.Await()
	require.NoError(t, err3)
	assert.Equal(t, 99, result3)
}

func TestConcurrentAwait(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		time.Sleep(10 * time.Millisecond)

		return 42, nil
	})

	// Launch multiple goroutines calling Await concurrently
	const numGoroutines = 10
	results := make(chan int, numGoroutines)
	errors := make(chan error, numGoroutines)

	for range numGoroutines {
		go func() {
			val, err := fut.Await()
			results <- val
			errors <- err
		}()
	}

	// Collect all results
	for range numGoroutines {
		result := <-results
		err := <-errors

		require.NoError(t, err)
		assert.Equal(t, 42, result)
	}
}

func TestConcurrentMixedReads(t *testing.T) {
	t.Parallel()

	fut := Go(func() (string, error) {
		time.Sleep(10 * time.Millisecond)

		return "concurrent", nil
	})

	const numGoroutines = 10
	done := make(chan bool, numGoroutines)

	ctx := t.Context()

	// Mix of Await and AwaitContext calls
	for range 5 {
		go func() {
			val, err := fut.Await()
			assert.NoError(t, err)
			assert.Equal(t, "concurrent", val)
			done <- true
		}()

		go func() {
			val, err := fut.AwaitContext(ctx)
			assert.NoError(t, err)
			assert.Equal(t, "concurrent", val)
			done <- true
		}()
	}

	// Wait for all goroutines
	for range numGoroutines {
		<-done
	}
}

func TestNewError(t *testing.T) {
	t.Parallel()

	fut := NewError[int](errTest)

	result, err := fut.Await()

	require.Error(t, err)
	assert.Equal(t, errTest, err)
	assert.Equal(t, 0, result)
}

func TestMap_NilFuture(t *testing.T) {
	t.Parallel()

	mapped := Map[int, string](nil, func(val int) (string, error) {
		return "test", nil
	})

	result, err := mapped.Await()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil future provided to Map")
	assert.Equal(t, "", result)
}

func TestMap_NilFunction(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	mapped := Map[int, string](fut, nil)

	result, err := mapped.Await()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nil function provided to Map")
	assert.Equal(t, "", result)
}

func TestMapContext_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 21, nil
	})

	mapped := MapContext(t.Context(), fut, func(ctx context.Context, val int) (int, error) {
		return val * 2, nil
	})

	result, err := mapped.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestMapContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	fut := Go(func() (int, error) {
		time.Sleep(50 * time.Millisecond)

		return 42, nil
	})

	mapped := MapContext(ctx, fut, func(ctx context.Context, val int) (int, error) {
		return val * 2, nil
	})

	// Cancel immediately
	cancel()

	result, err := mapped.AwaitContext(ctx)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 0, result)
}

func TestMapContext_PropagatesError(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 0, errSource
	})

	mapped := MapContext(t.Context(), fut, func(ctx context.Context, val int) (int, error) {
		return val * 2, nil
	})

	result, err := mapped.Await()

	require.Error(t, err)
	assert.Equal(t, errSource, err)
	assert.Equal(t, 0, result)
}

func TestFlatMapContext_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 21, nil
	})

	flatMapped := FlatMapContext(t.Context(), fut, func(val int) *Future[int] {
		return Go(func() (int, error) {
			return val * 2, nil
		})
	})

	result, err := flatMapped.Await()

	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestFlatMapContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	fut := Go(func() (int, error) {
		time.Sleep(50 * time.Millisecond)

		return 42, nil
	})

	flatMapped := FlatMapContext(ctx, fut, func(val int) *Future[int] {
		return Go(func() (int, error) {
			return val * 2, nil
		})
	})

	// Cancel immediately
	cancel()

	result, err := flatMapped.AwaitContext(ctx)

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, 0, result)
}

func TestCombineContext_Success(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		return 2, nil
	})

	fut3 := Go(func() (int, error) {
		return 3, nil
	})

	combined := CombineContext(t.Context(), fut1, fut2, fut3)

	results, err := combined.Await()

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, results)
}

func TestCombineContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	fut1 := Go(func() (int, error) {
		time.Sleep(10 * time.Millisecond)

		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		time.Sleep(100 * time.Millisecond)

		return 2, nil
	})

	combined := CombineContext(ctx, fut1, fut2)

	// Cancel after first future completes
	time.Sleep(20 * time.Millisecond)
	cancel()

	results, err := combined.Await()

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
	assert.Nil(t, results)
}

func TestCombineContext_EmptyFutures(t *testing.T) {
	t.Parallel()

	combined := CombineContext[int](t.Context())

	results, err := combined.Await()

	require.NoError(t, err)
	assert.Nil(t, results)
}

func TestCombineContextNoShortCircuit_Success(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		return 2, nil
	})

	fut3 := Go(func() (int, error) {
		return 3, nil
	})

	combined := CombineContextNoShortCircuit(t.Context(), fut1, fut2, fut3)

	results, err := combined.Await()

	require.NoError(t, err)
	assert.Equal(t, []int{1, 2, 3}, results)
}

func TestCombineContextNoShortCircuit_WithErrors(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		return 0, errTest
	})

	fut3 := Go(func() (int, error) {
		return 3, nil
	})

	combined := CombineContextNoShortCircuit(t.Context(), fut1, fut2, fut3)

	results, err := combined.Await()

	require.Error(t, err)
	assert.Contains(t, err.Error(), errTest.Error())
	assert.Nil(t, results)
}

func TestCombineContextNoShortCircuit_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	fut1 := Go(func() (int, error) {
		time.Sleep(10 * time.Millisecond)

		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		time.Sleep(100 * time.Millisecond)

		return 2, nil
	})

	combined := CombineContextNoShortCircuit(ctx, fut1, fut2)

	// Cancel after first future completes
	time.Sleep(20 * time.Millisecond)
	cancel()

	results, err := combined.Await()

	require.Error(t, err)
	// NoShortCircuit collects errors from all futures, so the context error gets joined
	assert.Contains(t, err.Error(), context.Canceled.Error())
	assert.Nil(t, results)
}

func TestToChannel_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	ch := fut.ToChannel()

	result := <-ch

	require.NoError(t, result.Error)
	assert.Equal(t, 42, result.Value)

	// Channel should be closed after receiving the result
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}

func TestToChannel_Error(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 0, errTest
	})

	ch := fut.ToChannel()

	result := <-ch

	require.Error(t, result.Error)
	assert.Equal(t, errTest, result.Error)
	assert.Equal(t, 0, result.Value)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}

func TestToChannel_SelectStatement(t *testing.T) {
	t.Parallel()

	fut1 := Go(func() (int, error) {
		time.Sleep(10 * time.Millisecond)

		return 1, nil
	})

	fut2 := Go(func() (int, error) {
		time.Sleep(20 * time.Millisecond)

		return 2, nil
	})

	ch1 := fut1.ToChannel()
	ch2 := fut2.ToChannel()

	// Should receive from ch1 first
	select {
	case result := <-ch1:
		require.NoError(t, result.Error)
		assert.Equal(t, 1, result.Value)
	case <-ch2:
		t.Fatal("received from ch2 before ch1")
	}

	// Then receive from ch2
	result := <-ch2
	require.NoError(t, result.Error)
	assert.Equal(t, 2, result.Value)
}

func TestToChannelContext_Success(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	ch := fut.ToChannelContext(t.Context())

	result := <-ch

	require.NoError(t, result.Error)
	assert.Equal(t, 42, result.Value)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}

func TestToChannelContext_Error(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 0, errTest
	})

	ch := fut.ToChannelContext(t.Context())

	result := <-ch

	require.Error(t, result.Error)
	assert.Equal(t, errTest, result.Error)
	assert.Equal(t, 0, result.Value)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}

func TestToChannelContext_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	fut := Go(func() (int, error) {
		time.Sleep(100 * time.Millisecond)

		return 42, nil
	})

	ch := fut.ToChannelContext(ctx)

	// Cancel context immediately
	cancel()

	result := <-ch

	require.Error(t, result.Error)
	assert.Equal(t, context.Canceled, result.Error)
	assert.Equal(t, 0, result.Value)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}

func TestToChannelContext_ContextTimeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Millisecond)
	defer cancel()

	fut := Go(func() (int, error) {
		time.Sleep(100 * time.Millisecond)

		return 42, nil
	})

	ch := fut.ToChannelContext(ctx)

	result := <-ch

	require.Error(t, result.Error)
	assert.Equal(t, context.DeadlineExceeded, result.Error)
	assert.Equal(t, 0, result.Value)
}

func TestToChannelContext_SelectStatement(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	fut := Go(func() (int, error) {
		time.Sleep(10 * time.Millisecond)

		return 42, nil
	})

	ch := fut.ToChannelContext(ctx)

	select {
	case result := <-ch:
		require.NoError(t, result.Error)
		assert.Equal(t, 42, result.Value)
	case <-ctx.Done():
		t.Fatal("context canceled before future completed")
	}
}

func TestToChannelContext_NilContext(t *testing.T) {
	t.Parallel()

	fut := Go(func() (int, error) {
		return 42, nil
	})

	// nil context should behave like regular Await (no cancellation)
	ch := fut.ToChannelContext(nil) //nolint:staticcheck

	result := <-ch

	require.NoError(t, result.Error)
	assert.Equal(t, 42, result.Value)

	// Channel should be closed
	_, ok := <-ch
	assert.False(t, ok, "channel should be closed")
}
