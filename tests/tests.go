// Package tests provides utilities for managing test context with unique identifiers
// and test metadata. It allows tests to carry test-specific information (test name, unique ID)
// through context.Context, making it easier to correlate test execution with external resources,
// logs, or debugging information.
//
// This package is useful when:
//   - Tests need to create uniquely-named resources (databases, files, etc.)
//   - Test execution needs to be tracked or correlated across systems
//   - Test metadata needs to be passed through function calls without explicit parameters
//
// Example usage:
//
//	func TestMyFeature(t *testing.T) {
//	    ctx := tests.GetUniqueContext(t)
//	    // ctx now contains unique test ID and test name
//
//	    info, ok := tests.GetTestInfo(ctx)
//	    if ok {
//	        fmt.Printf("Running test: %s with ID: %s\n", info.Name, info.Id)
//	    }
//	}
package tests

import (
	"context"
	"testing"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/envutil"
	"github.com/google/uuid"
)

// contextKey is a private type used for storing test metadata in context.Context.
// Using a custom type instead of string prevents collisions with other packages
// that might use the same key names.
type contextKey string

const (
	// testIdKey is the context key for storing the unique test identifier.
	// The test ID is a UUID prefixed with "test-" (e.g., "test-123e4567-e89b-12d3-a456-426614174000").
	testIdKey contextKey = "testId"

	// testNameKey is the context key for storing the test name.
	// The test name is obtained from testing.T.Name() and includes the full test path
	// (e.g., "TestMyFeature/subtest_name").
	testNameKey contextKey = "testName"

	// testTestKey is the context key for storing the testing.T instance.
	// This allows test utilities and helpers to access the original testing.T object
	// for operations like t.Helper(), t.Log(), t.Error(), etc.
	testTestKey contextKey = "testTest"
)

// GetUniqueContext creates a new context derived from t.Context() that includes:
//   - A unique test identifier (UUID with "test-" prefix)
//   - The test name from t.Name()
//
// This function marks itself as a test helper using t.Helper(), so any failures
// will be reported at the caller's location rather than within this function.
//
// The returned context is useful for:
//   - Creating uniquely-named test resources (databases, files, etc.)
//   - Correlating test execution with external systems
//   - Passing test metadata through function calls
//
// Example:
//
//	func TestDatabaseOperations(t *testing.T) {
//	    ctx := tests.GetUniqueContext(t)
//	    info, _ := tests.GetTestInfo(ctx)
//	    dbName := "test_db_" + info.Id // Use unique ID for database name
//	    // ... rest of test
//	}
func GetUniqueContext(t *testing.T) context.Context {
	t.Helper()

	return contexts.WithMultipleValues[contextKey](t.Context(), map[contextKey]any{
		testTestKey: t,
		testIdKey:   "test-" + uuid.New().String(),
		testNameKey: t.Name(),
	})
}

// CheckSkipped conditionally skips a test based on an environment variable.
// This function reads a boolean environment variable and skips the test if the value is true.
// It's useful for selectively disabling tests in different environments (CI, local, staging, etc.)
// without modifying test code.
//
// Parameters:
//   - ctx: Context for reading environment variables
//   - t: The testing.T instance, which will be used to skip the test if needed
//   - envKey: The environment variable name to check (e.g., "SKIP_INTEGRATION_TESTS")
//   - defaultValue: An optional default value (if len == 0, false is assumed)
//
// The function marks itself as a test helper using t.Helper(), so skip messages
// will be reported at the caller's location.
//
// Example:
//
//	func TestSlowIntegration(t *testing.T) {
//	    ctx := context.Background()
//	    // Skip this test if SKIP_SLOW_TESTS=true (default: skip in CI)
//	    tests.CheckSkipped(ctx, t, "SKIP_SLOW_TESTS", true)
//	    // ... rest of test only runs if not skipped
//	}
func CheckSkipped(ctx context.Context, t *testing.T, envKey string, defaultValue ...bool) {
	t.Helper()

	defl := false
	invert := false

	if len(defaultValue) > 0 {
		defl = defaultValue[0]
	}

	if len(defaultValue) > 1 {
		invert = defaultValue[1]
	}

	shouldSkip := envutil.Bool(ctx, envKey, envutil.Default(defl)).ValueOrElse(defl)

	original := shouldSkip

	if invert {
		shouldSkip = !shouldSkip
	}

	if shouldSkip {
		t.Skipf("Skipping test because of environment variable: %s=%v",
			envKey, original)
	}
}

