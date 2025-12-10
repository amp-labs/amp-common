package set

import (
	"errors"
	"iter"

	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/hashing"
)

// NewDefaultOrderedSet creates an OrderedSet that automatically generates default values for missing elements.
// When Contains is called with an element that doesn't exist, the getDefaultValue function
// is invoked to generate a value, which is then added to the set (at the end of the insertion order)
// and returned.
//
// The getDefaultValue function should return ErrNoDefaultValue when it cannot or chooses not
// to provide a default value. In that case, the set behaves as if the element doesn't exist.
//
// If storageSet is already a defaultOrderedSet, this function clones it and replaces the default
// value function with the new one provided.
//
// Unlike the standard Set, OrderedSet preserves the insertion order of elements. When a default value
// is generated and added, it's appended to the end of the current insertion order.
//
// Parameters:
//   - storageSet: The underlying OrderedSet implementation to use for storage
//   - getDefaultValue: Function that generates default values for missing elements
//
// Example:
//
//	// Create an ordered set that stores lowercase versions of strings
//	s := set.NewDefaultOrderedSet(
//	    set.NewOrderedSet[MyType](hashFunc),
//	    func(elem MyType) (MyType, error) {
//	        return elem.Normalize(), nil
//	    },
//	)
//	contains, _ := s.Contains(elem) // Returns true and adds normalized version to end
//
//	// Create a set that refuses to provide defaults
//	s2 := set.NewDefaultOrderedSet(
//	    set.NewOrderedSet[MyType](hashFunc),
//	    func(elem MyType) (MyType, error) {
//	        return zero.Value[MyType](), set.ErrNoDefaultValue
//	    },
//	)
//	contains, _ := s2.Contains(elem) // Returns false without adding
func NewDefaultOrderedSet[T collectable.Collectable[T]](
	storageSet OrderedSet[T],
	getDefaultValue func(T) (T, error),
) OrderedSet[T] {
	return &defaultOrderedSet[T]{
		s: storageSet,
		f: getDefaultValue,
	}
}

type defaultOrderedSet[T collectable.Collectable[T]] struct {
	s OrderedSet[T]      // Underlying ordered set for storage
	f func(T) (T, error) // Function to generate default values for missing elements
}

var _ OrderedSet[collectableWhatever[any]] = (*defaultOrderedSet[collectableWhatever[any]])(nil)

// AddAll adds multiple elements to the set in order.
// This operation bypasses the default value function and directly adds the provided elements.
// If an element already exists, it is not added again and its position in the order is not changed.
// Returns an error if any element causes a hash collision or if hashing fails.
func (d *defaultOrderedSet[T]) AddAll(elements ...T) error {
	return d.s.AddAll(elements...)
}

// Add inserts an element into the set.
// This operation bypasses the default value function and directly adds the provided element.
// If the element already exists, no error is returned and its position in the order is not changed.
// If the element is new, it's appended to the end of the insertion order.
// Returns an error if the element causes a hash collision or if hashing fails.
func (d *defaultOrderedSet[T]) Add(element T) error {
	return d.s.Add(element)
}

// Remove deletes the element from the set.
// If the element doesn't exist, this is a no-op and returns nil.
// Returns an error if hashing fails or a collision is detected.
func (d *defaultOrderedSet[T]) Remove(element T) error {
	return d.s.Remove(element)
}

// Clear removes all elements from the set, leaving it empty.
func (d *defaultOrderedSet[T]) Clear() {
	d.s.Clear()
}

// Contains checks if the given element exists in the set. If the element doesn't exist, attempts to
// generate and add a default value using the default value function:
//   - If the function succeeds, adds the default value and returns true
//   - If the function returns ErrNoDefaultValue, returns false
//   - If the function returns another error, returns that error
//
// Returns an error if hashing fails or a hash collision occurs during lookup or insertion.
func (d *defaultOrderedSet[T]) Contains(element T) (bool, error) {
	contains, err := d.s.Contains(element)
	if err != nil {
		return false, err
	}

	if contains {
		return true, nil
	}

	added, err := d.addDefaultForElement(element)
	if err != nil {
		return false, err
	}

	return added, nil
}

