// Package using provides a resource management pattern similar to C#'s "using" statement
// or Java's try-with-resources. It ensures that resources are properly cleaned up after use,
// even when errors occur.
//
// Example usage:
//
//	err := using.OpenFile("data.txt").Use(func(f *os.File) error {
//	    _, err := f.WriteString("hello world")
//	    return err
//	})
//	// File is automatically closed, even if an error occurred
package using

import (
	"errors"

	errors2 "github.com/amp-labs/amp-common/errors"
)

var (
	// ErrResourceNil is returned when a nil resource is passed to Use.
	ErrResourceNil = errors.New("resource is nil")
	// ErrFuncNil is returned when a nil function is passed to Use.
	ErrFuncNil = errors.New("f is nil")
)

// NewResource creates a Resource from a function that returns a value, closer, and error.
// The closer will be automatically invoked when the resource is used via Use().
func NewResource[V any](f func() (V, Closer, error)) *Resource[V] {
	return &Resource[V]{
		create: f,
	}
}

// Closer is a function that cleans up a resource. It follows the same signature as io.Closer.Close().
type Closer func() error

// Resource represents a managed resource that can be used safely with automatic cleanup.
// It is a function that produces a value, a closer for that value, and potentially an error.
type Resource[V any] struct {
	create   func() (V, Closer, error)
	released bool
}

// Use executes the provided function with the resource value, ensuring the resource
// is properly closed afterward. If both the function and closer return errors, both
// are collected and returned as a combined error.
func (p *Resource[V]) Use(userFunc func(value V) error) (errOut error) {
	if p == nil {
		return ErrResourceNil
	}

	p.released = false

	return use(p, userFunc)
}

// Release marks the resource as released, preventing automatic cleanup.
// When called, the resource's closer will not be invoked when Use() completes.
// This is useful when you need to transfer ownership of the resource outside
// the Use() block and manage its lifecycle manually.
func (p *Resource[V]) Release() {
	p.released = true
}

func use[V any](resource *Resource[V], userFunc func(value V) error) (errOut error) {
	if resource == nil {
		return ErrResourceNil
	}

	if userFunc == nil {
		return ErrFuncNil
	}

	val, closer, err := resource.create()
	if err != nil {
		return err
	}

	errs := errors2.Collection{}

	defer func() {
		if !resource.released && closer != nil {
			e := closer()
			if e != nil {
				errs.Add(e)
			}
		}

		if errs.HasError() {
			errOut = errs.GetError()
		} else {
			errOut = nil
		}
	}()

	e := userFunc(val)
	if e != nil {
		errs.Add(e)
	}

	return nil
}
