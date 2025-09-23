package lazy

import "sync"

// Of is a lazy value that is initialized at most once.
type Of[T any] struct {
	create func() T
	once   sync.Once
	value  T
}

// Get returns the value (and initializes it if necessary).
func (t *Of[T]) Get() T { //nolint:ireturn
	willTryToCreate := t.create != nil

	defer func() {
		if err := recover(); err != nil {
			t.once = sync.Once{} // reset the once state

			panic(err)
		}

		if willTryToCreate {
			t.create = nil
		}
	}()

	if willTryToCreate {
		t.once.Do(func() {
			t.value = t.create()
		})
	}

	return t.value
}

// Set lets you mutate the value. This is useful in some cases,
// but you should prefer the Get + callback pattern.
func (t *Of[T]) Set(value T) {
	t.create = nil
	t.value = value
}

// Initialized returns true if the value has been initialized.
// This is useful for testing and debugging, but should never
// be part of the normal code flow.
func (t *Of[T]) Initialized() bool {
	return t.create == nil
}

// New creates a new lazy value. The callback will be called later, when the
// value is first accessed.
func New[T any](f func() T) *Of[T] {
	return &Of[T]{create: f}
}
