package maps

import (
	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/set"
)

// FromSet converts a Set to a Map by applying a value function to each key.
// The getValue function is called for each key in the set to produce the corresponding value.
// The resulting map uses the same hash function as the input set and is pre-allocated to the set's size.
//
// Returns nil if the input set is nil.
//
// The iteration order is non-deterministic as it depends on the set's internal iteration order.
//
// Example:
//
//	// Create a map from a set of strings to their lengths
//	stringSet := set.NewSet[String](hashFunc)
//	stringSet.Add("hello")
//	stringSet.Add("world")
//	m := FromSet(stringSet, func(s String) int { return len(s.Value) })
//	// m contains: {"hello": 5, "world": 5}
func FromSet[K collectable.Collectable[K], V any](s set.Set[K], getValue func(key K) V) Map[K, V] {
	if s == nil {
		return nil
	}

	m := NewHashMapWithSize[K, V](s.HashFunction(), s.Size())

	for k := range s.Seq() {
		value := getValue(k)

		_ = m.Add(k, value)
	}

	return m
}

// FromOrderedSet converts an OrderedSet to an OrderedMap by applying a value function to each key.
// The getValue function is called for each key in the ordered set to produce the corresponding value.
// The resulting ordered map uses the same hash function as the input set and preserves the insertion order.
//
// Returns nil if the input set is nil.
//
// The iteration follows the insertion order of the set, and the resulting map maintains this same order.
//
// Example:
//
//	// Create an ordered map from an ordered set of strings to their lengths
//	stringSet := set.NewOrderedSet[String](hashFunc)
//	stringSet.Add("hello")
//	stringSet.Add("world")
//	m := FromOrderedSet(stringSet, func(s String) int { return len(s.Value) })
//	// m contains: {"hello": 5, "world": 5} in insertion order
func FromOrderedSet[K collectable.Collectable[K], V any](s set.OrderedSet[K], getValue func(key K) V) OrderedMap[K, V] {
	if s == nil {
		return nil
	}

	m := NewOrderedHashMap[K, V](s.HashFunction())

	for _, k := range s.Seq() {
		value := getValue(k)

		_ = m.Add(k, value)
	}

	return m
}
