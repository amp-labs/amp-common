// Package simultaneously provides parallel map transformation functions.
// This file contains utilities for concurrently transforming map entries,
// supporting both standard Go maps and amp-common Map implementations.
package simultaneously

import (
	"context"
	"sync"

	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/maps"
)

// Collectable is a type alias for collectable.Collectable, providing a shorter name
// for use in this package. It represents types that can be both hashed and compared
// for equality, which is required for use as map keys in amp-common maps.
type Collectable[T any] = collectable.Collectable[T]

// MapGoMap transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, producing a new map with potentially different key and value types.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use MapGoMapCtx.
//
// Returns nil if the input map is nil. The output map may have fewer entries if
// the transform produces duplicate keys (later entries overwrite earlier ones).
//
// Example:
//
//	// Convert map[string]int to map[int]string in parallel
//	input := map[string]int{"a": 1, "b": 2, "c": 3}
//	output, err := MapGoMap(2, input, func(ctx context.Context, k string, v int) (int, string, error) {
//	    return v, strings.ToUpper(k), nil
//	})
//	// output: map[int]string{1: "A", 2: "B", 3: "C"}
func MapGoMap[InKey comparable, InVal any, OutKey comparable, OutVal any](
	maxConcurrent int,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (map[OutKey]OutVal, error) {
	return MapGoMapCtx[InKey, InVal, OutKey, OutVal](context.Background(), maxConcurrent, input, transform)
}

// MapGoMapCtx transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, producing a new map with potentially different key and value types.
//
// This is the context-aware version of MapGoMap. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map may have fewer entries if
// the transform produces duplicate keys (later entries overwrite earlier ones).
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := MapGoMapCtx(ctx, 2, input, func(ctx context.Context, k string, v int) (int, string, error) {
//	    // Check context cancellation
//	    select {
//	    case <-ctx.Done():
//	        return 0, "", ctx.Err()
//	    default:
//	    }
//	    return v, strings.ToUpper(k), nil
//	})
func MapGoMapCtx[InKey comparable, InVal any, OutKey comparable, OutVal any](
	ctx context.Context,
	maxConcurrent int,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (map[OutKey]OutVal, error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, len(input))

	result, err := MapGoMapCtxWithExecutor(ctx, exec, input, transform)
	if closeErr := exec.Close(); closeErr != nil && err == nil {
		return nil, closeErr
	}

	return result, err
}

// MapGoMapWithExecutor transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, producing a new map with potentially different key and value types, using a custom executor.
// See MapGoMapCtxWithExecutor for more information.
func MapGoMapWithExecutor[InKey comparable, InVal any, OutKey comparable, OutVal any](
	exec Executor,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (map[OutKey]OutVal, error) {
	return MapGoMapCtxWithExecutor[InKey, InVal, OutKey, OutVal](context.Background(), exec, input, transform)
}

// MapGoMapCtxWithExecutor transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, producing a new map with potentially different key and value types, using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map may have fewer entries if
// the transform produces duplicate keys (later entries overwrite earlier ones).
//
// Example:
//
//	expectedSize := 100
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := MapGoMapCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, k string, v int) (int, string, error) {
//	        return v, strings.ToUpper(k), nil
//	    })
func MapGoMapCtxWithExecutor[InKey comparable, InVal any, OutKey comparable, OutVal any](
	ctx context.Context,
	exec Executor,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (map[OutKey]OutVal, error) {
	if input == nil {
		return nil, nil
	}

	in := maps.FromGoMap(input, hashing.Sha256)

	out, err := MapMapCtxWithExecutor[maps.Key[InKey], InVal, maps.Key[OutKey], OutVal](ctx, exec, in,
		func(ctx context.Context, key maps.Key[InKey], val InVal) (maps.Key[OutKey], OutVal, error) {
			resKey, resVal, err := transform(ctx, key.Key, val)
			if err != nil {
				var zero OutVal

				return maps.Key[OutKey]{}, zero, err
			}

			return maps.Key[OutKey]{
				Key: resKey,
			}, resVal, nil
		})
	if err != nil {
		return nil, err
	}

	return maps.ToGoMap(out), nil
}

// FlatMapGoMap transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening).
//
// Unlike MapGoMap which produces one output entry per input entry, FlatMapGoMap allows each
// transform to return an entire map of results, which are then merged into the final output map.
// This is useful when one input entry should expand into multiple output entries.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use FlatMapGoMapCtx.
//
// Returns nil if the input map is nil. If multiple transforms produce the same output key,
// later entries overwrite earlier ones (the order is non-deterministic due to parallelism).
//
// Example:
//
//	// Expand each entry into multiple entries
//	input := map[string]int{"a": 2, "b": 3}
//	output, err := FlatMapGoMap(2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
//	    // Create v entries for each input entry
//	    result := make(map[string]int)
//	    for i := 0; i < v; i++ {
//	        result[fmt.Sprintf("%s%d", k, i)] = i
//	    }
//	    return result, nil
//	})
//	// output: map[string]int{"a0": 0, "a1": 1, "b0": 0, "b1": 1, "b2": 2}
func FlatMapGoMap[InKey comparable, InVal any, OutKey comparable, OutVal any](
	maxConcurrent int,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (map[OutKey]OutVal, error),
) (map[OutKey]OutVal, error) {
	return FlatMapGoMapCtx(context.Background(), maxConcurrent, input, transform)
}

// FlatMapGoMapCtx transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening).
//
// This is the context-aware version of FlatMapGoMap. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike MapGoMapCtx which produces one output entry per input entry, FlatMapGoMapCtx allows each
// transform to return an entire map of results, which are then merged into the final output map.
// This is useful when one input entry should expand into multiple output entries.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. If multiple transforms produce the same output key,
// later entries overwrite earlier ones (the order is non-deterministic due to parallelism).
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := FlatMapGoMapCtx(ctx, 2, input, func(ctx context.Context, k string, v int) (map[string]int, error) {
//	    select {
//	    case <-ctx.Done():
//	        return nil, ctx.Err()
//	    default:
//	    }
//	    result := make(map[string]int)
//	    for i := 0; i < v; i++ {
//	        result[fmt.Sprintf("%s%d", k, i)] = i
//	    }
//	    return result, nil
//	})
func FlatMapGoMapCtx[InKey comparable, InVal any, OutKey comparable, OutVal any](
	ctx context.Context,
	maxConcurrent int,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (map[OutKey]OutVal, error),
) (map[OutKey]OutVal, error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, len(input))

	result, err := FlatMapGoMapCtxWithExecutor(ctx, exec, input, transform)
	if closeErr := exec.Close(); closeErr != nil && err == nil {
		return nil, closeErr
	}

	return result, err
}

// FlatMapGoMapWithExecutor transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening), using a custom executor.
// See FlatMapGoMapCtxWithExecutor for more information.
func FlatMapGoMapWithExecutor[InKey comparable, InVal any, OutKey comparable, OutVal any](
	exec Executor,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (map[OutKey]OutVal, error),
) (map[OutKey]OutVal, error) {
	return FlatMapGoMapCtxWithExecutor[InKey, InVal, OutKey, OutVal](context.Background(), exec, input, transform)
}

// FlatMapGoMapCtxWithExecutor transforms a standard Go map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening), using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike MapGoMapCtxWithExecutor which produces one output entry per input entry,
// FlatMapGoMapCtxWithExecutor allows each transform to return an entire map of results,
// which are then merged into the final output map.
// This is useful when one input entry should expand into multiple output entries.
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. If multiple transforms produce the same output key,
// later entries overwrite earlier ones (the order is non-deterministic due to parallelism).
//
// Example:
//
//	expectedSize := 100 // expected number of tasks per batch
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := FlatMapGoMapCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, k string, v int) (map[string]int, error) {
//	        result := make(map[string]int)
//	        for i := 0; i < v; i++ {
//	            result[fmt.Sprintf("%s%d", k, i)] = i
//	        }
//	        return result, nil
//	    })
func FlatMapGoMapCtxWithExecutor[InKey comparable, InVal any, OutKey comparable, OutVal any](
	ctx context.Context,
	exec Executor,
	input map[InKey]InVal,
	transform func(ctx context.Context, key InKey, val InVal) (map[OutKey]OutVal, error),
) (map[OutKey]OutVal, error) {
	if input == nil {
		return nil, nil
	}

	in := maps.FromGoMap(input, hashing.Sha256)

	out, err := FlatMapMapCtxWithExecutor[maps.Key[InKey], InVal, maps.Key[OutKey], OutVal](ctx, exec, in,
		func(ctx context.Context, key maps.Key[InKey], val InVal) (maps.Map[maps.Key[OutKey], OutVal], error) {
			res, err := transform(ctx, key.Key, val)
			if err != nil {
				return nil, err
			}

			return maps.FromGoMap(res, hashing.Sha256), nil
		})
	if err != nil {
		return nil, err
	}

	return maps.ToGoMap(out), nil
}

// MapMap transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, producing a new Map with potentially different key and value types.
//
// This function is similar to MapGoMap but works with amp-common Map types instead of
// standard Go maps. Keys must implement the Collectable interface (hashable and comparable).
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use MapMapCtx.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// The output map may have fewer entries if the transform produces duplicate keys.
//
// Example:
//
//	// Transform map entries while preserving map type
//	input := maps.NewHashMap[MyKey, int](hashing.Sha256)
//	output, err := MapMap(2, input, func(ctx context.Context, k MyKey, v int) (MyKey, string, error) {
//	    return k, strconv.Itoa(v), nil
//	})
func MapMap[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	maxConcurrent int,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.Map[OutKey, OutVal], error) {
	return MapMapCtx[InKey, InVal, OutKey, OutVal](context.Background(), maxConcurrent, input, transform)
}

// MapMapCtx transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, producing a new Map with potentially different key and value types.
//
// This is the context-aware version of MapMap. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// This function is similar to MapGoMapCtx but works with amp-common Map types instead of
// standard Go maps. Keys must implement the Collectable interface (hashable and comparable).
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// The output map may have fewer entries if the transform produces duplicate keys.
//
// Thread-safety: The output map is built with a mutex to handle concurrent additions,
// ensuring thread-safe construction even when transforms execute in parallel.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := MapMapCtx(ctx, 2, input, func(ctx context.Context, k MyKey, v int) (MyKey, string, error) {
//	    select {
//	    case <-ctx.Done():
//	        return MyKey{}, "", ctx.Err()
//	    default:
//	    }
//	    return k, strconv.Itoa(v), nil
//	})
func MapMapCtx[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	maxConcurrent int,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.Map[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())

	result, err := MapMapCtxWithExecutor(ctx, exec, input, transform)
	if closeErr := exec.Close(); closeErr != nil && err == nil {
		return nil, closeErr
	}

	return result, err
}

// MapMapWithExecutor transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, producing a new Map with potentially different key and value types, using a custom executor.
// See MapMapCtxWithExecutor for more information.
func MapMapWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	exec Executor,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.Map[OutKey, OutVal], error) {
	return MapMapCtxWithExecutor[InKey, InVal, OutKey, OutVal](context.Background(), exec, input, transform)
}

// MapMapCtxWithExecutor transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, producing a new Map with potentially different key and value types, using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// This function is similar to MapGoMapCtxWithExecutor but works with amp-common Map types instead of
// standard Go maps. Keys must implement the Collectable interface (hashable and comparable).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// The output map may have fewer entries if the transform produces duplicate keys.
//
// Thread-safety: The output map is built with a mutex to handle concurrent additions,
// ensuring thread-safe construction even when transforms execute in parallel.
//
// Example:
//
//	expectedSize := input.Size() // or set to the expected number of tasks
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := MapMapCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, k MyKey, v int) (MyKey, string, error) {
//	        return k, strconv.Itoa(v), nil
//	    })
func MapMapCtxWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	exec Executor,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.Map[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	var mut sync.Mutex

	out := maps.NewHashMapWithSize[OutKey, OutVal](input.HashFunction(), input.Size())

	callbacks := make([]func(context.Context) error, 0, input.Size())

	for key, value := range input.Seq() {
		func(key InKey, value InVal) {
			callbacks = append(callbacks, func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				resultKey, resultVal, err := transform(ctx, key, value)
				if err != nil {
					return err
				}

				mut.Lock()
				defer mut.Unlock()

				return out.Add(resultKey, resultVal)
			})
		}(key, value)
	}

	if err := DoCtxWithExecutor(ctx, exec, callbacks...); err != nil {
		return nil, err
	}

	return out, nil
}

// FlatMapMap transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening).
//
// Unlike MapMap which produces one output entry per input entry, FlatMapMap allows each
// transform to return an entire Map of results, which are then merged into the final output map.
// This is useful when one input entry should expand into multiple output entries.
//
// This function is similar to FlatMapGoMap but works with amp-common Map types instead of
// standard Go maps. Keys must implement the Collectable interface (hashable and comparable).
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use FlatMapMapCtx.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// If multiple transforms produce the same output key, later entries overwrite earlier ones
// (the order is non-deterministic due to parallelism).
//
// Example:
//
//	// Expand each entry into multiple entries
//	input := maps.NewHashMap[MyKey, int](hashing.Sha256)
//	output, err := FlatMapMap(2, input, func(ctx context.Context, k MyKey, v int) (maps.Map[MyKey, int], error) {
//	    result := maps.NewHashMap[MyKey, int](hashing.Sha256)
//	    for i := 0; i < v; i++ {
//	        result.Add(MyKey{ID: fmt.Sprintf("%s-%d", k.ID, i)}, i)
//	    }
//	    return result, nil
//	})
func FlatMapMap[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	maxConcurrent int,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.Map[OutKey, OutVal], error),
) (maps.Map[OutKey, OutVal], error) {
	return FlatMapMapCtx[InKey, InVal, OutKey, OutVal](context.Background(), maxConcurrent, input, transform)
}

// FlatMapMapCtx transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening).
//
// This is the context-aware version of FlatMapMap. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike MapMapCtx which produces one output entry per input entry, FlatMapMapCtx allows each
// transform to return an entire Map of results, which are then merged into the final output map.
// This is useful when one input entry should expand into multiple output entries.
//
// This function is similar to FlatMapGoMapCtx but works with amp-common Map types instead of
// standard Go maps. Keys must implement the Collectable interface (hashable and comparable).
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// If multiple transforms produce the same output key, later entries overwrite earlier ones
// (the order is non-deterministic due to parallelism).
//
// Thread-safety: The output map is built with a mutex to handle concurrent additions from
// all the flattened results, ensuring thread-safe construction even when transforms execute
// in parallel.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := FlatMapMapCtx(ctx, 2, input, func(ctx context.Context, k MyKey, v int) (maps.Map[MyKey, int], error) {
//	    select {
//	    case <-ctx.Done():
//	        return nil, ctx.Err()
//	    default:
//	    }
//	    result := maps.NewHashMap[MyKey, int](hashing.Sha256)
//	    for i := 0; i < v; i++ {
//	        result.Add(MyKey{ID: fmt.Sprintf("%s-%d", k.ID, i)}, i)
//	    }
//	    return result, nil
//	})
func FlatMapMapCtx[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	maxConcurrent int,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.Map[OutKey, OutVal], error),
) (maps.Map[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())

	result, err := FlatMapMapCtxWithExecutor(ctx, exec, input, transform)
	if closeErr := exec.Close(); closeErr != nil && err == nil {
		return nil, closeErr
	}

	return result, err
}

// FlatMapMapWithExecutor transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening), using a custom executor.
// See FlatMapMapCtxWithExecutor for more information.
func FlatMapMapWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	exec Executor,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.Map[OutKey, OutVal], error),
) (maps.Map[OutKey, OutVal], error) {
	return FlatMapMapCtxWithExecutor[InKey, InVal, OutKey, OutVal](context.Background(), exec, input, transform)
}

