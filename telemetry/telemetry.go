package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/amp-labs/amp-common/envutil"
	"github.com/amp-labs/amp-common/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

const (
	defaultServiceVersion = "1.0.0"
	defaultTimeout        = 5 * time.Second
)

var tracerProvider *sdktrace.TracerProvider

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
func LoadConfigFromEnv(runningEnv string) (*Config, error) {
	enabled := envutil.Bool("OTEL_ENABLED",
		envutil.Default(false)).
		ValueOrElse(false)

	// Default to GKE OpenTelemetry collector endpoint if running in GKE
	defaultEndpoint := ""
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" { // Check if running in Kubernetes
		// Running in Kubernetes, use GKE OpenTelemetry collector service endpoint
		defaultEndpoint = "http://opentelemetry-collector.opentelemetry.svc.cluster.local:4318"
	}

	serviceName := logger.GetSubsystem(context.Background())

	svcName, err := envutil.String("OTEL_SERVICE_NAME", envutil.Default(serviceName)).Value()
	if err != nil {
		return nil, err
	}

	svcVersion, err := envutil.String("OTEL_SERVICE_VERSION",
		envutil.Default(defaultServiceVersion)).
		Value()
	if err != nil {
		return nil, err
	}

	endpoint, err := envutil.String("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT",
		envutil.Default(defaultEndpoint)).
		Value()
	if err != nil {
		return nil, err
	}

	timeout, err := envutil.Duration("OTEL_EXPORTER_OTLP_TRACES_TIMEOUT",
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

	slog.Info("OpenTelemetry tracing initialized",
		"service", config.ServiceName,
		"version", config.ServiceVersion,
		"environment", config.Environment,
		"endpoint", config.Endpoint,
	)

	return nil
}

// Shutdown gracefully shuts down the OpenTelemetry tracer provider.
func Shutdown(ctx context.Context) error {
	if tracerProvider == nil {
		return nil
	}

	slog.Info("Shutting down OpenTelemetry tracer provider")

	return tracerProvider.Shutdown(ctx)
}
