package contexts

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAtomic(t *testing.T) {
	t.Parallel()

	t.Run("creates atomic context with initial context", func(t *testing.T) {
		t.Parallel()

		initialCtx := context.WithValue(t.Context(), contextKey("key"), "value")
		atomicCtx, swap := NewAtomic(initialCtx)

		require.NotNil(t, atomicCtx)
		require.NotNil(t, swap)

		// Verify the initial context is stored
		assert.Equal(t, "value", atomicCtx.Value(contextKey("key")))
	})

	t.Run("creates atomic context with background context", func(t *testing.T) {
		t.Parallel()

		atomicCtx, swap := NewAtomic(context.Background()) //nolint:usetesting
		require.NotNil(t, atomicCtx)
		require.NotNil(t, swap)
	})

	t.Run("swap function returns previous context", func(t *testing.T) {
		t.Parallel()

		initialCtx := context.WithValue(t.Context(), contextKey("key"), "initial")
		atomicCtx, swap := NewAtomic(initialCtx)

		newCtx := context.WithValue(t.Context(), contextKey("key"), "new")
		prevCtx := swap(newCtx)

		// Previous context should be the initial one
		assert.Equal(t, "initial", prevCtx.Value(contextKey("key")))

		// Atomic context should now have the new value
		assert.Equal(t, "new", atomicCtx.Value(contextKey("key")))
	})

	t.Run("multiple swaps work correctly", func(t *testing.T) {
		t.Parallel()

		ctx1 := context.WithValue(t.Context(), contextKey("key"), "ctx1")
		atomicCtx, swap := NewAtomic(ctx1)

		ctx2 := context.WithValue(t.Context(), contextKey("key"), "ctx2")
		prev := swap(ctx2)
		assert.Equal(t, "ctx1", prev.Value(contextKey("key")))
		assert.Equal(t, "ctx2", atomicCtx.Value(contextKey("key")))

		ctx3 := context.WithValue(t.Context(), contextKey("key"), "ctx3")
		prev = swap(ctx3)
		assert.Equal(t, "ctx2", prev.Value(contextKey("key")))
		assert.Equal(t, "ctx3", atomicCtx.Value(contextKey("key")))
	})
}

func TestAtomicContext_Deadline(t *testing.T) {
	t.Parallel()

	t.Run("returns no deadline for context without deadline", func(t *testing.T) {
		t.Parallel()

		atomicCtx, _ := NewAtomic(t.Context())
		deadline, ok := atomicCtx.Deadline()

		assert.False(t, ok)
		assert.True(t, deadline.IsZero())
	})

	t.Run("returns deadline from context with deadline", func(t *testing.T) {
		t.Parallel()

		expectedDeadline := time.Now().Add(1 * time.Hour)

		ctx, cancel := context.WithDeadline(t.Context(), expectedDeadline)
		defer cancel()

		atomicCtx, _ := NewAtomic(ctx)
		deadline, ok := atomicCtx.Deadline()

		assert.True(t, ok)
		assert.Equal(t, expectedDeadline, deadline)
	})

	t.Run("returns updated deadline after swap", func(t *testing.T) {
		t.Parallel()

		// Start with no deadline
		atomicCtx, swap := NewAtomic(t.Context())
		_, ok := atomicCtx.Deadline()
		assert.False(t, ok)

		// Swap to context with deadline
		expectedDeadline := time.Now().Add(1 * time.Hour)

		ctx, cancel := context.WithDeadline(t.Context(), expectedDeadline)
		defer cancel()

		swap(ctx)

		deadline, ok := atomicCtx.Deadline()

		assert.True(t, ok)
		assert.Equal(t, expectedDeadline, deadline)
	})

	t.Run("returns no deadline after swapping to context without deadline", func(t *testing.T) {
		t.Parallel()

		// Start with deadline
		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(1*time.Hour))
		defer cancel()

		atomicCtx, swap := NewAtomic(ctx)
		_, ok := atomicCtx.Deadline()
		assert.True(t, ok)

		// Swap to context without deadline
		swap(t.Context())

		_, ok = atomicCtx.Deadline()
		assert.False(t, ok)
	})
}

func TestAtomicContext_Done(t *testing.T) {
	t.Parallel()

	t.Run("returns nil channel for background context", func(t *testing.T) {
		t.Parallel()

		atomicCtx, _ := NewAtomic(context.Background()) //nolint:usetesting
		done := atomicCtx.Done()

		// context.Background().Done() returns nil
		assert.Nil(t, done)
	})

	t.Run("returns done channel that signals on cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		atomicCtx, _ := NewAtomic(ctx)
		done := atomicCtx.Done()

		// Should not be ready initially
		select {
		case <-done:
			t.Fatal("done channel should not be ready before cancel")
		default:
			// Expected
		}

		// Cancel and verify done channel signals
		cancel()

		select {
		case <-done:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("done channel should be ready after cancel")
		}
	})

	t.Run("returns new done channel after swap", func(t *testing.T) {
		t.Parallel()

		ctx1, cancel1 := context.WithCancel(t.Context())
		atomicCtx, swap := NewAtomic(ctx1)

		// Swap to a new context
		ctx2, cancel2 := context.WithCancel(t.Context())
		defer cancel2()

		swap(ctx2)

		done2 := atomicCtx.Done()

		// Cancel the first context - done2 should not be affected
		cancel1()

		select {
		case <-done2:
			t.Fatal("new done channel should not be ready when old context is cancelled")
		default:
			// Expected
		}

		// Cancel the second context - done2 should now be ready
		cancel2()

		select {
		case <-done2:
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("new done channel should be ready after new context is cancelled")
		}
	})
}

