package maps

import (
	"iter"
	"sync"

	"github.com/amp-labs/amp-common/hashing"
	"github.com/amp-labs/amp-common/optional"
	"github.com/amp-labs/amp-common/set"
)

// NewThreadSafeMap wraps an existing Map implementation with thread-safe access using sync.RWMutex.
// It provides concurrent read/write access to the underlying map while preserving the Map interface.
//
// The wrapper uses read-write locks to allow multiple concurrent readers or exclusive writer access.
// Write operations (Add, Remove, Clear) acquire exclusive locks, while read operations
// (Contains, Size, Seq, Union, Intersection, Clone) use shared read locks for better concurrency.
//
// Example usage:
//
//	unsafeMap := maps.New[string, int]()
//	safeMap := maps.NewThreadSafeMap(unsafeMap)
//	safeMap.Add("key", 42) // thread-safe
func NewThreadSafeMap[K any, V any](m Map[K, V]) Map[K, V] {
	if m == nil {
		return nil
	}

	tsm, ok := m.(*threadSafeMap[K, V])
	if ok {
		// Already thread-safe, return as-is
		return tsm
	}

	return &threadSafeMap[K, V]{
		internal: m,
	}
}

// threadSafeMap is a decorator that wraps any Map implementation with thread-safe access.
// It uses sync.RWMutex to coordinate concurrent access, allowing multiple simultaneous
// readers or a single exclusive writer.
type threadSafeMap[K any, V any] struct {
	mutex    sync.RWMutex // Protects access to internal map
	internal Map[K, V]    // Underlying map implementation
}

// Get retrieves the value for the given key with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Get calls without blocking each other.
// Returns the value and found=true if the key exists, or zero value and found=false if not.
// Returns ErrHashCollision if a hash collision occurs.
func (t *threadSafeMap[K, V]) Get(key K) (value V, found bool, err error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Get(key)
}

// GetOrElse retrieves the value for the given key, or returns defaultValue if the key doesn't exist.
// Acquires a read lock during the operation.
// Returns ErrHashCollision if a different key with the same hash exists in the map.
func (t *threadSafeMap[K, V]) GetOrElse(key K, defaultValue V) (value V, err error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.GetOrElse(key, defaultValue)
}

// Add inserts or updates a key-value pair in the map with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeMap[K, V]) Add(key K, value V) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.Add(key, value)
}

// Remove deletes a key-value pair from the map with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeMap[K, V]) Remove(key K) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.Remove(key)
}

// Clear removes all entries from the map with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeMap[K, V]) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.internal.Clear()
}

// Contains checks if a key exists in the map with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Contains calls without blocking each other.
func (t *threadSafeMap[K, V]) Contains(key K) (bool, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Contains(key)
}

// Size returns the number of entries in the map with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Size calls without blocking each other.
func (t *threadSafeMap[K, V]) Size() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Size()
}

// Seq returns an iterator over the map's key-value pairs with snapshot semantics.
// Acquires a read lock only during the snapshot creation, then releases it before returning.
//
// Implementation note: This creates a complete snapshot of the map under a read lock,
// then returns an iterator over that snapshot. This design ensures:
//   - The read lock is not held during iteration (preventing long-lived lock holding)
//   - Iteration sees a consistent view of the map at the time Seq() was called
//   - Multiple goroutines can iterate concurrently without blocking each other
//   - Changes made after Seq() is called are not visible to the iterator
//
// Trade-off: Uses O(n) memory to store the snapshot, but provides better concurrency
// characteristics than holding the lock during iteration.
func (t *threadSafeMap[K, V]) Seq() iter.Seq2[K, V] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Create snapshot under read lock
	accum := make([]KeyValuePair[K, V], 0, t.internal.Size())

	for key, val := range t.internal.Seq() {
		accum = append(accum, KeyValuePair[K, V]{Key: key, Value: val})
	}

	// Return iterator over snapshot (no lock held during iteration)
	return func(yield func(K, V) bool) {
		for _, kv := range accum {
			if !yield(kv.Key, kv.Value) {
				return
			}
		}
	}
}

// Union creates a new thread-safe map containing all entries from both this map and another.
// Acquires a read lock on this map during the operation. The returned map is also thread-safe.
//
// Note: The 'other' map is accessed directly without coordination. If 'other' is also being
// modified concurrently, the caller should handle that synchronization externally.
func (t *threadSafeMap[K, V]) Union(other Map[K, V]) (Map[K, V], error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	value, err := t.internal.Union(other)
	if err != nil {
		return nil, err
	}

	return NewThreadSafeMap(value), nil
}

// Intersection creates a new thread-safe map containing only entries present in both maps.
// Acquires a read lock on this map during the operation. The returned map is also thread-safe.
//
// Note: The 'other' map is accessed directly without coordination. If 'other' is also being
// modified concurrently, the caller should handle that synchronization externally.
func (t *threadSafeMap[K, V]) Intersection(other Map[K, V]) (Map[K, V], error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	value, err := t.internal.Intersection(other)
	if err != nil {
		return nil, err
	}

	return NewThreadSafeMap(value), nil
}

// Clone creates a deep copy of the map with independent thread-safe access.
// Acquires a read lock on this map during the clone operation.
// The returned map is a new thread-safe instance that can be modified independently.
func (t *threadSafeMap[K, V]) Clone() Map[K, V] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return NewThreadSafeMap(t.internal.Clone())
}

