package maps

import (
	"iter"
	"sync"

	"github.com/amp-labs/amp-common/collectable"
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
func NewThreadSafeMap[K collectable.Collectable[K], V any](m Map[K, V]) Map[K, V] {
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
type threadSafeMap[K collectable.Collectable[K], V any] struct {
	mutex    sync.RWMutex // Protects access to internal map
	internal Map[K, V]    // Underlying map implementation
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
