package contexts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithIgnoreLifecycle(t *testing.T) {
	t.Parallel()

	t.Run("returns nil when given nil context", func(t *testing.T) {
		t.Parallel()

		result := WithIgnoreLifecycle(nil) //nolint:staticcheck // Testing nil context behavior
		assert.Nil(t, result)
	})

	t.Run("returns non-nil context when given valid context", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		result := WithIgnoreLifecycle(ctx)
		assert.NotNil(t, result)
	})

	t.Run("wraps context in lifecycleInsensitiveContext", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		result := WithIgnoreLifecycle(ctx)

		// Verify it's the right type
		_, ok := result.(*lifecycleInsensitiveContext)
		assert.True(t, ok)
	})
}

func TestLifecycleInsensitiveContext_Done(t *testing.T) {
	t.Parallel()

	t.Run("returns channel that never closes", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		wrapped := WithIgnoreLifecycle(ctx)

		done := wrapped.Done()
		assert.NotNil(t, done)

		// Verify the channel doesn't close immediately
		select {
		case <-done:
			t.Fatal("Done channel should never close")
		case <-time.After(10 * time.Millisecond):
		}
	})

	t.Run("remains open even when parent context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		wrapped := WithIgnoreLifecycle(ctx)

		// Cancel the parent context
		cancel()

		// Verify parent is canceled
		select {
		case <-ctx.Done():
			// Expected
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Parent context should be cancelled")
		}
		// Verify wrapped context's Done channel still doesn't close
		select {
		case <-wrapped.Done():
			t.Fatal("Wrapped context's Done channel should never close")
		case <-time.After(10 * time.Millisecond):
		}
	})

	t.Run("remains open even when parent deadline expires", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		wrapped := WithIgnoreLifecycle(ctx)

		// Wait for parent to expire
		time.Sleep(10 * time.Millisecond)

		// Verify parent is done
		select {
		case <-ctx.Done():
			// Expected
		default:
			t.Fatal("Parent context should be done")
		}
		// Verify wrapped context's Done channel still doesn't close
		select {
		case <-wrapped.Done():
			t.Fatal("Wrapped context's Done channel should never close")
		case <-time.After(10 * time.Millisecond):
		}
	})

	t.Run("returns same shared channel instance", func(t *testing.T) {
		t.Parallel()

		ctx1 := WithIgnoreLifecycle(t.Context())
		ctx2 := WithIgnoreLifecycle(t.Context())

		// Both should return the same neverClosed channel
		// Verify both contexts return the same channel instance
		done1 := ctx1.Done()
		done2 := ctx2.Done()

		// Use reflect to compare the underlying channel pointers
		// This ensures they share the same channel
		assert.Equal(t, done1, done2)
	})
}

func TestLifecycleInsensitiveContext_Err(t *testing.T) {
	t.Parallel()

	t.Run("always returns nil", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		wrapped := WithIgnoreLifecycle(ctx)

		assert.NoError(t, wrapped.Err())
	})

	t.Run("returns nil even when parent context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		wrapped := WithIgnoreLifecycle(ctx)

		cancel()

		// Verify parent returns error
		require.Error(t, ctx.Err())
		assert.Equal(t, context.Canceled, ctx.Err())

		// Verify wrapped context returns nil
		assert.NoError(t, wrapped.Err())
	})

	t.Run("returns nil even when parent deadline expires", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		wrapped := WithIgnoreLifecycle(ctx)

		// Wait for parent to expire
		time.Sleep(10 * time.Millisecond)

		// Verify parent returns error
		require.Error(t, ctx.Err())
		assert.Equal(t, context.DeadlineExceeded, ctx.Err())

		// Verify wrapped context returns nil
		assert.NoError(t, wrapped.Err())
	})
}

func TestLifecycleInsensitiveContext_Deadline(t *testing.T) {
	t.Parallel()

	t.Run("returns zero time and false", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		wrapped := WithIgnoreLifecycle(ctx)

		deadline, ok := wrapped.Deadline()
		assert.False(t, ok)
		assert.True(t, deadline.IsZero())
	})

	t.Run("returns zero time even when parent has deadline", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(1*time.Hour))
		defer cancel()

		wrapped := WithIgnoreLifecycle(ctx)

		// Verify parent has deadline
		parentDeadline, parentOk := ctx.Deadline()
		assert.True(t, parentOk)
		assert.False(t, parentDeadline.IsZero())

		// Verify wrapped context has no deadline
		deadline, ok := wrapped.Deadline()
		assert.False(t, ok)
		assert.True(t, deadline.IsZero())
	})

	t.Run("returns zero time even when parent has timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
		defer cancel()

		wrapped := WithIgnoreLifecycle(ctx)

		// Verify parent has deadline
		_, parentOk := ctx.Deadline()
		assert.True(t, parentOk)

		// Verify wrapped context has no deadline
		deadline, ok := wrapped.Deadline()
		assert.False(t, ok)
		assert.True(t, deadline.IsZero())
	})
}

