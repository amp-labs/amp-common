// Package simultaneously provides parallel ordered set transformation functions.
// This file contains utilities for concurrently transforming ordered set elements
// while preserving insertion order.
package simultaneously

import (
	"context"

	"github.com/amp-labs/amp-common/set"
)

// MapOrderedSet transforms an OrderedSet by applying a transform function to each element
// in parallel, producing a new OrderedSet with potentially different element types.
//
// Unlike MapSet, this function preserves the insertion order of elements. The output set will have
// elements in the same order as the input set, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use MapOrderedSetCtx.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// Order preservation: outputs[i] corresponds to inputs[i] in insertion order.
//
// Example:
//
//	input := set.NewOrderedSet[hashing.HashableInt](hashing.Sha256)
//	input.Add(hashing.HashableInt(1))
//	input.Add(hashing.HashableInt(2))
//	input.Add(hashing.HashableInt(3))
//	output, err := MapOrderedSet(2, input,
//	    func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
//	        return hashing.HashableString(strconv.Itoa(int(v))), nil
//	    })
//	// output has elements in order: "1", "2", "3"
func MapOrderedSet[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	maxConcurrent int,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.OrderedSet[OutElem], error) {
	return MapOrderedSetCtx[InElem, OutElem](context.Background(), maxConcurrent, input, transform)
}

// MapOrderedSetCtx transforms an OrderedSet by applying a transform function to each element
// in parallel, producing a new OrderedSet with potentially different element types.
//
// This is the context-aware version of MapOrderedSet. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike MapSetCtx, this function preserves the insertion order of elements. The output set will have
// elements in the same order as the input set, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// Order preservation: outputs[i] corresponds to inputs[i] in insertion order.
//
// Thread-safety: Results are collected in parallel using MapSlice, then added to the output set
// in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := MapOrderedSetCtx(ctx, 2, input,
//	    func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
//	        select {
//	        case <-ctx.Done():
//	            return "", ctx.Err()
//	        default:
//	        }
//	        return hashing.HashableString(strconv.Itoa(int(v))), nil
//	    })
func MapOrderedSetCtx[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	maxConcurrent int,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.OrderedSet[OutElem], error) {
	if input == nil {
		return nil, nil
	}

	// Convert OrderedSet to slice
	elements := make([]InElem, 0, input.Size())
	for _, elem := range input.Seq() {
		elements = append(elements, elem)
	}

	// Transform using MapSliceCtx to preserve order
	transformed, err := MapSliceCtx(ctx, maxConcurrent, elements, transform)
	if err != nil {
		return nil, err
	}

	// Reconstruct OrderedSet from transformed elements
	out := set.NewOrderedSet[OutElem](input.HashFunction())

	for _, elem := range transformed {
		if err := out.Add(elem); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// FlatMapOrderedSet transforms an OrderedSet by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening).
//
// Unlike FlatMapSet, this function preserves order. Results from inputs[i] appear before results
// from inputs[i+1] in the output set's insertion order, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use FlatMapOrderedSetCtx.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// Order preservation: all results from inputs[i] appear before all results from inputs[i+1].
//
// Example:
//
//	input := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
//	input.Add(hashing.HashableString("ab"))
//	input.Add(hashing.HashableString("cd"))
//	output, err := FlatMapOrderedSet(2, input,
//	    func(ctx context.Context, s hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
//	        result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
//	        for _, ch := range string(s) {
//	            result.Add(hashing.HashableString(string(ch)))
//	        }
//	        return result, nil
//	    })
//	// output: "a", "b", "c", "d" (in this order)
func FlatMapOrderedSet[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	maxConcurrent int,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (set.OrderedSet[OutElem], error),
) (set.OrderedSet[OutElem], error) {
	return FlatMapOrderedSetCtx[InElem, OutElem](context.Background(), maxConcurrent, input, transform)
}

// FlatMapOrderedSetCtx transforms an OrderedSet by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening).
//
// This is the context-aware version of FlatMapOrderedSet. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike FlatMapSetCtx, this function preserves order. Results from inputs[i] appear before results
// from inputs[i+1] in the output set's insertion order, even though transforms execute in parallel.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// Order preservation: all results from inputs[i] appear before all results from inputs[i+1].
//
// Thread-safety: Results are collected in parallel using FlatMapSlice, then flattened and added
// to the output set in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := FlatMapOrderedSetCtx(ctx, 2, input,
//	    func(ctx context.Context, s hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
//	        select {
//	        case <-ctx.Done():
//	            return nil, ctx.Err()
//	        default:
//	        }
//	        result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
//	        for _, ch := range string(s) {
//	            result.Add(hashing.HashableString(string(ch)))
//	        }
//	        return result, nil
//	    })
func FlatMapOrderedSetCtx[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	maxConcurrent int,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (set.OrderedSet[OutElem], error),
) (result set.OrderedSet[OutElem], err error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())
	defer func() {
		if closeErr := exec.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return FlatMapOrderedSetCtxWithExecutor(ctx, exec, input, transform)
}

// MapOrderedSetWithExecutor transforms an OrderedSet by applying a transform function to each element
// in parallel, producing a new OrderedSet with potentially different element types, using a custom executor.
// See MapOrderedSetCtxWithExecutor for more information.
func MapOrderedSetWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	exec Executor,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.OrderedSet[OutElem], error) {
	return MapOrderedSetCtxWithExecutor[InElem, OutElem](context.Background(), exec, input, transform)
}

// MapOrderedSetCtxWithExecutor transforms an OrderedSet by applying a transform function to each element
// in parallel, producing a new OrderedSet with potentially different element types, using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike MapSetCtxWithExecutor, this function preserves the insertion order of elements. The output set
// will have elements in the same order as the input set, even though transforms execute in parallel.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// Order preservation: outputs[i] corresponds to inputs[i] in insertion order.
//
// Thread-safety: Results are collected in parallel using MapSliceCtxWithExecutor, then added to the output set
// in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	exec := newDefaultExecutor(2, expectedSize)
//	defer exec.Close()
//
//	output, err := MapOrderedSetCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, v hashing.HashableInt) (hashing.HashableString, error) {
//	        return hashing.HashableString(strconv.Itoa(int(v))), nil
//	    })
func MapOrderedSetCtxWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	exec Executor,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.OrderedSet[OutElem], error) {
	if input == nil {
		return nil, nil
	}

	// Convert OrderedSet to slice
	elements := make([]InElem, 0, input.Size())
	for _, elem := range input.Seq() {
		elements = append(elements, elem)
	}

	// Transform using MapSliceCtxWithExecutor to preserve order
	transformed, err := MapSliceCtxWithExecutor(ctx, exec, elements, transform)
	if err != nil {
		return nil, err
	}

	// Reconstruct OrderedSet from transformed elements
	out := set.NewOrderedSet[OutElem](input.HashFunction())

	for _, elem := range transformed {
		if err := out.Add(elem); err != nil {
			return nil, err
		}
	}

	return out, nil
}

