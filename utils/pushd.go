package utils

import (
	"errors"
	"fmt"
	"os"
)

// Pushd will chdir to a different directory, call the callback,
// and then restore the old working directory when the function exits.
func Pushd(path string, f func() error) error {
	if path == "." {
		return f()
	}

	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("os.Getwd: %v", err)
	}

	if err := os.Chdir(path); err != nil {
		return fmt.Errorf("os.Chdir: %v", err)
	}

	var errs []error

	if e := f(); e != nil {
		errs = append(errs, e)
	}

	if e := os.Chdir(wd); e != nil {
		errs = append(errs, fmt.Errorf("os.Chdir: %v", err))
	}

	if len(errs) == 0 {
		return nil
	}

	if len(errs) == 1 {
		return errs[0]
	}

	return errors.Join(errs...)
}
