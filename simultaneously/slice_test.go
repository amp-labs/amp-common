package simultaneously

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestAt3 = errors.New("error at 3")

func TestMapSlice_SuccessfulTransformation(t *testing.T) {
	t.Parallel()

	numbers := []int{1, 2, 3, 4, 5}
	doubled, err := MapSlice(2, numbers, func(ctx context.Context, n int) (int, error) {
		return n * 2, nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{2, 4, 6, 8, 10}, doubled)
}

func TestMapSlice_EmptySlice(t *testing.T) {
	t.Parallel()

	var empty []int
	result, err := MapSlice(2, empty, func(ctx context.Context, n int) (int, error) {
		return n * 2, nil
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestMapSlice_TypeConversion(t *testing.T) {
	t.Parallel()

	numbers := []int{1, 2, 3}
	strings, err := MapSlice(2, numbers, func(ctx context.Context, n int) (string, error) {
		return fmt.Sprintf("num-%d", n), nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"num-1", "num-2", "num-3"}, strings)
}

func TestMapSlice_OrderPreserved(t *testing.T) {
	t.Parallel()

	// Use different sleep times to ensure concurrent execution doesn't affect order
	values := []int{1, 2, 3, 4, 5}
	result, err := MapSlice(3, values, func(ctx context.Context, n int) (int, error) {
		// Later items sleep less to finish faster, testing order preservation
		time.Sleep(time.Duration(6-n) * time.Millisecond)

		return n * 10, nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int{10, 20, 30, 40, 50}, result)
}

func TestMapSlice_ErrorStopsExecution(t *testing.T) {
	t.Parallel()

	var execCount atomic.Int32

	values := []int{1, 2, 3, 4, 5}
	result, err := MapSlice(5, values, func(ctx context.Context, n int) (int, error) {
		execCount.Add(1)

		if n == 3 {
			return 0, errTestAt3
		}

		time.Sleep(50 * time.Millisecond)

		return n * 2, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at 3")
	assert.Nil(t, result)
}

func TestMapSlice_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	values := make([]int, 20)
	for i := range values {
		values[i] = i
	}

	_, err := MapSlice(3, values, func(ctx context.Context, n int) (int, error) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)

		// Track max concurrency
		for {
			maxVal := maxConcurrent.Load()
			if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		return n, nil
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(3))
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(1))
}

func TestMapSliceCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	var started atomic.Int32

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	values := make([]int, 10)
	result, err := MapSliceCtx(ctx, 2, values, func(ctx context.Context, n int) (int, error) {
		started.Add(1)
		time.Sleep(100 * time.Millisecond)

		return n, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
}

func TestMapSliceCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	values := []int{1, 2, 3}
	result, err := MapSliceCtx(t.Context(), 2, values, func(ctx context.Context, n int) (int, error) {
		if n == 2 {
			panic("intentional panic")
		}

		return n * 2, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, result)
}

func TestMapSliceCtx_UnlimitedConcurrency(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	values := make([]int, 10)
	for i := range values {
		values[i] = i
	}

	_, err := MapSliceCtx(t.Context(), 0, values, func(ctx context.Context, n int) (int, error) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)

		for {
			maxVal := maxConcurrent.Load()
			if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		return n, nil
	})

	require.NoError(t, err)
	// With unlimited concurrency, should run all 10 at once
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(5))
}

func TestFlatMapSlice_SuccessfulExpansion(t *testing.T) {
	t.Parallel()

	words := []string{"hello", "world"}
	chars, err := FlatMapSlice(2, words, func(ctx context.Context, word string) ([]rune, error) {
		return []rune(word), nil
	})

	require.NoError(t, err)

	expected := []rune{'h', 'e', 'l', 'l', 'o', 'w', 'o', 'r', 'l', 'd'}
	assert.Equal(t, expected, chars)
}

func TestFlatMapSlice_EmptyInput(t *testing.T) {
	t.Parallel()

	var empty []string
	result, err := FlatMapSlice(2, empty, func(ctx context.Context, s string) ([]rune, error) {
		return []rune(s), nil
	})

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestFlatMapSlice_EmptyOutputs(t *testing.T) {
	t.Parallel()

	values := []int{1, 2, 3}
	result, err := FlatMapSlice(2, values, func(ctx context.Context, n int) ([]int, error) {
		return []int{}, nil
	})

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestFlatMapSlice_MixedOutputSizes(t *testing.T) {
	t.Parallel()

	values := []int{0, 1, 2, 3}
	result, err := FlatMapSlice(2, values, func(ctx context.Context, n int) ([]int, error) {
		// Create n copies of n
		output := make([]int, n)
		for i := range output {
			output[i] = n
		}

		return output, nil
	})

	require.NoError(t, err)

	expected := []int{1, 2, 2, 3, 3, 3}
	assert.Equal(t, expected, result)
}

func TestFlatMapSlice_OrderPreserved(t *testing.T) {
	t.Parallel()

	values := []int{1, 2, 3}
	result, err := FlatMapSlice(3, values, func(ctx context.Context, n int) ([]int, error) {
		// Sleep different amounts to test order preservation
		time.Sleep(time.Duration(4-n) * time.Millisecond)

		return []int{n, n + 10}, nil
	})

	require.NoError(t, err)

	expected := []int{1, 11, 2, 12, 3, 13}
	assert.Equal(t, expected, result)
}

func TestFlatMapSlice_ErrorHandling(t *testing.T) {
	t.Parallel()

	values := []int{1, 2, 3, 4}
	result, err := FlatMapSlice(2, values, func(ctx context.Context, n int) ([]int, error) {
		if n == 3 {
			return nil, errTestAt3
		}

		return []int{n, n * 10}, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at 3")
	assert.Nil(t, result)
}

func TestFlatMapSliceCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	values := make([]int, 10)
	result, err := FlatMapSliceCtx(ctx, 2, values, func(ctx context.Context, n int) ([]int, error) {
		time.Sleep(100 * time.Millisecond)

		return []int{n}, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, result)
}

func TestFlatMapSliceCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	values := []int{1, 2, 3}
	result, err := FlatMapSliceCtx(t.Context(), 2, values, func(ctx context.Context, n int) ([]int, error) {
		if n == 2 {
			panic("intentional panic")
		}

		return []int{n}, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, result)
}

func TestFlatMapSliceCtx_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	values := make([]int, 20)
	for i := range values {
		values[i] = i
	}

	_, err := FlatMapSliceCtx(t.Context(), 4, values, func(ctx context.Context, n int) ([]int, error) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)

		for {
			maxVal := maxConcurrent.Load()
			if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		return []int{n}, nil
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(4))
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(1))
}

func TestFlatMapSlice_LargeExpansion(t *testing.T) {
	t.Parallel()

	values := []int{1, 2, 3}
	result, err := FlatMapSlice(2, values, func(ctx context.Context, n int) ([]int, error) {
		// Each input produces 100 outputs
		output := make([]int, 100)
		for i := range output {
			output[i] = n*100 + i
		}

		return output, nil
	})

	require.NoError(t, err)
	assert.Len(t, result, 300)
	// Check first few elements of each expansion
	assert.Equal(t, 100, result[0])   // First element of first expansion
	assert.Equal(t, 200, result[100]) // First element of second expansion
	assert.Equal(t, 300, result[200]) // First element of third expansion
}
