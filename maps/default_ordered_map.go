package maps

import (
	"errors"
	"iter"

	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/zero"
)

// NewDefaultOrderedMap creates an OrderedMap that automatically generates default values for missing keys.
// When Get or Contains is called with a key that doesn't exist, the getDefaultValue function
// is invoked to generate a value, which is then added to the map (at the end of the insertion order)
// and returned.
//
// The getDefaultValue function should return ErrNoDefaultValue when it cannot or chooses not
// to provide a default value. In that case, the map behaves as if the key doesn't exist.
//
// If storageMap is already a defaultOrderedMap, this function clones it and replaces the default
// value function with the new one provided.
//
// Unlike the standard Map, OrderedMap preserves the insertion order of keys. When a default value
// is generated and added, it's appended to the end of the current insertion order.
//
// Parameters:
//   - storageMap: The underlying OrderedMap implementation to use for storage
//   - getDefaultValue: Function that generates default values for missing keys
//
// Example:
//
//	// Create an ordered map that defaults to empty strings for missing keys
//	m := maps.NewDefaultOrderedMap(
//	    maps.NewOrderedHashMap[MyKey, string](hashFunc),
//	    func(k MyKey) (string, error) {
//	        return "", nil
//	    },
//	)
//	value, found, _ := m.Get(missingKey) // Returns ("", true, nil) and adds to end
//
//	// Create a map that refuses to provide defaults
//	m2 := maps.NewDefaultOrderedMap(
//	    maps.NewOrderedHashMap[MyKey, string](hashFunc),
//	    func(k MyKey) (string, error) {
//	        return "", maps.ErrNoDefaultValue
//	    },
//	)
//	value, found, _ := m2.Get(missingKey) // Returns ("", false, nil) without adding
func NewDefaultOrderedMap[K collectable.Collectable[K], V any](
	storageMap OrderedMap[K, V],
	getDefaultValue func(K) (V, error),
) OrderedMap[K, V] {
	dm, ok := storageMap.(*defaultOrderedMap[K, V])
	if ok && dm != nil {
		copied, ok := dm.Clone().(*defaultOrderedMap[K, V])
		if ok && copied != nil {
			copied.f = getDefaultValue

			return copied
		}
	}

	return &defaultOrderedMap[K, V]{
		m: storageMap,
		f: getDefaultValue,
	}
}

type defaultOrderedMap[K collectable.Collectable[K], V any] struct {
	m OrderedMap[K, V]   // Underlying ordered map for storage
	f func(K) (V, error) // Function to generate default values for missing keys
}

var _ OrderedMap[collectableWhatever[any], any] = (*defaultOrderedMap[collectableWhatever[any], any])(nil)

// Get retrieves the value for the given key. If the key exists, returns the value with found=true.
// If the key doesn't exist, attempts to generate a default value using the default value function:
//   - If the function succeeds, adds the default value to the map and returns it with found=true
//   - If the function returns ErrNoDefaultValue, returns zero value with found=false
//   - If the function returns another error, returns that error
//
// Returns ErrHashCollision if a hash collision occurs during lookup or insertion.
func (d *defaultOrderedMap[K, V]) Get(key K) (value V, found bool, err error) {
	value, found, err = d.m.Get(key)
	if found || err != nil {
		return value, found, err
	}

	newVal, added, err := d.addDefaultForKey(key)
	if err != nil {
		return zero.Value[V](), false, err
	}

	if !added {
		return zero.Value[V](), false, nil
	}

	return newVal, true, nil
}

// Add inserts or updates a key-value pair in the map.
// This operation bypasses the default value function and directly adds the provided value.
// If the key already exists, its value is replaced without changing the insertion order.
// If the key is new, it's appended to the end of the insertion order.
// Returns ErrHashCollision if a hash collision occurs.
func (d *defaultOrderedMap[K, V]) Add(key K, value V) error {
	return d.m.Add(key, value)
}

// Remove deletes the key-value pair from the map.
// If the key doesn't exist, this is a no-op and returns nil.
// Returns ErrHashCollision if a hash collision occurs.
func (d *defaultOrderedMap[K, V]) Remove(key K) error {
	return d.m.Remove(key)
}

