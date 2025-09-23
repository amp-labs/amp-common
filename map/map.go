package utils

import (
	"strings"
)

// CaseInsensitiveMap is a map that allows case-sensitive and case-insensitive key lookups.
type CaseInsensitiveMap[V any] struct {
	cs map[string]*stringMapEntry[V] // case-sensitive lookups (case-sensitive key -> value)
	ci map[string]*stringMapEntry[V] // case-insensitive lookups (case-insensitive key -> value)
	kl map[string]string             // key lookup (case-insensitive key -> case-sensitive key)
}

type stringMapEntry[A any] struct {
	key   string
	value A
}

// Add adds a key-value pair to the map.
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

	entry := &stringMapEntry[V]{key: key, value: value}

	s.cs[key] = entry
	s.ci[strings.ToLower(key)] = entry
	s.kl[strings.ToLower(key)] = key
}

// AddAll adds multiple key-value pairs to the map.
func (s *CaseInsensitiveMap[V]) AddAll(keyValueMap map[string]V) {
	for key, value := range keyValueMap {
		s.Add(key, value)
	}
}

// Remove removes a key-value pair from the map by doing a case-insensitive lookup.
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
func (s *CaseInsensitiveMap[V]) RemoveAll(keys ...string) {
	for _, key := range keys {
		s.Remove(key)
	}
}

// Get retrieves a value from the map by key. The caseSensitive parameter
// determines whether the key lookup is case-sensitive or case-insensitive.
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

// GetAll returns all case-sensitive key-value pairs in the map.
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
func (s *CaseInsensitiveMap[V]) Clear() {
	s.cs = make(map[string]*stringMapEntry[V])
	s.ci = make(map[string]*stringMapEntry[V])
	s.kl = make(map[string]string)
}

// Size returns the number of key-value pairs in the map.
func (s *CaseInsensitiveMap[V]) Size() int {
	return len(s.cs)
}

// Keys returns all keys in the map.
func (s *CaseInsensitiveMap[V]) Keys() []string {
	keys := make([]string, 0, len(s.cs))

	for key := range s.cs {
		keys = append(keys, key)
	}

	return keys
}

// Values returns all values in the map.
func (s *CaseInsensitiveMap[V]) Values() []V {
	values := make([]V, 0, len(s.cs))

	for _, value := range s.cs {
		values = append(values, value.value)
	}

	return values
}

// ContainsKey checks if a key exists in the map. The caseSensitive parameter
// determines whether the key lookup is case-sensitive or case-insensitive.
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

// IsEmpty checks if the map is empty.
func (s *CaseInsensitiveMap[V]) IsEmpty() bool {
	return len(s.cs) == 0
}

// Clone creates a copy of the map.
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

// Merge merges another map into the map.
func (s *CaseInsensitiveMap[V]) Merge(other *CaseInsensitiveMap[V]) {
	for key, value := range other.cs {
		s.Add(key, value.value)
	}
}

// MergeAll merges multiple maps into the map.
func (s *CaseInsensitiveMap[V]) MergeAll(others ...*CaseInsensitiveMap[V]) {
	for _, other := range others {
		for key, value := range other.GetAll() {
			s.Add(key, value)
		}
	}
}

// Filter returns a new map with key-value pairs that satisfy the predicate.
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

func NewCaseInsensitiveMap[V any](from map[string]V) *CaseInsensitiveMap[V] {
	sm := &CaseInsensitiveMap[V]{}
	sm.AddAll(from)

	return sm
}
