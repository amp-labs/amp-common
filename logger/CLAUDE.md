# Package: logger

Structured logging built on slog with OpenTelemetry integration and context-aware features.

## Usage

```go
import "github.com/amp-labs/amp-common/logger"

// Configure logging (call once at startup)
logger.ConfigureLoggingWithOptions(&logger.Options{
    MinLevel:   slog.LevelInfo,
    Output:     os.Stdout,
    AddSource:  false,
    EnableOtel: false,  // Opt-in OTel integration
    Subsystem:  "api",
})

// Use context-aware logging
logger.Info(ctx, "user logged in", "user_id", userId)
logger.Error(ctx, "failed to connect", "error", err)

// Get logger from context
log := logger.Get(ctx)
log.Info("message")
```

## Common Patterns

- Context-aware logging: Debug, Info, Warn, Error, Fatal
- `Get(ctx)` - Retrieve logger from context or default
- Options: MinLevel, Output, AddSource (file:line), EnableOtel (opt-in)
- Subsystem tracking per service/component
- OpenTelemetry integration (opt-in with EnableOtel: true)
- Runtime suppression: `WithSuppressOtel(ctx, true)` for selective OTel disabling

## Gotchas

- OpenTelemetry logging is OPT-IN (EnableOtel: false by default)
- AddSource tracks file:line but adds overhead
- Fatal() calls shutdown hooks and exits with code 1
- Subsystem can be context-overridden with `WithSubsystem()`
- OTel suppression allows per-request disabling even when globally enabled

## Related

- `telemetry` - OpenTelemetry tracing setup
- `shutdown` - Graceful shutdown for Fatal()
