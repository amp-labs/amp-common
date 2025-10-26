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
	"github.com/amp-labs/amp-common/should"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestAtElem2 = errors.New("error at element 2")

// TestMapOrderedSet_SuccessfulTransformation tests basic ordered set transformation.
func TestMapOrderedSet_SuccessfulTransformation(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
			return hashing.HashableString(strconv.Itoa(int(v))), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 3, output.Size())

	// Verify order is preserved
	entries := output.Entries()
	require.Len(t, entries, 3)
	assert.Equal(t, hashing.HashableString("1"), entries[0])
	assert.Equal(t, hashing.HashableString("2"), entries[1])
	assert.Equal(t, hashing.HashableString("3"), entries[2])
}

// TestMapOrderedSet_NilInput tests handling of nil input set.
func TestMapOrderedSet_NilInput(t *testing.T) {
	t.Parallel()

	var input set.OrderedSet[hashing.HashableInt]
	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
			return v, nil
		})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestMapOrderedSet_EmptySet tests handling of empty set.
func TestMapOrderedSet_EmptySet(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
			return v, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestMapOrderedSet_ErrorHandling tests error propagation.
func TestMapOrderedSet_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
			if v == 2 {
				return 0, errTestAtElem2
			}

			return v, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at element 2")
	assert.Nil(t, output)
}

// TestMapOrderedSetCtx_ContextCancellation tests context cancellation.
func TestMapOrderedSetCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSetCtx(ctx, 2, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
			time.Sleep(100 * time.Millisecond)

			return v, nil
		})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestMapOrderedSetCtx_PanicRecovery tests panic recovery.
func TestMapOrderedSetCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSetCtx(t.Context(), 2, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
			if v == 2 {
				panic("intentional panic")
			}

			return v, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestMapOrderedSet_ConcurrencyLimit tests that concurrency limiting works.
//
//nolint:dupl // Test pattern is intentionally similar to set_test but with OrderedSet types
func TestMapOrderedSet_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	for i := range 20 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	//nolint:lll // Type signature is unavoidably long
	_, err := MapOrderedSet(3, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
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

// TestMapOrderedSet_OrderPreservation tests that insertion order is preserved.
func TestMapOrderedSet_OrderPreservation(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	// Add in specific order
	for i := range 100 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedSet(10, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
			// Add some processing delay
			time.Sleep(time.Millisecond)

			return hashing.HashableString(fmt.Sprintf("val-%d", v)), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)

	// Verify order is exactly preserved
	entries := output.Entries()
	require.Len(t, entries, 100)

	for i := range 100 {
		expected := hashing.HashableString(fmt.Sprintf("val-%d", i))
		assert.Equal(t, expected, entries[i], "Entry %d should be %s", i, expected)
	}
}

// TestFlatMapOrderedSet_SuccessfulExpansion tests basic flat map expansion.
func TestFlatMapOrderedSet_SuccessfulExpansion(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableString], error) {
			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)

			for i := range int(v) {
				require.NoError(t, result.Add(hashing.HashableString(fmt.Sprintf("%d-%d", v, i))))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 5, output.Size()) // 2-0, 2-1, 3-0, 3-1, 3-2

	// Verify order is preserved
	entries := output.Entries()
	require.Len(t, entries, 5)
	assert.Equal(t, hashing.HashableString("2-0"), entries[0])
	assert.Equal(t, hashing.HashableString("2-1"), entries[1])
	assert.Equal(t, hashing.HashableString("3-0"), entries[2])
	assert.Equal(t, hashing.HashableString("3-1"), entries[3])
	assert.Equal(t, hashing.HashableString("3-2"), entries[4])
}

// TestFlatMapOrderedSet_NilInput tests handling of nil input.
func TestFlatMapOrderedSet_NilInput(t *testing.T) {
	t.Parallel()

	var input set.OrderedSet[hashing.HashableInt]
	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
			result := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
			require.NoError(t, result.Add(v))

			return result, nil
		})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestFlatMapOrderedSet_EmptyOutputs tests when transforms return empty sets.
func TestFlatMapOrderedSet_EmptyOutputs(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
			return set.NewOrderedSet[hashing.HashableInt](hashing.Sha256), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestFlatMapOrderedSet_ErrorHandling tests error propagation.
//
//nolint:dupl // Test pattern is intentionally similar to set_test but with OrderedSet types
func TestFlatMapOrderedSet_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))
	require.NoError(t, input.Add(hashing.HashableInt(3)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
			if v == 2 {
				return nil, errTestAtElem2
			}

			result := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
			require.NoError(t, result.Add(v))

			return result, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at element 2")
	assert.Nil(t, output)
}

// TestFlatMapOrderedSetCtx_ContextCancellation tests context cancellation.
//
//nolint:dupl // Test pattern is intentionally similar to set_test but with OrderedSet types
func TestFlatMapOrderedSetCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSetCtx(ctx, 2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
			time.Sleep(100 * time.Millisecond)

			result := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
			require.NoError(t, result.Add(v))

			return result, nil
		})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestFlatMapOrderedSetCtx_PanicRecovery tests panic recovery.
