package envutil

import (
	"context"
	"fmt"
	"maps"
	"os"
	"strings"
)

// Loader provides an isolated, mutable collection of environment variables.
// Unlike directly modifying os.Setenv, a Loader maintains its own environment
// map that can be manipulated independently of the process environment.
//
// The Loader supports multiple workflows:
//
//  1. Loading from files:
//     loader := envutil.NewLoader()
//     loader.LoadFile(".env")                    // Load base configuration
//     loader.LoadFile(".env.production")         // Override with environment-specific values
//
//  2. Programmatic manipulation:
//     loader.Set("DATABASE_URL", "postgres://localhost/test")
//     loader.Delete("LEGACY_CONFIG")
//     if loader.Contains("FEATURE_FLAG") {
//     // feature is enabled
//     }
//
//  3. Context integration:
//     ctx := loader.EnhanceContext(context.Background())
//     // All envutil Reader functions will check this context for overrides
//     port := envutil.IntCtx(ctx, "PORT", envutil.Default(8080)).Value()
//
//  4. Exporting to other systems:
//     cmd := exec.Command("my-app")
//     cmd.Env = loader.AsSlice()  // Pass environment to subprocess
//
// Thread-safety: Loader is NOT thread-safe. If you need concurrent access,
// you must synchronize access using a mutex or create separate Loader instances.
//
// The loader maintains an internal map of environment variables that can be
// modified using Set, Delete, and Load. The loader does not modify the actual
// process environment (os.Setenv is never called).
type Loader struct {
	environment map[string]string
}

// NewLoader creates a new empty Loader with no environment variables.
// The loader starts with an empty environment map, allowing you to build up
// your configuration from scratch by loading files or setting values programmatically.
//
// Unlike the previous behavior, NewLoader does NOT automatically load the process
// environment. Use LoadEnv() if you want to include the current process environment,
// or use LoadFile() to load from configuration files.
//
// Example usage:
//
//	// Create an empty loader
//	ldr := envutil.NewLoader()
//
//	// Option 1: Load from files only (no process env)
//	ldr.LoadFile(".env")
//	ldr.LoadFile(".env.production")
//
//	// Option 2: Start with process env, then layer files
//	ldr.LoadEnv()  // Load current process environment first
//	ldr.LoadFile(".env.local")  // Override with local config
//
//	// Option 3: Build environment programmatically
//	ldr.Set("PORT", "8080")
//	ldr.Set("DATABASE_URL", "postgres://localhost/test")
//
//	// Use the environment in a context
//	ctx := ldr.EnhanceContext(context.Background())
//
// Common use cases:
//   - Testing: Build isolated test environments without process env pollution
//   - Configuration layering: Explicitly control which sources to load and their priority
//   - Clean slate: Start fresh without inheriting unwanted process variables
//   - Programmatic config: Build configuration entirely in code
func NewLoader() *Loader {
	return &Loader{
		environment: make(map[string]string),
	}
}

// LoadEnv loads all environment variables from the current process into the loader.
// This captures a snapshot of the process environment at the time of the call,
// merging it into the loader's existing environment. Variables from the process
// environment will override any existing variables with the same key in the loader.
//
// This method is useful when you want to include the process environment as part
// of your configuration, either as a base layer or to merge with file-based config.
//
// Example usage:
//
//	// Start with process environment, then override with files
//	loader := envutil.NewLoader()
//	loader.LoadEnv()                    // Base: current process environment
//	loader.LoadFile(".env.local")       // Override with local config
//	loader.LoadFile(".env.production")  // Override with production config
//
//	// Modify specific values
//	loader.Set("DEBUG_MODE", "true")
//
// Common use cases:
//   - Baseline configuration: Use process env as a starting point
//   - Merging sources: Combine process env with file-based config
//   - Snapshot isolation: Capture process env at a specific point in time
//   - Testing: Load test environment without polluting process env
//
// Important notes:
//   - This does NOT modify the actual process environment (os.Setenv is not called)
//   - Variables are loaded once at call time; changes to process env afterward won't be reflected
//   - If called multiple times, each call will re-snapshot and override existing values
//   - Lines without '=' are silently ignored (malformed environment entries)
func (l *Loader) LoadEnv() {
	for _, line := range os.Environ() {
		// Split on first equals sign to allow values to contain '='
		parts := strings.SplitN(line, "=", 2) //nolint:mnd // 2 is required for SplitN to split on first '='
		if len(parts) != 2 {                  //nolint:mnd // len must be 2 for valid KEY=VALUE format
			continue
		}

		l.environment[parts[0]] = parts[1]
	}
}

