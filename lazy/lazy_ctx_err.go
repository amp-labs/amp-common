package lazy

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/amp-labs/amp-common/contexts"
)

// OfCtxErr is a lazy value that is initialized at most once, but which might error out.
type OfCtxErr[T any] struct {
	create atomic.Pointer[func(ctx context.Context) (T, error)]
	value  atomic.Pointer[T]

	done uint32
	m    sync.Mutex
}

// Get returns the value (and initializes it if necessary). If the initialization
// function returns an error, it will be returned by Get. Note that errors are
// NOT memoized, so if the initialization function returns an error, it will be
// invoked again on the next call to Get.
func (t *OfCtxErr[T]) Get(ctx context.Context) (T, error) { //nolint:ireturn
	createFn := t.create.Load()
	willTryToCreate := createFn != nil

	var errOut error

	defer func() {
		if err := recover(); err != nil {
			atomic.StoreUint32(&t.done, 0)

			panic(err)
		}

		// The function is done, it didn't panic, it didn't return
		// an error. We can now safely mark the value as memoized.
		if willTryToCreate && errOut == nil {
			t.create.Store(nil)
		}
	}()

	if willTryToCreate {
		errOut = t.doOrError(func() error {
			// Re-load create function inside lock to avoid race
			fn := t.create.Load()
			if fn == nil {
				return nil
			}

			value, err := (*fn)(contexts.WithIgnoreLifecycle(ctx))
			if err != nil {
				return err
			}

			t.value.Store(&value)
			t.create.Store(nil)

			return nil
		})
		if errOut != nil {
			var zero T

			return zero, errOut
		}
	}

	valPtr := t.value.Load()
	if valPtr != nil {
		return *valPtr, nil
	}

	var zero T

	return zero, nil
}

// Set lets you mutate the value. This is useful in some cases,
// but you should prefer the Get + callback pattern.
func (t *OfCtxErr[T]) Set(value T) {
	atomic.StoreUint32(&t.done, 1)
	t.create.Store(nil)
	t.value.Store(&value)
}

// Initialized returns true if the value has been initialized.
// This is useful for testing and debugging, but should never
// be part of the normal code flow.
func (t *OfCtxErr[T]) Initialized() bool {
	return t.create.Load() == nil
}

func (t *OfCtxErr[T]) doOrError(f func() error) error {
	if atomic.LoadUint32(&t.done) == 0 {
		// Outlined slow-path to allow inlining of the fast-path.
		return t.doSlowOrError(f)
	}

	return nil
}

func (t *OfCtxErr[T]) doSlowOrError(f func() error) error {
	t.m.Lock()
	defer t.m.Unlock()

	if t.done == 0 {
		err := f()
		if err != nil {
			return err
		}

		// The callback ran without error, now we can call it initialized
		atomic.StoreUint32(&t.done, 1)
	}

	return nil
}

func NewCtxErr[T any](f func(ctx context.Context) (T, error)) *OfCtxErr[T] {
	lazy := &OfCtxErr[T]{}
	lazy.create.Store(&f)

	return lazy
}