// FlatMapMapCtxWithExecutor transforms an amp-common Map by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening), using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike MapMapCtxWithExecutor which produces one output entry per input entry, FlatMapMapCtxWithExecutor allows each
// transform to return an entire Map of results, which are then merged into the final output map.
// This is useful when one input entry should expand into multiple output entries.
//
// This function is similar to FlatMapGoMapCtxWithExecutor but works with amp-common Map types instead of
// standard Go maps. Keys must implement the Collectable interface (hashable and comparable).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// If multiple transforms produce the same output key, later entries overwrite earlier ones
// (the order is non-deterministic due to parallelism).
//
// Thread-safety: The output map is built with a mutex to handle concurrent additions from
// all the flattened results, ensuring thread-safe construction even when transforms execute
// in parallel.
//
// Example:
//
//	expectedSize := input.Size() // or set to the expected number of tasks
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := FlatMapMapCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, k MyKey, v int) (maps.Map[MyKey, int], error) {
//	        result := maps.NewHashMap[MyKey, int](hashing.Sha256)
//	        for i := 0; i < v; i++ {
//	            result.Add(MyKey{ID: fmt.Sprintf("%s-%d", k.ID, i)}, i)
//	        }
//	        return result, nil
//	    })
func FlatMapMapCtxWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	exec Executor,
	input maps.Map[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.Map[OutKey, OutVal], error),
) (maps.Map[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	var mut sync.Mutex

	out := maps.NewHashMap[OutKey, OutVal](input.HashFunction())

	callbacks := make([]func(context.Context) error, 0, input.Size())

	for key, value := range input.Seq() {
		func(key InKey, value InVal) {
			callbacks = append(callbacks, func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				res, err := transform(ctx, key, value)
				if err != nil {
					return err
				}

				mut.Lock()
				defer mut.Unlock()

				for k, v := range res.Seq() {
					if err := out.Add(k, v); err != nil {
						return err
					}
				}

				return nil
			})
		}(key, value)
	}

	if err := DoCtxWithExecutor(ctx, exec, callbacks...); err != nil {
		return nil, err
	}

	return out, nil
}
