# Package: tests

Test utilities for managing test context with unique identifiers and metadata.

## Usage

```go
func TestMyFeature(t *testing.T) {
    ctx := tests.GetUniqueContext(t)

    info, ok := tests.GetTestInfo(ctx)
    // info.Id: unique UUID per test run
    // info.Name: test name from t.Name()

    // Use unique ID for test resources
    dbName := "test_db_" + info.Id

    // Conditionally skip based on env
    tests.CheckSkipped(ctx, t, "SKIP_INTEGRATION_TESTS")
}
```

## Common Patterns

- `GetUniqueContext()` - Create context with test ID and name
- `GetTestInfo()` - Retrieve test metadata from context
- `CheckSkipped()` - Skip tests based on environment variables
- Useful for creating unique test resources

## Gotchas

- Test ID is a UUID prefixed with "test-"
- Context includes testing.T for helper access
- CheckSkipped reads boolean env vars

## Related

- `contexts` - Context utilities used internally
