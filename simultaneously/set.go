// Package simultaneously provides parallel set transformation functions.
// This file contains utilities for concurrently transforming set elements.
package simultaneously

import (
	"context"
	"sync"

	"github.com/amp-labs/amp-common/set"
)

// MapSet transforms a Set by applying a transform function to each element
// in parallel, producing a new Set with potentially different element types.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use MapSetCtx.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// The output set may have fewer elements if the transform produces duplicate elements.
//
// Example:
//
//	// Transform string set to int set by converting to lengths in parallel
//	input := set.NewSet[hashing.HashableString](hashing.Sha256)
//	input.Add(hashing.HashableString("hello"))
//	input.Add(hashing.HashableString("world"))
//	output, err := MapSet(2, input, func(ctx context.Context, s hashing.HashableString) (hashing.HashableInt, error) {
//	    return hashing.HashableInt(len(s)), nil
//	})
//	// output contains: HashableInt(5) for both "hello" and "world"
func MapSet[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	maxConcurrent int,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.Set[OutElem], error) {
	return MapSetCtx[InElem, OutElem](context.Background(), maxConcurrent, input, transform)
}

// MapSetCtx transforms a Set by applying a transform function to each element
// in parallel, producing a new Set with potentially different element types.
//
// This is the context-aware version of MapSet. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// The output set may have fewer elements if the transform produces duplicate elements.
//
// Thread-safety: The output set is built with a mutex to handle concurrent additions,
// ensuring thread-safe construction even when transforms execute in parallel.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := MapSetCtx(ctx, 2, input,
//	    func(ctx context.Context, s hashing.HashableString) (hashing.HashableInt, error) {
//	        select {
//	        case <-ctx.Done():
//	            return 0, ctx.Err()
//	        default:
//	        }
//	        return hashing.HashableInt(len(s)), nil
//	    })
func MapSetCtx[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	maxConcurrent int,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (result set.Set[OutElem], err error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())
	defer func() {
		if closeErr := exec.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return MapSetCtxWithExecutor(ctx, exec, input, transform)
}

// FlatMapSet transforms a Set by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening).
//
// Unlike MapSet which produces one output element per input element, FlatMapSet allows each
// transform to return an entire Set of results, which are then merged into the final output set.
// This is useful when one input element should expand into multiple output elements.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set. If any transform
// returns an error, the operation stops and returns that error immediately, canceling
// any remaining transforms.
//
// This is the non-context version that uses context.Background(). For context-aware
// operations with cancellation support, use FlatMapSetCtx.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// If multiple transforms produce the same output element, duplicates are automatically handled
// by the set semantics.
//
// Example:
//
//	// Expand each string into its individual characters
//	input := set.NewSet[hashing.HashableString](hashing.Sha256)
//	input.Add(hashing.HashableString("hi"))
//	output, err := FlatMapSet(2, input,
//	    func(ctx context.Context, s hashing.HashableString) (set.Set[hashing.HashableString], error) {
//	        result := set.NewSet[hashing.HashableString](hashing.Sha256)
//	        for _, ch := range string(s) {
//	            result.Add(hashing.HashableString(string(ch)))
//	        }
//	        return result, nil
//	    })
//	// output contains: "h", "i"
func FlatMapSet[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	maxConcurrent int,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (set.Set[OutElem], error),
) (set.Set[OutElem], error) {
	return FlatMapSetCtx[InElem, OutElem](context.Background(), maxConcurrent, input, transform)
}

// FlatMapSetCtx transforms a Set by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening).
//
// This is the context-aware version of FlatMapSet. The provided context can be used to cancel
// the operation or set deadlines. If the context is canceled, the operation stops immediately
// and returns the context's error.
//
// Unlike MapSetCtx which produces one output element per input element, FlatMapSetCtx allows each
// transform to return an entire Set of results, which are then merged into the final output set.
// This is useful when one input element should expand into multiple output elements.
//
// The maxConcurrent parameter limits the number of concurrent transform operations.
// Set to 0 for unlimited concurrency (bounded only by available goroutines).
//
// The transform function is called for each element in the input set with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// If multiple transforms produce the same output element, duplicates are automatically handled
// by the set semantics.
//
// Thread-safety: The output set is built with a mutex to handle concurrent additions from
// all the flattened results, ensuring thread-safe construction even when transforms execute
// in parallel.
//
// Example:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	output, err := FlatMapSetCtx(ctx, 2, input,
//	    func(ctx context.Context, s hashing.HashableString) (set.Set[hashing.HashableString], error) {
//	        select {
//	        case <-ctx.Done():
//	            return nil, ctx.Err()
//	        default:
//	        }
//	        result := set.NewSet[hashing.HashableString](hashing.Sha256)
//	        for _, ch := range string(s) {
//	            result.Add(hashing.HashableString(string(ch)))
//	        }
//	        return result, nil
//	    })
func FlatMapSetCtx[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	maxConcurrent int,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (set.Set[OutElem], error),
) (result set.Set[OutElem], err error) {
	if input == nil {
		return nil, nil
	}

	exec := newDefaultExecutor(maxConcurrent, input.Size())
	defer func() {
		if closeErr := exec.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	return FlatMapSetCtxWithExecutor(ctx, exec, input, transform)
}

// MapSetWithExecutor transforms a Set by applying a transform function to each element
// in parallel, producing a new Set with potentially different element types, using a custom executor.
// See MapSetCtxWithExecutor for more information.
func MapSetWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	exec Executor,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.Set[OutElem], error) {
	return MapSetCtxWithExecutor[InElem, OutElem](context.Background(), exec, input, transform)
}

// MapSetCtxWithExecutor transforms a Set by applying a transform function to each element
// in parallel, producing a new Set with potentially different element types, using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// The transform function is called for each element in the input set with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// The output set may have fewer elements if the transform produces duplicate elements.
//
// Thread-safety: The output set is built with a mutex to handle concurrent additions,
// ensuring thread-safe construction even when transforms execute in parallel.
//
// Example:
//
//	exec := NewDefaultExecutor(2)
//	defer exec.Close()
//
//	output, err := MapSetCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, s hashing.HashableString) (hashing.HashableInt, error) {
//	        return hashing.HashableInt(len(s)), nil
//	    })
func MapSetCtxWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	exec Executor,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (OutElem, error),
) (set.Set[OutElem], error) {
	if input == nil {
		return nil, nil
	}

	var mut sync.Mutex

	out := set.NewSet[OutElem](input.HashFunction())

	callbacks := make([]func(context.Context) error, 0, input.Size())

	for elem := range input.Seq() {
		func(elem InElem) {
			callbacks = append(callbacks, func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				result, err := transform(ctx, elem)
				if err != nil {
					return err
				}

				mut.Lock()
				defer mut.Unlock()

				return out.Add(result)
			})
		}(elem)
	}

	if err := DoCtxWithExecutor(ctx, exec, callbacks...); err != nil {
		return nil, err
	}

	return out, nil
}

