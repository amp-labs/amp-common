package lazy

// This file contains context-based override functionality for lazy values, enabling
// dependency injection and testing scenarios. Named lazy values can be overridden
// with static values, provider functions, or error-returning providers.

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/google/uuid"
)

// contextKey is used to store lazy value overrides in context.
type contextKey string

// testKey is used to enable testing mode, which preserves create functions
// so that WithTestLocalCtx and WithTestLocalCtxErr can work correctly.
type testKey string

type preserveLifetimeKey string

// WithTestLocalCtx creates a test-local instance of a lazy value that shares the same
// initialization function but maintains separate state. This is useful for testing when
// you want to override a global lazy value with a fresh instance that will be initialized
// independently. The returned key can be used to access the test-local value via the context.
// Returns the key name and a getter function that retrieves the test-local value.
//
// Note: This function will panic if the lazy value's create function is nil.
func WithTestLocalCtx[T any](lazyValue *OfCtx[T]) (string, func(ctx context.Context) T) {
	createFn := lazyValue.create.Load()
	if createFn == nil || *createFn == nil {
		panic("createFn cannot be nil")
	}

	name := lazyValue.name
	if len(name) == 0 {
		name = contextKey(uuid.New().String())

		lazyValue.name = name
	}

	testLocalLazyValue := NewCtx[T](*createFn)

	return string(name), func(ctx context.Context) T {
		return testLocalLazyValue.Get(ctx)
	}
}

// WithTestLocalCtxErr creates a test-local instance of a lazy value (with error handling)
// that shares the same initialization function but maintains separate state. This is useful
// for testing when you want to override a global lazy value with a fresh instance that will
// be initialized independently. The returned key can be used to access the test-local value
// via the context. Returns the key name and a getter function that retrieves the test-local value.
//
// Note: This function will panic if the lazy value's create function is nil.
func WithTestLocalCtxErr[T any](lazyValue *OfCtxErr[T]) (string, func(ctx context.Context) (T, error)) {
	createFn := lazyValue.create.Load()
	if createFn == nil || *createFn == nil {
		panic("createFn cannot be nil")
	}

	name := lazyValue.name
	if len(name) == 0 {
		name = contextKey(uuid.New().String())

		lazyValue.name = name
	}

	testLocalLazyValue := NewCtxErr[T](*createFn)

	return string(name), func(ctx context.Context) (T, error) {
		return testLocalLazyValue.Get(ctx)
	}
}

// WithTestingEnabled enables or disables testing mode in the context. When testing mode is
// enabled, lazy values preserve their create functions even after initialization, allowing
// tools like WithTestLocalCtx and WithTestLocalCtxErr to work correctly. In normal (non-testing)
// mode, create functions are cleared after initialization to free memory.
func WithTestingEnabled(ctx context.Context, enabled bool) context.Context {
	return contexts.WithValue[testKey, bool](ctx, "testing-enabled", enabled)
}

// SetTestingEnabled configures the testing mode flag using a callback setter function.
// This is used with lazy value overrides to enable testing mode without directly
// manipulating a context. The setter function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - enabled: Whether to enable testing mode (preserves create functions after initialization)
//   - f: Callback function that stores the key-value pair. If nil, the function returns early.
func SetTestingEnabled(enabled bool, f func(key any, value any)) {
	if f == nil {
		return
	}

	f(testKey("testing-enabled"), enabled)
}

// WithLifecyclePreserved controls whether the context lifecycle is preserved when passed
// to lazy initialization functions. When enabled, the context is preserved as-is; when
// disabled (default), the context is wrapped to ignore lifecycle, preventing cancellation
// from affecting long-lived lazy values. This is useful when you want lazy initialization
// to respect context cancellation.
func WithLifecyclePreserved(ctx context.Context, preserveLifecycle bool) context.Context {
	return contexts.WithValue[preserveLifetimeKey, bool](ctx, "lifecycle-preserved", preserveLifecycle)
}

// SetLifecyclePreserved configures the lifecycle preservation flag using a callback setter function.
// This is used with lazy value overrides to control context lifecycle behavior without directly
// manipulating a context. The setter function is typically provided by lazy override mechanisms
// to store the value for later retrieval.
//
// Parameters:
//   - enabled: Whether to preserve context lifecycle (allows cancellation to affect initialization)
//   - f: Callback function that stores the key-value pair. If nil, the function returns early.
func SetLifecyclePreserved(enabled bool, f func(key any, value any)) {
	if f == nil {
		return
	}

	f(testKey("lifecycle-preserved"), enabled)
}

// WithValueOverride stores a value in the context that will override a named lazy value.
// When a lazy value with the matching name calls Get(), it will return this override value
// instead of performing lazy initialization. This is useful for testing and dependency injection.
// The key should match the name assigned to the lazy value via WithName().
func WithValueOverride[T any](ctx context.Context, key string, value T) context.Context {
	return contexts.WithValue[contextKey, T](ctx, contextKey(key), value)
}

// SetValueOverride configures a value override using a callback setter function.
// This is used with lazy value overrides to set override values without directly
// manipulating a context. The setter function is typically provided by lazy override
// mechanisms to store the key-value pair for later retrieval.
//
// Parameters:
//   - key: The name of the lazy value to override (must match the name set via WithName)
//   - value: The override value to use instead of lazy initialization
//   - f: Callback function that stores the key-value pair. If nil, the function returns early.
func SetValueOverride[T any](key string, value T, f func(any, any)) {
	if f == nil {
		return
	}

	f(contextKey(key), value)
}