// LoadFile reads environment variables from a file and merges them into the loader.
// The file format is automatically detected based on the file extension:
//   - .env files: KEY=VALUE pairs (one per line)
//   - .json files: Must have an "env" field with string key-value pairs
//   - .yml/.yaml files: Must have an "env" field with string key-value pairs
//
// Variables loaded from the file will override any existing variables with the same key
// in the loader's environment. This allows you to layer configurations by calling Load
// multiple times with different files:
//
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")                 // Base configuration
//	loader.LoadFile(".env.local")           // Local overrides (higher priority)
//	loader.LoadFile(".env.production")      // Production-specific overrides (highest priority)
//
// The method returns the number of variables loaded from the file and any error encountered.
// If the file doesn't exist, can't be read, or has an invalid format, an error is returned
// and the loader's state remains unchanged.
//
// Example .env file:
//
//	# Database configuration
//	DATABASE_URL=postgres://localhost/myapp
//	DB_POOL_SIZE=10
//
// Example JSON file:
//
//	{
//	  "env": {
//	    "DATABASE_URL": "postgres://localhost/myapp",
//	    "DB_POOL_SIZE": "10"
//	  }
//	}
//
// Returns:
//   - count: Number of environment variables loaded from the file
//   - error: Non-nil if the file cannot be read or parsed
//
// See LoadEnvFile for more details on supported file formats and parsing behavior.
func (l *Loader) LoadFile(filename string) (int64, error) {
	env, err := LoadEnvFile(filename)
	if err != nil {
		return 0, err
	}

	num := len(env)

	// Merge loaded variables into loader's environment (overwriting existing keys)
	maps.Copy(l.environment, env)

	return int64(num), nil
}

// Get retrieves the value of an environment variable from the loader.
// Returns the value and true if the key exists, or an empty string and false if not found.
//
// Example usage:
//
//	value, found := loader.Get("DATABASE_URL")
//	if !found {
//	    return errors.New("DATABASE_URL not configured")
//	}
//	fmt.Printf("Database: %s\n", value)
//
// Note: Unlike os.Getenv, this only checks the loader's internal environment,
// not the actual process environment. To check both, use the envutil Reader functions
// with a context created by EnhanceContext:
//
//	ctx := loader.EnhanceContext(context.Background())
//	dbURL := envutil.StringCtx(ctx, "DATABASE_URL", envutil.Required()).ValueOrFatal()
//
// Returns:
//   - value: The environment variable value if found, empty string otherwise
//   - found: True if the key exists in the loader's environment, false otherwise
func (l *Loader) Get(key string) (string, bool) {
	val, found := l.environment[key]

	return val, found
}

// Set adds or updates an environment variable in the loader.
// If the key already exists, its value is replaced. If the key doesn't exist,
// it is added to the loader's environment.
//
// This method does NOT modify the actual process environment (os.Setenv is not called).
// The change only affects this loader instance and any contexts created from it.
//
// Example usage:
//
//	loader := envutil.NewLoader()
//
//	// Set configuration values
//	loader.Set("PORT", "8080")
//	loader.Set("LOG_LEVEL", "debug")
//	loader.Set("FEATURE_XYZ_ENABLED", "true")
//
//	// Override existing values
//	loader.Set("DATABASE_URL", "postgres://localhost/test")  // Overrides value from .env
//
// Common use cases:
//   - Testing: Override specific variables for test scenarios
//   - Dynamic configuration: Set values based on runtime conditions
//   - Configuration templating: Start with base config and customize
//   - Feature flags: Enable/disable features programmatically
func (l *Loader) Set(key string, value string) {
	l.environment[key] = value
}

