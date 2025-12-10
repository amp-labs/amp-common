// Package errors provides error utilities with collection support for managing multiple errors.
package errors //nolint:revive // This is a fine package name, nuts to you

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrWrongType      = errors.New("wrong type")

	// ErrHashCollision is returned when two distinct keys produce the same hash value.
	// This error indicates that the hash function is not suitable for the given key space,
	// or that the key distribution is causing unexpected collisions. When this error occurs,
	// consider using a different hash function or implementing a collision resolution strategy.
	ErrHashCollision = errors.New("hashing collision")
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
