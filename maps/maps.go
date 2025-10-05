// Package maps provides generic map utilities, including a case-insensitive map implementation.
package maps

import (
	"strings"
)

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