// addDefaultForElement calls the default value function to generate a value for the given element,
// then adds it to the set if successful. Returns whether it was added and any error that occurred.
//   - If the function returns ErrNoDefaultValue, returns (false, nil)
//   - If the function returns another error, returns (false, error)
//   - If the function succeeds but Add fails, returns (false, error)
//   - If both succeed, returns (true, nil)
func (d *defaultOrderedSet[T]) addDefaultForElement(element T) (bool, error) {
	value, err := d.f(element)
	if err != nil {
		if errors.Is(err, ErrNoDefaultValue) {
			return false, nil
		}

		return false, err
	}

	if err := d.s.Add(value); err != nil { //nolint:noinlineerr // Inline error handling is clear here
		return false, err
	}

	return true, nil
}

// Size returns the number of elements currently stored in the set.
func (d *defaultOrderedSet[T]) Size() int {
	return d.s.Size()
}

// Entries returns all elements in the set as a slice in insertion order.
// The returned slice is a copy and modifications to it will not affect the set.
func (d *defaultOrderedSet[T]) Entries() []T {
	return d.s.Entries()
}

// Seq returns an iterator for ranging over all elements in insertion order.
// The iterator yields (index, element) tuples where index is the position in the
// insertion order (0-based). This guarantees deterministic iteration order that reflects
// the order in which elements were first added to the set.
//
// This method is compatible with Go 1.23+ range-over-func syntax:
//
//	for i, elem := range set.Seq() {
//	    // process index and element
//	}
func (d *defaultOrderedSet[T]) Seq() iter.Seq2[int, T] {
	return d.s.Seq()
}

// Union creates a new defaultOrderedSet containing all elements from both this set and other.
// Elements from this set are added first (preserving their order), followed by elements from other.
// The returned set uses the same default value function as this set.
// Returns an error if any element causes a hash collision or if hashing fails.
func (d *defaultOrderedSet[T]) Union(other OrderedSet[T]) (OrderedSet[T], error) {
	tmp, err := d.s.Union(other)
	if err != nil {
		return nil, err
	}

	return &defaultOrderedSet[T]{
		s: tmp,
		f: d.f,
	}, nil
}

// Intersection creates a new defaultOrderedSet containing only elements that exist in both sets.
// The insertion order is preserved from this set.
// The returned set uses the same default value function as this set.
// Returns an error if any element causes a hash collision or if hashing fails.
func (d *defaultOrderedSet[T]) Intersection(other OrderedSet[T]) (OrderedSet[T], error) {
	tmp, err := d.s.Intersection(other)
	if err != nil {
		return nil, err
	}

	return &defaultOrderedSet[T]{
		s: tmp,
		f: d.f,
	}, nil
}

// HashFunction returns the hash function used by the underlying ordered set.
// This allows callers to inspect the hash function or create compatible sets.
func (d *defaultOrderedSet[T]) HashFunction() hashing.HashFunc {
	return d.s.HashFunction()
}

// Clone creates a shallow copy of the ordered set, duplicating its structure and entries.
// The elements themselves are not deep-copied; they are referenced as-is.
// The returned set uses the same default value function as this set.
// Returns a new OrderedSet instance with the same entries in the same order.
func (d *defaultOrderedSet[T]) Clone() OrderedSet[T] {
	if d == nil {
		return nil
	}

	return &defaultOrderedSet[T]{
		s: d.s.Clone(),
		f: d.f,
	}
}

// Filter returns a new defaultOrderedSet containing only elements that satisfy the predicate.
// The predicate function is called for each element; if it returns true, the element is included.
// The insertion order is preserved in the resulting set.
// The returned set uses the same default value function as this set.
func (d *defaultOrderedSet[T]) Filter(predicate func(T) bool) OrderedSet[T] {
	out := d.Clone()
	out.Clear()

	for _, value := range d.Seq() {
		if predicate(value) {
			_ = out.Add(value)
		}
	}

	return out
}

// FilterNot returns a new defaultOrderedSet containing only elements that do not satisfy the predicate.
// The predicate function is called for each element; if it returns false, the element is included.
// The insertion order is preserved in the resulting set.
// The returned set uses the same default value function as this set.
func (d *defaultOrderedSet[T]) FilterNot(predicate func(T) bool) OrderedSet[T] {
	out := d.Clone()
	out.Clear()

	for _, value := range d.Seq() {
		if !predicate(value) {
			_ = out.Add(value)
		}
	}

	return out
}
