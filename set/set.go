package set

import (
	"iter"
	"sort"

	"facette.io/natsort"
	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/compare"
	errors2 "github.com/amp-labs/amp-common/errors"
	"github.com/amp-labs/amp-common/hashing"
)

// A Set is a collection of unique elements. Uniqueness is
// determined by the HashFunc provided when the Set is created,
// as well as how the object has implemented the Hashable and
// Comparable interfaces. If a collision is detected, an error
// is returned.
//
//nolint:interfacebloat // Set requires these methods for complete functionality
type Set[T collectable.Collectable[T]] interface {
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

	// Seq returns all elements in the set as an iter.Seq. The order is not guaranteed.
	Seq() iter.Seq[T]

	// Union returns a new set containing all elements from both sets. Returns an error
	// if any element causes a hash collision or if hashing fails.
	Union(other Set[T]) (Set[T], error)

	// Intersection returns a new set containing only elements present in both sets.
	// Returns an error if any element causes a hash collision or if hashing fails.
	Intersection(other Set[T]) (Set[T], error)

	// HashFunction returns the hash function used by this set.
	HashFunction() hashing.HashFunc
}

type setImpl[T collectable.Collectable[T]] struct {
	hash     hashing.HashFunc
	elements map[string]T
}

// NewSet creates a new Set with the provided hash function.
// The hash function is used to determine uniqueness of elements.
func NewSet[T collectable.Collectable[T]](hash hashing.HashFunc) Set[T] {
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
			return errors2.ErrHashCollision
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
			return true, errors2.ErrHashCollision
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

func (s *setImpl[T]) Seq() iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, item := range s.elements {
			if !yield(item) {
				return
			}
		}
	}
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

