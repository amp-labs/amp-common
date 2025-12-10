package utils

import (
	"errors"
	"fmt"
	"os"
)

var (
	// ErrGetwd is returned when os.Getwd fails.
	ErrGetwd = errors.New("os.Getwd failed")
	// ErrChdir is returned when os.Chdir fails.
	ErrChdir = errors.New("os.Chdir failed")
)

// Pushd will chdir to a different directory, call the callback,
// and then restore the old working directory when the function exits.
func Pushd(path string, callback func() error) error {
	if path == "." {
		return callback()
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGetwd, err)
	}

	if err := os.Chdir(path); err != nil { //nolint:noinlineerr // Inline error handling is clear here
		return fmt.Errorf("%w: %w", ErrChdir, err)
	}

	var errs []error

	e := callback()
	if e != nil {
		errs = append(errs, e)
	}

	e = os.Chdir(workingDir)
	if e != nil {
		errs = append(errs, fmt.Errorf("%w: %w", ErrChdir, e))
	}

	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errors.Join(errs...)
}
