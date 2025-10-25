package simultaneously

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/set"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestAtTwo = errors.New("error at 2")

// TestMapSet_SuccessfulTransformation tests basic set transformation.
func TestMapSet_SuccessfulTransformation(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	output, err := MapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
		return hashing.HashableString(strconv.Itoa(int(v))), nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 3, output.Size())

	// Verify all elements are present
	contains, err := output.Contains(hashing.HashableString("1"))
	require.NoError(t, err)
	assert.True(t, contains)

	contains, err = output.Contains(hashing.HashableString("2"))
	require.NoError(t, err)
	assert.True(t, contains)

	contains, err = output.Contains(hashing.HashableString("3"))
	require.NoError(t, err)
	assert.True(t, contains)
}

// TestMapSet_NilInput tests handling of nil input set.
func TestMapSet_NilInput(t *testing.T) {
	t.Parallel()

	var input set.Set[hashing.HashableInt]
	output, err := MapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
		return v, nil
	})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestMapSet_EmptySet tests handling of empty set.
func TestMapSet_EmptySet(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	output, err := MapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
		return v, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestMapSet_ErrorHandling tests error propagation.
func TestMapSet_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	output, err := MapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
		if v == 2 {
			return 0, errTestAtTwo
		}

		return v, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at 2")
	assert.Nil(t, output)
}

// TestMapSetCtx_ContextCancellation tests context cancellation.
func TestMapSetCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	output, err := MapSetCtx(ctx, 2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
		time.Sleep(100 * time.Millisecond)

		return v, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestMapSetCtx_PanicRecovery tests panic recovery.
func TestMapSetCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapSetCtx(t.Context(), 2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
		if v == 2 {
			panic("intentional panic")
		}

		return v, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestMapSet_ConcurrencyLimit tests that concurrency limiting works.
//
//nolint:dupl // Test pattern is intentionally similar to ordered_set_test but with Set types
func TestMapSet_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	for i := range 20 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	_, err := MapSet(3, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)

		for {
			maxVal := maxConcurrent.Load()
			if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		return v, nil
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(3))
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(1))
}

// TestMapSet_DuplicateElements tests that duplicate elements are handled by set semantics.
func TestMapSet_DuplicateElements(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	// Transform all elements to the same value
	output, err := MapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
		return hashing.HashableString("same"), nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	// All three inputs map to the same value, so output should have size 1
	assert.Equal(t, 1, output.Size())

	contains, err := output.Contains(hashing.HashableString("same"))
	require.NoError(t, err)
	assert.True(t, contains)
}

// TestFlatMapSet_SuccessfulExpansion tests basic flat map expansion.
func TestFlatMapSet_SuccessfulExpansion(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableString], error) {
		result := set.NewSet[hashing.HashableString](hashing.Sha256)
		for i := range int(v) {
			require.NoError(t, result.Add(hashing.HashableString(fmt.Sprintf("%d-%d", v, i))))
		}

		return result, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 5, output.Size()) // 2-0, 2-1, 3-0, 3-1, 3-2

	// Check specific entries
	contains, err := output.Contains(hashing.HashableString("2-0"))
	require.NoError(t, err)
	assert.True(t, contains)

	contains, err = output.Contains(hashing.HashableString("2-1"))
	require.NoError(t, err)
	assert.True(t, contains)

	contains, err = output.Contains(hashing.HashableString("3-2"))
	require.NoError(t, err)
	assert.True(t, contains)
}

// TestFlatMapSet_NilInput tests handling of nil input.
func TestFlatMapSet_NilInput(t *testing.T) {
	t.Parallel()

	var input set.Set[hashing.HashableInt]
	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
			result := set.NewSet[hashing.HashableInt](hashing.Sha256)
			require.NoError(t, result.Add(v))

			return result, nil
		})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestFlatMapSet_EmptyOutputs tests when transforms return empty sets.
