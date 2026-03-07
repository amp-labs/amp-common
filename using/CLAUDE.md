# Package: using

Resource management pattern like C# "using" or Java try-with-resources.

## Usage

```go
// Use with automatic cleanup
err := using.OpenFile("data.txt").Use(func(f *os.File) error {
    _, err := f.WriteString("hello")
    return err
})
// File automatically closed, even on error

// Custom resource
resource := using.NewResource(func() (*DB, using.Closer, error) {
    db, err := connectDB()
    return db, db.Close, err
})

err := resource.Use(func(db *DB) error {
    return db.Query(ctx, query)
})
```

## Common Patterns

- `Use()` - Execute function with resource, auto-cleanup
- `NewResource()` - Create custom managed resources
- `Release()` - Prevent auto-cleanup (transfer ownership)
- Package provides helpers: OpenFile, DialTCP, etc.

## Gotchas

- Errors from both function and closer are collected
- Call `Release()` to prevent automatic cleanup
- Resource must not be nil

## Related

- `closer` - Lower-level closer management
- `errors` - Error collection
