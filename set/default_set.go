package set

import (
	"errors"
	"iter"

	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/hashing"
)

// ErrNoDefaultValue is returned by the default value function when it cannot or chooses not to
// provide a default value for a given element. When this error is returned, the defaultSet will
// not add the element to the set and will behave as if the element simply doesn't exist.
var ErrNoDefaultValue = errors.New("no default value for this element")

// NewDefaultSet creates a Set that automatically generates default values for missing elements.
// When Contains is called with an element that doesn't exist, the getDefaultValue function
// is invoked to generate a value, which is then added to the set and returned.
//
// The getDefaultValue function should return ErrNoDefaultValue when it cannot or chooses not
// to provide a default value. In that case, the set behaves as if the element doesn't exist.
//
// If storageSet is already a defaultSet, this function clones it and replaces the default
// value function with the new one provided.
//
// Parameters:
//   - storageSet: The underlying Set implementation to use for storage
//   - getDefaultValue: Function that generates default values for missing elements
//
// Example:
//
//	// Create a set that stores lowercase versions of strings
//	s := set.NewDefaultSet(
//	    set.NewSet[MyType](hashFunc),
//	    func(elem MyType) (MyType, error) {
//	        return elem.Normalize(), nil
//	    },
//	)
//	contains, _ := s.Contains(elem) // Returns true and adds normalized version to set
//
//	// Create a set that refuses to provide defaults
//	s2 := set.NewDefaultSet(
//	    set.NewSet[MyType](hashFunc),
//	    func(elem MyType) (MyType, error) {
//	        return zero.Value[MyType](), set.ErrNoDefaultValue
//	    },
//	)
//	contains, _ := s2.Contains(elem) // Returns false without adding
func NewDefaultSet[T collectable.Collectable[T]](
	storageSet Set[T],
	getDefaultValue func(T) (T, error),
) Set[T] {
	return &defaultSet[T]{
		s: storageSet,
		f: getDefaultValue,
	}
}

type defaultSet[T collectable.Collectable[T]] struct {
	s Set[T]             // Underlying set for storage
	f func(T) (T, error) // Function to generate default values for missing elements
}

var _ Set[collectableWhatever[any]] = (*defaultSet[collectableWhatever[any]])(nil)

// AddAll adds multiple elements to the set.
// This operation bypasses the default value function and directly adds the provided elements.
// Returns an error if any element causes a hash collision or if hashing fails.
func (d *defaultSet[T]) AddAll(elements ...T) error {
	return d.s.AddAll(elements...)
}

// Add inserts an element into the set.
// This operation bypasses the default value function and directly adds the provided element.
// Returns an error if the element causes a hash collision or if hashing fails.
func (d *defaultSet[T]) Add(element T) error {
	return d.s.Add(element)
}

// Remove deletes the element from the set.
// If the element doesn't exist, this is a no-op and returns nil.
// Returns an error if hashing fails or a collision is detected.
func (d *defaultSet[T]) Remove(element T) error {
	return d.s.Remove(element)
}

// Clear removes all elements from the set, leaving it empty.
func (d *defaultSet[T]) Clear() {
	d.s.Clear()
}

// Contains checks if the given element exists in the set. If the element doesn't exist, attempts to
// generate and add a default value using the default value function:
//   - If the function succeeds, adds the default value and returns true
//   - If the function returns ErrNoDefaultValue, returns false
//   - If the function returns another error, returns that error
//
// Returns an error if hashing fails or a hash collision occurs during lookup or insertion.
func (d *defaultSet[T]) Contains(element T) (bool, error) {
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
func (d *defaultSet[T]) addDefaultForElement(element T) (bool, error) {
	value, err := d.f(element)
	if err != nil {
		if errors.Is(err, ErrNoDefaultValue) {
			return false, nil
		}

		return false, err
	}

	if err := d.s.Add(value); err != nil {
		return false, err
	}

	return true, nil
}

// Size returns the number of elements currently stored in the set.
func (d *defaultSet[T]) Size() int {
	return d.s.Size()
}

// Entries returns all elements in the set as a slice. The order is not guaranteed.
func (d *defaultSet[T]) Entries() []T {
	return d.s.Entries()
}

// Seq returns an iterator for ranging over all elements in the set.
// This method is compatible with Go 1.23+ range-over-func syntax:
//
//	for elem := range set.Seq() {
//	    // process element
//	}
//
// The iteration order is not guaranteed.
func (d *defaultSet[T]) Seq() iter.Seq[T] {
	return d.s.Seq()
}

// Union creates a new defaultSet containing all elements from both this set and other.
// The returned set uses the same default value function as this set.
// Returns an error if any element causes a hash collision or if hashing fails.
func (d *defaultSet[T]) Union(other Set[T]) (Set[T], error) {
	tmp, err := d.s.Union(other)
	if err != nil {
		return nil, err
	}

	return &defaultSet[T]{
		s: tmp,
		f: d.f,
	}, nil
}

// Intersection creates a new defaultSet containing only elements that exist in both sets.
// The returned set uses the same default value function as this set.
// Returns an error if any element causes a hash collision or if hashing fails.
func (d *defaultSet[T]) Intersection(other Set[T]) (Set[T], error) {
	tmp, err := d.s.Intersection(other)
	if err != nil {
		return nil, err
	}

	return &defaultSet[T]{
		s: tmp,
		f: d.f,
	}, nil
}

// HashFunction returns the hash function used by the underlying set.
// This allows callers to inspect the hash function or create compatible sets.
func (d *defaultSet[T]) HashFunction() hashing.HashFunc {
	return d.s.HashFunction()
}

// Clone creates a shallow copy of the set, duplicating its structure and entries.
// The elements themselves are not deep-copied; they are referenced as-is.
// The returned set uses the same default value function as this set.
// Returns a new Set instance with the same entries.
func (d *defaultSet[T]) Clone() Set[T] {
	return &defaultSet[T]{
		s: d.s.Clone(),
		f: d.f,
	}
}

// Filter returns a new defaultSet containing only elements that satisfy the predicate.
// The predicate function is called for each element; if it returns true, the element is included.
// The returned set uses the same default value function as this set.
func (d *defaultSet[T]) Filter(predicate func(T) bool) Set[T] {
	out := d.Clone()
	out.Clear()

	for value := range d.Seq() {
		if predicate(value) {
			_ = out.Add(value)
		}
	}

	return out
}

// FilterNot returns a new defaultSet containing only elements that do not satisfy the predicate.
// The predicate function is called for each element; if it returns false, the element is included.
// The returned set uses the same default value function as this set.
func (d *defaultSet[T]) FilterNot(predicate func(T) bool) Set[T] {
	out := d.Clone()
	out.Clear()

	for value := range d.Seq() {
		if !predicate(value) {
			_ = out.Add(value)
		}
	}

	return out
}
