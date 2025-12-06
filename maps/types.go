package maps

import (
	"iter"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/optional"
	"github.com/amp-labs/amp-common/set"
)

// KeyValuePair is a generic key-value pair struct used to represent entries in maps.
// It's particularly used by the OrderedMap.Seq() method to provide both the key and value
// in a single return value, along with an index to indicate insertion order.
//
// The Key must implement the collectable.Collectable interface (hashable and comparable),
// while the Value can be any type.
//
// Example:
//
//	// Returned by OrderedMap iteration
//	for i, entry := range orderedMap.Seq() {
//	    fmt.Printf("Index: %d, Key: %v, Value: %v\n", i, entry.Key, entry.Value)
//	}
type KeyValuePair[K any, V any] struct {
	Key   K
	Value V
}

// Map is a generic hash map interface for storing key-value pairs where keys must be
// both hashable and comparable. It provides set-like operations (Union, Intersection)
// in addition to standard map operations. All methods that modify the map or query for
// keys may return ErrHashCollision if the hash function produces collisions.
//
// Keys must implement the collectable.Collectable interface, which ensures they can be hashed
// for efficient lookup and compared for equality to resolve hash collisions.
//
// Thread-safety: Implementations are not guaranteed to be thread-safe unless
// explicitly documented. Concurrent access must be synchronized by the caller.
//
//nolint:interfacebloat,dupl // Map interface intentionally has 11 methods for cohesive API design
type Map[K any, V any] interface {
	// Get retrieves the value for the given key from the hash map.
	// If the key exists, returns the value with found=true. If the key doesn't exist, returns
	// a zero value with found=false.
	// Returns ErrHashCollision if a different key with the same hash exists in the map.
	Get(key K) (value V, found bool, err error)

	// GetOrElse retrieves the value for the given key, or returns defaultValue if the key doesn't exist.
	// Returns ErrHashCollision if a different key with the same hash exists in the map.
	GetOrElse(key K, defaultValue V) (value V, err error)

	// Add inserts or updates a key-value pair in the map.
	// If the key already exists, its value is replaced.
	// Returns ErrHashCollision if the hash function produces a collision with a different key.
	Add(key K, value V) error

	// Remove deletes the key-value pair from the map.
	// If the key doesn't exist, this is a no-op and returns nil.
	// Returns ErrHashCollision if the hash function produces a collision with a different key.
	Remove(key K) error

	// Clear removes all key-value pairs from the map, leaving it empty.
	Clear()

	// Contains checks if the given key exists in the map.
	// Returns true if the key exists, false otherwise.
	// Returns ErrHashCollision if the hash function produces a collision with a different key.
	Contains(key K) (bool, error)

	// Size returns the number of key-value pairs currently stored in the map.
	Size() int

	// Seq returns an iterator for ranging over all key-value pairs in the map.
	// The iteration order is non-deterministic. This method is compatible with
	// Go 1.23+ range-over-func syntax: for key, value := range map.Seq() { ... }
	Seq() iter.Seq2[K, V]

	// Union creates a new map containing all key-value pairs from both this map and other.
	// If a key exists in both maps, the value from other takes precedence.
	// Returns ErrHashCollision if any hash collision occurs during the operation.
	Union(other Map[K, V]) (Map[K, V], error)

	// Intersection creates a new map containing only key-value pairs whose keys exist in both maps.
	// The values are taken from this map, not from other.
	// Returns ErrHashCollision if any hash collision occurs during the operation.
	Intersection(other Map[K, V]) (Map[K, V], error)

	// Clone creates a shallow copy of the map, duplicating its structure and entries.
	// The keys and values themselves are not deep-copied; they are referenced as-is.
	// Returns a new Map instance with the same entries.
	Clone() Map[K, V]

	// HashFunction returns the hash function used by this map.
	// This allows callers to inspect the hash function or create compatible maps
	// that use the same hashing strategy, ensuring consistent key hashing across
	// different map instances.
	//
	// Example use cases:
	//   - Creating a new map with the same hash function
	//   - Verifying two maps use compatible hash functions before merging
	//   - Debugging hash collision issues
	HashFunction() hashing.HashFunc

	// Keys returns a set containing all keys from the map.
	// The returned set is a new instance and modifications to it do not affect the original map.
	Keys() set.Set[K]

	// ForEach applies the given function to each key-value pair in the map.
	// The iteration order is non-deterministic. This method is used for side effects only
	// and does not return a value.
	ForEach(f func(key K, value V))

	// ForAll tests whether a predicate holds for all key-value pairs in the map.
	// Returns true if the predicate returns true for all entries, false otherwise.
	// The iteration stops early if the predicate returns false for any entry.
	ForAll(predicate func(key K, value V) bool) bool

	// Filter creates a new map containing only key-value pairs for which the predicate returns true.
	// The predicate function is applied to each entry, and only matching entries are included
	// in the result map.
	Filter(predicate func(key K, value V) bool) Map[K, V]

	// FilterNot creates a new map containing only key-value pairs for which the predicate returns false.
	// This is the inverse of Filter - it excludes entries where the predicate returns true.
	FilterNot(predicate func(key K, value V) bool) Map[K, V]

	// Map transforms all key-value pairs in the map by applying the given function to each entry.
	// The function receives each key-value pair and returns a new key-value pair.
	// Returns a new map containing the transformed entries.
	// Note: If the transformation produces duplicate keys, the behavior depends on insertion order.
	Map(f func(key K, value V) (K, V)) Map[K, V]

	// FlatMap applies the given function to each key-value pair and flattens the results into a single map.
	// Each function call returns a map, and all returned maps are merged together.
	// Returns a new map containing all entries from the flattened results.
	// If duplicate keys exist across multiple results, later values take precedence.
	FlatMap(f func(key K, value V) Map[K, V]) Map[K, V]

	// Exists tests whether at least one key-value pair in the map satisfies the given predicate.
	// Returns true if the predicate returns true for any entry, false otherwise.
	// The iteration stops early as soon as a matching entry is found.
	Exists(predicate func(key K, value V) bool) bool

	// FindFirst searches for the first key-value pair that satisfies the given predicate.
	// Returns Some(KeyValuePair) if a matching entry is found, None otherwise.
	// The iteration order is non-deterministic, so "first" is not guaranteed to be consistent.
	FindFirst(predicate func(key K, value V) bool) optional.Value[KeyValuePair[K, V]]
}

