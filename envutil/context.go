package envutil

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
)

// envContextKey is a custom type used as the key for storing environment variable
// overrides in a context. Using a custom type prevents collisions with other context
// values and ensures type safety when retrieving values.
type envContextKey string

// WithEnvOverride returns a new context with a single environment variable override.
// This allows you to override environment variables for a specific operation without
// modifying the actual process environment.
//
// When envutil Reader functions (String, Int, Bool, etc.) are called with this context,
// they will first check for overrides in the context before reading from the actual
// environment.
//
// Example:
//
//	ctx := envutil.WithEnvOverride(context.Background(), "PORT", "9090")
//	port := envutil.IntCtx(ctx, "PORT", envutil.Default(8080)).Value()
//	// port will be 9090, even if PORT=8080 in the actual environment
//
// This is particularly useful for:
//   - Testing: Override environment variables without affecting other tests
//   - Request-scoped configuration: Different handlers can use different values
//   - Multi-tenant scenarios: Different tenants can have different configuration
func WithEnvOverride(ctx context.Context, key string, value string) context.Context {
	return contexts.WithValue[envContextKey, string](contexts.EnsureContext(ctx), envContextKey(key), value)
}

// SetEnvOverride configures a single environment variable override using a callback setter function.
// This is used with lazy value overrides to set environment variable overrides without directly
// manipulating a context. The set function is typically provided by lazy override mechanisms
// (e.g., lazy.SetValueOverride) to store the value for later retrieval.
//
// Parameters:
//   - key: The environment variable name to override
//   - value: The override value to use instead of the actual environment variable
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
//
// This function is typically used in conjunction with lazy value override systems
// where context values need to be configured before a context is created.
func SetEnvOverride(key string, value string, set func(any, any)) {
	if set == nil {
		return
	}

	set(envContextKey(key), value)
}

// WithEnvOverrides returns a new context with multiple environment variable overrides.
// This is a more efficient version of calling WithEnvOverride multiple times, as it
// stores all overrides in a single context operation.
//
// If the provided values map is empty, the original context is returned unchanged
// to avoid unnecessary context allocations.
//
// Example:
//
//	ctx := envutil.WithEnvOverrides(context.Background(), map[string]string{
//		"DATABASE_URL": "postgres://localhost/test",
//		"PORT":         "9090",
//		"LOG_LEVEL":    "debug",
//	})
//	// All envutil Reader functions will check these overrides first
//
// This is particularly useful for:
//   - Testing: Set up a complete test environment in one call
//   - Configuration injection: Pass environment-specific config through context
//   - Batch operations: Override multiple settings for a group of operations
func WithEnvOverrides(ctx context.Context, values map[string]string) context.Context {
	if len(values) == 0 {
		return ctx
	}

	vals := make(map[envContextKey]any, len(values))
	for k, v := range values {
		vals[envContextKey(k)] = v
	}

	return contexts.WithMultipleValues[envContextKey](contexts.EnsureContext(ctx), vals)
}

// SetEnvOverrides configures multiple environment variable overrides using a callback setter function.
// This is a more efficient version of calling SetEnvOverride multiple times. The set function is
// typically provided by lazy override mechanisms to store each key-value pair for later retrieval.
//
// Parameters:
//   - values: Map of environment variable names to override values
//   - set: Callback function that stores each key-value pair. If nil, the function returns early.
//
// This function is typically used in conjunction with lazy value override systems
// where context values need to be configured before a context is created.
func SetEnvOverrides(values map[string]string, set func(any, any)) {
	if set == nil {
		return
	}

	for k, v := range values {
		set(envContextKey(k), v)
	}
}

// getEnvOverride retrieves an environment variable override from the context.
// It returns the override value and true if found, or an empty string and false
// if no override exists for the given key.
//
// This is an internal function used by envutil Reader methods to check for
// context-based overrides before falling back to the actual environment.
// The lookup order in Reader methods is:
//  1. Check context for override (using this function)
//  2. Check actual environment (using os.Getenv)
//  3. Use default value if provided
//  4. Return error if required and not found
func getEnvOverride(ctx context.Context, key string) (string, bool) {
	return contexts.GetValue[envContextKey, string](ctx, envContextKey(key))
}
