# Package: telemetry

OpenTelemetry tracing integration with automatic service discovery and Kubernetes support.

## Usage

```go
import "github.com/amp-labs/amp-common/telemetry"

// Load config from environment
config, err := telemetry.LoadConfigFromEnv(ctx, "production")

// Initialize tracing
err := telemetry.Initialize(ctx, config)

// Use with spans package
ctx = spans.WithTracer(ctx, otel.Tracer("my-service"))
result, err := spans.StartValErr[int](ctx, "operation").Enter(...)
```

## Common Patterns

- `LoadConfigFromEnv()` - Read config from env vars
- `Initialize()` - Set up OTLP tracing with auto-discovery
- Environment variables: OTEL_ENABLED, OTEL_SERVICE_NAME, OTEL_EXPORTER_OTLP_TRACES_ENDPOINT
- Auto-detects Kubernetes and uses cluster-local collector
- Default sample rate: 10% (configurable)

## Gotchas

- Auto-discovers k8s collector at `opentelemetry-collector.opentelemetry.svc.cluster.local:4318`
- Disabled by default (set OTEL_ENABLED=true)
- Service name defaults to logger subsystem
- Supports multiple span processors (e.g., Sentry)

## Related

- `spans` - Fluent span creation API
- `logger` - Uses subsystem for service name
