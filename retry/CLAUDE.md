# Package: retry

Flexible retry mechanism with exponential backoff, jitter, retry budgets, and timeouts.

## Usage

```go
import "github.com/amp-labs/amp-common/retry"

// Simple retry
err := retry.Do(ctx, func(ctx context.Context) error {
    return makeAPICall()
})

// With custom options
err := retry.Do(ctx, operation,
    retry.WithAttempts(5),
    retry.WithBackoff(retry.ExpBackoff{
        Base: 100*time.Millisecond,
        Max: 5*time.Second,
        Factor: 2,
    }),
    retry.WithJitter(retry.FullJitter),
    retry.WithTimeout(30*time.Second),
)

// Reusable runner
runner := retry.NewRunner(retry.WithAttempts(3))
err := runner.Do(ctx, operation)

// With return value
result, err := retry.DoValue(ctx, func(ctx context.Context) (string, error) {
    return fetchData()
})
```

## Common Patterns

- `Do()` / `DoValue()` - One-shot retries
- `NewRunner()` / `NewValueRunner()` - Reusable retry runners
- Options: WithAttempts, WithBackoff, WithJitter, WithTimeout, WithBudget
- Default: 4 attempts, exponential backoff (100ms base, 2s max), full jitter
- Jitter strategies: NoJitter, FullJitter, EqualJitter, DecorrelatedJitter

## Gotchas

- Returns first error or permanent error
- Retry budgets for rate limiting retries across instances
- Attempts include initial call (4 attempts = 1 call + 3 retries)

## Related

- `future` - Async operations that may need retry
