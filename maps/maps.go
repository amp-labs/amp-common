// Package maps provides generic map utilities, including a case-insensitive map implementation.
package maps

import (
	"errors"
	"iter"
	"strings"

	"github.com/amp-labs/amp-common/compare"
	"github.com/amp-labs/amp-common/hashing"
)

// ErrHashCollision is returned when two distinct keys produce the same hash value.
// This error indicates that the hash function is not suitable for the given key space,
// or that the key distribution is causing unexpected collisions. When this error occurs,
// consider using a different hash function or implementing a collision resolution strategy.
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

// Map is a generic hash map interface for storing key-value pairs where keys must be
// both hashable and comparable. It provides set-like operations (Union, Intersection)
// in addition to standard map operations. All methods that modify the map or query for
// keys may return ErrHashCollision if the hash function produces collisions.
//
// Keys must implement the Collectable interface, which ensures they can be hashed
// for efficient lookup and compared for equality to resolve hash collisions.
//
// Thread-safety: Implementations are not guaranteed to be thread-safe unless
// explicitly documented. Concurrent access must be synchronized by the caller.
type Map[K Collectable[K], V any] interface {
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
}

type mapEntry[K Collectable[K], V any] struct {
	Key   K
	Value V
}

// NewMap creates a new hash-based Map implementation using the provided hash function.
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
func NewMap[K Collectable[K], V any](hash hashing.HashFunc) Map[K, V] {
	return &hashMap[K, V]{
		hash: hash,
		data: make(map[string]mapEntry[K, V]),
	}
}

type hashMap[K Collectable[K], V any] struct {
	hash hashing.HashFunc
	data map[string]mapEntry[K, V]
}

func (h *hashMap[K, V]) Add(key K, value V) error {
	hashVal, err := h.hash(key)
	if err != nil {
		return err
	}

	prev, ok := h.data[hashVal]

	if ok && !key.Equals(prev.Key) {
		// Hash collision detected
		return ErrHashCollision
	}

	h.data[hashVal] = mapEntry[K, V]{Key: key, Value: value}

	return nil
}

func (h *hashMap[K, V]) Remove(key K) error {
	hashVal, err := h.hash(key)
	if err != nil {
		return err
	}

	prev, ok := h.data[hashVal]

	if ok && !key.Equals(prev.Key) {
		// Hash collision detected - the stored key is different
		return ErrHashCollision
	}

	delete(h.data, hashVal)

	return nil
}

func (h *hashMap[K, V]) Clear() {
	h.data = make(map[string]mapEntry[K, V])
}

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
		return false, ErrHashCollision
	}

	return true, nil
}

func (h *hashMap[K, V]) Size() int {
	return len(h.data)
}

func (h *hashMap[K, V]) Seq() iter.Seq2[K, V] {
	return func(yield func(K, V) bool) {
		for _, entry := range h.data {
			if !yield(entry.Key, entry.Value) {
				return
			}
		}
	}
}

