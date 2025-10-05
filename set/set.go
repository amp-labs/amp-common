package set

import (
	"errors"
	"sort"

	"facette.io/natsort"
	"github.com/amp-labs/amp-common/compare"
	"github.com/amp-labs/amp-common/hashing"
)

// ErrHashCollision is returned when a hashing collision is detected.
// Specifically this refers to two different (non-equal) objects
// that have the same hashing value.
var ErrHashCollision = errors.New("hashing collision")

// Collectable is an interface that combines the Hashable and
// Comparable interfaces. This is useful for objects that need
// to be stored in a Set, where uniqueness is determined by
// the hashing value, and collisions are resolved by comparing
// the objects.
type Collectable[T any] interface {
	hashing.Hashable
	compare.Comparable[T]
}

// A Set is a collection of unique elements. Uniqueness is
// determined by the HashFunc provided when the Set is created,
// as well as how the object has implemented the Hashable and
// Comparable interfaces. If a collision is detected, an error
// is returned.
type Set[T Collectable[T]] interface {
	// AddAll adds multiple elements to the set. Returns an error if any element
	// causes a hash collision or if hashing fails.
	AddAll(elements ...T) error

	// Add adds a single element to the set. Returns an error if the element
	// causes a hash collision or if hashing fails. If the element already exists
	// in the set, no error is returned.
	Add(element T) error

	// Remove removes an element from the set. Returns an error if hashing fails.
	// If the element is not in the set, no error is returned.
	Remove(element T) error

	// Clear removes all elements from the set.
	Clear()

	// Contains checks if an element exists in the set. Returns true if the element
	// exists, false otherwise. Returns an error if hashing fails or a collision is detected.
	Contains(element T) (bool, error)

	// Size returns the number of elements in the set.
	Size() int

	// Entries returns all elements in the set as a slice. The order is not guaranteed.
	Entries() []T

	// Union returns a new set containing all elements from both sets. Returns an error
	// if any element causes a hash collision or if hashing fails.
	Union(other Set[T]) (Set[T], error)

	// Intersection returns a new set containing only elements present in both sets.
	// Returns an error if any element causes a hash collision or if hashing fails.
	Intersection(other Set[T]) (Set[T], error)
}

type setImpl[T Collectable[T]] struct {
	hash     hashing.HashFunc
	elements map[string]T
}

// NewSet creates a new Set with the provided hash function.
// The hash function is used to determine uniqueness of elements.
func NewSet[T Collectable[T]](hash hashing.HashFunc) Set[T] {
	return &setImpl[T]{
		hash:     hash,
		elements: make(map[string]T),
	}
}

func (s *setImpl[T]) AddAll(element ...T) error {
	for _, elem := range element {
		if err := s.Add(elem); err != nil {
			return err
		}
	}

	return nil
}

func (s *setImpl[T]) Add(element T) error {
	hashVal, err := s.hash(element)
	if err != nil {
		return err
	}

	prev, ok := s.elements[hashVal]
	if ok {
		if compare.Equals(prev, element) {
			return nil
		} else {
			return ErrHashCollision
		}
	}

	s.elements[hashVal] = element

	return nil
}

func (s *setImpl[T]) Clear() {
	s.elements = make(map[string]T)
}

func (s *setImpl[T]) Remove(element T) error {
	hashVal, err := s.hash(element)
	if err != nil {
		return err
	}

	prev, ok := s.elements[hashVal]
	if ok {
		if compare.Equals(prev, element) {
			delete(s.elements, hashVal)

			return nil
		}
	}

	return nil
}

func (s *setImpl[T]) Contains(element T) (bool, error) {
	hashVal, err := s.hash(element)
	if err != nil {
		return false, err
	}

	prev, ok := s.elements[hashVal]
	if ok {
		if compare.Equals(prev, element) {
			return true, nil
		} else {
			return true, ErrHashCollision
		}
	}

	return false, nil
}

func (s *setImpl[T]) Size() int {
	return len(s.elements)
}