func TestFlatMapSet_EmptyOutputs(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
			return set.NewSet[hashing.HashableInt](hashing.Sha256), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestFlatMapSet_ErrorHandling tests error propagation.
//
//nolint:dupl // Test pattern is intentionally similar to ordered_set_test but with Set types
func TestFlatMapSet_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
		if v == 2 {
			return nil, errTestAtTwo
		}

		result := set.NewSet[hashing.HashableInt](hashing.Sha256)
		require.NoError(t, result.Add(v))

		return result, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at 2")
	assert.Nil(t, output)
}

// TestFlatMapSetCtx_ContextCancellation tests context cancellation.
//
//nolint:dupl // Test pattern is intentionally similar to ordered_set_test but with Set types
func TestFlatMapSetCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSetCtx(ctx, 2, input, func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
		time.Sleep(100 * time.Millisecond)

		result := set.NewSet[hashing.HashableInt](hashing.Sha256)
		require.NoError(t, result.Add(v))

		return result, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestFlatMapSetCtx_PanicRecovery tests panic recovery.
//
//nolint:dupl // Test pattern is intentionally similar to ordered_set_test but with Set types
func TestFlatMapSetCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSetCtx(t.Context(), 2, input, func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
		if v == 2 {
			panic("intentional panic")
		}

		result := set.NewSet[hashing.HashableInt](hashing.Sha256)
		require.NoError(t, result.Add(v))

		return result, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestFlatMapSet_DuplicateElements tests handling of duplicate elements from different transforms.
func TestFlatMapSet_DuplicateElements(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSet(2, input, func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableString], error) {
		// Both transforms produce the same element
		result := set.NewSet[hashing.HashableString](hashing.Sha256)
		require.NoError(t, result.Add(hashing.HashableString("duplicate")))

		return result, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	// Set semantics should deduplicate
	assert.Equal(t, 1, output.Size())

	contains, err := output.Contains(hashing.HashableString("duplicate"))
	require.NoError(t, err)
	assert.True(t, contains)
}

// TestFlatMapSet_ConcurrencyLimit tests concurrency limiting for FlatMapSet.
func TestFlatMapSet_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := set.NewSet[hashing.HashableInt](hashing.Sha256)
	for i := range 20 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	_, err := FlatMapSet(4, input, func(ctx context.Context, v hashing.HashableInt) (set.Set[hashing.HashableInt], error) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)

		for {
			maxVal := maxConcurrent.Load()
			if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		result := set.NewSet[hashing.HashableInt](hashing.Sha256)
		require.NoError(t, result.Add(v))

		return result, nil
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(4))
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(1))
}

// TestMapSet_TypeTransformation tests transforming between different types.
func TestMapSet_TypeTransformation(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("5")))
	require.NoError(t, input.Add(hashing.HashableString("10")))
	require.NoError(t, input.Add(hashing.HashableString("15")))

	output, err := MapSet(2, input, func(ctx context.Context, v hashing.HashableString) (hashing.HashableInt, error) {
		val, parseErr := strconv.Atoi(string(v))
		if parseErr != nil {
			return 0, parseErr
		}

		return hashing.HashableInt(val), nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 3, output.Size())

	contains, err := output.Contains(hashing.HashableInt(5))
	require.NoError(t, err)
	assert.True(t, contains)

	contains, err = output.Contains(hashing.HashableInt(10))
	require.NoError(t, err)
	assert.True(t, contains)
}

// TestFlatMapSet_ExpandToMultiple tests expanding each element to multiple elements.
func TestFlatMapSet_ExpandToMultiple(t *testing.T) {
	t.Parallel()

	input := set.NewSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("ab")))
	require.NoError(t, input.Add(hashing.HashableString("cd")))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapSet(2, input, func(ctx context.Context, v hashing.HashableString) (set.Set[hashing.HashableString], error) {
		result := set.NewSet[hashing.HashableString](hashing.Sha256)
		// Split each string into individual characters
		for _, ch := range string(v) {
			require.NoError(t, result.Add(hashing.HashableString(string(ch))))
		}

		return result, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 4, output.Size()) // a, b, c, d

	// Verify all characters are present
	for _, expected := range []string{"a", "b", "c", "d"} {
		contains, err := output.Contains(hashing.HashableString(expected))
		require.NoError(t, err)
		assert.True(t, contains, "Expected %s to be in the output set", expected)
	}
}
