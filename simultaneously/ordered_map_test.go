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
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestAtKeyB = errors.New("error at key b")

// TestMapOrderedMap_SuccessfulTransformation tests basic ordered map transformation.
func TestMapOrderedMap_SuccessfulTransformation(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))
	require.NoError(t, input.Add(maps.Key[string]{Key: "c"}, 3))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], string, error) {
			return k, strconv.Itoa(v), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 3, output.Size())

	// Verify order is preserved
	entries := make([]maps.KeyValuePair[maps.Key[string], string], 0, output.Size())
	for _, entry := range output.Seq() {
		entries = append(entries, entry)
	}

	require.Len(t, entries, 3)
	assert.Equal(t, "a", entries[0].Key.Key)
	assert.Equal(t, "1", entries[0].Value)
	assert.Equal(t, "b", entries[1].Key.Key)
	assert.Equal(t, "2", entries[1].Value)
	assert.Equal(t, "c", entries[2].Key.Key)
	assert.Equal(t, "3", entries[2].Value)
}

// TestMapOrderedMap_NilInput tests handling of nil input map.
func TestMapOrderedMap_NilInput(t *testing.T) {
	t.Parallel()

	var input maps.OrderedMap[maps.Key[string], int]
	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
			return k, v, nil
		})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestMapOrderedMap_EmptyMap tests handling of empty map.
func TestMapOrderedMap_EmptyMap(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
			return k, v, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestMapOrderedMap_ErrorHandling tests error propagation.
func TestMapOrderedMap_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))
	require.NoError(t, input.Add(maps.Key[string]{Key: "c"}, 3))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
			if k.Key == "b" {
				return maps.Key[string]{}, 0, errTestAtKeyB
			}

			return k, v, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at key b")
	assert.Nil(t, output)
}

// TestMapOrderedMapCtx_ContextCancellation tests context cancellation.
//
//nolint:dupl // Test pattern is intentionally similar to map_test but with OrderedMap types
func TestMapOrderedMapCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := maps.NewOrderedHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMapCtx(ctx, 2, input,
		func(ctx context.Context, k maps.Key[int], v int) (maps.Key[int], int, error) {
			time.Sleep(100 * time.Millisecond)

			return k, v, nil
		})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestMapOrderedMapCtx_PanicRecovery tests panic recovery.
//
//nolint:dupl // Test pattern is intentionally similar to map_test but with OrderedMap types
func TestMapOrderedMapCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMapCtx(t.Context(), 2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
			if k.Key == "b" {
				panic("intentional panic")
			}

			return k, v, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestMapOrderedMap_ConcurrencyLimit tests that concurrency limiting works.
//
//nolint:dupl // Test pattern is intentionally similar to map_test but with OrderedMap types
func TestMapOrderedMap_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := maps.NewOrderedHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 20 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	//nolint:lll // Type signature is unavoidably long
	_, err := MapOrderedMap(3, input,
		func(ctx context.Context, k maps.Key[int], v int) (maps.Key[int], int, error) {
			current := concurrent.Add(1)
			defer concurrent.Add(-1)

			for {
				maxVal := maxConcurrent.Load()
				if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
					break
				}
			}

			time.Sleep(10 * time.Millisecond)

			return k, v, nil
		})

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(3))
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(1))
}

// TestMapOrderedMap_OrderPreservation tests that insertion order is preserved.
func TestMapOrderedMap_OrderPreservation(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[int], string](hashing.Sha256)
	// Add in specific order
	for i := range 100 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, fmt.Sprintf("val-%d", i)))
	}

	//nolint:lll // Type signature is unavoidably long
	output, err := MapOrderedMap(10, input,
		func(ctx context.Context, k maps.Key[int], v string) (maps.Key[int], string, error) {
			// Add some processing delay
			time.Sleep(time.Millisecond)

			return k, v + "-transformed", nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)

	// Verify order is exactly preserved
	entries := make([]maps.KeyValuePair[maps.Key[int], string], 0, output.Size())
	for _, entry := range output.Seq() {
		entries = append(entries, entry)
	}

	require.Len(t, entries, 100)

	for i := range 100 {
		assert.Equal(t, i, entries[i].Key.Key, "Entry %d should have key %d", i, i)
		assert.Equal(t, fmt.Sprintf("val-%d-transformed", i), entries[i].Value)
	}
}

