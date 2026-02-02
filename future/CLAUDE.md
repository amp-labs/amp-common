# Package: future

Future/Promise implementation for asynchronous programming with split read/write responsibilities.

## Usage

```go
import "github.com/amp-labs/amp-common/future"

// Simple async operation
fut := future.Go(func() (int, error) {
    return expensiveComputation(), nil
})
result, err := fut.Await()

// Manual control
fut, promise := future.New[string]()
go func() {
    result := asyncWork()
    promise.Success(result)
}()
value, err := fut.Await()

// With context
fut := future.GoContext(ctx, func(ctx context.Context) (int, error) {
    return contextAwareWork(ctx)
})
result, err := fut.AwaitContext(ctx)

// Map and combine
fut2 := future.Map(fut, func(x int) (string, error) {
    return fmt.Sprint(x), nil
})
combined, err := future.Combine(fut1, fut2)
```

## Common Patterns

- `Go()` / `GoContext()` - Fire and forget async operations
- `New()` - Manual Future/Promise pair for custom control
- `Await()` / `AwaitContext()` - Wait for result
- `Map()` / `FlatMap()` - Transform futures
- `Combine()` - Wait for multiple futures
- Callbacks: OnSuccess, OnError, OnResult

## Gotchas

- Write-once semantics (Promise can only complete once)
- Automatic panic recovery
- Results are memoized
- All operations thread-safe

## Related

- `retry` - Combine with retries for resilient async
