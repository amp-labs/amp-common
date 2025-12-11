package lazy

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/amp-labs/amp-common/contexts"
)

// OfCtx is a lazy value that is initialized at most once.
// If a name is assigned via WithName(), the lazy value can be overridden
// via context using WithValueOverride or WithValueOverrideProvider.
type OfCtx[T any] struct {
	name        contextKey // Optional name for context-based overrides
	create      atomic.Pointer[func(context.Context) T]
	once        atomic.Pointer[sync.Once]
	value       atomic.Pointer[T]
	initialized atomic.Bool // Thread-safe flag to track initialization state
}

// WithName assigns a name to this lazy value, enabling context-based overrides.
// Once named, the value can be overridden using WithValueOverride or WithValueOverrideProvider
// with the same key. Returns the receiver for method chaining.
func (t *OfCtx[T]) WithName(name string) *OfCtx[T] {
	t.name = contextKey(name)

	return t
}

// Get returns the value (and initializes it if necessary).
// If this lazy value has a name (set via WithName), Get will first check the context
// for any overrides set via WithValueOverride or WithValueOverrideProvider. If an override
// is found, it is returned immediately without performing lazy initialization. This allows
// for dependency injection and testing scenarios where you want to substitute mock values.
// If no override is found (or if the lazy value has no name), normal lazy initialization occurs.
//
// Override precedence (highest to lowest):
//  1. Direct value via WithValueOverride
//  2. Provider function via WithValueOverrideProvider
//  3. Lazy initialization via the create function
func (t *OfCtx[T]) Get(ctx context.Context) T { //nolint:ireturn
	// Check for context-based overrides first (if this lazy value has a name)
	if len(t.name) > 0 {
		// Highest precedence: direct value override
		value, found := contexts.GetValue[contextKey, T](ctx, t.name)
		if found {
			return value
		}

		// Second precedence: provider function override
		provider, found := contexts.GetValue[contextKey, func(ctx context.Context) T](ctx, t.name)
		if found && provider != nil {
			return provider(getSafeContext(ctx))
		}
	}

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
			result := (*createFn)(getSafeContext(ctx))
			// Mark as initialized. The create function is normally cleared to free memory,
			// but in testing mode it's preserved so WithTestLocalCtx can create independent
			// test instances that share the same initialization function.
			t.value.Store(&result)
			t.initialized.Store(true)

			if !isTestingEnabled(ctx) {
				t.create.Store(nil)
			}
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

// Set lets you mutate the value directly, bypassing lazy initialization.
// This is useful in some cases (e.g., setting up test fixtures), but you should
// prefer the Get + callback pattern for normal usage.
// The context parameter is used to check if testing mode is enabled; in testing mode,
// the create function is preserved to allow WithTestLocalCtx to work correctly.
func (t *OfCtx[T]) Set(ctx context.Context, value T) {
	if !isTestingEnabled(ctx) {
		t.create.Store(nil)
	}

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
