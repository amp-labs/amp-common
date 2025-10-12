package maps

import (
	"iter"

	"github.com/amp-labs/amp-common/collectable"
	errors2 "github.com/amp-labs/amp-common/errors"
	"github.com/amp-labs/amp-common/hashing"
)

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
type OrderedMap[K collectable.Collectable[K], V any] interface {
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
}

// NewOrderedHashMap creates a new ordered hash-based OrderedMap implementation using the provided hash function.
// The hash function must produce consistent hash values for equal keys and should
// minimize collisions to avoid ErrHashCollision errors during operations.
//
// Unlike the standard Map interface, the returned OrderedMap preserves insertion order when
// iterating through entries. The iteration order is deterministic and reflects the order
// in which keys were first added to the map.
//
// The returned map is not thread-safe. Concurrent access must be synchronized by the caller.
//
// Example:
//
//	// Using a custom hash function
//	m := maps.NewOrderedHashMap[MyKey, string](func(k hashing.Hashable) (string, error) {
//	    return k.Hash(), nil
//	})
//	m.Add(key1, "first")
//	m.Add(key2, "second")
//	// Iteration will always be in order: key1, key2
func NewOrderedHashMap[K collectable.Collectable[K], V any](hash hashing.HashFunc) OrderedMap[K, V] {
	return &orderedHashMap[K, V]{
		hash: hash,
		data: make(map[string]KeyValuePair[K, V]),
	}
}

// orderedHashMap is the concrete implementation of the OrderedMap interface using a hash table
// combined with a slice to maintain insertion order. It stores entries in a Go map indexed by
// string hash values for O(1) average-case lookup, while maintaining a separate slice of keys
// to track insertion order. Collision detection is performed by comparing the full key using
// the Comparable interface when hash values match. This ensures correctness even with imperfect
// hash functions.
//
// The implementation is not thread-safe and provides O(1) average-case lookup time with
// O(n) insertion-ordered iteration.
type orderedHashMap[K collectable.Collectable[K], V any] struct {
	orderedKeys []K                           // Slice of keys in insertion order
	hash        hashing.HashFunc              // Hash function for converting keys to string hashes
	data        map[string]KeyValuePair[K, V] // Internal storage indexed by hash values
}

// Add inserts or updates a key-value pair in the ordered hash map.
// If the key already exists (determined by both hash and equality), its value is replaced
// without changing its position in the insertion order. If the key is new, it's added to
// both the hash map and appended to the end of the orderedKeys slice.
// Returns ErrHashCollision if a different key produces the same hash value.
func (o *orderedHashMap[K, V]) Add(key K, value V) error {
	hashVal, err := o.hash(key)
	if err != nil {
		return err
	}

	prev, ok := o.data[hashVal]

	if ok && !key.Equals(prev.Key) {
		// Hash collision detected
		return errors2.ErrHashCollision
	}

	// If key doesn't exist, add it to orderedKeys
	if !ok {
		o.orderedKeys = append(o.orderedKeys, key)
	}

	o.data[hashVal] = KeyValuePair[K, V]{Key: key, Value: value}

	return nil
}

// Remove deletes a key-value pair from the ordered hash map.
// If the key doesn't exist, this is a no-op and returns nil. If the key exists, it's removed
// from both the hash map and the orderedKeys slice. This operation is O(n) due to the need
// to search and remove from the orderedKeys slice.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (o *orderedHashMap[K, V]) Remove(key K) error {
	hashVal, err := o.hash(key)
	if err != nil {
		return err
	}

	prev, ok := o.data[hashVal]

	if ok && !key.Equals(prev.Key) {
		// Hash collision detected - the stored key is different
		return errors2.ErrHashCollision
	}

	if ok {
		// Remove from orderedKeys
		for i, k := range o.orderedKeys {
			if k.Equals(key) {
				o.orderedKeys = append(o.orderedKeys[:i], o.orderedKeys[i+1:]...)

				break
			}
		}
	}

	delete(o.data, hashVal)

	return nil
}

// Clear removes all key-value pairs from the ordered hash map, resetting it to an empty state.
// The map remains usable after calling Clear. This operation is O(1) as it simply
// reallocates the internal storage and clears the orderedKeys slice, allowing the old data
// to be garbage collected.
func (o *orderedHashMap[K, V]) Clear() {
	o.orderedKeys = nil
	o.data = make(map[string]KeyValuePair[K, V])
}

// Contains checks whether a key exists in the ordered hash map.
// Returns true if the key exists, false otherwise.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (o *orderedHashMap[K, V]) Contains(key K) (bool, error) {
	hashVal, err := o.hash(key)
	if err != nil {
		return false, err
	}

	prev, ok := o.data[hashVal]

	if !ok {
		return false, nil
	}

	if !key.Equals(prev.Key) {
		// Hash collision detected - the stored key is different
		return false, errors2.ErrHashCollision
	}

	return true, nil
}