func TestAtomicContext_Err(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for active context", func(t *testing.T) {
		t.Parallel()

		atomicCtx, _ := NewAtomic(t.Context())
		assert.NoError(t, atomicCtx.Err())
	})

	t.Run("returns error for cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		atomicCtx, _ := NewAtomic(ctx)

		cancel()

		assert.ErrorIs(t, atomicCtx.Err(), context.Canceled)
	})

	t.Run("returns error for deadline exceeded", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		atomicCtx, _ := NewAtomic(ctx)

		time.Sleep(10 * time.Millisecond)

		assert.ErrorIs(t, atomicCtx.Err(), context.DeadlineExceeded)
	})

	t.Run("returns nil after swapping to fresh context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		atomicCtx, swap := NewAtomic(ctx)

		cancel()
		require.ErrorIs(t, atomicCtx.Err(), context.Canceled)

		// Swap to fresh context
		swap(t.Context())
		require.NoError(t, atomicCtx.Err())
	})

	t.Run("returns error after swapping to cancelled context", func(t *testing.T) {
		t.Parallel()

		atomicCtx, swap := NewAtomic(t.Context())
		require.NoError(t, atomicCtx.Err())

		// Swap to canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		swap(ctx)

		require.ErrorIs(t, atomicCtx.Err(), context.Canceled)
	})
}

func TestAtomicContext_Value(t *testing.T) {
	t.Parallel()

	t.Run("returns value from initial context", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey("key"), "value")
		atomicCtx, _ := NewAtomic(ctx)

		assert.Equal(t, "value", atomicCtx.Value(contextKey("key")))
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		t.Parallel()

		atomicCtx, _ := NewAtomic(t.Context())
		assert.Nil(t, atomicCtx.Value(contextKey("missing")))
	})

	t.Run("returns updated value after swap", func(t *testing.T) {
		t.Parallel()

		ctx1 := context.WithValue(t.Context(), contextKey("key"), "value1")
		atomicCtx, swap := NewAtomic(ctx1)

		assert.Equal(t, "value1", atomicCtx.Value(contextKey("key")))

		// Swap to context with different value
		ctx2 := context.WithValue(t.Context(), contextKey("key"), "value2")
		swap(ctx2)

		assert.Equal(t, "value2", atomicCtx.Value(contextKey("key")))
	})

	t.Run("handles multiple keys correctly", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey("key1"), "value1")
		ctx = context.WithValue(ctx, contextKey("key2"), 42)
		ctx = context.WithValue(ctx, contextKey("key3"), true)

		atomicCtx, _ := NewAtomic(ctx)

		assert.Equal(t, "value1", atomicCtx.Value(contextKey("key1")))
		assert.Equal(t, 42, atomicCtx.Value(contextKey("key2")))
		assert.Equal(t, true, atomicCtx.Value(contextKey("key3")))
	})

	t.Run("handles different value types", func(t *testing.T) {
		t.Parallel()

		type testStruct struct{ Name string }

		ctx := context.WithValue(t.Context(), contextKey("int"), 123)
		ctx = context.WithValue(ctx, contextKey("string"), "test")
		ctx = context.WithValue(ctx, contextKey("struct"), testStruct{Name: "foo"})
		ptr := &testStruct{Name: "bar"}
		ctx = context.WithValue(ctx, contextKey("pointer"), ptr)

		atomicCtx, _ := NewAtomic(ctx)

		assert.Equal(t, 123, atomicCtx.Value(contextKey("int")))
		assert.Equal(t, "test", atomicCtx.Value(contextKey("string")))
		assert.Equal(t, testStruct{Name: "foo"}, atomicCtx.Value(contextKey("struct")))
		assert.Equal(t, ptr, atomicCtx.Value(contextKey("pointer")))
	})
}

