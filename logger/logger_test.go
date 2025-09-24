package logger

import (
	"log"
	"log/slog"
	"testing"
)

func TestLogger(t *testing.T) { //nolint:paralleltest
	// Configure logging for JSON output
	ConfigureLoggingWithOptions(Options{
		Subsystem: "test",
		JSON:      true,
	})

	// Just use slog directly, as a point of comparison
	slog.Info("test info")

	// Use logger with no args (will embed subsystem but nothing else)
	Get().Info("should have the default subsystem")

	// Use logger with an embedded customer ID (should have customer ID and default subsystem)
	ctx := WithCustomerId(t.Context(), "1234")
	Get(ctx).Info("should have customer_id and default subsystem")

	// Use logger with an embedded subsystem (should have subsystem but no customer ID)
	ctx = WithSubsystem(t.Context(), "overridden")
	Get(ctx).Info("should have overridden subsystem")

	// Use logger with an embedded subsystem and customer ID (should have both)
	ctx = WithCustomerId(WithSubsystem(t.Context(), "overridden"), "1234")
	Get(ctx).Info("should have overridden subsystem and customer_id")

	// Use logger with an embedded sensitive flag (should have subsystem but no customer ID)
	ctx = WithSensitive(t.Context())
	Get(ctx).Info("should have only the subsystem")

	// Use logger with an embedded sensitive flag and customer ID (should have subsystem but no customer ID)
	ctx = WithSensitive(WithCustomerId(t.Context(), "1234"))
	Get(ctx).Info("should have only the subsystem")

	// Use logger with an embedded sensitive flag and subsystem (should have subsystem but no customer ID)
	ctx = WithSensitive(WithSubsystem(t.Context(), "overridden"))
	Get(ctx).Info("should have only the subsystem (overridden)")

	// Use logger with an embedded sensitive flag, subsystem, and customer ID (should have subsystem but no customer ID)
	ctx = WithSensitive(WithCustomerId(WithSubsystem(t.Context(), "overridden"), "1234"))
	Get(ctx).Info("should have only the subsystem (overridden)")

	// Use logger with an embedded routing to builder (should have log_project and default subsystem)
	ctx = WithRoutingToBuilder(t.Context(), "ampersand-project-id")
	Get(ctx).Info("should have log_project and default subsystem")

	// Use logger with an embedded routing to builder and subsystem (should have both)
	ctx = WithRoutingToBuilder(WithSubsystem(t.Context(), "overridden"), "ampersand-project-id")
	Get(ctx).Info("should have log_project and overridden subsystem")

	// Use logger with an embedded routing to builder, subsystem &
	// sensitive flag (should have subsystem but no log_project)
	ctx = WithSensitive(WithRoutingToBuilder(WithSubsystem(t.Context(), "overridden"), "builder-project-id"))
	Get(ctx).Info("should have only the overridden subsystem")
}

func TestLegacy(t *testing.T) { //nolint:paralleltest
	// Configure logging for JSON output
	ConfigureLoggingWithOptions(Options{
		Subsystem:   "test",
		JSON:        true,
		MinLevel:    slog.LevelDebug,
		LegacyLevel: slog.LevelInfo,
	})

	// Should output JSON
	log.Println("test")

	// Turn off JSON
	ConfigureLoggingWithOptions(Options{
		Subsystem: "test",
		JSON:      false,
	})

	// Should output text (slog text, just not JSON)
	log.Println("test")
}
