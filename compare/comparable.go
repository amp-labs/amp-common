// Package compare provides utilities for comparing values.
package compare

// Comparable is a generic interface for types that can compare themselves for equality.
// Types implementing this interface must provide their own Equals method that determines
// whether two values are equal according to the type's semantics.
type Comparable[T any] interface {
	Equals(other T) bool
}

// Equals compares two values using the Comparable interface.
// It delegates to the Equals method of the first argument.
func Equals[T any](a Comparable[T], b T) bool {
	return a.Equals(b)
}