// SetAll adds or updates multiple environment variables in the loader from a map.
// This is a convenience method for setting multiple variables at once, equivalent to
// calling Set() for each key-value pair in the map.
//
// Variables in the map will override any existing variables with the same key in the
// loader's environment. This method does NOT modify the actual process environment.
//
// Example usage:
//
//	loader := envutil.NewLoader()
//
//	// Set multiple configuration values at once
//	config := map[string]string{
//	    "DATABASE_URL":    "postgres://localhost/myapp",
//	    "REDIS_URL":       "redis://localhost:6379",
//	    "PORT":            "8080",
//	    "LOG_LEVEL":       "debug",
//	    "FEATURE_X_ENABLED": "true",
//	}
//	loader.SetAll(config)
//
//	// Merge with existing environment
//	loader.LoadEnv()  // Load process environment first
//	overrides := map[string]string{
//	    "DATABASE_URL": "postgres://localhost/test",  // Override for testing
//	    "LOG_LEVEL":    "error",                      // Override log level
//	}
//	loader.SetAll(overrides)
//
// Common use cases:
//   - Bulk configuration: Set multiple related variables at once
//   - Testing: Apply a complete test configuration in one call
//   - Merging configs: Combine configurations from different sources
//   - Dynamic configuration: Apply runtime-computed configuration maps
//
// Note: The order of iteration over the map is not guaranteed (Go map iteration is random).
// If you need deterministic ordering, set variables individually using Set() in a specific order.
func (l *Loader) SetAll(env map[string]string) {
	for k, v := range env {
		l.Set(k, v)
	}
}

// Delete removes an environment variable from the loader.
// If the key doesn't exist, this is a no-op (no error is returned).
//
// This method does NOT modify the actual process environment. The deletion
// only affects this loader instance and any contexts created from it.
//
// Example usage:
//
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")
//
//	// Remove sensitive variables before passing to subprocess
//	loader.Delete("SECRET_KEY")
//	loader.Delete("API_TOKEN")
//
//	cmd := exec.Command("./worker")
//	cmd.Env = loader.AsSlice()  // Worker won't have access to secrets
//
// Common use cases:
//   - Security: Remove sensitive variables before export
//   - Testing: Remove variables to test default behavior
//   - Cleanup: Remove legacy or deprecated configuration
//   - Isolation: Prevent specific variables from being inherited
func (l *Loader) Delete(key string) {
	delete(l.environment, key)
}

// Clear removes all environment variables from the loader, resetting it to an empty state.
// After calling Clear, the loader will contain no environment variables until new ones
// are loaded or set.
//
// This method does NOT modify the actual process environment. The clearing only affects
// this loader instance and any contexts created from it.
//
// Example usage:
//
//	loader := envutil.NewLoader()
//	loader.LoadEnv()  // Load process environment
//	loader.LoadFile(".env")  // Load file configuration
//
//	// ... do some work ...
//
//	// Clear everything and start fresh
//	loader.Clear()
//	loader.LoadFile(".env.test")  // Load test-specific config only
//
// Common use cases:
//   - Testing: Reset loader state between test cases
//   - Re-initialization: Start fresh without creating a new loader
//   - Conditional loading: Clear and reload based on runtime conditions
//   - Memory cleanup: Remove all variables when no longer needed
//
// Alternative: If you want to start completely fresh, consider creating a new loader
// with NewLoader() instead, which may be more explicit and easier to understand.
func (l *Loader) Clear() {
	clear(l.environment)
}

// Filter removes environment variables from the loader that don't match the predicate.
// The predicate function is called for each key-value pair, and only pairs where the
// predicate returns true are kept. All other variables are removed from the loader.
//
// This is a destructive operation that modifies the loader in-place. Variables that
// don't match the predicate are permanently removed. This method does NOT modify the
// actual process environment.
//
// Example usage:
//
//	loader := envutil.NewLoader()
//	loader.LoadEnv()  // Start with all process environment variables
//
//	// Keep only variables starting with "APP_"
//	loader.Filter(func(key, value string) bool {
//	    return strings.HasPrefix(key, "APP_")
//	})
//
//	// Keep only non-empty values
//	loader.Filter(func(key, value string) bool {
//	    return value != ""
//	})
//
//	// Keep specific variables by name
//	allowedVars := map[string]bool{
//	    "DATABASE_URL": true,
//	    "REDIS_URL":    true,
//	    "PORT":         true,
//	}
//	loader.Filter(func(key, value string) bool {
//	    return allowedVars[key]
//	})
//
//	// Remove sensitive variables before passing to subprocess
//	loader.Filter(func(key, value string) bool {
//	    sensitive := []string{"SECRET", "PASSWORD", "TOKEN", "KEY"}
//	    for _, s := range sensitive {
//	        if strings.Contains(key, s) {
//	            return false  // Remove sensitive variables
//	        }
//	    }
//	    return true  // Keep everything else
//	})
//
// Common use cases:
//   - Namespace isolation: Keep only variables with specific prefixes
//   - Security: Remove sensitive variables before export
//   - Cleanup: Remove empty or invalid values
//   - Allowlist/blocklist: Keep or remove specific sets of variables
//   - Validation: Remove variables that don't meet certain criteria
//
// Performance note: This method creates a new internal map and replaces the old one,
// so it's O(n) where n is the number of variables in the loader.
//
// Alternative: If you want to keep the original environment and create a filtered copy,
// use AsMap() to get a copy first, then create a new loader and filter that.
func (l *Loader) Filter(predicate func(key string, value string) (keep bool)) {
	accum := make(map[string]string)

	for key, value := range l.environment {
		if predicate(key, value) {
			accum[key] = value
		}
	}

	l.environment = accum
}

