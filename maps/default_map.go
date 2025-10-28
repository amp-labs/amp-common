package maps

import (
	"errors"
	"iter"

	"github.com/amp-labs/amp-common/collectable"
	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/zero"
)

// ErrNoDefaultValue is returned by the default value function when it cannot or chooses not to
// provide a default value for a given key. When this error is returned, the defaultMap will
// not add the key to the map and will behave as if the key simply doesn't exist.
var ErrNoDefaultValue = errors.New("no default value for this key")

// NewDefaultMap creates a Map that automatically generates default values for missing keys.
// When Get or Contains is called with a key that doesn't exist, the getDefaultValue function
// is invoked to generate a value, which is then added to the map and returned.
//
// The getDefaultValue function should return ErrNoDefaultValue when it cannot or chooses not
// to provide a default value. In that case, the map behaves as if the key doesn't exist.
//
// If storageMap is already a defaultMap, this function clones it and replaces the default
// value function with the new one provided.
//
// Parameters:
//   - storageMap: The underlying Map implementation to use for storage
//   - getDefaultValue: Function that generates default values for missing keys
//
// Example:
//
//	// Create a map that defaults to empty strings for missing keys
//	m := maps.NewDefaultMap(
//	    maps.NewHashMap[MyKey, string](hashFunc),
//	    func(k MyKey) (string, error) {
//	        return "", nil
//	    },
//	)
//	value, found, _ := m.Get(missingKey) // Returns ("", true, nil) and adds to map
//
//	// Create a map that refuses to provide defaults
//	m2 := maps.NewDefaultMap(
//	    maps.NewHashMap[MyKey, string](hashFunc),
//	    func(k MyKey) (string, error) {
//	        return "", maps.ErrNoDefaultValue
//	    },
//	)
//	value, found, _ := m2.Get(missingKey) // Returns ("", false, nil) without adding
func NewDefaultMap[K collectable.Collectable[K], V any](
	storageMap Map[K, V],
	getDefaultValue func(K) (V, error),
) Map[K, V] {
	dm, ok := storageMap.(*defaultMap[K, V])
	if ok && dm != nil {
		copied, ok := dm.Clone().(*defaultMap[K, V])
		if ok && copied != nil {
			copied.f = getDefaultValue

			return copied
		}
	}

	return &defaultMap[K, V]{
		m: storageMap,
		f: getDefaultValue,
	}
}

type defaultMap[K collectable.Collectable[K], V any] struct {
	m Map[K, V]          // Underlying map for storage
	f func(K) (V, error) // Function to generate default values for missing keys
}

var _ Map[collectableWhatever[any], any] = (*defaultMap[collectableWhatever[any], any])(nil)

// Get retrieves the value for the given key. If the key exists, returns the value with found=true.
// If the key doesn't exist, attempts to generate a default value using the default value function:
//   - If the function succeeds, adds the default value to the map and returns it with found=true
//   - If the function returns ErrNoDefaultValue, returns zero value with found=false
//   - If the function returns another error, returns that error
//
// Returns ErrHashCollision if a hash collision occurs during lookup or insertion.
func (d *defaultMap[K, V]) Get(key K) (value V, found bool, err error) {
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
// Returns ErrHashCollision if a hash collision occurs.
func (d *defaultMap[K, V]) Add(key K, value V) error {
	return d.m.Add(key, value)
}

// Remove deletes the key-value pair from the map.
// If the key doesn't exist, this is a no-op and returns nil.
// Returns ErrHashCollision if a hash collision occurs.
func (d *defaultMap[K, V]) Remove(key K) error {
	return d.m.Remove(key)
}

// Clear removes all key-value pairs from the map, leaving it empty.
func (d *defaultMap[K, V]) Clear() {
	d.m.Clear()
}

// Contains checks if the given key exists in the map. If the key doesn't exist, attempts to
// generate and add a default value using the default value function:
//   - If the function succeeds, adds the default value and returns true
//   - If the function returns ErrNoDefaultValue, returns false
//   - If the function returns another error, returns that error
//
// Returns ErrHashCollision if a hash collision occurs during lookup or insertion.
func (d *defaultMap[K, V]) Contains(key K) (bool, error) {
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
func (d *defaultMap[K, V]) addDefaultForKey(key K) (V, bool, error) {
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
func (d *defaultMap[K, V]) Size() int {
	return d.m.Size()
}

// Seq returns an iterator for ranging over all key-value pairs in the map.
// This method is compatible with Go 1.23+ range-over-func syntax:
//
//	for key, value := range map.Seq() {
//	    // process key and value
//	}
//
// The iteration order depends on the underlying Map implementation.
func (d *defaultMap[K, V]) Seq() iter.Seq2[K, V] {
	return d.m.Seq()
}

// Union creates a new defaultMap containing all key-value pairs from both this map and other.
// If a key exists in both maps, the value from other takes precedence.
// The returned map uses the same default value function as this map.
// Returns ErrHashCollision if any hash collision occurs during the operation.
func (d *defaultMap[K, V]) Union(other Map[K, V]) (Map[K, V], error) {
	tmp, err := d.m.Union(other)
	if err != nil {
		return nil, err
	}

	return &defaultMap[K, V]{
		m: tmp,
		f: d.f,
	}, nil
}

// Intersection creates a new defaultMap containing only key-value pairs whose keys exist in both maps.
// The values are taken from this map, not from other.
// The returned map uses the same default value function as this map.
// Returns ErrHashCollision if any hash collision occurs during the operation.
func (d *defaultMap[K, V]) Intersection(other Map[K, V]) (Map[K, V], error) {
	tmp, err := d.m.Intersection(other)
	if err != nil {
		return nil, err
	}

	return &defaultMap[K, V]{
		m: tmp,
		f: d.f,
	}, nil
}

// Clone creates a shallow copy of the map, duplicating its structure and entries.
// The keys and values themselves are not deep-copied; they are referenced as-is.
// Note: The cloned map does NOT preserve the default value function - it will not
// automatically generate default values for missing keys. Use the underlying map's
// Add method to explicitly add values to the cloned map.
// Returns a new Map instance with the same entries as this map.
func (d *defaultMap[K, V]) Clone() Map[K, V] {
	return &defaultMap[K, V]{
		m: d.m.Clone(),
		f: d.f,
	}
}

// HashFunction returns the hash function used by the underlying map.
// This allows callers to inspect the hash function or create compatible maps.
func (d *defaultMap[K, V]) HashFunction() hashing.HashFunc {
	return d.m.HashFunction()
}
