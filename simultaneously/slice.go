package simultaneously

import (
	"context"
	"sync"
)

// MapSlice transforms a slice of values in parallel by applying a function to each element.
// See MapSliceCtx for more information.
func MapSlice[Input, Output any](
	maxConcurrent int,
	values []Input,
	transform func(ctx context.Context, value Input) (Output, error),
) ([]Output, error) {
	return MapSliceCtx(context.Background(), maxConcurrent, values, transform)
}

// MapSliceCtx transforms a slice of values in parallel by applying a function to each element.
// It returns a new slice containing the transformed values in the same order as the input.
//
// The maxConcurrent parameter limits the number of concurrent transformations.
// If maxConcurrent is less than 1, all transformations will run at the same time.
//
// If any transformation returns an error, all remaining transformations are canceled
// (via their context) and the first error is returned. The output slice will be nil.
//
// Panics that occur within the transformation function are automatically recovered
// and converted to errors. Order is preserved: outputs[i] corresponds to values[i].
//
// Example:
//
//	numbers := []int{1, 2, 3, 4, 5}
//	doubled, err := MapSliceCtx(ctx, 2, numbers, func(ctx context.Context, n int) (int, error) {
//	    return n * 2, nil
//	})
//	// doubled = [2, 4, 6, 8, 10]
func MapSliceCtx[Input, Output any](
	ctx context.Context,
	maxConcurrent int,
	values []Input,
	transform func(ctx context.Context, value Input) (Output, error),
) ([]Output, error) {
	if len(values) == 0 {
		return nil, nil //nolint:nilnil
	}

	var mut sync.Mutex

	outputs := make([]Output, len(values))

	callbacks := make([]func(context.Context) error, 0, len(values))

	for idx, value := range values {
		func(idx int, value Input) {
			callbacks = append(callbacks, func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				result, err := transform(ctx, value)
				if err != nil {
					return err
				}

				mut.Lock()
				outputs[idx] = result
				mut.Unlock()

				return nil
			})
		}(idx, value)
	}

	if err := DoCtx(ctx, maxConcurrent, callbacks...); err != nil {
		return nil, err
	}

	return outputs, nil
}

// FlatMapSlice transforms a slice of values in parallel where each input produces zero or more outputs,
// then flattens the results into a single slice. See FlatMapSliceCtx for more information.
func FlatMapSlice[Input, Output any](
	maxConcurrent int,
	values []Input,
	transform func(ctx context.Context, value Input) ([]Output, error),
) ([]Output, error) {
	return FlatMapSliceCtx(context.Background(), maxConcurrent, values, transform)
}

// FlatMapSliceCtx transforms a slice of values in parallel where each input produces zero or more outputs,
// then flattens the results into a single slice. This is useful when each input element needs to be
// expanded into multiple output elements.
//
// The maxConcurrent parameter limits the number of concurrent transformations.
// If maxConcurrent is less than 1, all transformations will run at the same time.
//
// If any transformation returns an error, all remaining transformations are canceled
// (via their context) and the first error is returned. The output slice will be nil.
//
// Panics that occur within the transformation function are automatically recovered
// and converted to errors. Order is preserved: results from values[i] appear before
// results from values[i+1] in the flattened output.
//
// Example:
//
//	words := []string{"hello", "world"}
//	chars, err := FlatMapSliceCtx(ctx, 2, words, func(ctx context.Context, word string) ([]rune, error) {
//	    return []rune(word), nil
//	})
//	// chars = ['h', 'e', 'l', 'l', 'o', 'w', 'o', 'r', 'l', 'd']
func FlatMapSliceCtx[Input, Output any](
	ctx context.Context,
	maxConcurrent int,
	values []Input,
	transform func(ctx context.Context, value Input) ([]Output, error),
) ([]Output, error) {
	if len(values) == 0 {
		return nil, nil //nolint:nilnil
	}

	var mut sync.Mutex

	outputs := make([][]Output, len(values))

	callbacks := make([]func(context.Context) error, 0, len(values))

	for idx, value := range values {
		func(idx int, value Input) {
			callbacks = append(callbacks, func(ctx context.Context) error {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				result, err := transform(ctx, value)
				if err != nil {
					return err
				}

				mut.Lock()
				outputs[idx] = result
				mut.Unlock()

				return nil
			})
		}(idx, value)
	}

	if err := DoCtx(ctx, maxConcurrent, callbacks...); err != nil {
		return nil, err
	}

	totalSize := 0
	for _, val := range outputs {
		totalSize += len(val)
	}

	if totalSize == 0 {
		return nil, nil // nolint:nilnil
	}

	flattened := make([]Output, 0, totalSize)

	for _, val := range outputs {
		flattened = append(flattened, val...)
	}

	return flattened, nil
}