// FlatMapSetWithExecutor transforms a Set by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening), using a custom executor.
// See FlatMapSetCtxWithExecutor for more information.
func FlatMapSetWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	exec Executor,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (set.Set[OutElem], error),
) (set.Set[OutElem], error) {
	return FlatMapSetCtxWithExecutor[InElem, OutElem](context.Background(), exec, input, transform)
}

// FlatMapSetCtxWithExecutor transforms a Set by applying a transform function to each element
// in parallel, where each transform can produce multiple output elements (flattening), using a custom executor.
//
// This is useful when you want to reuse an executor across multiple batches of work or when you need
// custom execution behavior. The executor is not closed by this function, allowing it to be reused.
//
// Unlike MapSetCtxWithExecutor which produces one output element per input element, FlatMapSetCtxWithExecutor
// allows each transform to return an entire Set of results, which are then merged into the final output set.
//
// The transform function is called for each element in the input set with the provided context.
// If any transform returns an error, the operation stops and returns that error immediately,
// canceling any remaining transforms.
//
// Returns nil if the input set is nil. The output set uses the same hash function as the input.
// If multiple transforms produce the same output element, duplicates are automatically handled
// by the set semantics.
//
// Thread-safety: The output set is built with a mutex to handle concurrent additions from
// all the flattened results, ensuring thread-safe construction even when transforms execute
// in parallel.
//
// Example:
//
//	exec := NewDefaultExecutor(2)
//	defer exec.Close()
//
//	output, err := FlatMapSetCtxWithExecutor(ctx, exec, input,
//	    func(ctx context.Context, s hashing.HashableString) (set.Set[hashing.HashableString], error) {
//	        result := set.NewSet[hashing.HashableString](hashing.Sha256)
//	        for _, ch := range string(s) {
//	            result.Add(hashing.HashableString(string(ch)))
//	        }
//	        return result, nil
//	    })
func FlatMapSetCtxWithExecutor[InElem Collectable[InElem], OutElem Collectable[OutElem]](
	ctx context.Context,
	exec Executor,
	input set.Set[InElem],
	transform func(ctx context.Context, elem InElem) (set.Set[OutElem], error),
) (set.Set[OutElem], error) {
	if input == nil {
		return nil, nil
	}

	var mut sync.Mutex

	out := set.NewSet[OutElem](input.HashFunction())

	callbacks := make([]func(context.Context) error, 0, input.Size())

	for elem := range input.Seq() {
		func(elem InElem) {
			callbacks = append(callbacks, func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				res, err := transform(ctx, elem)
				if err != nil {
					return err
				}

				mut.Lock()
				defer mut.Unlock()

				for e := range res.Seq() {
					if err := out.Add(e); err != nil {
						return err
					}
				}

				return nil
			})
		}(elem)
	}

	if err := DoCtxWithExecutor(ctx, exec, callbacks...); err != nil {
		return nil, err
	}

	return out, nil
}