// Contains checks if an environment variable exists in the loader.
// Returns true if the key exists (regardless of value), false otherwise.
//
// Example usage:
//
//	if loader.Contains("DEBUG_MODE") {
//	    // Debug mode is configured (value might be "true", "false", or anything else)
//	}
//
//	if !loader.Contains("REQUIRED_CONFIG") {
//	    return errors.New("REQUIRED_CONFIG must be set")
//	}
//
// Note: This only checks for the key's existence, not the value. To check both
// existence and retrieve the value, use Get:
//
//	value, found := loader.Get("PORT")
//	if found {
//	    fmt.Printf("PORT is set to: %s\n", value)
//	}
//
// Returns true if the key exists in the loader's environment, false otherwise.
func (l *Loader) Contains(key string) bool {
	_, found := l.environment[key]

	return found
}

// Keys returns a slice of all environment variable names in the loader.
// The order of keys is not guaranteed (map iteration order is random in Go).
//
// Example usage:
//
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")
//
//	// Print all configured variables
//	for _, key := range loader.Keys() {
//	    value, _ := loader.Get(key)
//	    fmt.Printf("%s=%s\n", key, value)
//	}
//
//	// Check for required variables
//	requiredKeys := []string{"DATABASE_URL", "API_KEY", "PORT"}
//	loadedKeys := loader.Keys()
//	for _, required := range requiredKeys {
//	    if !slices.Contains(loadedKeys, required) {
//	        return fmt.Errorf("missing required variable: %s", required)
//	    }
//	}
//
// Returns a new slice containing all environment variable names.
// The returned slice is a copy; modifying it won't affect the loader.
func (l *Loader) Keys() []string {
	keys := make([]string, 0, len(l.environment))

	for k := range l.environment {
		keys = append(keys, k)
	}

	return keys
}

// AsMap returns a copy of the loader's environment as a map.
// The returned map is independent of the loader; modifying it won't affect
// the loader's internal state.
//
// Example usage:
//
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")
//
//	// Get a snapshot of the environment
//	snapshot := loader.AsMap()
//
//	// Pass to a function that expects map[string]string
//	config := parseConfig(snapshot)
//
//	// Create a modified copy without affecting the loader
//	modified := loader.AsMap()
//	modified["PORT"] = "9090"  // Doesn't change loader
//
// This is useful when you need:
//   - A snapshot of the current environment state
//   - To pass environment to functions expecting map[string]string
//   - To create a modified copy without affecting the original loader
//   - To marshal the environment to JSON/YAML
//
// Returns a new map containing all environment variables.
func (l *Loader) AsMap() map[string]string {
	return maps.Clone(l.environment)
}

// AsSlice returns the loader's environment as a slice of "KEY=VALUE" strings.
// This format is compatible with exec.Cmd.Env and other systems that expect
// environment variables in this format.
//
// The order of elements is not guaranteed (map iteration order is random in Go).
//
// Example usage:
//
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")
//	loader.Set("APP_ENV", "production")
//
//	// Pass environment to a subprocess
//	cmd := exec.Command("./worker")
//	cmd.Env = loader.AsSlice()
//	if err := cmd.Run(); err != nil {
//	    return err
//	}
//
//	// Write environment to a file
//	file, err := os.Create(".env.generated")
//	if err != nil {
//	    return err
//	}
//	defer file.Close()
//	for _, line := range loader.AsSlice() {
//	    fmt.Fprintln(file, line)
//	}
//
// Common use cases:
//   - Passing environment to subprocesses (exec.Cmd.Env)
//   - Generating .env files programmatically
//   - Exporting configuration for container environments
//   - Debugging: Print all variables in a readable format
//
// Returns a new slice of "KEY=VALUE" strings.
func (l *Loader) AsSlice() []string {
	out := make([]string, 0, len(l.environment))

	for k := range l.environment {
		out = append(out, fmt.Sprintf("%s=%s", k, l.environment[k]))
	}

	return out
}

