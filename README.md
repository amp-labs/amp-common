# amp-common

This repository contains shared Go libraries and utilities used across Ampersand projects. It provides a collection of reusable packages for common functionality like actor models, object pooling, concurrent execution, environment variable parsing, telemetry, and more.

## Overview

`amp-common` is a Go library (not a standalone application) that provides shared utilities and packages. It uses Go 1.24.6 and is published as `github.com/amp-labs/amp-common`.

## Prerequisites

* **Go 1.24.6+**
* **SSH Key Setup**: This project uses private GitHub repositories. You need to set up SSH authentication:
  1. Ensure you have an SSH key configured for GitHub access to private amp-labs repositories
  2. Configure git to use SSH for GitHub:
     ```bash
     git config --global url."git@github.com:".insteadOf "https://github.com/"
     ```
  3. Set the GOPRIVATE environment variable (add to your shell profile):
     ```bash
     export GOPRIVATE=github.com/amp-labs/*
     ```

## Development

### Testing

```bash
make test              # Run all tests
go test -v ./...       # Run tests with verbose output
go test -v -run TestName ./package-name  # Run a specific test
```

### Linting and Formatting

```bash
make fix               # Run all formatters and linters with auto-fix
make fix/sort          # Same as fix but with sorted output
make fix-files FILES="file1.go file2.go"  # Fix specific files
```

The linting stack includes:
- `wsl` - Whitespace linter (allows cuddle declarations)
- `gci` - Go import formatter
- `golangci-lint` - Comprehensive Go linter (configured via `.golangci.yml`)

## Core Packages

### Concurrency & Actor Model

**`actor`** - Actor model implementation with message passing
- Generic `Actor[Request, Response]` with concurrent message processing
- Actors have mailboxes (channels) and process messages sequentially
- Includes Prometheus metrics for monitoring
- Methods: `Send`, `SendCtx`, `Request`, `RequestCtx`, `Publish`, `PublishCtx`
- Graceful panic recovery

**`pool`** - Generic object pooling with lifecycle management
- Thread-safe pool for any `io.Closer` objects
- Dynamic growth, configurable idle cleanup
- Prometheus metrics for monitoring
- Uses channels and semaphores for concurrency control

**`simultaneously`** - Parallel execution utility
- `Do(maxConcurrent int, ...func(context.Context) error)` - Run functions in parallel
- Returns first error encountered, cancels remaining on error
- Automatic panic recovery with stack traces
- Semaphore-based concurrency limiting

### Configuration & Environment

**`envutil`** - Type-safe environment variable parsing
- Fluent API with `Reader[T]` type for chaining operations
- Built-in support for: strings, ints, bools, durations, URLs, UUIDs, file paths, etc.
- Options pattern: `Default()`, `Required()`, `Validate()`, etc.
- Example: `envutil.Int("PORT", envutil.Default(8080)).Value()`

### Observability

**`telemetry`** - OpenTelemetry tracing integration
- `Initialize(ctx, config)` - Set up OTLP tracing
- `LoadConfigFromEnv()` - Load config from environment variables
- Auto-detects Kubernetes environments and uses cluster-local collector
- Environment variables: `OTEL_ENABLED`, `OTEL_SERVICE_NAME`, `OTEL_EXPORTER_OTLP_TRACES_ENDPOINT`

**`logger`** - Structured logging utilities
- Built on Go's `slog` package
- Integrates with OpenTelemetry context

### CLI & Commands

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
- **`maps`** - Generic map utilities
- **`set`** - Generic set implementation
- **`tuple`** - Generic tuple types
- **`compare`** - Comparison utilities
- **`errors`** - Error utilities
- **`assert`** - Assertion utilities for testing
- **`hashing`** - Hashing utilities
- **`sanitize`** - String sanitization
- **`script`** - Script execution utilities
- **`build`** - Build information utilities
- **`http/transport`** - HTTP transport configuration with DNS caching

## Dependency Management

This is a Go module (`github.com/amp-labs/amp-common`). When changes are pushed to `main`, Cloud Build automatically:
1. Creates a PR in the `server` repository to update the `amp-common` dependency
2. Closes old auto-update PRs
3. Auto-merges the new PR

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

## Testing Philosophy

- Tests use `github.com/stretchr/testify` for assertions
- Package `debug` is for local debugging only and should not be imported in production code

## Prometheus Metrics

Many packages expose Prometheus metrics:
- **Actor**: message counts, processing time, panics, queue depth
- **Pool**: object counts, creation/close events, errors
- Metrics use subsystem labels for multi-tenancy

## Troubleshooting

### SSH Authentication Issues

If you encounter errors like `Permission denied (publickey)` or module download failures:

1. **Verify your SSH key is added to GitHub** and has access to amp-labs repositories

2. **Test GitHub SSH connection**:
   ```bash
   ssh -T git@github.com
   ```
   You should see: `Hi username! You've successfully authenticated...`

3. **Verify Go environment**:
   ```bash
   go env GOPRIVATE  # Should show: github.com/amp-labs/*
   ```

4. **Test module access**:
   ```bash
   go list -m github.com/amp-labs/amp-common
   ```

### Common Issues

**Problem:** Module download fails with authentication errors
- **Solution:** Ensure `git config --global url."git@github.com:".insteadOf "https://github.com/"` is set

**Problem:** IDE shows "module not found" errors
- **Solution:** Restart your IDE after setting environment variables, ensure SSH key is loaded

**Problem:** Tests fail to import private dependencies
- **Solution:** Verify `GOPRIVATE` is set in your environment and SSH authentication is working