func (s *setImpl[T]) Entries() []T {
	items := make([]T, 0, len(s.elements))
	for _, item := range s.elements {
		items = append(items, item)
	}

	return items
}

func (s *setImpl[T]) Union(other Set[T]) (Set[T], error) {
	ns := NewSet[T](s.hash)

	myItems := s.Entries()
	otherItems := other.Entries()

	if err := ns.AddAll(myItems...); err != nil {
		return nil, err
	}

	if err := ns.AddAll(otherItems...); err != nil {
		return nil, err
	}

	return ns, nil
}

func (s *setImpl[T]) Intersection(other Set[T]) (Set[T], error) {
	ns := NewSet[T](s.hash)

	for _, item := range s.Entries() {
		if contains, err := other.Contains(item); err != nil {
			return nil, err
		} else if contains {
			if err := ns.Add(item); err != nil {
				return nil, err
			}
		}
	}

	return ns, nil
}

// StringSet is a specialized Set implementation for string elements.
// It provides additional methods for sorting entries.
type StringSet struct {
	hash hashing.HashFunc
	set  Set[hashing.HashableString]
}

// NewStringSet creates a new StringSet with the provided hash function.
func NewStringSet(hash hashing.HashFunc) *StringSet {
	return &StringSet{
		hash: hash,
		set:  NewSet[hashing.HashableString](hash),
	}
}

// AddAll adds multiple string elements to the set.
func (s *StringSet) AddAll(element ...string) error {
	for _, elem := range element {
		if err := s.Add(elem); err != nil {
			return err
		}
	}

	return nil
}

// Add adds a single string element to the set.
func (s *StringSet) Add(element string) error {
	return s.set.Add(hashing.HashableString(element))
}

// Clear removes all elements from the set.
func (s *StringSet) Clear() {
	s.set.Clear()
}

// Remove removes a string element from the set.
func (s *StringSet) Remove(element string) error {
	return s.set.Remove(hashing.HashableString(element))
}

// Contains checks if a string element exists in the set.
func (s *StringSet) Contains(element string) (bool, error) {
	return s.set.Contains(hashing.HashableString(element))
}

// Size returns the number of elements in the set.
func (s *StringSet) Size() int {
	return s.set.Size()
}

// Entries returns all string elements in the set. The order is not guaranteed.
func (s *StringSet) Entries() []string {
	items := make([]string, 0, s.Size())

	for _, item := range s.set.Entries() {
		items = append(items, string(item))
	}

	return items
}

// SortedEntries returns all string elements in the set sorted alphabetically.
func (s *StringSet) SortedEntries() []string {
	items := s.Entries()

	sort.Strings(items)

	return items
}

// NaturalSortedEntries returns all string elements in the set sorted using natural sort order.
// Natural sort treats numbers within strings numerically (e.g., "file2" comes before "file10").
func (s *StringSet) NaturalSortedEntries() []string {
	items := s.Entries()

	natsort.Sort(items)

	return items
}

// Union returns a new StringSet containing all elements from both sets.
func (s *StringSet) Union(other *StringSet) (*StringSet, error) {
	ns := NewStringSet(s.hash)

	myItems := s.Entries()
	otherItems := other.Entries()

	if err := ns.AddAll(myItems...); err != nil {
		return nil, err
	}

	if err := ns.AddAll(otherItems...); err != nil {
		return nil, err
	}

	return ns, nil
}

// Intersection returns a new StringSet containing only elements present in both sets.
func (s *StringSet) Intersection(other *StringSet) (*StringSet, error) {
	ns := NewSet[hashing.HashableString](s.hash)

	for _, item := range s.Entries() {
		if contains, err := other.Contains(item); err != nil {
			return nil, err
		} else if contains {
			if err := ns.Add(hashing.HashableString(item)); err != nil {
				return nil, err
			}
		}
	}

	return &StringSet{
		hash: s.hash,
		set:  ns,
	}, nil
}
