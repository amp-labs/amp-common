package maps

import (
	"iter"

	"github.com/amp-labs/amp-common/collectable"
	errors2 "github.com/amp-labs/amp-common/errors"
	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/optional"
	"github.com/amp-labs/amp-common/set"
	"github.com/amp-labs/amp-common/zero"
)

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
type Map[K collectable.Collectable[K], V any] interface {
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

// NewHashMap creates a new hash-based Map implementation using the provided hash function.
// The hash function must produce consistent hash values for equal keys and should
// minimize collisions to avoid ErrHashCollision errors during operations.
//
// The returned map is not thread-safe. Concurrent access must be synchronized by the caller.
//
// Example:
//
//	// Using a custom hash function
//	m := maps.NewMap[MyKey, string](func(k hashing.Hashable) (string, error) {
//	    return k.Hash(), nil
//	})
//	m.Add(key, "value")
func NewHashMap[K collectable.Collectable[K], V any](hash hashing.HashFunc) Map[K, V] {
	return &hashMap[K, V]{
		hash: hash,
		data: make(map[string]KeyValuePair[K, V]),
	}
}

// NewHashMapWithSize creates a new hash-based Map implementation with pre-allocated capacity.
// This function is similar to NewHashMap but allows specifying an initial capacity hint to
// optimize memory allocation when the expected map size is known in advance.
//
// The size parameter pre-allocates space for approximately 'size' entries, reducing the need
// for memory reallocation during initial insertions. This can improve performance when building
// large maps. The map will still grow dynamically if more entries are added beyond the initial size.
//
// The hash function must produce consistent hash values for equal keys and should minimize
// collisions to avoid ErrHashCollision errors during operations.
//
// The returned map is not thread-safe. Concurrent access must be synchronized by the caller.
//
// Use this function when:
//   - You know the approximate number of entries in advance
//   - You're building a large map and want to avoid multiple reallocations
//   - Performance during initial population is critical
//
// Example:
//
//	// Creating a map for 1000 expected entries
//	m := maps.NewHashMapWithSize[MyKey, string](hashFunc, 1000)
//	for i := 0; i < 1000; i++ {
//	    m.Add(keys[i], values[i])
//	}
func NewHashMapWithSize[K collectable.Collectable[K], V any](hash hashing.HashFunc, size int) Map[K, V] {
	return &hashMap[K, V]{
		hash: hash,
		data: make(map[string]KeyValuePair[K, V], size),
	}
}

// hashMap is the concrete implementation of the Map interface using a hash table.
// It stores entries in a Go map indexed by string hash values. Collision detection
// is performed by comparing the full key using the Comparable interface when hash
// values match. This ensures correctness even with imperfect hash functions.
//
// The implementation is not thread-safe and uses O(1) average-case lookup time.
type hashMap[K collectable.Collectable[K], V any] struct {
	hash hashing.HashFunc              // Hash function for converting keys to string hashes
	data map[string]KeyValuePair[K, V] // Internal storage indexed by hash values
}

// Get retrieves the value for the given key from the hash map.
// If the key exists, returns the value with found=true. If the key doesn't exist, returns
// a zero value with found=false.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (h *hashMap[K, V]) Get(key K) (value V, found bool, errOut error) {
	hashVal, err := h.hash(key)
	if err != nil {
		return zero.Value[V](), false, err
	}

	kv, ok := h.data[hashVal]
	if !ok {
		return zero.Value[V](), false, nil
	}

	if !kv.Key.Equals(key) {
		return zero.Value[V](), false, errors2.ErrHashCollision
	}

	return kv.Value, true, nil
}

// GetOrElse retrieves the value for the given key, or returns defaultValue if the key doesn't exist.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (h *hashMap[K, V]) GetOrElse(key K, defaultValue V) (value V, err error) {
	value, found, err := h.Get(key)
	if err != nil {
		return zero.Value[V](), err
	}

	if !found {
		return defaultValue, nil
	}

	return value, nil
}

// Add inserts or updates a key-value pair in the hash map.
// If the key already exists (determined by both hash and equality), its value is replaced.
// Returns ErrHashCollision if a different key produces the same hash value.
func (h *hashMap[K, V]) Add(key K, value V) error {
	hashVal, err := h.hash(key)
	if err != nil {
		return err
	}

	prev, ok := h.data[hashVal]

	if ok && !key.Equals(prev.Key) {
		// Hash collision detected
		return errors2.ErrHashCollision
	}

	h.data[hashVal] = KeyValuePair[K, V]{Key: key, Value: value}

	return nil
}