//
//nolint:dupl // Test pattern is intentionally similar to set_test but with OrderedSet types
func TestFlatMapOrderedSetCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableInt(1)))
	require.NoError(t, input.Add(hashing.HashableInt(2)))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSetCtx(t.Context(), 2, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
			if v == 2 {
				panic("intentional panic")
			}

			result := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
			require.NoError(t, result.Add(v))

			return result, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestFlatMapOrderedSet_OrderPreservation tests that insertion order is preserved.
func TestFlatMapOrderedSet_OrderPreservation(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	// Add entries in specific order
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSet(5, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableString], error) {
			// Add some processing delay
			time.Sleep(time.Millisecond)

			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)

			// Each element produces 2 outputs
			for i := range 2 {
				elem := hashing.HashableString(fmt.Sprintf("%d-%d", v, i))
				require.NoError(t, result.Add(elem))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 20, output.Size())

	// Verify order is preserved: 0-0, 0-1, 1-0, 1-1, 2-0, 2-1, ...
	entries := output.Entries()
	require.Len(t, entries, 20)

	for i := range 10 {
		baseIdx := i * 2

		expected0 := hashing.HashableString(fmt.Sprintf("%d-0", i))
		expected1 := hashing.HashableString(fmt.Sprintf("%d-1", i))

		assert.Equal(t, expected0, entries[baseIdx])
		assert.Equal(t, expected1, entries[baseIdx+1])
	}
}

// TestFlatMapOrderedSet_ExpandToMultiple tests expanding each element to multiple elements.
func TestFlatMapOrderedSet_ExpandToMultiple(t *testing.T) {
	t.Parallel()

	input := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("ab")))
	require.NoError(t, input.Add(hashing.HashableString("cd")))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSet(2, input,
		func(ctx context.Context, v hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
			// Split each string into individual characters
			for _, ch := range string(v) {
				require.NoError(t, result.Add(hashing.HashableString(string(ch))))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 4, output.Size()) // a, b, c, d

	// Verify order: a, b (from "ab"), then c, d (from "cd")
	entries := output.Entries()
	require.Len(t, entries, 4)
	assert.Equal(t, hashing.HashableString("a"), entries[0])
	assert.Equal(t, hashing.HashableString("b"), entries[1])
	assert.Equal(t, hashing.HashableString("c"), entries[2])
	assert.Equal(t, hashing.HashableString("d"), entries[3])
}

// TestMapOrderedSetWithExecutor_SuccessfulExecution tests MapOrderedSetWithExecutor with successful transformation.
func TestMapOrderedSetWithExecutor_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 3)
	defer should.Close(exec, "closing executor")

	input := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("hello")))
	require.NoError(t, input.Add(hashing.HashableString("world")))
	require.NoError(t, input.Add(hashing.HashableString("test")))

	output, err := MapOrderedSetWithExecutor(exec, input,
		func(ctx context.Context, v hashing.HashableString) (hashing.HashableInt, error) {
			return hashing.HashableInt(len(v)), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 2, output.Size())
}

// TestMapOrderedSetWithExecutor_ExecutorReuse tests executor reuse for MapOrderedSet.
func TestMapOrderedSetWithExecutor_ExecutorReuse(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 5)
	defer should.Close(exec, "closing executor")

	// First batch
	input1 := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input1.Add(hashing.HashableString("hello")))
	require.NoError(t, input1.Add(hashing.HashableString("world")))

	output1, err := MapOrderedSetWithExecutor(exec, input1,
		func(ctx context.Context, v hashing.HashableString) (hashing.HashableInt, error) {
			return hashing.HashableInt(len(v)), nil
		})

	require.NoError(t, err)
	assert.Equal(t, 1, output1.Size())

	// Second batch
	input2 := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input2.Add(hashing.HashableString("abc")))
	require.NoError(t, input2.Add(hashing.HashableString("def")))

	output2, err := MapOrderedSetWithExecutor(exec, input2,
		func(ctx context.Context, v hashing.HashableString) (hashing.HashableInt, error) {
			return hashing.HashableInt(len(v)), nil
		})

	require.NoError(t, err)
	assert.Equal(t, 1, output2.Size())
}

