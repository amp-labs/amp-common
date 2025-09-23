package assert

import (
	"fmt"

	"github.com/amp-labs/amp-common/errors"
)

// Type asserts that the given value is of the expected type T.
// If the assertion fails, it returns an error indicating the mismatch.
//
//nolint:ireturn
func Type[T any](val any) (T, error) {
	of, ok := val.(T)
	if !ok {
		return of, fmt.Errorf("%w: expected type %T, but received %T", errors.ErrWrongType, of, val)
	}

	return of, nil
}