// Remove deletes a key-value pair from the hash map.
// If the key doesn't exist, this is a no-op and returns nil.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (h *hashMap[K, V]) Remove(key K) error {
	hashVal, err := h.hash(key)
	if err != nil {
		return err
	}

	prev, ok := h.data[hashVal]

	if ok && !key.Equals(prev.Key) {
		// Hash collision detected - the stored key is different
		return errors2.ErrHashCollision
	}

	delete(h.data, hashVal)

	return nil
}

// Clear removes all key-value pairs from the hash map, resetting it to an empty state.
// The map remains usable after calling Clear. This operation is O(1) as it simply
// reallocates the internal storage, allowing the old data to be garbage collected.
func (h *hashMap[K, V]) Clear() {
	h.data = make(map[string]KeyValuePair[K, V])
}

// Contains checks whether a key exists in the hash map.
// Returns true if the key exists, false otherwise.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (h *hashMap[K, V]) Contains(key K) (bool, error) {
	hashVal, err := h.hash(key)
	if err != nil {
		return false, err
	}

	prev, ok := h.data[hashVal]

	if !ok {
		return false, nil
	}

	if !key.Equals(prev.Key) {
		// Hash collision detected - the stored key is different
		return false, errors2.ErrHashCollision
	}

	return true, nil
}

// Size returns the number of key-value pairs currently stored in the hash map.
// This operation is O(1) as it simply returns the length of the internal storage.
func (h *hashMap[K, V]) Size() int {
	return len(h.data)
}

// Seq returns an iterator for ranging over all key-value pairs in the hash map.
// The iteration order is non-deterministic as it depends on the internal hash map iteration order.
// This method is compatible with Go 1.23+ range-over-func syntax:
//
//	for key, value := range map.Seq() {
//	    // process key and value
//	}
//
// The iterator stops early if the yield function returns false.
func (h *hashMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, entry := range h.data {
			if !yield(entry.Key, entry.Value) {
				return
			}
		}
	}
}

