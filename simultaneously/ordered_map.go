// Package simultaneously provides parallel ordered map transformation functions.
// This file contains utilities for concurrently transforming ordered map entries
// while preserving insertion order.
package simultaneously

import (
	"context"

	"github.com/amp-labs/amp-common/maps"
)

// MapOrderedMap transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, producing a new OrderedMap with potentially different key and value types.
//
// Unlike MapMap, this function preserves the insertion order of entries. The output map will have
// entries in the same order as the input map, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use MapOrderedMapCtx.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// Order preservation: outputs[i] corresponds to inputs[i] in insertion order.
//
// Example:
//
//	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
//	input.Add(maps.Key[string]{Key: "first"}, 1)
//	input.Add(maps.Key[string]{Key: "second"}, 2)
//	output, err := MapOrderedMap(2, input,
//	    func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], string, error) {
//	        return k, strconv.Itoa(v), nil
//	    })
//	// output has entries in order: "first" -> "1", "second" -> "2"
func MapOrderedMap[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	maxConcurrent int,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	return MapOrderedMapCtx[InKey, InVal, OutKey, OutVal](context.Background(), maxConcurrent, input, transform)
}

// MapOrderedMapCtx transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, producing a new OrderedMap with potentially different key and value types.
//
// This is the context-aware version of MapOrderedMap. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike MapMapCtx, this function preserves the insertion order of entries. The output map will have
// entries in the same order as the input map, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// Order preservation: outputs[i] corresponds to inputs[i] in insertion order.
//
// Thread-safety: Results are collected in parallel with a mutex, then added to the output map
// in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := MapOrderedMapCtx(ctx, 2, input,
//	    func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], string, error) {
//	        select {
//	        case <-ctx.Done():
//	            return maps.Key[string]{}, "", ctx.Err()
//	        default:
//	        }
//	        return k, strconv.Itoa(v), nil
//	    })
func MapOrderedMapCtx[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	maxConcurrent int,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())

	result, err := MapOrderedMapCtxWithExecutor(ctx, exec, input, transform)
	if closeErr := exec.Close(); closeErr != nil && err == nil {
		return nil, closeErr
	}

	return result, err
}

// FlatMapOrderedMap transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening).
//
// Unlike FlatMapMap, this function preserves order. Results from inputs[i] appear before results
// from inputs[i+1] in the output map's insertion order, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use FlatMapOrderedMapCtx.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// Order preservation: all results from inputs[i] appear before all results from inputs[i+1].
//
// Example:
//
//	input := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
//	input.Add(maps.Key[string]{Key: "a"}, 2)
//	input.Add(maps.Key[string]{Key: "b"}, 3)
//	output, err := FlatMapOrderedMap(2, input,
//	    func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
//	        result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
//	        for i := 0; i < v; i++ {
//	            key := maps.Key[string]{Key: fmt.Sprintf("%s%d", k.Key, i)}
//	            result.Add(key, i)
//	        }
//	        return result, nil
//	    })
//	// output: a0->0, a1->1, b0->0, b1->1, b2->2 (in this order)
func FlatMapOrderedMap[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	maxConcurrent int,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.OrderedMap[OutKey, OutVal], error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	return FlatMapOrderedMapCtx[InKey, InVal, OutKey, OutVal](context.Background(), maxConcurrent, input, transform)
}

// FlatMapOrderedMapCtx transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening).
//
// This is the context-aware version of FlatMapOrderedMap. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike FlatMapMapCtx, this function preserves order. Results from inputs[i] appear before results
// from inputs[i+1] in the output map's insertion order, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each entry in the input map with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// Order preservation: all results from inputs[i] appear before all results from inputs[i+1].
//
// Thread-safety: Results are collected in parallel with a mutex, then flattened and added to
// the output map in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := FlatMapOrderedMapCtx(ctx, 2, input,
//	    func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
//	        select {
//	        case <-ctx.Done():
//	            return nil, ctx.Err()
//	        default:
//	        }
//	        result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
//	        for i := 0; i < v; i++ {
//	            result.Add(maps.Key[string]{Key: fmt.Sprintf("%s%d", k.Key, i)}, i)
//	        }
//	        return result, nil
//	    })
func FlatMapOrderedMapCtx[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	maxConcurrent int,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.OrderedMap[OutKey, OutVal], error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())

	result, err := FlatMapOrderedMapCtxWithExecutor(ctx, exec, input, transform)
	if closeErr := exec.Close(); closeErr != nil && err == nil {
		return nil, closeErr
	}

	return result, err
}

// MapOrderedMapWithExecutor transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, producing a new OrderedMap with potentially different key and value types, using a custom executor.
// See MapOrderedMapCtxWithExecutor for more information.
func MapOrderedMapWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	exec Executor,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	return MapOrderedMapCtxWithExecutor[InKey, InVal, OutKey, OutVal](context.Background(), exec, input, transform)
}