// Size returns the number of key-value pairs currently stored in the ordered hash map.
// This operation is O(1) as it simply returns the length of the internal storage.
func (o *orderedHashMap[K, V]) Size() int {
	return len(o.data)
}

// Seq returns an iterator for ranging over all key-value pairs in insertion order.
// The iterator yields (index, KeyValuePair) tuples where index is the position in the
// insertion order (0-based). This guarantees deterministic iteration order that reflects
// the order in which keys were first added to the map.
//
// This method is compatible with Go 1.23+ range-over-func syntax:
//
//	for i, entry := range map.Seq() {
//	    // process index and entry.Key, entry.Value
//	}
//
// The iterator stops early if the yield function returns false.
func (o *orderedHashMap[K, V]) Seq() iter.Seq2[int, KeyValuePair[K, V]] {
	return func(yield func(int, KeyValuePair[K, V]) bool) {
		for i, key := range o.orderedKeys {
			hashVal, err := o.hash(key)
			if err != nil {
				return
			}

			entry, ok := o.data[hashVal]
			if !ok {
				continue
			}

			if !yield(i, entry) {
				return
			}
		}
	}
}

// Union creates a new ordered hash map containing all key-value pairs from both this map and other.
// Entries from this map are added first, preserving their insertion order. Then entries from other
// are added. If a key exists in both maps, the value from other takes precedence, but the key
// maintains its original position from this map (it's not moved to the end).
// Returns a new OrderedMap instance with entries from both maps merged together.
// Returns ErrHashCollision if any hash collision occurs during the operation.
//
// The time complexity is O(n + m) where n is the size of this map and m is the size of other.
func (o *orderedHashMap[K, V]) Union(other OrderedMap[K, V]) (OrderedMap[K, V], error) {
	result := NewOrderedHashMap[K, V](o.hash)

	// Add all entries from this map (maintains order)
	for _, entry := range o.Seq() {
		if err := result.Add(entry.Key, entry.Value); err != nil {
			return nil, err
		}
	}

	// Add all entries from the other map (overwrites duplicates, adds new ones at the end)
	for _, entry := range other.Seq() {
		if err := result.Add(entry.Key, entry.Value); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Intersection creates a new ordered hash map containing only key-value pairs whose keys exist in both maps.
// The values are taken from this map, not from other. The insertion order is preserved from this map,
// so the result maintains the relative order of keys as they appeared in this map.
// Keys are compared using both hash and equality.
// Returns a new OrderedMap instance with only the common entries.
// Returns ErrHashCollision if any hash collision occurs during the operation.
//
// The time complexity is O(n) where n is the size of this map.
func (o *orderedHashMap[K, V]) Intersection(other OrderedMap[K, V]) (OrderedMap[K, V], error) {
	result := NewOrderedHashMap[K, V](o.hash)

	// Only add entries that exist in both maps (maintains order from this map)
	for _, entry := range o.Seq() {
		contains, err := other.Contains(entry.Key)
		if err != nil {
			return nil, err
		}

		if contains {
			if err := result.Add(entry.Key, entry.Value); err != nil {
				return nil, err
			}
		}
	}

	return result, nil
}

// Clone creates a shallow copy of the ordered hash map, duplicating its structure, entries,
// and insertion order. The keys and values themselves are not deep-copied; they are referenced
// as-is in the new map. Returns a new OrderedMap instance with the same entries in the same order
// as this map.
//
// If the receiver is nil, returns nil. The cloned map uses the same hash function as the original
// and is completely independent - modifications to one map do not affect the other.
//
// This operation is O(n) where n is the number of entries in the map, as it iterates through
// all entries to populate the new map in order.
//
// Note: Since the map was already validated during construction, Add operations during cloning
// should not fail with hash collisions. Any errors are silently ignored.
//
// Example:
//
//	original := maps.NewOrderedHashMap[MyKey, string](hashFunc)
//	original.Add(key1, "value1")
//	original.Add(key2, "value2")
//
//	cloned := original.Clone()
//	cloned.Add(key3, "value3")  // Does not affect original
//	// Iteration order is preserved: key1, key2, key3
func (o *orderedHashMap[K, V]) Clone() OrderedMap[K, V] {
	if o == nil {
		return nil
	}

	result := NewOrderedHashMap[K, V](o.hash)

	for _, entry := range o.Seq() {
		_ = result.Add(entry.Key, entry.Value) // Add should not fail here
	}

	return result
}
