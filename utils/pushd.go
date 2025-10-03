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
func Pushd(path string, fn func() error) error {
	if path == "." {
		return fn()
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrGetwd, err)
	}

	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("%w: %w", ErrChdir, err)
	}

	var errs []error
	if e := fn(); e != nil {
		errs = append(errs, e)
	}

	if e := os.Chdir(wd); e != nil {
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