// MapOrderedMapCtxWithExecutor transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, producing a new OrderedMap with potentially different key and value types, using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike MapMapCtxWithExecutor, this function preserves the insertion order of entries. The output map will have
// entries in the same order as the input map, even though transforms execute in parallel.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// Order preservation: outputs[i] corresponds to inputs[i] in insertion order.
//
// Thread-safety: Results are collected in parallel with a mutex, then added to the output map
// in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := MapOrderedMapCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, k maps.Key[string], v int) (maps.Key[string], string, error) {
//	        return k, strconv.Itoa(v), nil
//	    })
func MapOrderedMapCtxWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	exec Executor,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (OutKey, OutVal, error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	// Convert OrderedMap to slice of key-value pairs
	type inPair struct {
		key InKey
		val InVal
	}

	type outPair struct {
		key OutKey
		val OutVal
	}

	pairs := make([]inPair, 0, input.Size())
	for _, entry := range input.Seq() {
		pairs = append(pairs, inPair{key: entry.Key, val: entry.Value})
	}

	// Transform using MapSliceCtxWithExecutor to preserve order
	transformed, err := MapSliceCtxWithExecutor(ctx, exec, pairs,
		func(ctx context.Context, pair inPair) (outPair, error) {
			k, v, err := transform(ctx, pair.key, pair.val)
			if err != nil {
				var zero outPair

				return zero, err
			}

			return outPair{key: k, val: v}, nil
		})
	if err != nil {
		return nil, err
	}

	// Reconstruct OrderedMap from transformed pairs
	out := maps.NewOrderedHashMap[OutKey, OutVal](input.HashFunction())

	for _, pair := range transformed {
		if err := out.Add(pair.key, pair.val); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// FlatMapOrderedMapWithExecutor transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening), using a custom executor.
// See FlatMapOrderedMapCtxWithExecutor for more information.
func FlatMapOrderedMapWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	exec Executor,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.OrderedMap[OutKey, OutVal], error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	return FlatMapOrderedMapCtxWithExecutor[InKey, InVal, OutKey, OutVal](context.Background(), exec, input, transform)
}

// FlatMapOrderedMapCtxWithExecutor transforms an OrderedMap by applying a transform function to each key-value pair
// in parallel, where each transform can produce multiple output entries (flattening), using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike FlatMapMapCtxWithExecutor, this function preserves order. Results from inputs[i] appear before results
// from inputs[i+1] in the output map's insertion order, even though transforms execute in parallel.
//
// Returns nil if the input map is nil. The output map uses the same hash function as the input.
// Order preservation: all results from inputs[i] appear before all results from inputs[i+1].
//
// Thread-safety: Results are collected in parallel with a mutex, then flattened and added to
// the output map in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := FlatMapOrderedMapCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, k maps.Key[string], v int) (maps.OrderedMap[maps.Key[string], int], error) {
//	        result := maps.NewOrderedHashMap[maps.Key[string], int](hashing.Sha256)
//	        for i := 0; i < v; i++ {
//	            result.Add(maps.Key[string]{Key: fmt.Sprintf("%s%d", k.Key, i)}, i)
//	        }
//	        return result, nil
//	    })
func FlatMapOrderedMapCtxWithExecutor[InKey Collectable[InKey], InVal any, OutKey Collectable[OutKey], OutVal any](
	ctx context.Context,
	exec Executor,
	input maps.OrderedMap[InKey, InVal],
	transform func(ctx context.Context, key InKey, val InVal) (maps.OrderedMap[OutKey, OutVal], error),
) (maps.OrderedMap[OutKey, OutVal], error) {
	if input == nil {
		return nil, nil
	}

	// Convert OrderedMap to slice of key-value pairs
	type inPair struct {
		key InKey
		val InVal
	}

	type outPair struct {
		key OutKey
		val OutVal
	}

	pairs := make([]inPair, 0, input.Size())
	for _, entry := range input.Seq() {
		pairs = append(pairs, inPair{key: entry.Key, val: entry.Value})
	}

	// Transform using FlatMapSliceCtxWithExecutor to preserve order
	flattened, err := FlatMapSliceCtxWithExecutor(ctx, exec, pairs,
		func(ctx context.Context, pair inPair) ([]outPair, error) {
			res, err := transform(ctx, pair.key, pair.val)
			if err != nil {
				return nil, err
			}

			// Convert OrderedMap result to slice of pairs
			outPairs := make([]outPair, 0, res.Size())
			for _, entry := range res.Seq() {
				outPairs = append(outPairs, outPair{key: entry.Key, val: entry.Value})
			}

			return outPairs, nil
		})
	if err != nil {
		return nil, err
	}

	// Reconstruct OrderedMap from flattened pairs
	out := maps.NewOrderedHashMap[OutKey, OutVal](input.HashFunction())

	for _, pair := range flattened {
		if err := out.Add(pair.key, pair.val); err != nil {
			return nil, err
		}
	}

	return out, nil
}
