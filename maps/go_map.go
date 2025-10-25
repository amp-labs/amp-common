package maps

import (
	"hash"

	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/hashing"
)

// Key is a generic wrapper type that adapts any comparable type to be used as a map key.
// It implements the collectable.Collectable interface, making comparable types compatible
// with the Map interface that requires hashable and comparable keys.
//
// This type bridges the gap between Go's built-in comparable constraint and the amp-common
// collectable.Collectable interface, allowing standard Go types (int, string, etc.) to be
// used with hash-based maps.
//
// Example:
//
//	// Wrapping a string
//	key := Key[string]{Key: "my-key"}
//
//	// Using with a hash map
//	m := NewHashMap[Key[string], int](hashing.Sha256)
//	m.Add(Key[string]{Key: "count"}, 42)
type Key[T comparable] struct {
	Key T
}

// UpdateHash writes the key's hash representation to the provided hash.Hash.
// It converts the comparable key to a collectable.Collectable and delegates hashing to it.
// This allows any comparable type to participate in the hashing process.
func (m Key[T]) UpdateHash(h hash.Hash) error {
	return collectable.FromComparable(m.Key).UpdateHash(h)
}

// Equals compares this Key with another Key for equality.
// Two Keys are equal if their wrapped values are equal according to Go's == operator.
func (m Key[T]) Equals(other Key[T]) bool {
	return m.Key == other.Key
}

// FromGoMap converts a standard Go map to an amp-common Map implementation.
// It creates a new HashMap and populates it with all key-value pairs from the input map.
//
// The hash parameter specifies the hash function to use for the map (e.g., hashing.Sha256).
// Returns nil if the input map is nil.
//
// Panics if adding a key-value pair fails due to a hash collision. This should be rare
// with a good hash function like SHA-256.
//
// Example:
//
//	goMap := map[string]int{"a": 1, "b": 2}
//	ampMap := FromGoMap(goMap, hashing.Sha256)
//	// ampMap can now use Map interface methods
func FromGoMap[K comparable, V any](m map[K]V, hash hashing.HashFunc) Map[Key[K], V] {
	if m == nil {
		return nil
	}

	out := NewHashMap[Key[K], V](hash)

	for k, v := range m {
		if err := out.Add(Key[K]{k}, v); err != nil {
			panic(err)
		}
	}

	return out
}

// ToGoMap converts an amp-common Map to a standard Go map.
// It extracts all key-value pairs from the Map and returns them in a native Go map[K]V.
//
// Returns nil if the input map is nil. The iteration order is non-deterministic for
// standard maps and follows insertion order for ordered maps.
//
// Example:
//
//	ampMap := NewHashMap[Key[string], int](hashing.Sha256)
//	ampMap.Add(Key[string]{Key: "a"}, 1)
//
//	goMap := ToGoMap(ampMap)
//	// goMap is now map[string]int{"a": 1}
func ToGoMap[K comparable, V any](m Map[Key[K], V]) map[K]V {
	if m == nil {
		return nil
	}

	out := make(map[K]V, m.Size())

	for k, v := range m.Seq() {
		out[k.Key] = v
	}

	return out
}
