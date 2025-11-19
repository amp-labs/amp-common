package lazy

import (
	"context"
	"sync"
	"sync/atomic"
)

// OfCtx is a lazy value that is initialized at most once.
type OfCtx[T any] struct {
	create      atomic.Pointer[func(context.Context) T]
	once        atomic.Pointer[sync.Once]
	value       atomic.Pointer[T]
	initialized atomic.Bool // Thread-safe flag to track initialization state
}

// Get returns the value (and initializes it if necessary).
func (t *OfCtx[T]) Get(ctx context.Context) T { //nolint:ireturn
	// Load the once value - initialize if needed
	once := t.once.Load()
	if once == nil {
		newOnce := &sync.Once{}
		if t.once.CompareAndSwap(nil, newOnce) {
			once = newOnce
		} else {
			once = t.once.Load()
		}
	}

	defer func() {
		if err := recover(); err != nil {
			// Reset the once state on panic so initialization can be retried
			t.once.Store(&sync.Once{})

			panic(err)
		}
	}()

	once.Do(func() {
		// Only initialize if create function is set
		createFn := t.create.Load()
		if createFn != nil {
			result := (*createFn)(ctx)
			// Mark as initialized and clear the create function
			t.value.Store(&result)
			t.initialized.Store(true)
			t.create.Store(nil)
		}
	})

	// Return the value - may be nil pointer if never initialized
	valPtr := t.value.Load()
	if valPtr != nil {
		return *valPtr
	}

	var zero T

	return zero
}

// Set lets you mutate the value. This is useful in some cases,
// but you should prefer the Get + callback pattern.
func (t *OfCtx[T]) Set(value T) {
	t.create.Store(nil)
	t.value.Store(&value)
	t.initialized.Store(true)
}

// Initialized returns true if the value has been initialized.
// This is useful for testing and debugging, but should never
// be part of the normal code flow.
func (t *OfCtx[T]) Initialized() bool {
	return t.initialized.Load()
}

// NewCtx creates a new lazy value. The callback will be called later, when the
// value is first accessed. The callback includes a context parameter.
func NewCtx[T any](f func(ctx context.Context) T) *OfCtx[T] {
	lazy := &OfCtx[T]{}
	lazy.create.Store(&f)

	return lazy
}