// HashFunction returns the hash function used by the underlying map.
// This allows callers to inspect or reuse the hash function for creating compatible maps.
func (t *threadSafeMap[K, V]) HashFunction() hashing.HashFunc {
	return t.internal.HashFunction()
}

// Keys returns a set containing all keys from the map.
// Acquires a read lock during the operation.
// The returned set is a new instance and modifications to it do not affect the original map.
func (t *threadSafeMap[K, V]) Keys() set.Set[K] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Keys()
}

// ForEach applies the given function to each key-value pair in the map.
// Acquires a read lock only during snapshot creation to avoid holding the lock during callback execution.
// The iteration is performed on a snapshot of the map, so changes made after ForEach is called
// are not visible to the iteration.
func (t *threadSafeMap[K, V]) ForEach(f func(key K, value V)) {
	t.mutex.RLock()
	// Create snapshot under read lock
	accum := make([]KeyValuePair[K, V], 0, t.internal.Size())

	for key, val := range t.internal.Seq() {
		accum = append(accum, KeyValuePair[K, V]{Key: key, Value: val})
	}
	t.mutex.RUnlock()

	// Execute function on snapshot without holding lock
	for _, kv := range accum {
		f(kv.Key, kv.Value)
	}
}

// ForAll tests whether a predicate holds for all key-value pairs in the map.
// Acquires a read lock only during snapshot creation to avoid holding the lock during callback execution.
// Returns true if the predicate returns true for all entries, false otherwise.
// The iteration is performed on a snapshot of the map.
func (t *threadSafeMap[K, V]) ForAll(predicate func(key K, value V) bool) bool {
	t.mutex.RLock()
	// Create snapshot under read lock
	accum := make([]KeyValuePair[K, V], 0, t.internal.Size())

	for key, val := range t.internal.Seq() {
		accum = append(accum, KeyValuePair[K, V]{Key: key, Value: val})
	}
	t.mutex.RUnlock()

	// Test predicate on snapshot without holding lock
	for _, kv := range accum {
		if !predicate(kv.Key, kv.Value) {
			return false
		}
	}

	return true
}

// Filter creates a new thread-safe map containing only key-value pairs for which the predicate returns true.
// Acquires a read lock on this map during the operation. The returned map is also thread-safe.
func (t *threadSafeMap[K, V]) Filter(predicate func(key K, value V) bool) Map[K, V] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return NewThreadSafeMap(t.internal.Filter(predicate))
}

// FilterNot creates a new thread-safe map containing only key-value pairs for which the predicate returns false.
// Acquires a read lock on this map during the operation. The returned map is also thread-safe.
func (t *threadSafeMap[K, V]) FilterNot(predicate func(key K, value V) bool) Map[K, V] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return NewThreadSafeMap(t.internal.FilterNot(predicate))
}

// Map transforms all key-value pairs in the map by applying the given function to each entry.
// Acquires a read lock on this map during the operation. The returned map is also thread-safe.
func (t *threadSafeMap[K, V]) Map(f func(key K, value V) (K, V)) Map[K, V] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return NewThreadSafeMap(t.internal.Map(f))
}

// FlatMap applies the given function to each key-value pair and flattens the results into a single map.
// Acquires a read lock on this map during the operation. The returned map is also thread-safe.
func (t *threadSafeMap[K, V]) FlatMap(f func(key K, value V) Map[K, V]) Map[K, V] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return NewThreadSafeMap(t.internal.FlatMap(f))
}

// Exists tests whether at least one key-value pair in the map satisfies the given predicate.
// Acquires a read lock only during snapshot creation to avoid holding the lock during callback execution.
// Returns true if the predicate returns true for any entry, false otherwise.
// The iteration is performed on a snapshot of the map.
func (t *threadSafeMap[K, V]) Exists(predicate func(key K, value V) bool) bool {
	t.mutex.RLock()
	// Create snapshot under read lock
	accum := make([]KeyValuePair[K, V], 0, t.internal.Size())

	for key, val := range t.internal.Seq() {
		accum = append(accum, KeyValuePair[K, V]{Key: key, Value: val})
	}
	t.mutex.RUnlock()

	// Test predicate on snapshot without holding lock
	for _, kv := range accum {
		if predicate(kv.Key, kv.Value) {
			return true
		}
	}

	return false
}

// FindFirst searches for the first key-value pair that satisfies the given predicate.
// Acquires a read lock only during snapshot creation to avoid holding the lock during callback execution.
// Returns Some(KeyValuePair) if a matching entry is found, None otherwise.
// The iteration is performed on a snapshot of the map.
func (t *threadSafeMap[K, V]) FindFirst(predicate func(key K, value V) bool) optional.Value[KeyValuePair[K, V]] {
	t.mutex.RLock()
	// Create snapshot under read lock
	accum := make([]KeyValuePair[K, V], 0, t.internal.Size())

	for key, val := range t.internal.Seq() {
		accum = append(accum, KeyValuePair[K, V]{Key: key, Value: val})
	}
	t.mutex.RUnlock()

	// Search in snapshot without holding lock
	for _, kv := range accum {
		if predicate(kv.Key, kv.Value) {
			return optional.Some(kv)
		}
	}

	return optional.None[KeyValuePair[K, V]]()
}