// Union creates a new hash map containing all key-value pairs from both this map and other.
// If a key exists in both maps, the value from other takes precedence in the result.
// Returns a new Map instance with entries from both maps merged together.
// Returns ErrHashCollision if any hash collision occurs during the operation.
//
// The time complexity is O(n + m) where n is the size of this map and m is the size of other.
func (h *hashMap[K, V]) Union(other Map[K, V]) (Map[K, V], error) {
	result := NewHashMap[K, V](h.hash)

	// Add all entries from this map
	for key, value := range h.Seq() {
		if err := result.Add(key, value); err != nil {
			return nil, err
		}
	}

	// Add all entries from the other map (overwrites duplicates)
	for key, value := range other.Seq() {
		if err := result.Add(key, value); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Intersection creates a new hash map containing only key-value pairs whose keys exist in both maps.
// The values are taken from this map, not from other. Keys are compared using both hash and equality.
// Returns a new Map instance with only the common entries.
// Returns ErrHashCollision if any hash collision occurs during the operation.
//
// The time complexity is O(n) where n is the size of the smaller map.
func (h *hashMap[K, V]) Intersection(other Map[K, V]) (Map[K, V], error) {
	result := NewHashMap[K, V](h.hash)

	// Only add entries that exist in both maps
	for key, value := range h.Seq() {
		contains, err := other.Contains(key)
		if err != nil {
			return nil, err
		}

		if contains {
			if err := result.Add(key, value); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// Clone creates a shallow copy of the hash map, duplicating its structure and entries.
// The keys and values themselves are not deep-copied; they are referenced as-is in the new map.
// Returns a new Map instance with the same entries as this map.
//
// If the receiver is nil, returns nil. The cloned map uses the same hash function as the original
// and is completely independent - modifications to one map do not affect the other.
//
// This operation is O(n) where n is the number of entries in the map, as it iterates through
// all entries to populate the new map.
//
// Note: Since the map was already validated during construction, Add operations during cloning
// should not fail with hash collisions. Any errors are silently ignored.
//
// Example:
//
//	original := maps.NewHashMap[MyKey, string](hashFunc)
//	original.Add(key1, "value1")
//
//	cloned := original.Clone()
//	cloned.Add(key2, "value2")  // Does not affect original
func (h *hashMap[K, V]) Clone() Map[K, V] {
	if h == nil {
		return nil
	}

	result := NewHashMap[K, V](h.hash)

	for key, value := range h.Seq() {
		_ = result.Add(key, value) // Add should not fail here
	}

	return result
}

// HashFunction returns the hash function used by this map.
// This allows callers to inspect or reuse the hash function for creating compatible maps.
func (h *hashMap[K, V]) HashFunction() hashing.HashFunc {
	return h.hash
}

// Keys returns a set containing all keys from the map.
// The returned set is a new instance and modifications to it do not affect the original map.
func (h *hashMap[K, V]) Keys() set.Set[K] {
	keys := set.NewSet[K](h.hash)

	for key := range h.Seq() {
		_ = keys.Add(key) // Add should not fail for existing keys
	}

	return keys
}

// ForEach applies the given function to each key-value pair in the map.
// The iteration order is non-deterministic. This method is used for side effects only
// and does not return a value.
func (h *hashMap[K, V]) ForEach(f func(key K, value V)) {
	for key, value := range h.Seq() {
		f(key, value)
	}
}

// ForAll tests whether a predicate holds for all key-value pairs in the map.
// Returns true if the predicate returns true for all entries, false otherwise.
// The iteration stops early if the predicate returns false for any entry.
func (h *hashMap[K, V]) ForAll(predicate func(key K, value V) bool) bool {
	for key, value := range h.Seq() {
		if !predicate(key, value) {
			return false
		}
	}

	return true
}

// Filter creates a new map containing only key-value pairs for which the predicate returns true.
// The predicate function is applied to each entry, and only matching entries are included
// in the result map.
func (h *hashMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	result := NewHashMap[K, V](h.hash)

	for key, value := range h.Seq() {
		if predicate(key, value) {
			_ = result.Add(key, value) // Add should not fail for valid keys
		}
	}

	return result
}

// FilterNot creates a new map containing only key-value pairs for which the predicate returns false.
// This is the inverse of Filter - it excludes entries where the predicate returns true.
func (h *hashMap[K, V]) FilterNot(predicate func(key K, value V) bool) Map[K, V] {
	result := NewHashMap[K, V](h.hash)

	for key, value := range h.Seq() {
		if !predicate(key, value) {
			_ = result.Add(key, value) // Add should not fail for valid keys
		}
	}

	return result
}

// Map transforms all key-value pairs in the map by applying the given function to each entry.
// The function receives each key-value pair and returns a new key-value pair.
// Returns a new map containing the transformed entries.
// Note: If the transformation produces duplicate keys, the behavior depends on insertion order.
func (h *hashMap[K, V]) Map(f func(key K, value V) (K, V)) Map[K, V] {
	result := NewHashMap[K, V](h.hash)

	for key, value := range h.Seq() {
		newKey, newValue := f(key, value)
		_ = result.Add(newKey, newValue) // Duplicate keys will be overwritten
	}

	return result
}

// FlatMap applies the given function to each key-value pair and flattens the results into a single map.
// Each function call returns a map, and all returned maps are merged together.
// Returns a new map containing all entries from the flattened results.
// If duplicate keys exist across multiple results, later values take precedence.
func (h *hashMap[K, V]) FlatMap(f func(key K, value V) Map[K, V]) Map[K, V] {
	result := NewHashMap[K, V](h.hash)

	for key, value := range h.Seq() {
		mappedResult := f(key, value)
		for newKey, newValue := range mappedResult.Seq() {
			_ = result.Add(newKey, newValue) // Duplicate keys will be overwritten
		}
	}

	return result
}

// Exists tests whether at least one key-value pair in the map satisfies the given predicate.
// Returns true if the predicate returns true for any entry, false otherwise.
// The iteration stops early as soon as a matching entry is found.
func (h *hashMap[K, V]) Exists(predicate func(key K, value V) bool) bool {
	for key, value := range h.Seq() {
		if predicate(key, value) {
			return true
		}
	}

	return false
}

// FindFirst searches for the first key-value pair that satisfies the given predicate.
// Returns Some(KeyValuePair) if a matching entry is found, None otherwise.
// The iteration order is non-deterministic, so "first" is not guaranteed to be consistent.
func (h *hashMap[K, V]) FindFirst(predicate func(key K, value V) bool) optional.Value[KeyValuePair[K, V]] {
	for key, value := range h.Seq() {
		if predicate(key, value) {
			return optional.Some(KeyValuePair[K, V]{Key: key, Value: value})
		}
	}

	return optional.None[KeyValuePair[K, V]]()
}
