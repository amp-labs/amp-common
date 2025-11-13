package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithAttempts_Option(t *testing.T) {
	t.Parallel()

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return errors.New("always fail") //nolint:err113 // Test error
	}, WithAttempts(7))

	require.Error(t, err)
	assert.Equal(t, 7, callCount)
}

func TestWithBackoff_Option(t *testing.T) {
	t.Parallel()

	customBackoff := ExpBackoff{
		Base:   50 * time.Millisecond,
		Max:    500 * time.Millisecond,
		Factor: 3.0,
	}

	callTimes := []time.Time{}
	err := Do(t.Context(), func(ctx context.Context) error {
		callTimes = append(callTimes, time.Now())
		if len(callTimes) < 3 {
			return errors.New("retry") //nolint:err113 // Test error
		}

		return nil
	}, WithBackoff(customBackoff), WithJitter(WithoutJitter))

	require.NoError(t, err)
	assert.Len(t, callTimes, 3)

	// Verify backoff delays match expected pattern
	delay1 := callTimes[1].Sub(callTimes[0])
	assert.GreaterOrEqual(t, delay1.Milliseconds(), int64(50))
}

func TestWithTimeout_Option(t *testing.T) {
	t.Parallel()

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++
		if callCount == 1 {
			time.Sleep(150 * time.Millisecond)
		}

		return nil
	}, WithTimeout(Timeout(50*time.Millisecond)), WithAttempts(3))

	require.NoError(t, err, "should succeed after timeout on first attempt")
	assert.Equal(t, 2, callCount)
}

func TestWithJitter_Option(t *testing.T) {
	t.Parallel()

	// Test with no jitter
	callTimes := []time.Time{}
	err := Do(t.Context(), func(ctx context.Context) error {
		callTimes = append(callTimes, time.Now())
		if len(callTimes) < 3 {
			return errors.New("retry") //nolint:err113 // Test error
		}

		return nil
	}, WithJitter(WithoutJitter), WithBackoff(ExpBackoff{
		Base:   100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 2.0,
	}))

	require.NoError(t, err)
	assert.Len(t, callTimes, 3)

	// With no jitter, delays should be exact
	delay1 := callTimes[1].Sub(callTimes[0])
	assert.InDelta(t, 100, delay1.Milliseconds(), 20, "delay should be close to 100ms")
}

func TestWithBudget_Option(t *testing.T) {
	t.Parallel()

	budget := &Budget{
		Rate:  1.0, // Very low rate threshold
		Ratio: 0.0, // No retries allowed
	}

	// Warm up the budget with some initial calls
	for range 5 {
		budget.sendOK(false)
		time.Sleep(10 * time.Millisecond)
	}

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return errors.New("always fail") //nolint:err113 // Test error
	}, WithBudget(budget), WithAttempts(10))

	require.Error(t, err)
	// Budget should have prevented most retries
	assert.Less(t, callCount, 10, "budget should limit retries")
}

func TestMultipleOptions(t *testing.T) {
	t.Parallel()

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return errors.New("retry") //nolint:err113 // Test error
		}

		return nil
	},
		WithAttempts(5),
		WithBackoff(ExpBackoff{Base: 10 * time.Millisecond, Max: 100 * time.Millisecond, Factor: 2.0}),
		WithJitter(WithoutJitter),
		WithTimeout(Timeout(1*time.Second)),
	)

	require.NoError(t, err)
	assert.Equal(t, 3, callCount)
}

func TestDefaultOptions(t *testing.T) {
	t.Parallel()

	// Test that NewRunner uses sensible defaults
	runner := NewRunner()

	callCount := 0
	err := runner.Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return errors.New("always fail") //nolint:err113 // Test error
	})

	require.Error(t, err)
	assert.Equal(t, 4, callCount, "default should be 4 attempts")
}

func TestOptionsIsolation(t *testing.T) {
	t.Parallel()

	// Create two runners with different options
	runner1 := NewRunner(WithAttempts(3))
	runner2 := NewRunner(WithAttempts(5))

	// Runner1 should use 3 attempts
	callCount1 := 0
	err1 := runner1.Do(t.Context(), func(ctx context.Context) error {
		callCount1++

		return errors.New("fail") //nolint:err113 // Test error
	})
	require.Error(t, err1)
	assert.Equal(t, 3, callCount1)

	// Runner2 should use 5 attempts
	callCount2 := 0
	err2 := runner2.Do(t.Context(), func(ctx context.Context) error {
		callCount2++

		return errors.New("fail") //nolint:err113 // Test error
	})
	require.Error(t, err2)
	assert.Equal(t, 5, callCount2)
}