func TestLifecycleInsensitiveContext_Value(t *testing.T) {
	t.Parallel()

	t.Run("retrieves values from parent context", func(t *testing.T) {
		t.Parallel()

		type testKeyType string

		key := testKeyType("testKey")
		value := "testValue"

		ctx := context.WithValue(t.Context(), key, value)
		wrapped := WithIgnoreLifecycle(ctx)

		assert.Equal(t, value, wrapped.Value(key))
	})

	t.Run("retrieves values even after parent is cancelled", func(t *testing.T) {
		t.Parallel()

		key := contextKey("requestID")
		value := "req-123"

		ctx, cancel := context.WithCancel(t.Context())
		ctx = context.WithValue(ctx, key, value)
		wrapped := WithIgnoreLifecycle(ctx)

		// Cancel parent
		cancel()

		// Verify parent is canceled
		require.Error(t, ctx.Err())

		// Verify we can still access values
		assert.Equal(t, value, wrapped.Value(key))
	})

	t.Run("retrieves values even after parent deadline expires", func(t *testing.T) {
		t.Parallel()

		key := contextKey("traceID")
		value := "trace-456"

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		ctx = context.WithValue(ctx, key, value)
		wrapped := WithIgnoreLifecycle(ctx)

		// Wait for parent to expire
		time.Sleep(10 * time.Millisecond)

		// Verify parent is done
		require.Error(t, ctx.Err())

		// Verify we can still access values
		assert.Equal(t, value, wrapped.Value(key))
	})

	t.Run("returns nil for non-existent keys", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		wrapped := WithIgnoreLifecycle(ctx)

		assert.Nil(t, wrapped.Value("nonexistent"))
	})

	t.Run("supports multiple values", func(t *testing.T) {
		t.Parallel()

		type testKey string

		ctx := t.Context()
		ctx = context.WithValue(ctx, testKey("key1"), "value1")
		ctx = context.WithValue(ctx, testKey("key2"), 123)
		ctx = context.WithValue(ctx, testKey("key3"), true)

		wrapped := WithIgnoreLifecycle(ctx)

		assert.Equal(t, "value1", wrapped.Value(testKey("key1")))
		assert.Equal(t, 123, wrapped.Value(testKey("key2")))
		assert.Equal(t, true, wrapped.Value(testKey("key3")))
	})

	t.Run("supports custom key types", func(t *testing.T) {
		t.Parallel()

		type customKey struct{ id int }

		key := customKey{id: 42}
		value := "customValue"

		ctx := context.WithValue(t.Context(), key, value)
		wrapped := WithIgnoreLifecycle(ctx)

		assert.Equal(t, value, wrapped.Value(key))
	})
}

func TestLifecycleInsensitiveContext_ImplementsInterface(t *testing.T) {
	t.Parallel()

	t.Run("satisfies context.Context interface", func(t *testing.T) {
		t.Parallel()

		ctx := WithIgnoreLifecycle(t.Context())

		// This will compile only if ctx implements context.Context
		_ = ctx

		// Verify all methods are callable
		_ = ctx.Done()
		_ = ctx.Err()
		_, _ = ctx.Deadline()
		_ = ctx.Value("any")
	})
}

func TestLifecycleInsensitiveContext_RealWorldUseCases(t *testing.T) {
	t.Parallel()

	t.Run("cleanup after request cancellation", func(t *testing.T) {
		t.Parallel()

		// Simulate a request context that gets canceled
		requestCtx, cancel := context.WithCancel(t.Context())
		requestCtx = context.WithValue(requestCtx, contextKey("requestID"), "req-789")

		// Create cleanup context
		cleanupCtx := WithIgnoreLifecycle(requestCtx)

		// Cancel the request
		cancel()

		// Verify request context is canceled
		require.Error(t, requestCtx.Err())

		// Simulate cleanup operation that should continue
		// even though request is canceled
		cleanupComplete := make(chan bool)

		go func() {
			// This would normally be blocked by canceled context,
			// but cleanup context ignores the cancellation
			select {
			case <-cleanupCtx.Done():
				cleanupComplete <- false
			case <-time.After(20 * time.Millisecond):
				// Cleanup completed successfully
				requestID := cleanupCtx.Value(contextKey("requestID"))
				assert.Equal(t, "req-789", requestID)

				cleanupComplete <- true
			}
		}()

		select {
		case success := <-cleanupComplete:
			assert.True(t, success, "Cleanup should complete successfully")
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Cleanup did not complete in time")
		}
	})

	t.Run("final flush after deadline expires", func(t *testing.T) {
		t.Parallel()

		// Simulate an operation with a tight deadline
		operationCtx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		operationCtx = context.WithValue(operationCtx, contextKey("batchID"), "batch-999")

		// Create flush context that ignores deadline
		flushCtx := WithIgnoreLifecycle(operationCtx)

		// Wait for operation deadline to expire
		time.Sleep(10 * time.Millisecond)

		// Verify operation context is done
		require.Error(t, operationCtx.Err())

		// Flush should still be able to proceed
		flushed := false

		select {
		case <-flushCtx.Done():
			t.Fatal("Flush context should never be done")
		default:
			// Can still access batch ID for logging
			batchID := flushCtx.Value(contextKey("batchID"))
			assert.Equal(t, "batch-999", batchID)

			flushed = true
		}

		assert.True(t, flushed, "Flush should have completed")
	})

	t.Run("nested lifecycle-insensitive contexts", func(t *testing.T) {
		t.Parallel()

		type levelKey string

		ctx := context.WithValue(t.Context(), levelKey("level"), 1)
		wrapped1 := WithIgnoreLifecycle(ctx)
		wrapped2 := WithIgnoreLifecycle(wrapped1)

		// Both levels should still access values
		assert.Equal(t, 1, wrapped1.Value(levelKey("level")))
		assert.Equal(t, 1, wrapped2.Value(levelKey("level")))

		// Both should never be done
		assert.NoError(t, wrapped1.Err())
		assert.NoError(t, wrapped2.Err())
	})
}
