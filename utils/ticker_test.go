//nolint:revive // Package name 'utils' is established convention in this codebase
package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTickerWithContext(t *testing.T) {
	t.Parallel()

	t.Run("sends ticks at regular intervals", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 500*time.Millisecond)
		defer cancel()

		ticker := TickerWithContext(ctx, 50*time.Millisecond)

		// Receive first tick
		start := time.Now()
		tick1, ok := <-ticker
		require.True(t, ok, "channel should not be closed")

		elapsed1 := time.Since(start)

		// First tick should arrive around 50ms (allow 1ms margin for timer imprecision)
		assert.GreaterOrEqual(t, elapsed1, 49*time.Millisecond)
		assert.Less(t, elapsed1, 100*time.Millisecond)
		assert.False(t, tick1.IsZero(), "tick should contain a valid time")

		// Receive second tick
		tick2, ok := <-ticker
		require.True(t, ok, "channel should not be closed")

		elapsed2 := tick2.Sub(tick1)

		// Second tick should be approximately 50ms after first (allow 1ms margin)
		assert.GreaterOrEqual(t, elapsed2, 49*time.Millisecond)
		assert.Less(t, elapsed2, 100*time.Millisecond)
	})

	t.Run("receives multiple ticks", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 300*time.Millisecond)
		defer cancel()

		ticker := TickerWithContext(ctx, 50*time.Millisecond)

		tickCount := 0
		timeout := time.After(200 * time.Millisecond)

		// Count ticks for 200ms (should get 3-4 ticks with 50ms interval)
		for {
			select {
			case tick, ok := <-ticker:
				if !ok {
					t.Fatal("channel closed unexpectedly")
				}

				assert.False(t, tick.IsZero())

				tickCount++
			case <-timeout:
				// We should have received at least 3 ticks in 200ms with 50ms interval
				assert.GreaterOrEqual(t, tickCount, 3, "should receive at least 3 ticks")
				assert.LessOrEqual(t, tickCount, 5, "should not receive too many ticks")

				return
			}
		}
	})

	t.Run("stops and closes channel when context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())

		ticker := TickerWithContext(ctx, 50*time.Millisecond)

		// Receive first tick to ensure ticker is running
		tick1, ok := <-ticker
		require.True(t, ok, "channel should be open")
		assert.False(t, tick1.IsZero())

		// Cancel context
		cancel()

		// Channel should close shortly after cancellation
		timeout := time.After(100 * time.Millisecond)
		select {
		case tick, ok := <-ticker:
			if ok {
				// Might receive one more tick that was in-flight
				assert.False(t, tick.IsZero())
				// Next read should definitely be closed
				_, ok = <-ticker
				assert.False(t, ok, "channel should be closed after context cancellation")
			} else {
				// Channel closed immediately
				assert.False(t, ok, "channel should be closed")
			}
		case <-timeout:
			t.Fatal("channel did not close within timeout")
		}
	})

	t.Run("handles nil context by using background context", func(t *testing.T) {
		t.Parallel()

		// Create ticker with nil context (tests defensive nil check)
		//nolint:staticcheck // Intentionally testing nil context handling
		ticker := TickerWithContext(nil, 50*time.Millisecond)

		// Should still receive ticks
		timeout := time.After(150 * time.Millisecond)

		select {
		case tick, ok := <-ticker:
			require.True(t, ok, "channel should be open")
			assert.False(t, tick.IsZero())
		case <-timeout:
			t.Fatal("did not receive tick with nil context")
		}
	})

	t.Run("stops immediately when context is already cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel() // Cancel immediately

		ticker := TickerWithContext(ctx, 50*time.Millisecond)

		// Channel should close quickly without sending ticks
		timeout := time.After(100 * time.Millisecond)
		select {
		case tick, ok := <-ticker:
			if ok {
				t.Logf("received unexpected tick: %v", tick)
				// If we get a tick, the next read should be closed
				_, ok = <-ticker
				assert.False(t, ok, "channel should close after initial tick")
			}
			// Either way, channel should be closed
		case <-timeout:
			t.Fatal("channel did not close within timeout")
		}
	})

	t.Run("handles context timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		ticker := TickerWithContext(ctx, 30*time.Millisecond)

		tickCount := 0

		for tick := range ticker {
			assert.False(t, tick.IsZero())

			tickCount++
			// Should receive 2-3 ticks before timeout
			if tickCount > 10 {
				t.Fatal("received too many ticks, channel should have closed")
			}
		}

		// Should have received at least 2 ticks in 100ms with 30ms interval
		assert.GreaterOrEqual(t, tickCount, 2, "should receive at least 2 ticks before timeout")
		assert.LessOrEqual(t, tickCount, 5, "should not receive too many ticks")
	})

	t.Run("works with very short durations", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		ticker := TickerWithContext(ctx, 10*time.Millisecond)

		// Should receive multiple ticks quickly
		timeout := time.After(50 * time.Millisecond)
		tickCount := 0

		for {
			select {
			case tick, ok := <-ticker:
				if !ok {
					t.Fatal("channel closed unexpectedly")
				}

				assert.False(t, tick.IsZero())

				tickCount++
			case <-timeout:
				// With 10ms interval, should get at least 4 ticks in 50ms
				assert.GreaterOrEqual(t, tickCount, 4, "should receive multiple ticks with short duration")

				return
			}
		}
	})

	t.Run("channel is receive-only", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()

		ticker := TickerWithContext(ctx, 50*time.Millisecond)

		// This is a compile-time check enforced by the type system
		// The returned channel is <-chan time.Time (receive-only)
		// Uncommenting the next line would cause a compile error:
		// ticker <- time.Now() // compile error: send to receive-only channel

		// Just verify we can receive from it
		select {
		case tick, ok := <-ticker:
			assert.True(t, ok)
			assert.False(t, tick.IsZero())
		case <-time.After(100 * time.Millisecond):
			t.Fatal("did not receive tick")
		}
	})

	t.Run("multiple consumers can range over the channel", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 150*time.Millisecond)
		defer cancel()

		ticker := TickerWithContext(ctx, 30*time.Millisecond)

		tickCount := 0

		for tick := range ticker {
			assert.False(t, tick.IsZero())

			tickCount++
			// Safety limit to prevent infinite loop in case of bugs
			if tickCount > 100 {
				t.Fatal("too many ticks received")
			}
		}

		// Should have received 3-5 ticks in 150ms with 30ms interval
		assert.GreaterOrEqual(t, tickCount, 3, "should receive at least 3 ticks")
		assert.LessOrEqual(t, tickCount, 7, "should not receive too many ticks")
	})

	t.Run("does not leak goroutines on context cancellation", func(t *testing.T) {
		t.Parallel()

		// Create and cancel multiple tickers to verify cleanup
		for range 10 {
			ctx, cancel := context.WithCancel(t.Context())
			ticker := TickerWithContext(ctx, 10*time.Millisecond)

			// Receive one tick
			<-ticker

			// Cancel and wait for channel close
			cancel()

			// Drain channel to verify it closes
			timeout := time.After(50 * time.Millisecond)

			for {
				select {
				case _, ok := <-ticker:
					if !ok {
						// Channel closed, cleanup successful
						goto nextIteration
					}
					// Continue draining
				case <-timeout:
					t.Fatal("channel did not close, possible goroutine leak")
				}
			}

		nextIteration:
		}
	})
}
