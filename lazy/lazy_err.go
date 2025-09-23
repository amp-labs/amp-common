package lazy

import (
	"sync"
	"sync/atomic"
)

// OfErr is a lazy value that is initialized at most once, but which might error out.
type OfErr[T any] struct {
	create func() (T, error)
	value  T

	done uint32
	m    sync.Mutex
}

// Get returns the value (and initializes it if necessary). If the initialization
// function returns an error, it will be returned by Get. Note that errors are
// NOT memoized, so if the initialization function returns an error, it will be
// invoked again on the next call to Get.
func (t *OfErr[T]) Get() (T, error) { //nolint:ireturn
	willTryToCreate := t.create != nil

	var errOut error

	defer func() {
		if err := recover(); err != nil {
			atomic.StoreUint32(&t.done, 0)

			panic(err)
		}

		// The function is done, it didn't panic, it didn't return
		// an error. We can now safely mark the value as memoized.
		if willTryToCreate && errOut == nil {
			t.create = nil
		}
	}()

	if willTryToCreate {
		errOut = t.doOrError(func() error {
			value, err := t.create()
			if err != nil {
				return err
			}

			t.value = value
			t.create = nil

			return nil
		})
		if errOut != nil {
			var zero T

			return zero, errOut
		}
	}

	return t.value, nil
}

// Set lets you mutate the value. This is useful in some cases,
// but you should prefer the Get + callback pattern.
func (t *OfErr[T]) Set(value T) {
	atomic.StoreUint32(&t.done, 1)
	t.create = nil
	t.value = value
}

// Initialized returns true if the value has been initialized.
// This is useful for testing and debugging, but should never
// be part of the normal code flow.
func (t *OfErr[T]) Initialized() bool {
	return t.create == nil
}

func (t *OfErr[T]) doOrError(f func() error) error {
	if atomic.LoadUint32(&t.done) == 0 {
		// Outlined slow-path to allow inlining of the fast-path.
		return t.doSlowOrError(f)
	}

	return nil
}

func (t *OfErr[T]) doSlowOrError(f func() error) error {
	t.m.Lock()
	defer t.m.Unlock()

	if t.done == 0 {
		if err := f(); err != nil {
			return err
		}

		// The callback ran without error, now we can call it initialized
		atomic.StoreUint32(&t.done, 1)
	}

	return nil
}

func NewErr[T any](f func() (T, error)) *OfErr[T] {
	return &OfErr[T]{create: f}
}
