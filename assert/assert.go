// Package assert provides type assertion utilities with error handling.
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

// True asserts that the given value is true.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func True(value bool, args ...any) {
	if value {
		return
	}

	if len(args) == 0 {
		panic("assertion failed")
	}

	first := args[0]
	remaining := args[1:]

	if firstStr, ok := first.(string); ok {
		panic(fmt.Sprintf(firstStr, remaining...))
	} else {
		panic(fmt.Sprintf("assertion failed: %v", args))
	}
}

// False asserts that the given value is false.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func False(value bool, args ...any) {
	True(!value, args...)
}

// Nil asserts that the given value is nil.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func Nil(value any, args ...any) {
	True(value == nil, args...)
}

// NotNil asserts that the given value is not nil.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func NotNil(value any, args ...any) {
	True(value != nil, args...)
}