// HashFunction returns the hash function used by this set.
func (s *setImpl[T]) HashFunction() hashing.HashFunc {
	return s.hash
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

// Seq returns all string elements in the set as an iter.Seq. The order is not guaranteed.
func (s *StringSet) Seq() iter.Seq[string] {
	return func(yield func(string) bool) {
		for item := range s.set.Seq() {
			if !yield(string(item)) {
				return
			}
		}
	}
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

// OrderedSet is a Set that maintains insertion order of elements.
// Unlike the regular Set where Entries() returns elements in arbitrary order,
// OrderedSet.Entries() returns elements in the order they were added.
//
//nolint:interfacebloat // OrderedSet requires these methods for complete functionality
type OrderedSet[T collectable.Collectable[T]] interface {
	// AddAll adds multiple elements to the set in order. Returns an error if any element
	// causes a hash collision or if hashing fails. If an element already exists, it is not
	// added again and its position in the order is not changed.
	AddAll(elements ...T) error

	// Add adds a single element to the set. Returns an error if the element
	// causes a hash collision or if hashing fails. If the element already exists
	// in the set, no error is returned and its position in the order is not changed.
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

	// Entries returns all elements in the set as a slice in insertion order.
	// The returned slice is a copy and modifications to it will not affect the set.
	Entries() []T

	// Seq returns all elements in the set as an iter.Seq in insertion order.
	Seq() iter.Seq2[int, T]

	// Union returns a new ordered set containing all elements from both sets.
	// Elements from the current set appear first in insertion order, followed by
	// elements from the other set that are not already present. Returns an error
	// if any element causes a hash collision or if hashing fails.
	Union(other OrderedSet[T]) (OrderedSet[T], error)

	// Intersection returns a new ordered set containing only elements present in both sets.
	// The order is preserved from the current set. Returns an error if any element causes
	// a hash collision or if hashing fails.
	Intersection(other OrderedSet[T]) (OrderedSet[T], error)

	// HashFunction returns the hash function used by this ordered set.
	HashFunction() hashing.HashFunc
}

type orderedSetImpl[T collectable.Collectable[T]] struct {
	hash  hashing.HashFunc
	set   Set[T]
	order []T
}

// NewOrderedSet creates a new OrderedSet with the provided hash function.
// The hash function is used to determine uniqueness of elements.
// Elements are returned in insertion order by Entries().
func NewOrderedSet[T collectable.Collectable[T]](hash hashing.HashFunc) OrderedSet[T] {
	return &orderedSetImpl[T]{
		hash:  hash,
		set:   NewSet[T](hash),
		order: make([]T, 0),
	}
}

func (s *orderedSetImpl[T]) AddAll(elements ...T) error {
	for _, elem := range elements {
		if err := s.Add(elem); err != nil {
			return err
		}
	}

	return nil
}

func (s *orderedSetImpl[T]) Add(element T) error {
	// Check if element already exists
	contains, err := s.set.Contains(element)
	if err != nil {
		return err
	}

	// If already exists, don't add again (preserve original position)
	if contains {
		return nil
	}

	// Add to underlying set
	if err := s.set.Add(element); err != nil {
		return err
	}

	// Add to order slice
	s.order = append(s.order, element)

	return nil
}

func (s *orderedSetImpl[T]) Remove(element T) error {
	// Check if element exists in set
	contains, err := s.set.Contains(element)
	if err != nil {
		return err
	}

	if !contains {
		return nil
	}

	// Remove from underlying set
	if err := s.set.Remove(element); err != nil {
		return err
	}

	// Remove from order slice by filtering
	filtered := make([]T, 0, len(s.order)-1)

	for _, item := range s.order {
		if !compare.Equals(item, element) {
			filtered = append(filtered, item)
		}
	}

	s.order = filtered

	return nil
}

func (s *orderedSetImpl[T]) Clear() {
	s.set.Clear()
	s.order = make([]T, 0)
}

func (s *orderedSetImpl[T]) Contains(element T) (bool, error) {
	return s.set.Contains(element)
}

func (s *orderedSetImpl[T]) Size() int {
	return s.set.Size()
}

func (s *orderedSetImpl[T]) Entries() []T {
	// Return a copy of the order slice to prevent external modifications
	result := make([]T, len(s.order))
	copy(result, s.order)

	return result
}

func (s *orderedSetImpl[T]) Seq() iter.Seq2[int, T] {
	return func(yield func(int, T) bool) {
		for i, item := range s.order {
			if !yield(i, item) {
				return
			}
		}
	}
}

func (s *orderedSetImpl[T]) Union(other OrderedSet[T]) (OrderedSet[T], error) {
	ns := NewOrderedSet[T](s.hash)

	// Add all elements from current set (preserves order)
	myItems := s.Entries()
	if err := ns.AddAll(myItems...); err != nil {
		return nil, err
	}

	// Add all elements from other set (preserves order, skips duplicates)
	otherItems := other.Entries()
	if err := ns.AddAll(otherItems...); err != nil {
		return nil, err
	}

	return ns, nil
}

func (s *orderedSetImpl[T]) Intersection(other OrderedSet[T]) (OrderedSet[T], error) {
	ns := NewOrderedSet[T](s.hash)

	// Iterate in order, only add elements that exist in both sets
	for _, item := range s.order {
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

// HashFunction returns the hash function used by this ordered set.
func (s *orderedSetImpl[T]) HashFunction() hashing.HashFunc {
	return s.hash
}

// StringOrderedSet is a specialized OrderedSet implementation for string elements
// that maintains insertion order. It provides additional methods for sorted access
// while preserving the original insertion order in the main Entries() method.
type StringOrderedSet struct {
	hash hashing.HashFunc
	set  OrderedSet[hashing.HashableString]
}

// NewStringOrderedSet creates a new StringOrderedSet with the provided hash function.
// Elements are maintained in insertion order and can be retrieved in that order via Entries().
func NewStringOrderedSet(hash hashing.HashFunc) *StringOrderedSet {
	return &StringOrderedSet{
		hash: hash,
		set:  NewOrderedSet[hashing.HashableString](hash),
	}
}

// AddAll adds multiple string elements to the set in order. If an element already exists,
// it is not added again and its position in the order is not changed.
func (s *StringOrderedSet) AddAll(element ...string) error {
	for _, elem := range element {
		if err := s.Add(elem); err != nil {
			return err
		}
	}

	return nil
}

// Add adds a single string element to the set. If the element already exists,
// no error is returned and its position in the order is not changed.
func (s *StringOrderedSet) Add(element string) error {
	return s.set.Add(hashing.HashableString(element))
}

// Clear removes all elements from the set.
func (s *StringOrderedSet) Clear() {
	s.set.Clear()
}

// Remove removes a string element from the set.
func (s *StringOrderedSet) Remove(element string) error {
	return s.set.Remove(hashing.HashableString(element))
}

// Contains checks if a string element exists in the set.
func (s *StringOrderedSet) Contains(element string) (bool, error) {
	return s.set.Contains(hashing.HashableString(element))
}

// Size returns the number of elements in the set.
func (s *StringOrderedSet) Size() int {
	return s.set.Size()
}

// Entries returns all string elements in the set in insertion order.
// The returned slice is a copy and modifications to it will not affect the set.
func (s *StringOrderedSet) Entries() []string {
	items := make([]string, 0, s.Size())

	for _, item := range s.set.Entries() {
		items = append(items, string(item))
	}

	return items
}

// Seq returns all string elements in the set as an iter.Seq2 in insertion order.
func (s *StringOrderedSet) Seq() iter.Seq2[int, string] {
	return func(yield func(int, string) bool) {
		for i, item := range s.set.Seq() {
			if !yield(i, string(item)) {
				return
			}
		}
	}
}

// SortedEntries returns all string elements in the set sorted alphabetically.
// This does not affect the insertion order maintained by the set.
func (s *StringOrderedSet) SortedEntries() []string {
	items := s.Entries()

	sort.Strings(items)

	return items
}

// NaturalSortedEntries returns all string elements in the set sorted using natural sort order.
// Natural sort treats numbers within strings numerically (e.g., "file2" comes before "file10").
// This does not affect the insertion order maintained by the set.
func (s *StringOrderedSet) NaturalSortedEntries() []string {
	items := s.Entries()

	natsort.Sort(items)

	return items
}

// Union returns a new StringOrderedSet containing all elements from both sets.
// Elements from the current set appear first in insertion order, followed by
// elements from the other set that are not already present, also in their insertion order.
func (s *StringOrderedSet) Union(other *StringOrderedSet) (*StringOrderedSet, error) {
	ns := NewStringOrderedSet(s.hash)

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

// Intersection returns a new StringOrderedSet containing only elements present in both sets.
// The order is preserved from the current set.
func (s *StringOrderedSet) Intersection(other *StringOrderedSet) (*StringOrderedSet, error) {
	ns := NewOrderedSet[hashing.HashableString](s.hash)

	for _, item := range s.Entries() {
		if contains, err := other.Contains(item); err != nil {
			return nil, err
		} else if contains {
			if err := ns.Add(hashing.HashableString(item)); err != nil {
				return nil, err
			}
		}
	}

	return &StringOrderedSet{
		hash: s.hash,
		set:  ns,
	}, nil
}

// HashFunction returns the hash function used by this set.
func (s *StringOrderedSet) HashFunction() hashing.HashFunc {
	return s.hash
}
