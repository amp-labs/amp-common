// Package telemetry provides OpenTelemetry tracing integration with automatic service discovery and configuration.
package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/logger"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	defaultServiceVersion = "1.0.0"
	defaultTimeout        = 5 * time.Second
)

var (
	tracerProvider *sdktrace.TracerProvider
	loggerProvider *sdklog.LoggerProvider
)

// Config holds the OpenTelemetry configuration.
type Config struct {
	ServiceName    string
	ServiceVersion string
	Environment    string
	Endpoint       string
	Enabled        bool
	Timeout        time.Duration
}

// LoadConfigFromEnv loads OpenTelemetry configuration from environment variables.
func LoadConfigFromEnv(ctx context.Context, runningEnv string) (*Config, error) {
	enabled := envutil.Bool(ctx, "OTEL_ENABLED",
		envutil.Default(false)).
		ValueOrElse(false)

	// Default to GKE OpenTelemetry collector endpoint if running in GKE
	defaultEndpoint := ""
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" { // Check if running in Kubernetes
		// Running in Kubernetes, use GKE OpenTelemetry collector service endpoint
		defaultEndpoint = "http://opentelemetry-collector.opentelemetry.svc.cluster.local:4318"
	}

	serviceName := logger.GetSubsystem(ctx)

	svcName, err := envutil.String(ctx, "OTEL_SERVICE_NAME", envutil.Default(serviceName)).Value()
	if err != nil {
		return nil, err
	}

	svcVersion, err := envutil.String(ctx, "OTEL_SERVICE_VERSION",
		envutil.Default(defaultServiceVersion)).
		Value()
	if err != nil {
		return nil, err
	}

	endpoint, err := envutil.String(ctx, "OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		envutil.Default(defaultEndpoint)).
		Value()
	if err != nil {
		return nil, err
	}

	timeout, err := envutil.Duration(ctx, "OTEL_EXPORTER_OTLP_TRACES_TIMEOUT",
		envutil.Default(defaultTimeout)).
		Value()
	if err != nil {
		return nil, err
	}

	return &Config{
		ServiceName:    svcName,
		ServiceVersion: svcVersion,
		Environment:    runningEnv,
		Endpoint:       endpoint,
		Enabled:        enabled,
		Timeout:        timeout,
	}, nil
}

// Initialize sets up OpenTelemetry tracing with the given configuration.
func Initialize(ctx context.Context, config *Config) error {
	if !config.Enabled {
		slog.Info("OpenTelemetry tracing is disabled")

		return nil
	}

	if config.Endpoint == "" {
		slog.Warn("OpenTelemetry endpoint not configured, tracing will be disabled")

		return nil
	}

	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP trace exporter
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(config.Endpoint),
		otlptracehttp.WithTimeout(config.Timeout),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP trace exporter: %w", err)
	}

	// Create trace provider
	tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	// Set the global trace provider
	otel.SetTracerProvider(tracerProvider)

	// Set the global propagator to support trace context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	// Create OTLP log exporter (uses same endpoint as traces)
	logExporter, err := otlploghttp.New(ctx,
		otlploghttp.WithEndpointURL(config.Endpoint),
		otlploghttp.WithTimeout(config.Timeout),
	)
	if err != nil {
		return fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	// Create log provider with the same resource as traces
	loggerProvider = sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(res),
	)

	// Create slog handler that bridges to OpenTelemetry
	otelHandler := otelslog.NewHandler(config.ServiceName,
		otelslog.WithLoggerProvider(loggerProvider),
	)

	// Get the current slog handler (to preserve existing behavior)
	currentHandler := slog.Default().Handler()

	// Create a multi-handler that sends logs to both OTel and the existing handler
	multiHandler := NewMultiHandler(otelHandler, currentHandler)

	// Set as the default slog handler
	slog.SetDefault(slog.New(multiHandler))

	slog.Info("OpenTelemetry tracing and logging initialized",
		"service", config.ServiceName,
		"version", config.ServiceVersion,
		"environment", config.Environment,
		"endpoint", config.Endpoint,
	)

	return nil
}

// Shutdown gracefully shuts down the OpenTelemetry tracer and logger providers.
func Shutdown(ctx context.Context) error {
	var shutdownErr error

	if tracerProvider != nil {
		slog.Info("Shutting down OpenTelemetry tracer provider")

		err := tracerProvider.Shutdown(ctx)
		if err != nil {
			shutdownErr = fmt.Errorf("failed to shutdown tracer provider: %w", err)
		}
	}

	if loggerProvider != nil {
		slog.Info("Shutting down OpenTelemetry logger provider")

		err := loggerProvider.Shutdown(ctx)
		if err != nil {
			if shutdownErr != nil {
				shutdownErr = fmt.Errorf("%w; failed to shutdown logger provider: %w", shutdownErr, err)
			} else {
				shutdownErr = fmt.Errorf("failed to shutdown logger provider: %w", err)
			}
		}
	}

	return shutdownErr
}

// MultiHandler is a slog.Handler that sends logs to multiple handlers.
// This allows logs to go to both OpenTelemetry (for trace correlation) and
// the existing handler (for local logging/debugging).
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler creates a new MultiHandler that forwards logs to all provided handlers.
func NewMultiHandler(handlers ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: handlers}
}

// Enabled reports whether the handler handles records at the given level.
// Returns true if any handler is enabled for this level.
func (h *MultiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

// Handle handles the Record by forwarding it to all handlers.
func (h *MultiHandler) Handle(ctx context.Context, record slog.Record) error {
	for _, handler := range h.handlers {
		if handler.Enabled(ctx, record.Level) {
			err := handler.Handle(ctx, record)
			if err != nil {
				// Continue to other handlers even if one fails
				slog.Error("MultiHandler: failed to handle record", "error", err, "handler", fmt.Sprintf("%T", handler))
			}
		}
	}

	return nil
}

// WithAttrs returns a new Handler whose attributes consist of
// both the receiver's attributes and the arguments.
func (h *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithAttrs(attrs)
	}

	return &MultiHandler{handlers: newHandlers}
}

// WithGroup returns a new Handler with the given group appended to
// the receiver's existing groups.
func (h *MultiHandler) WithGroup(name string) slog.Handler {
	newHandlers := make([]slog.Handler, len(h.handlers))
	for i, handler := range h.handlers {
		newHandlers[i] = handler.WithGroup(name)
	}

	return &MultiHandler{handlers: newHandlers}
}
