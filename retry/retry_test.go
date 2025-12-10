package retry

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/atomic"
)

func TestDo_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestDo_SuccessAfterRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("temporary error") //nolint:err113 // Test error
		}

		return nil
	}, WithAttempts(5))

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDo_ExhaustsRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	testErr := errors.New("permanent failure") //nolint:err113 // Test error
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return testErr
	}, WithAttempts(3))

	require.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Equal(t, 3, callCount)
}

func TestDo_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	callCount := 0

	// Cancel immediately
	cancel()

	err := Do(ctx, func(ctx context.Context) error {
		callCount++

		return errors.New("should not be called") //nolint:err113 // Test error
	}, WithAttempts(5))

	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestDo_PermanentError(t *testing.T) {
	t.Parallel()

	callCount := 0
	testErr := errors.New("validation error") //nolint:err113 // Test error
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return Abort(testErr)
	}, WithAttempts(5))

	require.Error(t, err)
	require.ErrorIs(t, err, testErr, "should be able to unwrap to original error")
	assert.Equal(t, 1, callCount, "should not retry permanent errors")
}

func TestDo_WithTimeout(t *testing.T) {
	t.Parallel()

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			// First attempt: wait for context to timeout
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(100 * time.Millisecond):
				return errors.New("timeout didn't work") //nolint:err113 // Test error
			}
		}
		// Second attempt: succeed immediately
		return nil
	}, WithAttempts(3), WithTimeout(Timeout(30*time.Millisecond)), WithBackoff(ExpBackoff{
		Base:   5 * time.Millisecond,
		Max:    20 * time.Millisecond,
		Factor: 2.0,
	}), WithJitter(WithoutJitter))

	require.NoError(t, err, "should succeed on second attempt")
	assert.Equal(t, 2, callCount, "should have attempted twice")
}

func TestDoValue_Success(t *testing.T) {
	t.Parallel()

	result, err := DoValue(t.Context(), func(ctx context.Context) (string, error) {
		return "success", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "success", result)
}

func TestDoValue_SuccessAfterRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	result, err := DoValue(t.Context(), func(ctx context.Context) (int, error) {
		callCount++
		if callCount < 3 {
			return 0, errors.New("temporary error") //nolint:err113 // Test error
		}

		return 42, nil
	}, WithAttempts(5))

	require.NoError(t, err)
	assert.Equal(t, 42, result)
	assert.Equal(t, 3, callCount)
}

func TestDoValue_ExhaustsRetries(t *testing.T) {
	t.Parallel()

	testErr := errors.New("permanent failure") //nolint:err113 // Test error
	result, err := DoValue(t.Context(), func(ctx context.Context) (string, error) {
		return "", testErr
	}, WithAttempts(3))

	require.Error(t, err)
	assert.Equal(t, testErr, err)
	assert.Empty(t, result, "should return zero value on error")
}

func TestNewRunner_CustomOptions(t *testing.T) {
	t.Parallel()

	runner := NewRunner(
		WithAttempts(10),
		WithBackoff(ExpBackoff{
			Base:   50 * time.Millisecond,
			Max:    1 * time.Second,
			Factor: 3.0,
		}),
		WithJitter(WithoutJitter),
	)

	callCount := 0
	err := runner.Do(t.Context(), func(ctx context.Context) error {
		callCount++
		if callCount < 5 {
			return errors.New("retry me") //nolint:err113 // Test error
		}

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 5, callCount)
}

func TestNewValueRunner_CustomOptions(t *testing.T) {
	t.Parallel()

	runner := NewValueRunner[string](
		WithAttempts(10),
		WithBackoff(ExpBackoff{
			Base:   50 * time.Millisecond,
			Max:    1 * time.Second,
			Factor: 3.0,
		}),
	)

	callCount := 0
	result, err := runner.Do(t.Context(), func(ctx context.Context) (string, error) {
		callCount++
		if callCount < 3 {
			return "", errors.New("retry me") //nolint:err113 // Test error
		}

		return "done", nil
	})

	require.NoError(t, err)
	assert.Equal(t, "done", result)
	assert.Equal(t, 3, callCount)
}

func TestDo_BackoffDelay(t *testing.T) {
	t.Parallel()

	callTimes := []time.Time{}
	err := Do(t.Context(), func(ctx context.Context) error {
		callTimes = append(callTimes, time.Now())
		if len(callTimes) < 3 {
			return errors.New("retry me") //nolint:err113 // Test error
		}

		return nil
	}, WithAttempts(3), WithBackoff(ExpBackoff{
		Base:   100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 2.0,
	}), WithJitter(WithoutJitter))

	require.NoError(t, err)
	require.Len(t, callTimes, 3)

	// Check that delays increase exponentially
	delay1 := callTimes[1].Sub(callTimes[0])
	delay2 := callTimes[2].Sub(callTimes[1])

	assert.GreaterOrEqual(t, delay1.Milliseconds(), int64(100), "first delay should be >= 100ms")
	assert.GreaterOrEqual(t, delay2.Milliseconds(), int64(200), "second delay should be >= 200ms")
}

func TestCallWithTimeout_Success(t *testing.T) {
	t.Parallel()

	called := false

	var mut sync.Mutex

	running := atomic.NewBool(true)

	err := callWithTimeout(t.Context(), func(ctx context.Context) error {
		called = true

		return nil
	}, Timeout(1*time.Second), &mut, running)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestCallWithTimeout_Exceeds(t *testing.T) {
	t.Parallel()

	var mut sync.Mutex

	running := atomic.NewBool(true)

	err := callWithTimeout(t.Context(), func(ctx context.Context) error {
		return utils.SleepCtx(ctx, 200*time.Millisecond)
	}, Timeout(50*time.Millisecond), &mut, running)

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
}

func TestDo_RespectsContextDeadline(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
	defer cancel()

	callCount := atomic.NewInt64(0)
	err := Do(ctx, func(ctx context.Context) error {
		callCount.Inc()

		_ = utils.SleepCtx(ctx, 30*time.Millisecond)

		return errors.New("should timeout") //nolint:err113 // Test error
	}, WithAttempts(10), WithBackoff(ExpBackoff{
		Base:   5 * time.Millisecond,
		Max:    10 * time.Millisecond,
		Factor: 2.0,
	}), WithJitter(WithoutJitter))

	require.Error(t, err)
	assert.Equal(t, context.DeadlineExceeded, err)
	// Should have attempted at least once but not all 10 times
	assert.GreaterOrEqual(t, callCount.Load(), int64(1))
	assert.Less(t, callCount.Load(), int64(10))
}