// SetTestId configures the test identifier using a callback setter function.
// This is used with lazy value overrides to set the test ID without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - id: The unique test identifier (typically a UUID with "test-" prefix)
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetTestId(id string, set func(any, any)) {
	if set == nil {
		return
	}

	set(testIdKey, id)
}

// SetTestName configures the test name using a callback setter function.
// This is used with lazy value overrides to set the test name without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - name: The test name including subtest path (e.g., "TestMyFeature/subtest")
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetTestName(name string, set func(any, any)) {
	if set == nil {
		return
	}

	set(testNameKey, name)
}

// SetTestTest configures the testing.T instance using a callback setter function.
// This is used with lazy value overrides to set the test object without directly
// manipulating a context. The set function is typically provided by lazy override
// mechanisms to store the value for later retrieval.
//
// Parameters:
//   - t: The testing.T instance from the current test
//   - set: Callback function that stores the key-value pair. If nil, the function returns early.
func SetTestTest(t *testing.T, set func(any, any)) {
	t.Helper()

	if set == nil {
		return
	}

	set(testTestKey, t)
}

// GetTestName retrieves the test name from the context.
// The test name is the full test path including any subtests (e.g., "TestMyFeature/subtest").
//
// Returns:
//   - string: The test name if present in the context
//   - bool: true if the test name was found, false otherwise
//
// Example:
//
//	name, ok := tests.GetTestName(ctx)
//	if ok {
//	    fmt.Printf("Running test: %s\n", name)
//	}
func GetTestName(ctx context.Context) (string, bool) {
	return contexts.GetValue[contextKey, string](ctx, testNameKey)
}

// GetTestId retrieves the unique test identifier from the context.
// The test ID is a UUID prefixed with "test-" (e.g., "test-123e4567-e89b-12d3-a456-426614174000").
//
// Returns:
//   - string: The test ID if present in the context
//   - bool: true if the test ID was found, false otherwise
//
// Example:
//
//	id, ok := tests.GetTestId(ctx)
//	if ok {
//	    resourceName := "resource_" + id
//	}
func GetTestId(ctx context.Context) (string, bool) {
	return contexts.GetValue[contextKey, string](ctx, testIdKey)
}

// GetTest retrieves the testing.T instance from the context.
// The testing.T instance provides access to test utilities like Helper(), Error(), and Log().
//
// Returns:
//   - *testing.T: The testing.T instance if present in the context
//   - bool: true if the testing.T was found, false otherwise
//
// Example:
//
//	t, ok := tests.GetTest(ctx)
//	if ok {
//	    t.Helper()
//	    t.Logf("Performing operation in test: %s", t.Name())
//	}
func GetTest(ctx context.Context) (*testing.T, bool) {
	return contexts.GetValue[contextKey, *testing.T](ctx, testTestKey)
}

// Info represents test metadata containing both the unique identifier and test name.
// This struct is JSON-serializable, making it useful for logging or sending test
// information to external systems.
type Info struct {
	Test *testing.T `json:"-"`
	Id   string     `json:"id"`   // Unique test identifier (UUID with "test-" prefix)
	Name string     `json:"name"` // Full test name including subtest path
}

// GetTestInfo retrieves both the test ID and test name from the context as a single Info struct.
// This is a convenience function that combines GetTestId and GetTestName.
//
// Returns:
//   - Info: A struct containing the test ID and name. If only one value is present,
//     the other field will be an empty string.
//   - bool: true if at least one value (ID or name) was found in the context,
//     false if neither value is present
//
// Example:
//
//	info, ok := tests.GetTestInfo(ctx)
//	if ok {
//	    fmt.Printf("Test: %s (ID: %s)\n", info.Name, info.Id)
//	    // Log to external system, create resources, etc.
//	}
func GetTestInfo(ctx context.Context) (Info, bool) {
	name, nameOk := GetTestName(ctx)
	id, idOk := GetTestId(ctx)
	t, tOk := GetTest(ctx)

	if !nameOk && !idOk && !tOk {
		return Info{}, false
	}

	return Info{
		Test: t,
		Id:   id,
		Name: name,
	}, true
}
