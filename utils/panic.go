package utils //nolint:revive // utils is an appropriate package name for utility functions

import (
	"fmt"

	"github.com/amp-labs/amp-common/errors"
)

// GetPanicRecoveryError converts a recovered panic value and optional stack trace
// into a standard error. If the panic value is nil, it returns nil.
// If the panic value is an error, it wraps it with ErrPanicRecovery.
// If the panic value is not an error, it formats it as a string and wraps it.
// If a stack trace is provided, it appends it to the error message.
func GetPanicRecoveryError(err any, stack []byte) error {
	if err == nil {
		return nil
	}

	errErr, ok := err.(error)
	if ok {
		if stack != nil {
			return fmt.Errorf("%w: %w\nstack trace:\n%s", errors.ErrPanicRecovery, errErr, string(stack))
		}

		return fmt.Errorf("%w: %w", errors.ErrPanicRecovery, errErr)
	} else {
		if stack != nil {
			return fmt.Errorf("%w: %v\nstack trace:\n%s", errors.ErrPanicRecovery, err, string(stack))
		}

		return fmt.Errorf("%w: %v", errors.ErrPanicRecovery, err)
	}
}
