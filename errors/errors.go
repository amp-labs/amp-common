package errors

import "errors"

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrWrongType      = errors.New("wrong type")
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
