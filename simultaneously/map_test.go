package simultaneously

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestAtB = errors.New("error at b")

// TestMapGoMap_SuccessfulTransformation tests basic map transformation.
func TestMapGoMap_SuccessfulTransformation(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2, "c": 3}
	output, err := MapGoMap(2, input, func(ctx context.Context, k string, v int) (int, string, error) {
		return v, strings.ToUpper(k), nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output, 3)
	assert.Equal(t, "A", output[1])
	assert.Equal(t, "B", output[2])
	assert.Equal(t, "C", output[3])
}

// TestMapGoMap_NilInput tests handling of nil input map.
func TestMapGoMap_NilInput(t *testing.T) {
	t.Parallel()

	var input map[string]int
	output, err := MapGoMap(2, input, func(ctx context.Context, k string, v int) (int, string, error) {
		return v, k, nil
	})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestMapGoMap_EmptyMap tests handling of empty map.
func TestMapGoMap_EmptyMap(t *testing.T) {
	t.Parallel()

	input := map[string]int{}
	output, err := MapGoMap(2, input, func(ctx context.Context, k string, v int) (int, string, error) {
		return v, k, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output)
}

// TestMapGoMap_ErrorHandling tests error propagation.
func TestMapGoMap_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2, "c": 3}
	output, err := MapGoMap(2, input, func(ctx context.Context, k string, v int) (int, string, error) {
		if k == "b" {
			return 0, "", errTestAtB
		}

		return v, k, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at b")
	assert.Nil(t, output)
}

// TestMapGoMapCtx_ContextCancellation tests context cancellation.
func TestMapGoMapCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := make(map[int]int)
	for i := range 10 {
		input[i] = i
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	output, err := MapGoMapCtx(ctx, 2, input, func(ctx context.Context, k, v int) (int, int, error) {
		time.Sleep(100 * time.Millisecond)

		return k, v, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestMapGoMapCtx_PanicRecovery tests panic recovery.
func TestMapGoMapCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2}
	output, err := MapGoMapCtx(t.Context(), 2, input, func(ctx context.Context, k string, v int) (string, int, error) {
		if k == "b" {
			panic("intentional panic")
		}

		return k, v, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestMapGoMap_ConcurrencyLimit tests that concurrency limiting works.
func TestMapGoMap_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := make(map[int]int)
	for i := range 20 {
		input[i] = i
	}

	_, err := MapGoMap(3, input, func(ctx context.Context, k, v int) (int, int, error) {
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

// TestFlatMapGoMap_SuccessfulExpansion tests basic flat map expansion.
func TestFlatMapGoMap_SuccessfulExpansion(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 2, "b": 3}
	output, err := FlatMapGoMap(2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
		result := make(map[string]int)
		for i := range v {
			result[fmt.Sprintf("%s%d", k, i)] = i
		}

		return result, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output, 5) // a0, a1, b0, b1, b2

	// Check specific entries
	assert.Equal(t, 0, output["a0"])
	assert.Equal(t, 1, output["a1"])
	assert.Equal(t, 0, output["b0"])
	assert.Equal(t, 1, output["b1"])
	assert.Equal(t, 2, output["b2"])
}

// TestFlatMapGoMap_NilInput tests handling of nil input.
func TestFlatMapGoMap_NilInput(t *testing.T) {
	t.Parallel()

	var input map[string]int
	output, err := FlatMapGoMap(2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
		return map[string]int{k: v}, nil
	})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestFlatMapGoMap_EmptyOutputs tests when transforms return empty maps.
func TestFlatMapGoMap_EmptyOutputs(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2}
	output, err := FlatMapGoMap(2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
		return map[string]int{}, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Empty(t, output)
}

// TestFlatMapGoMap_ErrorHandling tests error propagation.
func TestFlatMapGoMap_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2, "c": 3}
	output, err := FlatMapGoMap(2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
		if k == "b" {
			return nil, errTestAtB
		}

		return map[string]int{k: v}, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at b")
	assert.Nil(t, output)
}

// TestFlatMapGoMapCtx_ContextCancellation tests context cancellation.
func TestFlatMapGoMapCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := make(map[int]int)
	for i := range 10 {
		input[i] = i
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	output, err := FlatMapGoMapCtx(ctx, 2, input, func(ctx context.Context, k, v int) (map[int]int, error) {
		time.Sleep(100 * time.Millisecond)

		return map[int]int{k: v}, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestFlatMapGoMapCtx_PanicRecovery tests panic recovery.
func TestFlatMapGoMapCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2}
	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapGoMapCtx(t.Context(), 2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
		if k == "b" {
			panic("intentional panic")
		}

		return map[string]int{k: v}, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestFlatMapGoMap_DuplicateKeys tests handling of duplicate keys from different transforms.
func TestFlatMapGoMap_DuplicateKeys(t *testing.T) {
	t.Parallel()

	input := map[string]int{"a": 1, "b": 2}
	output, err := FlatMapGoMap(2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
		// Both transforms produce the same key
		return map[string]int{"duplicate": v}, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Len(t, output, 1)
	// One of the values should be present (non-deterministic which one due to concurrency)
	val, exists := output["duplicate"]
	assert.True(t, exists)
	assert.True(t, val == 1 || val == 2)
}

// TestMapMap_SuccessfulTransformation tests amp-common Map transformation.
func TestMapMap_SuccessfulTransformation(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))
	require.NoError(t, input.Add(maps.Key[string]{Key: "c"}, 3))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], string, error) {
		return k, strconv.Itoa(v), nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 3, output.Size())

	// Verify values by iterating
	found := make(map[string]string)
	for key, value := range output.Seq() {
		found[key.Key] = value
	}

	assert.Equal(t, "1", found["a"])
}

// TestMapMap_NilInput tests nil input map.
func TestMapMap_NilInput(t *testing.T) {
	t.Parallel()

	var input maps.Map[maps.Key[string], int]
	output, err := MapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
		return k, v, nil
	})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestMapMap_EmptyMap tests empty map.
func TestMapMap_EmptyMap(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	output, err := MapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
		return k, v, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestMapMapCtx_ContextCancellation tests context cancellation.
func TestMapMapCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	output, err := MapMapCtx(ctx, 2, input, func(ctx context.Context, k maps.Key[int], v int) (maps.Key[int], int, error) {
		time.Sleep(100 * time.Millisecond)

		return k, v, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestMapMapCtx_PanicRecovery tests panic recovery.
func TestMapMapCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := MapMapCtx(t.Context(), 2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
		if k.Key == "b" {
			panic("intentional panic")
		}

		return k, v, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestMapMap_ErrorHandling tests error propagation.
func TestMapMap_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	output, err := MapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], int, error) {
		if k.Key == "b" {
			return maps.Key[string]{}, 0, errTestAtB
		}

		return k, v, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at b")
	assert.Nil(t, output)
}

// TestFlatMapMap_SuccessfulExpansion tests flat map expansion for amp-common Maps.
func TestFlatMapMap_SuccessfulExpansion(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 2))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 3))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Map[maps.Key[string], int], error) {
		result := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)

		for i := range v {
			key := maps.Key[string]{Key: fmt.Sprintf("%s%d", k.Key, i)}
			require.NoError(t, result.Add(key, i))
		}

		return result, nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 5, output.Size()) // a0, a1, b0, b1, b2

	// Verify specific entries by iterating
	found := make(map[string]int)
	for key, value := range output.Seq() {
		found[key.Key] = value
	}

	assert.Equal(t, 0, found["a0"])
	assert.Equal(t, 2, found["b2"])
}

// TestFlatMapMap_NilInput tests nil input.
func TestFlatMapMap_NilInput(t *testing.T) {
	t.Parallel()

	var input maps.Map[maps.Key[string], int]
	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Map[maps.Key[string], int], error) {
		result := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
		require.NoError(t, result.Add(k, v))

		return result, nil
	})

	require.NoError(t, err)
	assert.Nil(t, output)
}