// TestMapOrderedSetCtxWithExecutor_SuccessfulExecution tests
// MapOrderedSetCtxWithExecutor with successful transformation.
//
//nolint:dupl // Test code duplicated across set types for clarity and independence
func TestMapOrderedSetCtxWithExecutor_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 3)
	defer should.Close(exec, "closing executor")

	input := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("hello")))
	require.NoError(t, input.Add(hashing.HashableString("world")))
	require.NoError(t, input.Add(hashing.HashableString("test")))

	output, err := MapOrderedSetCtxWithExecutor(t.Context(), exec, input,
		func(ctx context.Context, v hashing.HashableString) (hashing.HashableInt, error) {
			return hashing.HashableInt(len(v)), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 2, output.Size())
}

// TestMapOrderedSetCtxWithExecutor_ContextCancellation tests context cancellation with executor.
func TestMapOrderedSetCtxWithExecutor_ContextCancellation(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 10)
	defer should.Close(exec, "closing executor")

	ctx, cancel := context.WithCancel(t.Context())

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	output, err := MapOrderedSetCtxWithExecutor(ctx, exec, input,
		func(ctx context.Context, v hashing.HashableInt) (hashing.HashableInt, error) {
			time.Sleep(100 * time.Millisecond)

			return v * 2, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, output)
}

// TestFlatMapOrderedSetWithExecutor_SuccessfulExecution tests FlatMapOrderedSetWithExecutor with successful expansion.
func TestFlatMapOrderedSetWithExecutor_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	input := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("ab")))
	require.NoError(t, input.Add(hashing.HashableString("cd")))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSetWithExecutor(exec, input,
		func(ctx context.Context, v hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
			for _, ch := range string(v) {
				require.NoError(t, result.Add(hashing.HashableString(string(ch))))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 4, output.Size())
}

// TestFlatMapOrderedSetWithExecutor_ExecutorReuse tests executor reuse for FlatMapOrderedSet.
func TestFlatMapOrderedSetWithExecutor_ExecutorReuse(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 3)
	defer should.Close(exec, "closing executor")

	// First batch
	input1 := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input1.Add(hashing.HashableString("ab")))

	//nolint:lll // Type signature is unavoidably long
	output1, err := FlatMapOrderedSetWithExecutor(exec, input1,
		func(ctx context.Context, v hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
			for _, ch := range string(v) {
				require.NoError(t, result.Add(hashing.HashableString(string(ch))))
			}

			return result, nil
		})

	require.NoError(t, err)
	assert.Equal(t, 2, output1.Size())

	// Second batch
	input2 := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input2.Add(hashing.HashableString("xyz")))

	//nolint:lll // Type signature is unavoidably long
	output2, err := FlatMapOrderedSetWithExecutor(exec, input2,
		func(ctx context.Context, v hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
			for _, ch := range string(v) {
				require.NoError(t, result.Add(hashing.HashableString(string(ch))))
			}

			return result, nil
		})

	require.NoError(t, err)
	assert.Equal(t, 3, output2.Size())
}

// TestFlatMapOrderedSetCtxWithExecutor_SuccessfulExecution tests
// FlatMapOrderedSetCtxWithExecutor with successful expansion.
func TestFlatMapOrderedSetCtxWithExecutor_SuccessfulExecution(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 2)
	defer should.Close(exec, "closing executor")

	input := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
	require.NoError(t, input.Add(hashing.HashableString("ab")))
	require.NoError(t, input.Add(hashing.HashableString("cd")))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSetCtxWithExecutor(t.Context(), exec, input,
		func(ctx context.Context, v hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
			result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
			for _, ch := range string(v) {
				require.NoError(t, result.Add(hashing.HashableString(string(ch))))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 4, output.Size())
}

// TestFlatMapOrderedSetCtxWithExecutor_ContextCancellation tests
// context cancellation for FlatMapOrderedSetCtxWithExecutor.
func TestFlatMapOrderedSetCtxWithExecutor_ContextCancellation(t *testing.T) {
	t.Parallel()

	exec := newDefaultExecutor(2, 10)
	defer should.Close(exec, "closing executor")

	ctx, cancel := context.WithCancel(t.Context())

	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(hashing.HashableInt(i)))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedSetCtxWithExecutor(ctx, exec, input,
		func(ctx context.Context, v hashing.HashableInt) (set.OrderedSet[hashing.HashableInt], error) {
			time.Sleep(100 * time.Millisecond)

			result := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
			require.NoError(t, result.Add(v))

			return result, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
	assert.Nil(t, output)
}
