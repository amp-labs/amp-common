# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`amp-common` is a Go library repository containing shared utilities and packages used across Ampersand projects. This is not a standalone application but a collection of reusable Go packages.

## Commands

### Testing

```bash
make test       # Run all tests
make race       # Run tests with race detection
go test -v ./...  # Run tests with verbose output
```

### Linting and Formatting

```bash
make fix               # Run all formatters and linters with auto-fix
make fix/sort          # Same as fix but with sorted output
make lint              # Run linters without auto-fix
make format            # Alias for 'make fix'
make fix-markdown      # Fix markdown files
```

The linting stack includes:

- `wsl` - Whitespace linter (allows cuddle declarations)
- `gci` - Go import formatter
- `golangci-lint` - Comprehensive Go linter (configured via `.golangci.yml`)

### Running a Single Test

```bash
go test -v -run TestName ./package-name
```

## Architecture

### Core Packages

**`actor`** - Actor model implementation with message passing

- Provides generic `Actor[Request, Response]` with concurrent message processing
- Actors have mailboxes (channels) and process messages sequentially
- Includes Prometheus metrics for monitoring actor performance
- `Ref` type provides methods: `Send`, `SendCtx`, `Request`, `RequestCtx`, `Publish`, `PublishCtx`
- Actors can panic-recover gracefully and notify callers of failures

**`pool`** - Generic object pooling with lifecycle management

- Thread-safe pool for any `io.Closer` objects
- Dynamic growth, configurable idle cleanup
- Includes Prometheus metrics for pool monitoring
- Uses channels and semaphores for concurrency control

**`simultaneously`** - Parallel execution utility

- `Do(maxConcurrent int, ...func(context.Context) error)` - Run functions in parallel
- Returns first error encountered, cancels remaining on error
- Automatic panic recovery with stack traces
- Semaphore-based concurrency limiting

**`envutil`** - Type-safe environment variable parsing

- Fluent API with `Reader[T]` type for chaining operations
- Built-in support for: strings, ints, bools, durations, URLs, UUIDs, file paths, etc.
- Options pattern: `Default()`, `IfMissing()`, `Fallback()`, `Validate()`, etc.
- Example: `envutil.Int(ctx, "PORT", envutil.Default(8080)).Value()`
- Recording: Disabled by default. If you enable recording for testing/debugging, be careful not to capture secrets. See package documentation and SECURITY.md for details.

**`startup`** - Application initialization and environment configuration

- Load environment variables from files specified in ENV_FILE
- Semicolon-separated file paths support (e.g., `/path/to/.env;/path/to/.env.local`)
- Configurable override behavior for existing environment variables
- Functions: `ConfigureEnvironment()`, `ConfigureEnvironmentFromFiles()`, `WithAllowOverride()`

**`telemetry`** - OpenTelemetry tracing integration

- `Initialize(ctx, config)` - Set up OTLP tracing
- `LoadConfigFromEnv()` - Load config from environment variables
- Auto-detects Kubernetes environments and uses cluster-local collector
- Environment variables: `OTEL_ENABLED`, `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`

**`logger`** - Structured logging utilities

- Built on Go's `slog` package
- Integrates with OpenTelemetry context
- Optional OpenTelemetry integration via `go.opentelemetry.io/contrib/bridges/otelslog`
  - Enable with `Options{EnableOtel: true}` when configuring logging
  - When enabled, logs are sent to both console and OpenTelemetry
  - Disabled by default (opt-in feature)
  - Allows logs to be correlated with traces and exported via OTLP
  - Runtime suppression: Use `WithSuppressOtel(ctx, true)` to selectively suppress OTel logging while keeping console output
    - Useful for high-frequency operations or non-sampled contexts
    - If no suppression flag is present and OTel is configured, OTel logging runs by default
- Source code location tracking via `AddSource` option
  - Enable with `Options{AddSource: true}` or environment variable `LOG_ADD_SOURCE=true`
  - When enabled, logs include file name and line number where the log was generated
  - Applied to both slog handlers (JSON/Text) and OpenTelemetry handler
  - Useful for debugging but adds overhead - typically disabled in production

**`cli`** - CLI utilities for terminal interaction

- Banner/divider generation with Unicode box drawing
- `BannerAutoWidth()`, `DividerAutoWidth()` - Auto-detect terminal size
- Prompt utilities for user input
- Set `AMP_NO_BANNER=true` to suppress banners

**`cmd`** - Command execution wrapper

- Fluent API for building `exec.Cmd` instances
- Methods: `SetDir()`, `SetStdin()`, `SetStdout()`, `SetStderr()`, `AppendEnv()`, etc.
- Returns exit code and error separately

### Utility Packages