// OrderedMap is a generic ordered hash map interface for storing key-value pairs where keys must be
// both hashable and comparable. Unlike the standard Map interface, OrderedMap preserves insertion
// order when iterating. It provides set-like operations (Union, Intersection) in addition to standard
// map operations. All methods that modify the map or query for keys may return ErrHashCollision if
// the hash function produces collisions.
//
// Keys must implement the collectable.Collectable interface, which ensures they can be hashed
// for efficient lookup and compared for equality to resolve hash collisions.
//
// Thread-safety: Implementations are not guaranteed to be thread-safe unless
// explicitly documented. Concurrent access must be synchronized by the caller.
//
//nolint:interfacebloat,dupl // OrderedMap interface intentionally has 11 methods for cohesive API design
type OrderedMap[K any, V any] interface {
	// Get retrieves the value for the given key from the hash map.
	// If the key exists, returns the value with found=true. If the key doesn't exist, returns
	// a zero value with found=false.
	// Returns ErrHashCollision if a different key with the same hash exists in the map.
	Get(key K) (value V, found bool, err error)

	// GetOrElse retrieves the value for the given key, or returns defaultValue if the key doesn't exist.
	// Returns ErrHashCollision if a different key with the same hash exists in the map.
	GetOrElse(key K, defaultValue V) (value V, err error)

	// Add inserts or updates a key-value pair in the map.
	// If the key already exists, its value is replaced without changing the insertion order.
	// If the key is new, it's appended to the end of the insertion order.
	// Returns ErrHashCollision if the hash function produces a collision with a different key.
	Add(key K, value V) error

	// Remove deletes the key-value pair from the map.
	// If the key doesn't exist, this is a no-op and returns nil.
	// Returns ErrHashCollision if the hash function produces a collision with a different key.
	Remove(key K) error

	// Clear removes all key-value pairs from the map, leaving it empty.
	Clear()

	// Contains checks if the given key exists in the map.
	// Returns true if the key exists, false otherwise.
	// Returns ErrHashCollision if the hash function produces a collision with a different key.
	Contains(key K) (bool, error)

	// Size returns the number of key-value pairs currently stored in the map.
	Size() int

	// Seq returns an iterator for ranging over all key-value pairs in insertion order.
	// The iterator yields (index, KeyValuePair) tuples where index represents the insertion order.
	// This method is compatible with Go 1.23+ range-over-func syntax:
	// for i, entry := range map.Seq() { ... }
	Seq() iter.Seq2[int, KeyValuePair[K, V]]

	// Union creates a new map containing all key-value pairs from both this map and other.
	// Entries from this map are added first (preserving their order), followed by entries from other.
	// If a key exists in both maps, the value from other takes precedence, but the key maintains
	// its original position from this map.
	// Returns ErrHashCollision if any hash collision occurs during the operation.
	Union(other OrderedMap[K, V]) (OrderedMap[K, V], error)

	// Intersection creates a new map containing only key-value pairs whose keys exist in both maps.
	// The values are taken from this map, not from other, and the insertion order is preserved
	// from this map.
	// Returns ErrHashCollision if any hash collision occurs during the operation.
	Intersection(other OrderedMap[K, V]) (OrderedMap[K, V], error)

	// Clone creates a shallow copy of the map, duplicating its structure, entries, and insertion order.
	// The keys and values themselves are not deep-copied; they are referenced as-is.
	// Returns a new OrderedMap instance with the same entries in the same order.
	Clone() OrderedMap[K, V]

	// HashFunction returns the hash function used by this ordered map.
	// This allows callers to inspect the hash function or create compatible ordered maps
	// that use the same hashing strategy, ensuring consistent key hashing across
	// different map instances.
	//
	// Example use cases:
	//   - Creating a new ordered map with the same hash function
	//   - Verifying two ordered maps use compatible hash functions before merging
	//   - Debugging hash collision issues
	HashFunction() hashing.HashFunc

	// Keys returns a set containing all keys from the map, in insertion order.
	// The returned set is a new instance and modifications to it do not affect the original map.
	Keys() set.OrderedSet[K]

	// ForEach applies the given function to each key-value pair in the map.
	// The iteration order is non-deterministic. This method is used for side effects only
	// and does not return a value.
	ForEach(f func(key K, value V))

	// ForAll tests whether a predicate holds for all key-value pairs in the map.
	// Returns true if the predicate returns true for all entries, false otherwise.
	// The iteration stops early if the predicate returns false for any entry.
	ForAll(predicate func(key K, value V) bool) bool

	// Filter creates a new map containing only key-value pairs for which the predicate returns true.
	// The predicate function is applied to each entry, and only matching entries are included
	// in the result map.
	Filter(predicate func(key K, value V) bool) OrderedMap[K, V]

	// FilterNot creates a new map containing only key-value pairs for which the predicate returns false.
	// This is the inverse of Filter - it excludes entries where the predicate returns true.
	FilterNot(predicate func(key K, value V) bool) OrderedMap[K, V]

	// Map transforms all key-value pairs in the map by applying the given function to each entry.
	// The function receives each key-value pair and returns a new key-value pair.
	// Returns a new map containing the transformed entries.
	// Note: If the transformation produces duplicate keys, the behavior depends on insertion order.
	Map(f func(key K, value V) (K, V)) OrderedMap[K, V]

	// FlatMap applies the given function to each key-value pair and flattens the results into a single map.
	// Each function call returns a map, and all returned maps are merged together.
	// Returns a new map containing all entries from the flattened results.
	// If duplicate keys exist across multiple results, later values take precedence.
	FlatMap(f func(key K, value V) OrderedMap[K, V]) OrderedMap[K, V]

	// Exists tests whether at least one key-value pair in the map satisfies the given predicate.
	// Returns true if the predicate returns true for any entry, false otherwise.
	// The iteration stops early as soon as a matching entry is found.
	Exists(predicate func(key K, value V) bool) bool

	// FindFirst searches for the first key-value pair that satisfies the given predicate.
	// Returns Some(KeyValuePair) if a matching entry is found, None otherwise.
	// The iteration order is non-deterministic, so "first" is not guaranteed to be consistent.
	FindFirst(predicate func(key K, value V) bool) optional.Value[KeyValuePair[K, V]]
}
