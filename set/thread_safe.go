package set

import (
	"iter"
	"sync"

	"github.com/amp-labs/amp-common/collectable"
)

// NewThreadSafeSet wraps an existing Set implementation with thread-safe access using sync.RWMutex.
// It provides concurrent read/write access to the underlying set while preserving the Set interface.
//
// The wrapper uses read-write locks to allow multiple concurrent readers or exclusive writer access.
// Write operations (Add, AddAll, Remove, Clear) acquire exclusive locks, while read operations
// (Contains, Size, Entries, Seq, Union, Intersection) use shared read locks for better concurrency.
//
// Example usage:
//
//	unsafeSet := set.NewSet[string](hashing.Sha256)
//	safeSet := set.NewThreadSafeSet(unsafeSet)
//	safeSet.Add("element") // thread-safe
func NewThreadSafeSet[T collectable.Collectable[T]](s Set[T]) Set[T] {
	if s == nil {
		return nil
	}

	tss, ok := s.(*threadSafeSet[T])
	if ok {
		// Already thread-safe, return as-is
		return tss
	}

	return &threadSafeSet[T]{
		internal: s,
	}
}

// threadSafeSet is a decorator that wraps any Set implementation with thread-safe access.
// It uses sync.RWMutex to coordinate concurrent access, allowing multiple simultaneous
// readers or a single exclusive writer.
type threadSafeSet[T collectable.Collectable[T]] struct {
	mutex    sync.RWMutex // Protects access to internal set
	internal Set[T]       // Underlying set implementation
}

// AddAll adds multiple elements to the set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeSet[T]) AddAll(elements ...T) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.AddAll(elements...)
}

// Add adds a single element to the set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeSet[T]) Add(element T) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.Add(element)
}

// Remove removes an element from the set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeSet[T]) Remove(element T) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.Remove(element)
}

// Clear removes all elements from the set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeSet[T]) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.internal.Clear()
}

// Contains checks if an element exists in the set with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Contains calls without blocking each other.
func (t *threadSafeSet[T]) Contains(element T) (bool, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Contains(element)
}

// Size returns the number of elements in the set with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Size calls without blocking each other.
func (t *threadSafeSet[T]) Size() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Size()
}

// Entries returns all elements in the set with shared read lock protection.
// Acquires a read lock during the operation to ensure a consistent snapshot.
func (t *threadSafeSet[T]) Entries() []T {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Entries()
}

// Seq returns an iterator over the set's elements with snapshot semantics.
// Acquires a read lock only during the snapshot creation, then releases it before returning.
//
// Implementation note: This creates a complete snapshot of the set under a read lock,
// then returns an iterator over that snapshot. This design ensures:
//   - The read lock is not held during iteration (preventing long-lived lock holding)
//   - Iteration sees a consistent view of the set at the time Seq() was called
//   - Multiple goroutines can iterate concurrently without blocking each other
//   - Changes made after Seq() is called are not visible to the iterator
//
// Trade-off: Uses O(n) memory to store the snapshot, but provides better concurrency
// characteristics than holding the lock during iteration.
func (t *threadSafeSet[T]) Seq() iter.Seq[T] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Create snapshot under read lock
	accum := t.internal.Entries()

	// Return iterator over snapshot (no lock held during iteration)
	return func(yield func(T) bool) {
		for _, element := range accum {
			if !yield(element) {
				return
			}
		}
	}
}

// Union creates a new thread-safe set containing all elements from both this set and another.
// Acquires a read lock on this set during the operation. The returned set is also thread-safe.
//
// Note: The 'other' set is accessed directly without coordination. If 'other' is also being
// modified concurrently, the caller should handle that synchronization externally.
func (t *threadSafeSet[T]) Union(other Set[T]) (Set[T], error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	value, err := t.internal.Union(other)
	if err != nil {
		return nil, err
	}

	return NewThreadSafeSet(value), nil
}

// Intersection creates a new thread-safe set containing only elements present in both sets.
// Acquires a read lock on this set during the operation. The returned set is also thread-safe.
//
// Note: The 'other' set is accessed directly without coordination. If 'other' is also being
// modified concurrently, the caller should handle that synchronization externally.
func (t *threadSafeSet[T]) Intersection(other Set[T]) (Set[T], error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	value, err := t.internal.Intersection(other)
	if err != nil {
		return nil, err
	}

	return NewThreadSafeSet(value), nil
}

// NewThreadSafeOrderedSet wraps an existing OrderedSet implementation with thread-safe access using sync.RWMutex.
// It provides concurrent read/write access to the underlying ordered set while preserving the OrderedSet interface.
//
// The wrapper uses read-write locks to allow multiple concurrent readers or exclusive writer access.
// Write operations (Add, AddAll, Remove, Clear) acquire exclusive locks, while read operations
// (Contains, Size, Entries, Seq, Union, Intersection) use shared read locks for better concurrency.
//
// Example usage:
//
//	unsafeSet := set.NewOrderedSet[string](hashing.Sha256)
//	safeSet := set.NewThreadSafeOrderedSet(unsafeSet)
//	safeSet.Add("element") // thread-safe
func NewThreadSafeOrderedSet[T collectable.Collectable[T]](s OrderedSet[T]) OrderedSet[T] {
	if s == nil {
		return nil
	}

	tsos, ok := s.(*threadSafeOrderedSet[T])
	if ok {
		// Already thread-safe, return as-is
		return tsos
	}

	return &threadSafeOrderedSet[T]{
		internal: s,
	}
}

