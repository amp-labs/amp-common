package validator

import (
	"fmt"

	"github.com/amp-labs/amp-common/statemachine"
)

// ValidateOTELInstrumentation checks that spans are created correctly.
// This validator ensures that the state machine is properly instrumented for OpenTelemetry tracing.
func ValidateOTELInstrumentation(config *statemachine.Config) []ValidationError {
	var errors []ValidationError

	// Check that all states will create spans
	// This is a structural check - the actual span creation happens at runtime
	// We validate that the configuration doesn't have patterns that would prevent tracing
	if config == nil {
		errors = append(errors, ValidationError{
			Code:    "OTEL_CONFIG_EXISTS",
			Message: "config is nil - cannot validate OTEL instrumentation",
		})

		return errors
	}

	// Note: Initial state and final states validation is handled by core validation

	// Check that all states have valid names (required for state span naming)
	for _, state := range config.States {
		if state.Name == "" {
			errors = append(errors, ValidationError{
				Code:    "OTEL_STATE_NAMING",
				Message: "state with empty name - cannot create state span",
				Location: Location{
					State: state.Name,
				},
			})
		}

		// Check that actions have valid names (required for action span naming)
		for i, action := range state.Actions {
			if action.Name == "" {
				errors = append(errors, ValidationError{
					Code:    "OTEL_ACTION_NAMING",
					Message: fmt.Sprintf("action %d in state '%s' has empty name - cannot create action span", i, state.Name),
					Location: Location{
						State: state.Name,
					},
				})
			}
		}
	}

	// Verify span attribute completeness
	// Note: Actual span attributes are set from Context at runtime,
	// so we can't validate them here. This is a placeholder for future validation.

	// Check for potential span leaks
	// The framework uses defer span.End(), so leaks are unlikely.
	// This is a structural check for common patterns that might cause issues.

	return errors
}

// ValidateOTELContextMetadata validates that the Context has required metadata for span attributes.
// This is a runtime validation that should be called after Context is created.
// Returns warnings, not errors, since missing metadata doesn't prevent execution.
func ValidateOTELContextMetadata(ctx *statemachine.Context) []ValidationWarning {
	var warnings []ValidationWarning

	// Check for recommended metadata fields
	if ctx.ToolName == "" {
		warnings = append(warnings, ValidationWarning{
			Code:    "OTEL_CONTEXT_TOOL",
			Message: "ToolName not set in Context - span attribute 'tool' will be empty",
		})
	}

	if ctx.SessionID == "" {
		warnings = append(warnings, ValidationWarning{
			Code:    "OTEL_CONTEXT_SESSION",
			Message: "SessionID not set in Context - span attribute 'session_id' will be empty",
		})
	}

	if ctx.ProjectID == "" {
		warnings = append(warnings, ValidationWarning{
			Code:    "OTEL_CONTEXT_PROJECT",
			Message: "ProjectID not set in Context - span attribute 'project_id_hash' will be empty",
		})
	}

	if ctx.Provider == "" {
		warnings = append(warnings, ValidationWarning{
			Code:    "OTEL_CONTEXT_PROVIDER",
			Message: "Provider not set in Context - span attribute 'provider' will be empty",
		})
	}

	return warnings
}

// ValidateOTELDebugMode validates that debug mode is properly configured.
func ValidateOTELDebugMode() []ValidationWarning {
	// This is a no-op validator since debug mode is controlled by environment variable
	// and doesn't require validation. Kept for future use.
	return nil
}
