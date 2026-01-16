package helpers

import (
	"context"
	"log/slog"
)

// LogSamplingAttempt logs a sampling request attempt.
func LogSamplingAttempt(ctx context.Context, operation string, prompt string, opts map[string]any) {
	fields := []any{
		"operation", operation,
		"prompt_length", len(prompt),
	}
	for k, v := range opts {
		fields = append(fields, k, v)
	}

	slog.InfoContext(ctx, "attempting sampling", fields...)
}

// LogSamplingResult logs a sampling response.
func LogSamplingResult(ctx context.Context, operation string, source string, resultLength int, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "sampling failed",
			"operation", operation,
			"source", source,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "sampling succeeded",
			"operation", operation,
			"source", source,
			"result_length", resultLength,
		)
	}
}

// LogElicitationAttempt logs an elicitation request attempt.
func LogElicitationAttempt(ctx context.Context, operation string, mode string, message string) {
	slog.InfoContext(ctx, "attempting elicitation",
		"operation", operation,
		"mode", mode,
		"message_length", len(message),
	)
}

// LogElicitationResult logs an elicitation response.
func LogElicitationResult(ctx context.Context, operation string, action string, declined bool, err error) {
	if err != nil {
		slog.ErrorContext(ctx, "elicitation failed",
			"operation", operation,
			"action", action,
			"error", err,
		)
	} else {
		slog.InfoContext(ctx, "elicitation completed",
			"operation", operation,
			"action", action,
			"declined", declined,
		)
	}
}

// LogGracefulDegradation logs when a fallback is used.
func LogGracefulDegradation(ctx context.Context, operation string, reason string, fallbackType string) {
	slog.InfoContext(ctx, "using fallback value",
		"operation", operation,
		"reason", reason,
		"fallback_type", fallbackType,
	)
}

// LogCapabilityCheck logs capability checking results.
func LogCapabilityCheck(ctx context.Context, capability string, available bool) {
	slog.InfoContext(ctx, "capability check",
		"capability", capability,
		"available", available,
	)
}
