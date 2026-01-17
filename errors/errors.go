// Package errors provides error utilities with collection support for managing multiple errors.
package errors //nolint:revive // This is a fine package name, nuts to you

import (
	"errors"
	"fmt"
	"runtime/debug"
)

var (
	// ErrNotImplemented is returned when a feature or method has not been implemented yet.
	ErrNotImplemented = errors.New("not implemented")

	// ErrWrongType is returned when a type assertion or type conversion fails due to an unexpected type.
	ErrWrongType = errors.New("wrong type")

	// ErrHashCollision is returned when two distinct keys produce the same hash value.
	// This error indicates that the hash function is not suitable for the given key space,
	// or that the key distribution is causing unexpected collisions. When this error occurs,
	// consider using a different hash function or implementing a collision resolution strategy.
	ErrHashCollision = errors.New("hashing collision")

	// ErrValidation is returned when a value fails validation checks.
	// This error is used as a sentinel error that wraps the underlying validation failure,
	// allowing callers to distinguish validation errors from other error types using errors.Is().
	// The validate package automatically wraps validation failures with this error.
	// When this error occurs, the underlying wrapped error will contain specific details
	// about what validation failed.
	ErrValidation = errors.New("validation error")

	// ErrPanicRecovery is returned when a panic has been recovered during error collection.
	// This error wraps the recovered panic value along with a stack trace to help debug
	// the source of the panic. Used by the Collect function to safely handle panics.
	ErrPanicRecovery = errors.New("recovered from panic")
)

// Collection is a thread-unsafe utility for accumulating multiple errors.
// It provides methods to add errors, check for errors, and retrieve them as a single combined error.
// Use this when you need to collect errors from multiple operations and return them together.
type Collection struct {
	errors []error
}

// Add appends an error to the collection. Nil errors are automatically ignored.
func (c *Collection) Add(err error) {
	if err != nil {
		c.errors = append(c.errors, err)
	}
}

// Clear removes all errors from the collection, resetting it to an empty state.
func (c *Collection) Clear() {
	c.errors = nil
}

// HasError returns true if the collection contains at least one error.
func (c *Collection) HasError() bool {
	return len(c.errors) > 0
}

// GetError returns the collected errors as a single error.
// Returns nil if the collection is empty, the single error if there's only one,
// or a joined error (using errors.Join) if there are multiple errors.
func (c *Collection) GetError() error {
	switch len(c.errors) {
	case 0:
		return nil
	case 1:
		return c.errors[0]
	default:
		return errors.Join(c.errors...)
	}
}

// Collect provides a safe way to accumulate errors from multiple operations.
// It creates a Collection, executes the provided collector function, and returns
// any accumulated errors. If the collector function panics, the panic is recovered
// and added to the error collection as an ErrPanicRecovery with stack trace.
// Returns nil if no errors were collected, a single error if only one was collected,
// or a joined error if multiple errors were collected.
func Collect(collector func(errs *Collection)) error {
	c := new(Collection)

	collect(c, collector)

	if c.HasError() {
		return c.GetError()
	}

	return nil
}

// getPanicRecoveryError converts a recovered panic value into a properly formatted error.
// If the panic value is already an error, it wraps it with ErrPanicRecovery.
// Otherwise, it formats the value and wraps it with ErrPanicRecovery.
// The stack trace is included in the error message if provided.
func getPanicRecoveryError(err any, stack []byte) error {
	if err == nil {
		return nil
	}

	errErr, ok := err.(error)
	if ok {
		if stack != nil {
			return fmt.Errorf("%w: %w\nstack trace:\n%s", ErrPanicRecovery, errErr, string(stack))
		}

		return fmt.Errorf("%w: %w", ErrPanicRecovery, errErr)
	} else {
		if stack != nil {
			return fmt.Errorf("%w: %v\nstack trace:\n%s", ErrPanicRecovery, err, string(stack))
		}

		return fmt.Errorf("%w: %v", ErrPanicRecovery, err)
	}
}

// collect is the internal implementation that executes the collector function with panic recovery.
// If the collector function panics, the panic is recovered, converted to an error using
// getPanicRecoveryError, and added to the Collection. This ensures that panics during
// error collection don't propagate to callers.
func collect(c *Collection, collector func(errs *Collection)) {
	if collector == nil {
		return
	}

	defer func() {
		if e := recover(); e != nil {
			err := getPanicRecoveryError(e, debug.Stack())

			c.Add(err)
		}
	}()

	collector(c)
}