func (h *hashMap[K, V]) Union(other Map[K, V]) (Map[K, V], error) {
	result := NewMap[K, V](h.hash)

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

func (h *hashMap[K, V]) Intersection(other Map[K, V]) (Map[K, V], error) {
	result := NewMap[K, V](h.hash)

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

// CaseInsensitiveMap is a map that allows both case-sensitive and case-insensitive key lookups.
// It maintains the original casing of keys while also supporting case-insensitive retrieval.
// The zero value is not ready to use; use NewCaseInsensitiveMap to create instances.
//
// Example:
//
//	m := maps.NewCaseInsensitiveMap(map[string]string{"Content-Type": "application/json"})
//	key, val, ok := m.Get("content-type", false) // case-insensitive: returns "Content-Type", "application/json", true
//	key, val, ok := m.Get("Content-Type", true)  // case-sensitive: returns "Content-Type", "application/json", true
type CaseInsensitiveMap[V any] struct {
	cs map[string]*stringMapEntry[V] // case-sensitive lookups (case-sensitive key -> value)
	ci map[string]*stringMapEntry[V] // case-insensitive lookups (lowercase key -> value)
	kl map[string]string             // key lookup (lowercase key -> original case-sensitive key)
}

// stringMapEntry stores a key-value pair with the original key casing preserved.
type stringMapEntry[A any] struct {
	key   string
	value A
}

// Add adds a key-value pair to the map.
// If a key with different casing already exists, it removes the old entry first
// and adds the new one with the provided casing. This ensures only one entry per
// case-insensitive key exists, with the most recently added casing preserved.
func (s *CaseInsensitiveMap[V]) Add(key string, value V) {
	if s.cs == nil {
		s.cs = make(map[string]*stringMapEntry[V])
	}

	if s.ci == nil {
		s.ci = make(map[string]*stringMapEntry[V])
	}

	if s.kl == nil {
		s.kl = make(map[string]string)
	}

	// Remove old entry if a case-insensitive match exists
	ciKey := strings.ToLower(key)
	if oldCsKey, exists := s.kl[ciKey]; exists && oldCsKey != key {
		delete(s.cs, oldCsKey)
	}

	entry := &stringMapEntry[V]{key: key, value: value}

	s.cs[key] = entry
	s.ci[ciKey] = entry
	s.kl[ciKey] = key
}

// AddAll adds multiple key-value pairs to the map.
// This is a convenience method that calls Add for each entry in the provided map.
func (s *CaseInsensitiveMap[V]) AddAll(keyValueMap map[string]V) {
	for key, value := range keyValueMap {
		s.Add(key, value)
	}
}

// Remove removes a key-value pair from the map using case-insensitive lookup.
// If the key doesn't exist, this is a no-op.
func (s *CaseInsensitiveMap[V]) Remove(key string) {
	ciKey := strings.ToLower(key)
	csKey := s.kl[ciKey]

	if s.cs != nil {
		delete(s.cs, csKey)
	}

	if s.ci != nil {
		delete(s.ci, strings.ToLower(key))
	}
}

// RemoveAll removes multiple key-value pairs from the map.
// This is a convenience method that calls Remove for each provided key.
func (s *CaseInsensitiveMap[V]) RemoveAll(keys ...string) {
	for _, key := range keys {
		s.Remove(key)
	}
}

// Get retrieves a value from the map by key.
// The caseSensitive parameter determines whether the lookup is case-sensitive.
// Returns the original key (with preserved casing), value, and whether the key was found.
// When caseSensitive is false, returns the original casing of the stored key.
//
// Example:
//
//	m.Add("Content-Type", "application/json")
//	key, val, ok := m.Get("content-type", false) // returns "Content-Type", "application/json", true
//	key, val, ok := m.Get("content-type", true)  // returns "content-type", zero, false
//
// nolint:ireturn
func (s *CaseInsensitiveMap[V]) Get(key string, caseSensitive bool) (string, V, bool) {
	if caseSensitive {
		entry, ok := s.cs[key]
		if !ok {
			var zero V

			return key, zero, false
		}

		return key, entry.value, true
	}

	entry, ok := s.ci[strings.ToLower(key)]
	if !ok {
		var zero V

		return key, zero, false
	}

	return entry.key, entry.value, true
}

// GetAll returns all key-value pairs in the map with original key casing preserved.
// Returns nil if the map is empty.
func (s *CaseInsensitiveMap[V]) GetAll() map[string]V {
	if s.cs == nil {
		return nil
	}

	out := make(map[string]V)

	for _, entry := range s.cs {
		out[entry.key] = entry.value
	}

	return out
}

// Clear removes all key-value pairs from the map.
// After calling Clear, the map is empty but still usable.
func (s *CaseInsensitiveMap[V]) Clear() {
	s.cs = make(map[string]*stringMapEntry[V])
	s.ci = make(map[string]*stringMapEntry[V])
	s.kl = make(map[string]string)
}

// Size returns the number of key-value pairs in the map.
func (s *CaseInsensitiveMap[V]) Size() int {
	return len(s.cs)
}

// Keys returns all keys in the map with original casing preserved.
// The order of keys is non-deterministic.
func (s *CaseInsensitiveMap[V]) Keys() []string {
	keys := make([]string, 0, len(s.cs))

	for key := range s.cs {
		keys = append(keys, key)
	}

	return keys
}

// Values returns all values in the map.
// The order of values is non-deterministic.
func (s *CaseInsensitiveMap[V]) Values() []V {
	values := make([]V, 0, len(s.cs))

	for _, value := range s.cs {
		values = append(values, value.value)
	}

	return values
}

// ContainsKey checks if a key exists in the map.
// The caseSensitive parameter determines whether the lookup is case-sensitive.
// Returns whether the key exists and the original casing of the stored key (if found).
func (s *CaseInsensitiveMap[V]) ContainsKey(key string, caseSensitive bool) (bool, string) {
	if caseSensitive {
		_, ok := s.cs[key]

		return ok, key
	}

	entry, ok := s.ci[strings.ToLower(key)]

	if ok {
		return true, entry.key
	}

	return false, key
}

// IsEmpty returns true if the map contains no key-value pairs.
func (s *CaseInsensitiveMap[V]) IsEmpty() bool {
	return len(s.cs) == 0
}

// Clone creates a deep copy of the map.
// Returns nil if the receiver is nil.
func (s *CaseInsensitiveMap[V]) Clone() *CaseInsensitiveMap[V] {
	if s == nil {
		return nil
	}

	if s.cs == nil {
		return &CaseInsensitiveMap[V]{}
	}

	keyValueMap := make(map[string]V)

	for key, value := range s.cs {
		keyValueMap[key] = value.value
	}

	out := &CaseInsensitiveMap[V]{}
	out.AddAll(keyValueMap)

	return out
}

// Merge merges another map into this map.
// Existing keys (case-insensitive match) are replaced with values from the other map.
func (s *CaseInsensitiveMap[V]) Merge(other *CaseInsensitiveMap[V]) {
	for key, value := range other.cs {
		s.Add(key, value.value)
	}
}

// MergeAll merges multiple maps into this map.
// Maps are merged in order, with later maps overwriting earlier ones for duplicate keys.
func (s *CaseInsensitiveMap[V]) MergeAll(others ...*CaseInsensitiveMap[V]) {
	for _, other := range others {
		for key, value := range other.GetAll() {
			s.Add(key, value)
		}
	}
}

// Filter returns a new map containing only the key-value pairs that satisfy the predicate.
// The predicate receives the original key (with preserved casing) and value.
func (s *CaseInsensitiveMap[V]) Filter(predicate func(string, V) bool) *CaseInsensitiveMap[V] {
	values := make(map[string]V)

	for key, value := range s.GetAll() {
		if predicate(key, value) {
			values[key] = value
		}
	}

	out := &CaseInsensitiveMap[V]{}
	out.AddAll(values)

	return out
}

// NewCaseInsensitiveMap creates a new CaseInsensitiveMap initialized with the provided key-value pairs.
// Pass nil or an empty map to create an empty map.
func NewCaseInsensitiveMap[V any](from map[string]V) *CaseInsensitiveMap[V] {
	sm := &CaseInsensitiveMap[V]{}
	sm.AddAll(from)

	return sm
}