// TestFlatMapMap_EmptyOutputs tests when transforms return empty maps.
func TestFlatMapMap_EmptyOutputs(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Map[maps.Key[string], int], error) {
		return maps.NewHashMap[maps.Key[string], int](hashing.Sha256), nil
	})

	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}

// TestFlatMapMapCtx_ContextCancellation tests context cancellation.
func TestFlatMapMapCtx_ContextCancellation(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(t.Context())

	input := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 10 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapMapCtx(ctx, 2, input, func(ctx context.Context, k maps.Key[int], v int) (maps.Map[maps.Key[int], int], error) {
		time.Sleep(100 * time.Millisecond)

		result := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
		require.NoError(t, result.Add(k, v))

		return result, nil
	})

	require.Error(t, err)
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, output)
}

// TestFlatMapMapCtx_PanicRecovery tests panic recovery.
func TestFlatMapMapCtx_PanicRecovery(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapMapCtx(t.Context(), 2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Map[maps.Key[string], int], error) {
		if k.Key == "b" {
			panic("intentional panic")
		}

		result := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
		require.NoError(t, result.Add(k, v))

		return result, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic recovered")
	assert.Contains(t, err.Error(), "intentional panic")
	assert.Nil(t, output)
}

// TestFlatMapMap_ErrorHandling tests error propagation.
func TestFlatMapMap_ErrorHandling(t *testing.T) {
	t.Parallel()

	input := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
	require.NoError(t, input.Add(maps.Key[string]{Key: "a"}, 1))
	require.NoError(t, input.Add(maps.Key[string]{Key: "b"}, 2))

	//nolint:lll // Type signature is unavoidably long
	output, err := FlatMapMap(2, input, func(ctx context.Context, k maps.Key[string], v int) (maps.Map[maps.Key[string], int], error) {
		if k.Key == "b" {
			return nil, errTestAtB
		}

		result := maps.NewHashMap[maps.Key[string], int](hashing.Sha256)
		require.NoError(t, result.Add(k, v))

		return result, nil
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "error at b")
	assert.Nil(t, output)
}

// TestMapMap_ConcurrencyLimit tests concurrency limiting.
func TestMapMap_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 20 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	_, err := MapMap(3, input, func(ctx context.Context, k maps.Key[int], v int) (maps.Key[int], int, error) {
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

// TestFlatMapMap_ConcurrencyLimit tests concurrency limiting for FlatMapMap.
func TestFlatMapMap_ConcurrencyLimit(t *testing.T) {
	t.Parallel()

	var concurrent atomic.Int32

	var maxConcurrent atomic.Int32

	input := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
	for i := range 20 {
		require.NoError(t, input.Add(maps.Key[int]{Key: i}, i))
	}

	//nolint:lll // Type signature is unavoidably long
	_, err := FlatMapMap(4, input, func(ctx context.Context, k maps.Key[int], v int) (maps.Map[maps.Key[int], int], error) {
		current := concurrent.Add(1)
		defer concurrent.Add(-1)

		for {
			maxVal := maxConcurrent.Load()
			if current <= maxVal || maxConcurrent.CompareAndSwap(maxVal, current) {
				break
			}
		}

		time.Sleep(10 * time.Millisecond)

		result := maps.NewHashMap[maps.Key[int], int](hashing.Sha256)
		require.NoError(t, result.Add(k, v))

		return result, nil
	})

	require.NoError(t, err)
	assert.LessOrEqual(t, maxConcurrent.Load(), int32(4))
	assert.GreaterOrEqual(t, maxConcurrent.Load(), int32(1))
}