// Clear removes all key-value pairs from the map, leaving it empty.
func (d *defaultOrderedMap[K, V]) Clear() {
	d.m.Clear()
}

// Contains checks if the given key exists in the map. If the key doesn't exist, attempts to
// generate and add a default value using the default value function:
//   - If the function succeeds, adds the default value and returns true
//   - If the function returns ErrNoDefaultValue, returns false
//   - If the function returns another error, returns that error
//
// Returns ErrHashCollision if a hash collision occurs during lookup or insertion.
func (d *defaultOrderedMap[K, V]) Contains(key K) (bool, error) {
	contains, err := d.m.Contains(key)
	if err != nil {
		return false, err
	}

	if contains {
		return true, nil
	}

	_, added, err := d.addDefaultForKey(key)
	if err != nil {
		return false, err
	}

	return added, nil
}

// addDefaultForKey calls the default value function to generate a value for the given key,
// then adds it to the map if successful. Returns the generated value, whether it was added,
// and any error that occurred.
//   - If the function returns ErrNoDefaultValue, returns (zero, false, nil)
//   - If the function returns another error, returns (zero, false, error)
//   - If the function succeeds but Add fails, returns (zero, false, error)
//   - If both succeed, returns (value, true, nil)
func (d *defaultOrderedMap[K, V]) addDefaultForKey(key K) (V, bool, error) {
	value, err := d.f(key)
	if err != nil {
		var zeroVal V

		if errors.Is(err, ErrNoDefaultValue) {
			return zeroVal, false, nil
		}

		return zeroVal, false, err
	}

	if err := d.m.Add(key, value); err != nil {
		var zeroVal V

		return zeroVal, false, err
	}

	return value, true, nil
}

// Size returns the number of key-value pairs currently stored in the map.
func (d *defaultOrderedMap[K, V]) Size() int {
	return d.m.Size()
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
func (d *defaultOrderedMap[K, V]) Seq() iter.Seq2[int, KeyValuePair[K, V]] {
	return d.m.Seq()
}

// Union creates a new defaultOrderedMap containing all key-value pairs from both this map and other.
// Entries from this map are added first (preserving their order), followed by entries from other.
// If a key exists in both maps, the value from other takes precedence, but the key maintains
// its original position from this map.
// The returned map uses the same default value function as this map.
// Returns ErrHashCollision if any hash collision occurs during the operation.
func (d *defaultOrderedMap[K, V]) Union(other OrderedMap[K, V]) (OrderedMap[K, V], error) {
	tmp, err := d.m.Union(other)
	if err != nil {
		return nil, err
	}

	return &defaultOrderedMap[K, V]{
		m: tmp,
		f: d.f,
	}, nil
}

// Intersection creates a new defaultOrderedMap containing only key-value pairs whose keys exist in both maps.
// The values are taken from this map, not from other, and the insertion order is preserved from this map.
// The returned map uses the same default value function as this map.
// Returns ErrHashCollision if any hash collision occurs during the operation.
func (d *defaultOrderedMap[K, V]) Intersection(other OrderedMap[K, V]) (OrderedMap[K, V], error) {
	tmp, err := d.m.Intersection(other)
	if err != nil {
		return nil, err
	}

	return &defaultOrderedMap[K, V]{
		m: tmp,
		f: d.f,
	}, nil
}

// Clone creates a shallow copy of the map, duplicating its structure, entries, and insertion order.
// The keys and values themselves are not deep-copied; they are referenced as-is.
// Note: The cloned map does NOT preserve the default value function - it will not
// automatically generate default values for missing keys. Use the underlying map's
// Add method to explicitly add values to the cloned map.
// Returns a new OrderedMap instance with the same entries in the same order as this map.
func (d *defaultOrderedMap[K, V]) Clone() OrderedMap[K, V] {
	return &defaultOrderedMap[K, V]{
		m: d.m.Clone(),
		f: d.f,
	}
}

// HashFunction returns the hash function used by the underlying ordered map.
// This allows callers to inspect the hash function or create compatible maps.
func (d *defaultOrderedMap[K, V]) HashFunction() hashing.HashFunc {
	return d.m.HashFunction()
}
