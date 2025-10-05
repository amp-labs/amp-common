// Package pointer provides utilities for working with pointers in Go.
// It includes generic functions for creating pointers and safely dereferencing them.
package pointer

// To returns a pointer to the given value.
// This is useful when you need to take the address of a literal or expression result.
//
// Example:
//
//	s := pointer.To("hello")  // *string
//	i := pointer.To(42)       // *int
func To[T any](v T) *T {
	return &v
}

// Value safely dereferences a pointer and returns the value and a boolean indicating success.
// If the pointer is nil, it returns the zero value of type T and false.
// If the pointer is non-nil, it returns the dereferenced value and true.
//
// Example:
//
//	var p *string
//	val, ok := pointer.Value(p)  // "", false
//
//	s := pointer.To("hello")
//	val, ok := pointer.Value(s)  // "hello", true
func Value[T any](p *T) (T, bool) {
	if p == nil {
		var zero T

		return zero, false
	}

	return *p, true
}