// threadSafeOrderedSet is a decorator that wraps any OrderedSet implementation with thread-safe access.
// It uses sync.RWMutex to coordinate concurrent access, allowing multiple simultaneous
// readers or a single exclusive writer.
type threadSafeOrderedSet[T collectable.Collectable[T]] struct {
	mutex    sync.RWMutex  // Protects access to internal ordered set
	internal OrderedSet[T] // Underlying ordered set implementation
}

// AddAll adds multiple elements to the ordered set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeOrderedSet[T]) AddAll(elements ...T) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.AddAll(elements...)
}

// Add adds a single element to the ordered set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeOrderedSet[T]) Add(element T) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.Add(element)
}

// Remove removes an element from the ordered set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeOrderedSet[T]) Remove(element T) error {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	return t.internal.Remove(element)
}

// Clear removes all elements from the ordered set with exclusive lock protection.
// Acquires a write lock to ensure no other goroutines can read or write during the operation.
func (t *threadSafeOrderedSet[T]) Clear() {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	t.internal.Clear()
}

// Contains checks if an element exists in the ordered set with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Contains calls without blocking each other.
func (t *threadSafeOrderedSet[T]) Contains(element T) (bool, error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Contains(element)
}

// Size returns the number of elements in the ordered set with shared read lock protection.
// Acquires a read lock, allowing multiple concurrent Size calls without blocking each other.
func (t *threadSafeOrderedSet[T]) Size() int {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Size()
}

// Entries returns all elements in the ordered set in insertion order with shared read lock protection.
// Acquires a read lock during the operation to ensure a consistent snapshot.
func (t *threadSafeOrderedSet[T]) Entries() []T {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	return t.internal.Entries()
}

// Seq returns an iterator over the ordered set's elements in insertion order with snapshot semantics.
// Acquires a read lock only during the snapshot creation, then releases it before returning.
//
// Implementation note: This creates a complete snapshot of the ordered set under a read lock,
// then returns an iterator over that snapshot. This design ensures:
//   - The read lock is not held during iteration (preventing long-lived lock holding)
//   - Iteration sees a consistent view of the set at the time Seq() was called
//   - Multiple goroutines can iterate concurrently without blocking each other
//   - Changes made after Seq() is called are not visible to the iterator
//
// Trade-off: Uses O(n) memory to store the snapshot, but provides better concurrency
// characteristics than holding the lock during iteration.
func (t *threadSafeOrderedSet[T]) Seq() iter.Seq2[int, T] {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	// Create snapshot under read lock
	accum := t.internal.Entries()

	// Return iterator over snapshot (no lock held during iteration)
	return func(yield func(int, T) bool) {
		for i, element := range accum {
			if !yield(i, element) {
				return
			}
		}
	}
}

// Union creates a new thread-safe ordered set containing all elements from both sets.
// Acquires a read lock on this set during the operation. The returned set is also thread-safe.
// Elements from the current set appear first in insertion order, followed by elements from
// the other set that are not already present.
//
// Note: The 'other' set is accessed directly without coordination. If 'other' is also being
// modified concurrently, the caller should handle that synchronization externally.
func (t *threadSafeOrderedSet[T]) Union(other OrderedSet[T]) (OrderedSet[T], error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	value, err := t.internal.Union(other)
	if err != nil {
		return nil, err
	}

	return NewThreadSafeOrderedSet(value), nil
}

// Intersection creates a new thread-safe ordered set containing only elements present in both sets.
// Acquires a read lock on this set during the operation. The returned set is also thread-safe.
// The order is preserved from the current set.
//
// Note: The 'other' set is accessed directly without coordination. If 'other' is also being
// modified concurrently, the caller should handle that synchronization externally.
func (t *threadSafeOrderedSet[T]) Intersection(other OrderedSet[T]) (OrderedSet[T], error) {
	t.mutex.RLock()
	defer t.mutex.RUnlock()

	value, err := t.internal.Intersection(other)
	if err != nil {
		return nil, err
	}

	return NewThreadSafeOrderedSet(value), nil
}

// NewThreadSafeStringSet wraps an existing StringSet with thread-safe access.
// Returns a StringSet that can be safely used from multiple goroutines.
func NewThreadSafeStringSet(s *StringSet) *StringSet {
	if s == nil {
		return nil
	}

	return &StringSet{
		hash: s.hash,
		set:  NewThreadSafeSet(s.set),
	}
}

// NewThreadSafeStringOrderedSet wraps an existing StringOrderedSet with thread-safe access.
// Returns a StringOrderedSet that can be safely used from multiple goroutines.
func NewThreadSafeStringOrderedSet(s *StringOrderedSet) *StringOrderedSet {
	if s == nil {
		return nil
	}

	return &StringOrderedSet{
		hash: s.hash,
		set:  NewThreadSafeOrderedSet(s.set),
	}
}
