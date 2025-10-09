package lazy

import (
	"sync"
	"sync/atomic"
)

// Of is a lazy value that is initialized at most once.
type Of[T any] struct {
	create      func() T
	once        sync.Once
	value       T
	initialized atomic.Bool // Thread-safe flag to track initialization state
}

// Get returns the value (and initializes it if necessary).
func (t *Of[T]) Get() T { //nolint:ireturn
	defer func() {
		if err := recover(); err != nil {
			// Reset the once state on panic so initialization can be retried
			t.once = sync.Once{}

			panic(err)
		}
	}()

	t.once.Do(func() {
		// Only initialize if create function is set
		if t.create != nil {
			t.value = t.create()
			// Mark as initialized and clear the create function
			t.initialized.Store(true)
			t.create = nil
		}
	})

	return t.value
}

// Set lets you mutate the value. This is useful in some cases,
// but you should prefer the Get + callback pattern.
func (t *Of[T]) Set(value T) {
	t.create = nil
	t.value = value
	t.initialized.Store(true)
}

// Initialized returns true if the value has been initialized.
// This is useful for testing and debugging, but should never
// be part of the normal code flow.
func (t *Of[T]) Initialized() bool {
	return t.initialized.Load()
}

// New creates a new lazy value. The callback will be called later, when the
// value is first accessed.
func New[T any](f func() T) *Of[T] {
	return &Of[T]{create: f}
}
