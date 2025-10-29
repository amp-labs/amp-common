package maps

import "hash"

// collectableWhatever is a test/example type that implements the collectable.Collectable interface
// for any type T. It's used for compile-time interface verification with the var _ idiom.
// This type intentionally panics on any method call as it's never meant to be instantiated.
type collectableWhatever[T any] struct{}

// UpdateHash panics as this type is never meant to be instantiated or used at runtime.
func (c collectableWhatever[T]) UpdateHash(h hash.Hash) error {
	panic("should never happen")
}

// Equals panics as this type is never meant to be instantiated or used at runtime.
func (c collectableWhatever[T]) Equals(other collectableWhatever[T]) bool {
	panic("should never happen")
}