func TestAtomicContext_ThreadSafety(t *testing.T) {
	t.Parallel()

	t.Run("concurrent swaps are safe", func(t *testing.T) {
		t.Parallel()

		atomicCtx, swap := NewAtomic(t.Context())

		const numGoroutines = 100

		var waitGroup sync.WaitGroup
		waitGroup.Add(numGoroutines)

		// Concurrently swap contexts
		for i := range numGoroutines {
			go func(idx int) {
				defer waitGroup.Done()

				newCtx := context.WithValue(t.Context(), contextKey("id"), idx)
				swap(newCtx)
			}(i)
		}

		waitGroup.Wait()

		// Should have some valid ID value
		value := atomicCtx.Value(contextKey("id"))
		assert.NotNil(t, value)
	})

	t.Run("concurrent reads and swaps are safe", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey("key"), "initial")
		atomicCtx, swap := NewAtomic(ctx)

		const (
			numReaders = 50
			numWriters = 50
		)

		var waitGroup sync.WaitGroup
		waitGroup.Add(numReaders + numWriters)

		// Start readers
		for range numReaders {
			go func() {
				defer waitGroup.Done()

				for range 100 {
					_ = atomicCtx.Value(contextKey("key"))
					_ = atomicCtx.Err()
					_ = atomicCtx.Done()
					_, _ = atomicCtx.Deadline()
				}
			}()
		}

		// Start writers
		for i := range numWriters {
			go func(idx int) {
				defer waitGroup.Done()

				for range 10 {
					newCtx := context.WithValue(t.Context(), contextKey("key"), idx)
					swap(newCtx)
				}
			}(i)
		}

		waitGroup.Wait()

		// Should complete without panics or races
		value := atomicCtx.Value(contextKey("key"))
		assert.NotNil(t, value)
	})

	t.Run("concurrent deadline checks during swaps", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(1*time.Hour))
		defer cancel()

		atomicCtx, swap := NewAtomic(ctx)

		const numGoroutines = 100

		var waitGroup sync.WaitGroup
		waitGroup.Add(numGoroutines * 2)

		// Readers checking deadline
		for range numGoroutines {
			go func() {
				defer waitGroup.Done()

				for range 100 {
					_, _ = atomicCtx.Deadline()
				}
			}()
		}

		// Writers swapping contexts
		for range numGoroutines {
			go func() {
				defer waitGroup.Done()

				for range 10 {
					newCtx, newCancel := context.WithDeadline(t.Context(), time.Now().Add(2*time.Hour))
					defer newCancel()

					swap(newCtx)
				}
			}()
		}

		waitGroup.Wait()
	})
}

func TestAtomicContext_RealWorldScenarios(t *testing.T) {
	t.Parallel()

	t.Run("replacing cancelled context with fresh one", func(t *testing.T) {
		t.Parallel()

		ctx1, cancel1 := context.WithCancel(t.Context())
		atomicCtx, swap := NewAtomic(ctx1)

		// Cancel the first context
		cancel1()
		require.ErrorIs(t, atomicCtx.Err(), context.Canceled)

		// Replace with fresh context
		ctx2 := t.Context()
		swap(ctx2)

		// Should now be alive again
		assert.NoError(t, atomicCtx.Err())
	})

	t.Run("updating context values during long-running operation", func(t *testing.T) {
		t.Parallel()

		ctx1 := context.WithValue(t.Context(), contextKey("phase"), "initialization")
		atomicCtx, swap := NewAtomic(ctx1)

		assert.Equal(t, "initialization", atomicCtx.Value(contextKey("phase")))

		// Update to processing phase
		ctx2 := context.WithValue(t.Context(), contextKey("phase"), "processing")
		swap(ctx2)
		assert.Equal(t, "processing", atomicCtx.Value(contextKey("phase")))

		// Update to completion phase
		ctx3 := context.WithValue(t.Context(), contextKey("phase"), "completion")
		swap(ctx3)
		assert.Equal(t, "completion", atomicCtx.Value(contextKey("phase")))
	})

	t.Run("extending deadline during long operation", func(t *testing.T) {
		t.Parallel()

		// Start with short deadline
		deadline1 := time.Now().Add(100 * time.Millisecond)

		ctx1, cancel1 := context.WithDeadline(t.Context(), deadline1)
		defer cancel1()

		atomicCtx, swap := NewAtomic(ctx1)
		dl, ok := atomicCtx.Deadline()
		require.True(t, ok)
		assert.Equal(t, deadline1, dl)

		// Extend deadline before it expires
		deadline2 := time.Now().Add(1 * time.Hour)

		ctx2, cancel2 := context.WithDeadline(t.Context(), deadline2)
		defer cancel2()

		swap(ctx2)

		dl, ok = atomicCtx.Deadline()
		require.True(t, ok)
		assert.Equal(t, deadline2, dl)

		// Original deadline should have passed but context is still valid
		time.Sleep(150 * time.Millisecond)
		assert.NoError(t, atomicCtx.Err())
	})

	t.Run("shared atomic context across goroutines", func(t *testing.T) {
		t.Parallel()

		atomicCtx, swap := NewAtomic(t.Context())

		// Start a worker that uses the atomic context
		workerDone := make(chan struct{})

		go func() {
			defer close(workerDone)

			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()

			for {
				select {
				case <-atomicCtx.Done():
					return
				case <-ticker.C:
					// Simulate work
				}
			}
		}()

		// Give worker time to start
		time.Sleep(50 * time.Millisecond)

		// Cancel the context by swapping to canceled context
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		swap(ctx)

		// Worker should stop
		select {
		case <-workerDone:
			// Expected
		case <-time.After(200 * time.Millisecond):
			t.Fatal("worker should have stopped when context was cancelled")
		}
	})
}
