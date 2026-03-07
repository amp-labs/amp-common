# Frequently Asked Questions (FAQ)

## Table of Contents

- [Testing](#testing)
- [Environment Variables](#environment-variables)
- [Code Style](#code-style)
- [Error Handling](#error-handling)
- [Observability](#observability)
- [Development Setup](#development-setup)
- [Concurrency](#concurrency)

## Testing

### Why is t.Parallel() required in all tests?

**Short answer:** It's enforced by the `paralleltest` linter to ensure tests are thread-safe and run faster.

**Long answer:**

Calling `t.Parallel()` at the start of every test provides several benefits:

1. **Catches concurrency bugs early** - If your code isn't thread-safe, parallel tests will expose it
2. **Significantly faster test execution** - Tests run concurrently instead of sequentially
3. **Enforces test isolation** - Each test must be independent and not rely on global state
4. **Better resource utilization** - Makes use of all CPU cores during testing

```go
func TestMyFunction(t *testing.T) {
    t.Parallel()  // Required at the top

    t.Run("sub-test", func(t *testing.T) {
        t.Parallel()  // Required in sub-tests too
        // Test code
    })
}
```

**Exception:** Sequential tests that modify global state (like environment variables) should be clearly documented and justified.

### What's the difference between require and assert?

**require** - Stops the test immediately on failure (use for prerequisites)

```go
result, err := setupFunction()
require.NoError(t, err)  // Test stops here if there's an error
require.NotNil(t, result) // Won't be reached if previous require failed
```

**assert** - Records the failure but continues the test (use for validations)

```go
assert.Equal(t, expected, actual)  // Records failure but continues
assert.True(t, condition)          // Still executed even if previous assert failed
```

**Rule of thumb:**

- Use `require` for setup and preconditions
- Use `assert` for actual test validations
- If a failure makes the rest of the test meaningless, use `require`

### How do I run a single test?

```bash
# Run a specific test by name
go test -v -run TestMyFunction ./package-name

# Run tests matching a pattern
go test -v -run TestMyFunction/sub-test ./package-name

# Run all tests in a specific package
go test -v ./package-name
```

### How do I test context cancellation?

Create a canceled context and verify your code respects it:

```go
func TestWithCanceledContext(t *testing.T) {
    t.Parallel()

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    result, err := YourFunction(ctx)
    assert.Error(t, err)
    assert.True(t, errors.Is(err, context.Canceled))
}
```

For timeout testing:

```go
func TestWithTimeout(t *testing.T) {
    t.Parallel()

    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
    defer cancel()

    time.Sleep(1 * time.Millisecond) // Ensure timeout occurs

    result, err := YourFunction(ctx)
    assert.Error(t, err)
    assert.True(t, errors.Is(err, context.DeadlineExceeded))
}
```

## Environment Variables

### How do I read environment variables safely?

Use the `envutil` package for type-safe, validated environment variable access:

```go
import "github.com/amp-labs/amp-common/envutil"

// With default value
port := envutil.Int("PORT", envutil.Default(8080)).Value()

// Required (fails if missing)
apiKey := envutil.Secret("API_KEY", envutil.Required()).Value()

// With validation
timeout := envutil.Duration("TIMEOUT",
    envutil.Default(30*time.Second),
    envutil.Validate(func(d time.Duration) error {
        if d < time.Second {
            return errors.New("timeout too short")
        }
        return nil
    }),
).Value()
```

### How do I handle secrets in tests?

Disable recording when working with secrets:

```go
func TestWithSecrets(t *testing.T) {
    t.Parallel()

    // Disable recording to prevent secrets from being captured
    cleanup := envutil.WithRecording(false)
    defer cleanup()

    // Set test secret
    t.Setenv("API_KEY", "test-secret")

    // Read secret safely
    apiKey := envutil.Secret("API_KEY").Value()

    // Use in test
    assert.NotEmpty(t, apiKey)
}
```

### What if I need to test with different environment variable values?

Use `t.Setenv()` in tests (Go 1.17+):

```go
func TestWithEnvVar(t *testing.T) {
    t.Parallel()

    t.Setenv("MY_VAR", "test-value")
    // MY_VAR is set only for this test and cleaned up automatically

    value := envutil.String("MY_VAR").Value()
    assert.Equal(t, "test-value", value)
}
```

Or use context overrides:

```go
func TestWithContextOverride(t *testing.T) {
    t.Parallel()

    ctx := envutil.WithOverride(context.Background(), "PORT", "9000")
    port := envutil.Int("PORT").ValueContext(ctx)
    assert.Equal(t, 9000, port)
}
```

## Code Style

### What's the correct import order?

Imports must be grouped in this order (enforced by `gci`):

```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "time"

    // 2. External dependencies
    "github.com/stretchr/testify/assert"
    "github.com/prometheus/client_golang/prometheus"

    // 3. amp-common packages
    "github.com/amp-labs/amp-common/errors"
    "github.com/amp-labs/amp-common/try"
)
```

Run `make fix` to automatically fix import order.

### Why does the linter complain about my variable names?

Short variable names are only allowed within 15 lines of usage:

```go
// ✅ Good: Short name within 15 lines
func process(data []byte) {
    d := decode(data)
    // ... use d within ~15 lines
}

// ❌ Bad: Short name in long function
func longFunction(data []byte) {
    d := decode(data)
    // ... 50 lines of code ...
    return d // Too far from declaration
}

// ✅ Good: Descriptive name
func longFunction(data []byte) {
    decoded := decode(data)
    // ... 50 lines of code ...
    return decoded
}
```

## Error Handling

### Should I use %w or %v for errors?

**Use `%w` for errors** (to preserve error wrapping):

```go
// ✅ Good: Preserves error chain
if err != nil {
    return fmt.Errorf("failed to fetch user: %w", err)
}

// Now caller can use errors.Is() and errors.As()
```

**Use `%v` only for non-error values** (like recovered panic values):

```go
// ✅ Good: Panic value is not an error type
if r := recover(); r != nil {
    return fmt.Errorf("panic: %v", r)
}
```

**Never use `%v` for actual errors:**

```go
// ❌ Bad: Breaks error chain
if err != nil {
    return fmt.Errorf("failed: %v", err)
}
```

### Where should error classification happen?

Error classification (checking error types, status codes, etc.) should happen in the `errors` package, not in the `retry` package.

```go
// ✅ Good: Classification in errors package
if errors.IsRetryable(err) {
    // Retry the operation
}

// ❌ Bad: Classification in retry package
retry.Do(func() error {
    // retry package shouldn't classify errors
})
```

## Observability

### How do I add Prometheus metrics?

Use `promauto` for automatic registration and include subsystem labels:

```go
import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    requestsTotal = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Namespace: "amp",
            Subsystem: "mypackage",
            Name:      "requests_total",
            Help:      "Total number of requests",
        },
        []string{"status"}, // Labels
    )
)

// Use metrics
requestsTotal.WithLabelValues("success").Inc()
```

### How do I add tracing to my code?

Use the `telemetry` package:

```go
import (
    "context"
    "github.com/amp-labs/amp-common/telemetry"
)

func init() {
    // Initialize once at startup
    config := telemetry.LoadConfigFromEnv()
    telemetry.Initialize(context.Background(), config)
}

func MyFunction(ctx context.Context) error {
    ctx, span := telemetry.NewSpan(ctx, "MyFunction")
    defer span.End()

    // Your code here
    // Errors are automatically recorded in spans
}
```

### How do I add structured logging?

Use the `logger` package:

```go
import (
    "context"
    "log/slog"
    "github.com/amp-labs/amp-common/logger"
)

func MyFunction(ctx context.Context) {
    // Get logger from context or create new one
    log := logger.FromContext(ctx)

    log.Info("processing request",
        slog.String("user_id", userID),
        slog.Int("count", count),
    )

    if err != nil {
        log.Error("operation failed",
            slog.String("operation", "fetch"),
            slog.Any("error", err),
        )
    }
}
```

## Development Setup

### SSH authentication fails - how do I fix it?

1. **Verify SSH key is added to GitHub:**

   ```bash
   ssh -T git@github.com
   # Should show: "Hi username! You've successfully authenticated..."
   ```

2. **Configure Git to use SSH:**

   ```bash
   git config --global url."git@github.com:".insteadOf "https://github.com/"
   ```

3. **Set GOPRIVATE:**

   ```bash
   export GOPRIVATE=github.com/amp-labs/*
   # Add to your shell profile (.bashrc, .zshrc, etc.)
   ```

4. **Verify Go environment:**

   ```bash
   go env GOPRIVATE  # Should show: github.com/amp-labs/*
   ```

5. **Clear module cache if needed:**

   ```bash
   go clean -modcache
   go mod download
   ```

### Module download fails - what should I do?

1. **Check GOPRIVATE is set:**

   ```bash
   go env GOPRIVATE
   ```

2. **Verify SSH authentication works:**

   ```bash
   ssh -T git@github.com
   ```

3. **Check Git SSH config:**

   ```bash
   git config --global --get url."git@github.com:".insteadOf
   # Should show: https://github.com/
   ```

4. **Clear cache and retry:**

   ```bash
   go clean -modcache
   go mod download
   ```

### My IDE shows "module not found" errors - how do I fix it?

1. **Restart your IDE** after setting environment variables
2. **Ensure SSH key is loaded:**

   ```bash
   ssh-add -l  # List loaded keys
   ssh-add ~/.ssh/id_ed25519  # Add if needed
   ```

3. **Check IDE Go environment** includes GOPRIVATE setting
4. **Invalidate IDE caches** (if available)

## Concurrency

### How do I run operations in parallel with a limit?

Use the `simultaneously` package:

```go
import "github.com/amp-labs/amp-common/simultaneously"

ctx := context.Background()

// Run up to 10 operations concurrently
err := simultaneously.Do(10,
    func(ctx context.Context) error {
        return operation1(ctx)
    },
    func(ctx context.Context) error {
        return operation2(ctx)
    },
    func(ctx context.Context) error {
        return operation3(ctx)
    },
    // ... more operations
)
```

### How do I create a Future for async operations?

Use the `future` package:

```go
import "github.com/amp-labs/amp-common/future"

// Create a future
fut := future.Go(func() (User, error) {
    return fetchUser(userID)
})

// Wait for result
user, err := fut.Await()
```

With context:

```go
ctx := context.Background()

fut := future.GoContext(ctx, func(ctx context.Context) (User, error) {
    return fetchUserWithContext(ctx, userID)
})

user, err := fut.AwaitContext(ctx)
```

### How do I use the actor pattern?

Use the `actor` package:

```go
import "github.com/amp-labs/amp-common/actor"

// Create an actor
myActor := actor.New(func(ctx context.Context, msg string) (string, error) {
    // Process message
    return "response", nil
})

// Start the actor
go myActor.Run(context.Background())

// Send messages
ref := myActor.Ref()
response, err := ref.Request(context.Background(), "hello")
```

## Still Have Questions?

- Check the [Contributing Guide](CONTRIBUTING.md)
- Review package READMEs (e.g., `future/README.md`)
- Look at existing code examples
- Open a GitHub issue with the `question` label
