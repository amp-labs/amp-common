package lazy

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/amp-labs/amp-common/contexts"
)

// OfCtxErr is a lazy value that is initialized at most once, but which might error out.
// If a name is assigned via WithName(), the lazy value can be overridden via context
// using WithValueOverride, WithValueOverrideProvider, or WithValueOverrideErrorProvider.
type OfCtxErr[T any] struct {
	name   contextKey // Optional name for context-based overrides
	create atomic.Pointer[func(ctx context.Context) (T, error)]
	value  atomic.Pointer[T]
	done   uint32
	m      sync.Mutex
}

// WithName assigns a name to this lazy value, enabling context-based overrides.
// Once named, the value can be overridden using WithValueOverride, WithValueOverrideProvider,
// or WithValueOverrideErrorProvider with the same key. Returns the receiver for method chaining.
func (t *OfCtxErr[T]) WithName(name string) *OfCtxErr[T] {
	t.name = contextKey(name)

	return t
}

// Get returns the value (and initializes it if necessary). If the initialization
// function returns an error, it will be returned by Get. Note that errors are
// NOT memoized, so if the initialization function returns an error, it will be
// invoked again on the next call to Get.
//
// If this lazy value has a name (set via WithName), Get will first check the context
// for any overrides set via WithValueOverride, WithValueOverrideProvider, or
// WithValueOverrideErrorProvider. If an override is found, it is returned immediately
// without performing lazy initialization. This allows for dependency injection and
// testing scenarios where you want to substitute mock values.
//
// Override precedence (highest to lowest):
//  1. Direct value via WithValueOverride
//  2. Non-error provider via WithValueOverrideProvider
//  3. Error-returning provider via WithValueOverrideErrorProvider
//  4. Lazy initialization via the create function
func (t *OfCtxErr[T]) Get(ctx context.Context) (T, error) { //nolint:ireturn
	// Check for context-based overrides first (if this lazy value has a name)
	if len(t.name) > 0 {
		// Highest precedence: direct value override
		value, found := contexts.GetValue[contextKey, T](ctx, t.name)
		if found {
			return value, nil
		}

		// Second precedence: non-error provider function
		providerA, found := contexts.GetValue[contextKey, func(ctx context.Context) T](ctx, t.name)
		if found && providerA != nil {
			return providerA(getSafeContext(ctx)), nil
		}

		// Third precedence: error-returning provider function
		providerB, found := contexts.GetValue[contextKey, func(ctx context.Context) (T, error)](ctx, t.name)
		if found && providerB != nil {
			return providerB(getSafeContext(ctx))
		}
	}

	createFn := t.create.Load()
	willTryToCreate := createFn != nil

	var errOut error

	defer func() {
		if err := recover(); err != nil {
			atomic.StoreUint32(&t.done, 0)

			panic(err)
		}

		// The initialization completed without panic or error. Clear the create function
		// to free memory, unless testing mode is enabled (which preserves the create
		// function so WithTestLocalCtxErr can create independent test instances).
		if willTryToCreate && errOut == nil && !isTestingEnabled(ctx) {
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

			value, err := (*fn)(getSafeContext(ctx))
			if err != nil {
				return err
			}

			t.value.Store(&value)

			// Clear create function to free memory, unless testing mode is enabled
			if !isTestingEnabled(ctx) {
				t.create.Store(nil)
			}

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

// Set lets you mutate the value directly, bypassing lazy initialization.
// This is useful in some cases (e.g., setting up test fixtures), but you should
// prefer the Get + callback pattern for normal usage.
// The context parameter is used to check if testing mode is enabled; in testing mode,
// the create function is preserved to allow WithTestLocalCtxErr to work correctly.
func (t *OfCtxErr[T]) Set(ctx context.Context, value T) {
	atomic.StoreUint32(&t.done, 1)

	if !isTestingEnabled(ctx) {
		t.create.Store(nil)
	}

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

// NewCtxErr creates a new lazy value that can return an error during initialization.
// The callback will be called later, when the value is first accessed. The callback
// includes a context parameter and can return an error. If an error is returned,
// the value is NOT memoized, and the callback will be invoked again on the next Get call.
func NewCtxErr[T any](f func(ctx context.Context) (T, error)) *OfCtxErr[T] {
	lazy := &OfCtxErr[T]{}
	lazy.create.Store(&f)

	return lazy
}
