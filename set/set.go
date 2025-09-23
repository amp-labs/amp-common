package set

import (
	"errors"
	"sort"

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
	AddAll(elements ...T) error
	Add(element T) error
	Remove(element T) error
	Clear()
	Contains(element T) (bool, error)
	Size() int
	Entries() []T
	Union(other Set[T]) (Set[T], error)
	Intersection(other Set[T]) (Set[T], error)
}

type setImpl[T Collectable[T]] struct {
	hash     hashing.HashFunc
	elements map[string]T
}

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

type StringSet struct {
	hash hashing.HashFunc
	set  Set[hashing.HashableString]
}

func NewStringSet(hash hashing.HashFunc) *StringSet {
	return &StringSet{
		hash: hash,
		set:  NewSet[hashing.HashableString](hash),
	}
}

func (s *StringSet) AddAll(element ...string) error {
	for _, elem := range element {
		if err := s.Add(elem); err != nil {
			return err
		}
	}

	return nil
}

func (s *StringSet) Add(element string) error {
	return s.set.Add(hashing.HashableString(element))
}

func (s *StringSet) Clear() {
	s.set.Clear()
}

func (s *StringSet) Remove(element string) error {
	return s.set.Remove(hashing.HashableString(element))
}

func (s *StringSet) Contains(element string) (bool, error) {
	return s.set.Contains(hashing.HashableString(element))
}

func (s *StringSet) Size() int {
	return s.set.Size()
}

func (s *StringSet) Entries() []string {
	items := make([]string, 0, s.Size())

	for _, item := range s.set.Entries() {
		items = append(items, string(item))
	}

	return items
}

func (s *StringSet) SortedEntries() []string {
	items := s.Entries()

	sort.Strings(items)

	return items
}

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