// EnhanceContext creates a new context with all loader environment variables as overrides.
// This allows envutil Reader functions (String, Int, Bool, etc.) to use the loader's
// environment instead of the process environment when called with this context.
//
// The enhanced context provides a layered configuration approach:
//  1. Check context overrides (from this loader)
//  2. Check actual process environment (os.Getenv)
//  3. Use default value if provided
//  4. Return error if required and not found
//
// Example usage:
//
//	// Load configuration from files
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")
//	loader.LoadFile(".env.test")
//
//	// Create context with loader's environment
//	ctx := loader.EnhanceContext(context.Background())
//
//	// All envutil readers will use the loader's environment
//	port := envutil.IntCtx(ctx, "PORT", envutil.Default(8080)).Value()
//	dbURL := envutil.StringCtx(ctx, "DATABASE_URL", envutil.Required()).ValueOrFatal()
//	debug := envutil.BoolCtx(ctx, "DEBUG_MODE", envutil.Default(false)).Value()
//
// Common use cases:
//   - Testing: Inject test-specific configuration without modifying process environment
//   - Request handlers: Different requests can have different configuration contexts
//   - Multi-tenant apps: Each tenant can have isolated configuration
//   - Configuration isolation: Run operations with specific config without side effects
//
// Example with HTTP handler:
//
//	func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//	    loader := envutil.NewLoader()
//	    loader.Set("TENANT_ID", r.Header.Get("X-Tenant-ID"))
//	    loader.Set("REQUEST_ID", r.Header.Get("X-Request-ID"))
//
//	    ctx := loader.EnhanceContext(r.Context())
//	    h.processRequest(ctx)  // Handler uses tenant-specific config
//	}
//
// Example with testing:
//
//	func TestDatabaseConnection(t *testing.T) {
//	    loader := envutil.NewLoader()
//	    loader.Set("DATABASE_URL", "postgres://localhost/test")
//	    loader.Set("DB_POOL_SIZE", "5")
//
//	    ctx := loader.EnhanceContext(context.Background())
//	    db, err := connectDatabase(ctx)  // Uses test database
//	    require.NoError(t, err)
//	}
//
// Returns a new context with all loader environment variables as overrides.
// The original context is not modified.
//
// See WithEnvOverrides for more details on context-based environment overrides.
func (l *Loader) EnhanceContext(ctx context.Context) context.Context {
	env := l.AsMap()

	return WithEnvOverrides(ctx, env)
}

// SetContext configures environment variable overrides using a callback setter function.
// This is used with lazy value override systems where context values need to be
// configured before a context is created.
//
// The setter function is typically provided by lazy initialization mechanisms
// (e.g., lazy.SetValueOverride) that store key-value pairs for later retrieval
// when the actual context is created.
//
// Parameters:
//   - setter: Callback function that stores each key-value pair from the loader's
//     environment. If nil, this method returns immediately without effect.
//
// Example usage with lazy initialization:
//
//	// Configure lazy context with loader's environment
//	loader := envutil.NewLoader()
//	loader.LoadFile(".env")
//	loader.LoadFile(".env.production")
//
//	// Set overrides using lazy mechanism
//	lazyCtx := lazy.NewContext()
//	loader.SetContext(lazyCtx.SetValueOverride)
//
//	// Later, when context is materialized, it will have loader's environment
//	ctx := lazyCtx.Get()
//	// envutil readers will use the loader's environment
//	port := envutil.IntCtx(ctx, "PORT", envutil.Default(8080)).Value()
//
// Advanced example - custom setter:
//
//	// Custom setter that filters sensitive variables
//	sensitiveKeys := map[string]bool{"API_KEY": true, "SECRET": true}
//	loader.SetContext(func(key, value any) {
//	    keyStr := key.(string)
//	    if !sensitiveKeys[keyStr] {
//	        myContextBuilder.Set(key, value)
//	    }
//	})
//
// This method is less commonly used than EnhanceContext. Use EnhanceContext when:
//   - You have a context and want to add loader's environment to it
//   - You're working with standard context.Context
//
// Use SetContext when:
//   - Working with lazy initialization frameworks
//   - You need to configure overrides before context creation
//   - Integrating with custom context management systems
//
// See SetEnvOverrides for more details on the callback-based override mechanism.
func (l *Loader) SetContext(setter func(any, any)) {
	env := l.AsMap()

	SetEnvOverrides(env, setter)
}