// TestFlatMapOrderedMap_SuccessfulExpansion tests basic flat map expansion.
func TestFlatMapOrderedMap_SuccessfulExpansion(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 2))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 3))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
			result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)

			for i := range v {
				key := maps.Key[string]{Key: fmt.Sprintf("%s%d", k.Key, i)}
				require.NoError(t, result.Add(key, i))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 5, output.Size()) // a0, a1, b0, b1, b2

	// Verify order is preserved
	entries := make([]maps.KeyValuePair[maps.Key[string], int], 0, output.Size())
	for _, entry := range output.Seq() {
		entries = append(entries, entry)
	}

	require.Len(t, entries, 5)
	assert.Equal(t, "a0", entries[0].Key.Key)
	assert.Equal(t, 0, entries[0].Value)
	assert.Equal(t, "a1", entries[1].Key.Key)
	assert.Equal(t, 1, entries[1].Value)
	assert.Equal(t, "b0", entries[2].Key.Key)
	assert.Equal(t, 0, entries[2].Value)
	assert.Equal(t, "b1", entries[3].Key.Key)
	assert.Equal(t, 1, entries[3].Value)
	assert.Equal(t, "b2", entries[4].Key.Key)
	assert.Equal(t, 2, entries[4].Value)
}

// TestFlatMapOrderedMap_NilInput tests handling of nil input.
func TestFlatMapOrderedMap_NilInput(t *testing.T) {
	t.Parallel()

	var input maps.OrderedMap[maps.Key[string], int]
	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
			result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
			require.NoError(t, result.Add(k, v))

			return result, nil
		})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestFlatMapOrderedMap_EmptyOutputs tests when transforms return empty maps.
func TestFlatMapOrderedMap_EmptyOutputs(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
			return maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256), nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestFlatMapOrderedMap_ErrorHandling tests error propagation.
//
//nolint:dupl // Test pattern is intentionally similar to map_test but with OrderedMap types
func TestFlatMapOrderedMap_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMap(2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
			if k.Key == "b" {
				return nil, errTestAtKeyB
			}

			result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
			require.NoError(t, result.Add(k, v))

			return result, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at key b")
	assert.Nil(t, output)
}

// TestFlatMapOrderedMapCtx_ContextCancellation tests context cancellation.
//
//nolint:dupl // Test pattern is intentionally similar to map_test but with OrderedMap types
func TestFlatMapOrderedMapCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := maps.NewOrderedHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMapCtx(ctx, 2, input,
		func(ctx context.Context, k maps.Key[int], v int) (maps.OrderedMap[maps.Key[int], int], error) {
			time.Sleep(100 * time.Millisecond)

			result := maps.NewOrderedHashMap[maps.Key[int], int](hashing.Sha256)
			require.NoError(t, result.Add(k, v))

			return result, nil
		})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestFlatMapOrderedMapCtx_PanicRecovery tests panic recovery.
//
//nolint:dupl // Test pattern is intentionally similar to map_test but with OrderedMap types
func TestFlatMapOrderedMapCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMapCtx(t.Context(), 2, input,
		func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
			if k.Key == "b" {
				panic("intentional panic")
			}

			result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
			require.NoError(t, result.Add(k, v))

			return result, nil
		})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "recovered from panic")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestFlatMapOrderedMap_OrderPreservation tests that insertion order is preserved.
func TestFlatMapOrderedMap_OrderPreservation(t *testing.T) {
	t.Parallel()

	input := maps.NewOrderedHashMap[maps.Key[int], int](hashing.Sha256)
	// Add entries in specific order
	for i := range 10 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, 2)) // Each produces 2 outputs
	}

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapOrderedMap(5, input,
		func(ctx context.Context, k maps.Key[int], v int) (maps.OrderedMap[maps.Key[int], int], error) {
			// Add some processing delay
			time.Sleep(time.Millisecond)

			result := maps.NewOrderedHashMap[maps.Key[int], int](hashing.Sha256)

			for i := range v {
				key := maps.Key[int]{Key: k.Key*10 + i}
				require.NoError(t, result.Add(key, i))
			}

			return result, nil
		})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 20, output.Size())

	// Verify order is preserved: 0->0, 0->1, 10->0, 10->1, 20->0, 20->1, ...
	entries := make([]maps.KeyValuePair[maps.Key[int], int], 0, output.Size())
	for _, entry := range output.Seq() {
		entries = append(entries, entry)
	}

	require.Len(t, entries, 20)

	for i := range 10 {
		baseIdx := i * 2
		assert.Equal(t, i*10+0, entries[baseIdx].Key.Key)
		assert.Equal(t, 0, entries[baseIdx].Value)
		assert.Equal(t, i*10+1, entries[baseIdx+1].Key.Key)
		assert.Equal(t, 1, entries[baseIdx+1].Value)
	}
}