- **`lazy`** - Lazy initialization with thread-safety
- **`try`** - Result type for error handling (`Try[T]` with `Value` and `Error`)
- **`should`** - Utilities for cleanup operations (e.g., `should.Close()`)
- **`shutdown`** - Graceful shutdown coordination
- **`bgworker`** - Background worker management
- **`utils`** - Misc utilities (channels, context, JSON, sleep, dedup)
- **`xform`** - Type transformations and conversions
- **`maps`** - Generic map utilities with red-black tree implementation
- **`set`** - Generic set implementation with red-black tree backing
- **`tuple`** - Generic tuple types
- **`compare`** - Comparison utilities
- **`sortable`** - Sortable interface with `LessThan` comparison for ordering
- **`collectable`** - Interface combining `Hashable` and `Comparable` for use in Map/Set data structures
- **`errors`** - Error utilities with collection support
- **`retry`** - Flexible retry mechanism with exponential backoff, jitter, and retry budgets
- **`validate`** - Validation interfaces (`HasValidate`, `HasValidateWithContext`) with panic recovery and Prometheus metrics
- **`assert`** - Assertion utilities for testing
- **`hashing`** - Hashing utilities
- **`sanitize`** - String sanitization
- **`jsonpath`** - JSONPath bracket notation utilities for field mapping (parsing, validation, nested path operations)
- **`script`** - Script execution utilities
- **`build`** - Build information utilities
- **`http/transport`** - HTTP transport configuration with DNS caching
- **`channels`** - Channel utilities (`CloseChannelIgnorePanic`)
- **`closer`** - Resource management utilities for `io.Closer` (`Closer` collector, `CloseOnce`, `HandlePanic`, `CustomCloser`)
- **`optional`** - Type-safe Optional/Maybe type (`Some[T]`, `None[T]`, `Map`, `FlatMap`)
- **`pointer`** - Pointer utilities (`To[T]`, `Value[T]`)
- **`stage`** - Environment detection (local, test, dev, staging, prod)
- **`using`** - Resource management pattern (try-with-resources/using statement)
- **`future`** - Future/Promise implementation for async programming (`Go`, `GoContext`, `Await`, `Map`, `Combine`)
- **`envtypes`** - Common environment variable types (HostPort, Path)
- **`contexts`** - Context utilities (`EnsureContext`, `IsContextAlive`, `WithValue[K,V]`, `GetValue[K,V]`)
- **`emoji`** - Emoji constants for terminal output and UI (Rocket, Fire, ThumbsUp, Warning, etc.)
- **`zero`** - Zero value utilities for generic types (`Value[T]()`, `IsZero[T](value)`)
- **`debug`** - Debugging utilities (for local development only, not for production use)

## Dependency Management

This repository is a Go module (`github.com/amp-labs/amp-common`). It uses Go 1.25.5.

### Private Dependencies

The codebase uses private GitHub repositories. When working with this code:

- Set `GOPRIVATE="github.com/amp-labs/*"`
- SSH authentication is required for private repos

### Updating Dependencies

When changes are pushed to `main`, Cloud Build automatically:

1. Creates a PR in the `server` repository to update `amp-common` dependency
2. Closes old auto-update PRs
3. Auto-merges the new PR

## Testing Philosophy

- Tests use `github.com/stretchr/testify` for assertions
- Package `debug` is for local debugging only and should not be imported in production code

## Testing Requirements

### Mandatory t.Parallel()

**All tests MUST call `t.Parallel()`** at the beginning. This is enforced by the `paralleltest` linter.

```go
func TestMyFunction(t *testing.T) {
    t.Parallel()  // REQUIRED at the top of every test function

    t.Run("sub-test name", func(t *testing.T) {
        t.Parallel()  // REQUIRED in every sub-test too

        // Test code here
    })
}
```

**Why?**

- **Test isolation**: Forces tests to be independent and thread-safe
- **Catches concurrency bugs**: If your code isn't thread-safe, parallel tests will expose it early
- **Significantly faster**: Tests run concurrently instead of sequentially, utilizing all CPU cores
- **Better resource utilization**: Makes efficient use of build server capacity

**Exceptions:**

Sequential tests that modify global state (like environment variables) should:

1. Use `//nolint:tparallel` to disable the linter for that test
2. Clearly document why `t.Parallel()` is omitted
3. Be justified - most tests should be parallelizable

Example of legitimate exception:

```go
//nolint:tparallel // Cannot use t.Parallel() with subtests that call t.Setenv()
func TestWithEnvVar(t *testing.T) {
    t.Run("with value", func(t *testing.T) {
        t.Setenv("MY_VAR", "value")  // Modifies process-wide state
        // Test code
    })
}

## Linter Configuration

The `.golangci.yml` enables most linters but disables:

- `gochecknoinits` - Allows `init()` functions
- `exhaustruct` - Zero-valued fields are acceptable
- `testpackage` - Not doing black-box testing
- `wrapcheck` - Too noisy
- `funlen`, `cyclop`, `gocognit` - Function complexity checks disabled
- `gochecknoglobals` - Global variables allowed for legitimate use cases

Special rules:

- Variable naming accepts both "Id" and "ID" (via revive)
- Short variable names allowed within 15 lines (via varnamelen)

## Error Handling

### Error Wrapping

Use `%w` for errors to preserve error wrapping and allow `errors.Is()` and `errors.As()` to work:

```go
// ✅ Good: Preserves error chain
if err != nil {
    return fmt.Errorf("failed to fetch user: %w", err)
}

// Now callers can use errors.Is() and errors.As()
if errors.Is(err, sql.ErrNoRows) {
    // Handle not found
}
```

Use `%v` **only** for non-error values like recovered panic values:

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
    return fmt.Errorf("failed: %v", err)  // Don't do this!
}
```

### Error Classification

Error classification (checking error types, determining if retryable, etc.) should happen in the `errors` package, not in other packages like `retry`.

```go
// ✅ Good: Classification in errors package
if errors.IsRetryable(err) {
    // Retry the operation
}

// ❌ Bad: Classification logic scattered in retry package
// The retry package should use errors package for classification
```

## Prometheus Metrics

Many packages expose Prometheus metrics:

- Actor: message counts, processing time, panics, queue depth
- Pool: object counts, creation/close events, errors
- Metrics use subsystem labels for multi-tenancy