// WithValueOverrideProvider stores a provider function in the context that will override
// a named lazy value. When a lazy value with the matching name calls Get(), it will invoke
// this provider function instead of performing lazy initialization. This is useful when the
// override value needs to be computed lazily or depends on the context.
// The key should match the name assigned to the lazy value via WithName().
func WithValueOverrideProvider[T any](
	ctx context.Context, key string, provider func(ctx context.Context) T,
) context.Context {
	return contexts.WithValue[contextKey, func(ctx context.Context) T](ctx, contextKey(key), provider)
}

// SetValueOverrideProvider configures a provider function override using a callback setter function.
// This is used with lazy value overrides to set override providers without directly
// manipulating a context. The setter function is typically provided by lazy override
// mechanisms to store the key-value pair for later retrieval.
//
// Parameters:
//   - key: The name of the lazy value to override (must match the name set via WithName)
//   - provider: Function that will be invoked to provide the override value
//   - f: Callback function that stores the key-value pair. If nil, the function returns early.
func SetValueOverrideProvider[T any](key string, provider func(ctx context.Context) T, f func(any, any)) {
	if f == nil {
		return
	}

	f(contextKey(key), provider)
}

// WithValueOverrideErrorProvider stores a provider function (that can return errors) in the
// context that will override a named lazy value. When a lazy value with the matching name calls
// Get(), it will invoke this provider function instead of performing lazy initialization.
// This is useful when the override value needs to be computed lazily and may fail.
// The key should match the name assigned to the lazy value via WithName().
func WithValueOverrideErrorProvider[T any](
	ctx context.Context, key string, provider func(ctx context.Context) (T, error),
) context.Context {
	return contexts.WithValue[contextKey, func(ctx context.Context) (T, error)](ctx, contextKey(key), provider)
}

// SetValueOverrideErrorProvider configures an error-returning provider function override using a callback
// setter function. This is used with lazy value overrides to set override providers (that can return errors)
// without directly manipulating a context. The setter function is typically provided by lazy override
// mechanisms to store the key-value pair for later retrieval.
//
// Parameters:
//   - key: The name of the lazy value to override (must match the name set via WithName)
//   - provider: Function that will be invoked to provide the override value (may return an error)
//   - f: Callback function that stores the key-value pair. If nil, the function returns early.
func SetValueOverrideErrorProvider[T any](key string, provider func(ctx context.Context) (T, error), f func(any, any)) {
	if f == nil {
		return
	}

	f(contextKey(key), provider)
}

// WithMultipleValues stores multiple override values in the context at once.
// This is a convenience function for setting up multiple overrides in a single call.
// The values can be static values, provider functions (func(ctx context.Context) T),
// or error-returning provider functions (func(ctx context.Context) (T, error)).
// Each value will override the corresponding named lazy value when its Get() is called.
// The keys should match the names assigned to lazy values via WithName().
func WithMultipleValues(ctx context.Context, values map[string]any) context.Context {
	vals := make(map[contextKey]any, len(values))

	for k, v := range values {
		vals[contextKey(k)] = v
	}

	return contexts.WithMultipleValues(ctx, vals)
}

// SetMultipleValues configures multiple value overrides using a callback setter function.
// This is a convenience function for setting multiple overrides at once without directly
// manipulating a context. The setter function is typically provided by lazy override
// mechanisms to store each key-value pair for later retrieval.
//
// Parameters:
//   - ctx: Context (currently unused, but kept for consistency with WithMultipleValues)
//   - values: Map of override names to values (can be static values, providers, or error-returning providers)
//   - f: Callback function that stores each key-value pair. If nil, the function returns early.
func SetMultipleValues(ctx context.Context, values map[string]any, f func(any, any)) {
	if f == nil {
		return
	}

	for k, v := range values {
		f(contextKey(k), v)
	}
}

// isTestingEnabled checks if testing mode is enabled in the context.
// Testing mode preserves create functions after initialization.
func isTestingEnabled(ctx context.Context) bool {
	value, found := contexts.GetValue[testKey, bool](ctx, "testing-enabled")

	return found && value
}

// isLifecyclePreserved checks if context lifecycle preservation is enabled.
// When enabled, contexts passed to lazy initialization functions maintain their
// original lifecycle, allowing cancellation to affect initialization.
func isLifecyclePreserved(ctx context.Context) bool {
	value, found := contexts.GetValue[preserveLifetimeKey, bool](ctx, "lifecycle-preserved")

	return found && value
}

// getSafeContext prepares a context for use in lazy initialization. By default, it wraps
// the context to ignore lifecycle (cancellation), ensuring that lazy values aren't affected
// by context cancellation. If lifecycle preservation is enabled via WithLifecyclePreserved,
// the context is preserved as-is, allowing cancellation to propagate to initialization.
func getSafeContext(ctx context.Context) context.Context {
	if isLifecyclePreserved(ctx) {
		return contexts.EnsureContext(ctx)
	}

	return contexts.EnsureContext(contexts.WithIgnoreLifecycle(ctx))
}
