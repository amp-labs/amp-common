package utils

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSleepCtx(t *testing.T) {
	t.Parallel()

	t.Run("sleeps for specified duration", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		start := time.Now()
		err := SleepCtx(ctx, 50*time.Millisecond)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
		assert.Less(t, elapsed, 100*time.Millisecond) // reasonable upper bound
	})

	t.Run("returns immediately for zero duration", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		start := time.Now()
		err := SleepCtx(ctx, 0)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed, 10*time.Millisecond)
	})

	t.Run("returns immediately for negative duration", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		start := time.Now()
		err := SleepCtx(ctx, -100*time.Millisecond)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, elapsed, 10*time.Millisecond)
	})

	t.Run("returns error when context is cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())

		// Cancel context after 20ms
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := SleepCtx(ctx, 1*time.Second)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.ErrorIs(t, err, context.Canceled)
		assert.Less(t, elapsed, 100*time.Millisecond) // should return quickly after cancel
	})

	t.Run("returns error when context deadline exceeded", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := SleepCtx(ctx, 1*time.Second)
		elapsed := time.Since(start)

		require.Error(t, err)
		require.ErrorIs(t, err, context.DeadlineExceeded)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})

	t.Run("returns nil when context is already done but duration is zero", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		err := SleepCtx(ctx, 0)
		require.NoError(t, err)
	})

	t.Run("completes sleep when context is not cancelled", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := SleepCtx(ctx, 50*time.Millisecond)
		elapsed := time.Since(start)

		require.NoError(t, err)
		assert.GreaterOrEqual(t, elapsed, 50*time.Millisecond)
		assert.Less(t, elapsed, 100*time.Millisecond)
	})
}
