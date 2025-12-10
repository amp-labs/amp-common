package maps

import (
	"errors"
	"iter"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/optional"
	"github.com/amp-labs/amp-common/set"
	"github.com/amp-labs/amp-common/zero"
)

// NewDefaultZeroOrderedMap creates an OrderedMap that automatically provides zero values for missing keys.
// This is a convenience wrapper around NewDefaultOrderedMap that uses the zero value of type V as
// the default value for any key that doesn't exist in the map.
//
// When Get or Contains is called with a key that doesn't exist, the zero value for type V is
// automatically generated and added to the map (at the end of the insertion order), then returned.
// This eliminates the need to manually check if a key exists before accessing it.
//
// Unlike NewDefaultOrderedMap, there's no way for this map to refuse providing a default value -
// it will always succeed in generating a zero value for missing keys. This makes it ideal for
// use cases where you want default initialization without custom logic.
//
// The map preserves insertion order of keys. When a zero value is auto-generated for a missing key,
// that key is appended to the end of the current insertion order.
//
// Common zero values by type:
//   - Numeric types (int, float64, etc.): 0
//   - Strings: ""
//   - Booleans: false
//   - Pointers: nil
//   - Slices, maps, channels: nil
//   - Structs: struct with all fields set to their zero values
//
// Parameters:
//   - storageMap: The underlying OrderedMap implementation to use for storage
//
// Example usage:
//
//	// Create an ordered map that defaults to 0 for missing integer keys
//	m := maps.NewDefaultZeroOrderedMap[string, int](
//	    maps.NewOrderedHashMap[string, int](hashing.NewGoHasher[string]()),
//	)
//	value, found, _ := m.Get("missing") // Returns (0, true, nil) and adds key with value 0
//
//	// Create an ordered map that defaults to empty strings
//	m2 := maps.NewDefaultZeroOrderedMap[int, string](
//	    maps.NewOrderedHashMap[int, string](hashing.NewGoHasher[int]()),
//	)
//	value, found, _ := m2.Get(42) // Returns ("", true, nil) and adds key 42 with value ""
//
//	// Works with structs too - all fields initialized to their zero values
//	type Config struct {
//	    Enabled bool
//	    Retries int
//	}
//	m3 := maps.NewDefaultZeroOrderedMap[string, Config](
//	    maps.NewOrderedHashMap[string, Config](hashing.NewGoHasher[string]()),
//	)
//	cfg, found, _ := m3.Get("app") // Returns (Config{Enabled: false, Retries: 0}, true, nil)
func NewDefaultZeroOrderedMap[K any, V any](
	storageMap OrderedMap[K, V],
) OrderedMap[K, V] {
	return NewDefaultOrderedMap[K, V](storageMap, func(k K) (V, error) {
		return zero.Value[V](), nil
	})
}

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
func NewDefaultOrderedMap[K any, V any](
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

type defaultOrderedMap[K any, V any] struct {
	m OrderedMap[K, V]   // Underlying ordered map for storage
	f func(K) (V, error) // Function to generate default values for missing keys
}

var _ OrderedMap[string, string] = (*defaultOrderedMap[string, string])(nil)

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

// GetOrElse retrieves the value for the given key, or returns defaultValue if the key doesn't exist.
// If the key doesn't exist, the default value function is NOT invoked - the provided defaultValue
// parameter is returned directly instead.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (d *defaultOrderedMap[K, V]) GetOrElse(key K, defaultValue V) (value V, err error) {
	var found bool

	value, found, err = d.m.Get(key)
	if found || err != nil {
		return value, err
	}

	newVal, added, err := d.addDefaultForKey(key)
	if err != nil {
		return zero.Value[V](), err
	}

	if !added {
		return defaultValue, nil
	}

	return newVal, nil
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

	if err := d.m.Add(key, value); err != nil { //nolint:noinlineerr // Inline error handling is clear here
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

// Keys returns a set containing all keys from the map, in insertion order.
// The returned set is a new instance and modifications to it do not affect the original map.
func (d *defaultOrderedMap[K, V]) Keys() set.OrderedSet[K] {
	return d.m.Keys()
}

// ForEach applies the given function to each key-value pair in the map.
// The iteration follows insertion order. This method is used for side effects only
// and does not return a value.
func (d *defaultOrderedMap[K, V]) ForEach(f func(key K, value V)) {
	d.m.ForEach(f)
}

// ForAll tests whether a predicate holds for all key-value pairs in the map.
// Returns true if the predicate returns true for all entries, false otherwise.
// The iteration follows insertion order and stops early if the predicate returns false for any entry.
func (d *defaultOrderedMap[K, V]) ForAll(predicate func(key K, value V) bool) bool {
	return d.m.ForAll(predicate)
}

// Filter creates a new defaultOrderedMap containing only key-value pairs for which the predicate returns true.
// The predicate function is applied to each entry in insertion order, and only matching entries are
// included in the result map. The returned map uses the same default value function as this map.
func (d *defaultOrderedMap[K, V]) Filter(predicate func(key K, value V) bool) OrderedMap[K, V] {
	return &defaultOrderedMap[K, V]{
		m: d.m.Filter(predicate),
		f: d.f,
	}
}

// FilterNot creates a new defaultOrderedMap containing only key-value pairs for which the predicate returns false.
// This is the inverse of Filter - it excludes entries where the predicate returns true.
// The returned map uses the same default value function as this map.
func (d *defaultOrderedMap[K, V]) FilterNot(predicate func(key K, value V) bool) OrderedMap[K, V] {
	return &defaultOrderedMap[K, V]{
		m: d.m.FilterNot(predicate),
		f: d.f,
	}
}

// Map transforms all key-value pairs in the map by applying the given function to each entry.
// The function receives each key-value pair in insertion order and returns a new key-value pair.
// Returns a new defaultOrderedMap containing the transformed entries with the same default value function.
// Note: If the transformation produces duplicate keys, the behavior depends on insertion order.
func (d *defaultOrderedMap[K, V]) Map(f func(key K, value V) (K, V)) OrderedMap[K, V] {
	return &defaultOrderedMap[K, V]{
		m: d.m.Map(f),
		f: d.f,
	}
}

// FlatMap applies the given function to each key-value pair and flattens the results into a single ordered map.
// Each function call receives entries in insertion order and returns an ordered map. All returned maps are
// merged together in the order they were produced.
// Returns a new defaultOrderedMap containing all entries from the flattened results
// with the same default value function.
// If duplicate keys exist across multiple results, later values take precedence.
func (d *defaultOrderedMap[K, V]) FlatMap(f func(key K, value V) OrderedMap[K, V]) OrderedMap[K, V] {
	return &defaultOrderedMap[K, V]{
		m: d.m.FlatMap(f),
		f: d.f,
	}
}

// Exists tests whether at least one key-value pair in the map satisfies the given predicate.
// Returns true if the predicate returns true for any entry, false otherwise.
// The iteration follows insertion order and stops early as soon as a matching entry is found.
func (d *defaultOrderedMap[K, V]) Exists(predicate func(key K, value V) bool) bool {
	return d.m.Exists(predicate)
}

// FindFirst searches for the first key-value pair that satisfies the given predicate.
// Returns Some(KeyValuePair) if a matching entry is found, None otherwise.
// The iteration follows insertion order, so "first" refers to the earliest inserted entry that matches.
func (d *defaultOrderedMap[K, V]) FindFirst(predicate func(key K, value V) bool) optional.Value[KeyValuePair[K, V]] {
	return d.m.FindFirst(predicate)
}