// FlatMapOrderedSetWithExecutor transforms an OrderedSet by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening), using a custom executor.
// See FlatMapOrderedSetCtxWithExecutor for more information.
func FlatMapOrderedSetWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	exec Executor,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (set.OrderedSet[OutElem], error),
) (set.OrderedSet[OutElem], error) {
	return FlatMapOrderedSetCtxWithExecutor[InElem, OutElem](context.Background(), exec, input, transform)
}

// FlatMapOrderedSetCtxWithExecutor transforms an OrderedSet by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening), using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike FlatMapSetCtxWithExecutor, this function preserves order. Results from inputs[i] appear before results
// from inputs[i+1] in the output set's insertion order, even though transforms execute in parallel.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// Order preservation: all results from inputs[i] appear before all results from inputs[i+1].
//
// Thread-safety: Results are collected in parallel using FlatMapSliceCtxWithExecutor, then flattened and added
// to the output set in the original insertion order to preserve ordering semantics.
//
// Example:
//
//	exec := newDefaultExecutor(2, input.Size())
//	defer exec.Close()
//
//	output, err := FlatMapOrderedSetCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, s hashing.HashableString) (set.OrderedSet[hashing.HashableString], error) {
//	        result := set.NewOrderedSet[hashing.HashableString](hashing.Sha256)
//	        for _, ch := range string(s) {
//	            result.Add(hashing.HashableString(string(ch)))
//	        }
//	        return result, nil
//	    })
func FlatMapOrderedSetCtxWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	exec Executor,
	input set.OrderedSet[InElem],
	transform func(ctx context.Context, elem InElem) (set.OrderedSet[OutElem], error),
) (set.OrderedSet[OutElem], error) {
	if input == nil {
		return nil, nil
	}

	// Convert OrderedSet to slice
	elements := make([]InElem, 0, input.Size())
	for _, elem := range input.Seq() {
		elements = append(elements, elem)
	}

	// Transform using FlatMapSliceCtxWithExecutor to preserve order
	flattened, err := FlatMapSliceCtxWithExecutor(ctx, exec, elements,
		func(ctx context.Context, elem InElem) ([]OutElem, error) {
			res, err := transform(ctx, elem)
			if err != nil {
				return nil, err
			}

			// Convert OrderedSet result to slice
			outElems := make([]OutElem, 0, res.Size())
			for _, e := range res.Seq() {
				outElems = append(outElems, e)
			}

			return outElems, nil
		})
	if err != nil {
		return nil, err
	}

	// Reconstruct OrderedSet from flattened elements
	out := set.NewOrderedSet[OutElem](input.HashFunction())

	for _, elem := range flattened {
		if err := out.Add(elem); err != nil {
			return nil, err
		}
	}

	return out, nil
}
