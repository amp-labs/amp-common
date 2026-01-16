package validator

import (
	"fmt"
	"strings"

	"github.com/amp-labs/amp-common/statemachine"
)

// ValidationResult contains the results of validating a state machine config.
type ValidationResult struct {
	Valid       bool
	Errors      []ValidationError
	Warnings    []ValidationWarning
	Suggestions []Suggestion
}

// ValidationError represents a validation error with fix suggestions.
type ValidationError struct {
	Code     string   // Error code like "UNREACHABLE_STATE", "MISSING_TRANSITION"
	Message  string   // Human-readable error message
	Location Location // Where the error occurred
	Fix      *Fix     // Optional auto-fix suggestion
}

// ValidationWarning represents a non-critical issue.
type ValidationWarning struct {
	Code     string   // Warning code
	Message  string   // Human-readable warning message
	Location Location // Where the warning occurred
}

// Suggestion provides improvement recommendations.
type Suggestion struct {
	Message string // Suggestion description
	Example string // Code example showing the improvement
}

// Location identifies where an issue occurred.
type Location struct {
	File   string // Config file path
	Line   int    // Line number (0 if unknown)
	Column int    // Column number (0 if unknown)
	State  string // State name if applicable
}

// Validate performs comprehensive validation on a state machine config.
func Validate(config *statemachine.Config) ValidationResult {
	return ValidateWithRules(config, DefaultRules())
}

// ValidateFile loads a config from a file and validates it.
func ValidateFile(path string) (ValidationResult, error) {
	return ValidateFileWithOptions(path, false)
}

// ValidateFileStrict loads a config from a file and validates it in strict mode.
func ValidateFileStrict(path string) (ValidationResult, error) {
	return ValidateFileWithOptions(path, true)
}

// ValidateFileWithOptions loads a config from a file and validates it with options.
func ValidateFileWithOptions(path string, strict bool) (ValidationResult, error) {
	config, err := statemachine.LoadConfig(path)
	if err != nil {
		return ValidationResult{
			Valid: false,
			Errors: []ValidationError{
				{
					Code:     "CONFIG_LOAD_FAILED",
					Message:  fmt.Sprintf("Failed to load config: %v", err),
					Location: Location{File: path},
				},
			},
		}, err
	}

	var result ValidationResult
	if strict {
		result = ValidateWithRulesStrict(config, DefaultRules())
	} else {
		result = Validate(config)
	}

	// Set file location for all errors and warnings
	for i := range result.Errors {
		if result.Errors[i].Location.File == "" {
			result.Errors[i].Location.File = path
		}
	}

	for i := range result.Warnings {
		if result.Warnings[i].Location.File == "" {
			result.Warnings[i].Location.File = path
		}
	}

	return result, nil
}

// ValidateWithRules validates using custom rules.
func ValidateWithRules(config *statemachine.Config, rules []Rule) ValidationResult {
	var result ValidationResult

	result.Valid = true

	// Run all validation rules
	for _, rule := range rules {
		ruleResult := rule.Check(config)
		result.Errors = append(result.Errors, ruleResult.Errors...)
		result.Warnings = append(result.Warnings, ruleResult.Warnings...)
	}

	// If any errors found, mark as invalid
	if len(result.Errors) > 0 {
		result.Valid = false
	}

	// Add general suggestions
	result.Suggestions = generateSuggestions(config)

	return result
}

// ValidateWithRulesStrict validates with strict mode (treats warnings as errors).
func ValidateWithRulesStrict(config *statemachine.Config, rules []Rule) ValidationResult {
	result := ValidateWithRules(config, rules)

	// In strict mode, treat warnings as errors
	for _, warning := range result.Warnings {
		result.Errors = append(result.Errors, ValidationError{
			Code:     warning.Code,
			Message:  warning.Message,
			Location: warning.Location,
		})
	}

	// Clear warnings since they're now errors
	result.Warnings = nil

	// Mark as invalid if there are errors
	if len(result.Errors) > 0 {
		result.Valid = false
	}

	return result
}

// generateSuggestions provides general improvement suggestions.
func generateSuggestions(config *statemachine.Config) []Suggestion {
	var suggestions []Suggestion

	// Suggest naming conventions
	hasNonSnakeCase := false

	for _, state := range config.States {
		if containsUpperCase(state.Name) {
			hasNonSnakeCase = true

			break
		}
	}

	if hasNonSnakeCase {
		suggestions = append(suggestions, Suggestion{
			Message: "Consider using snake_case for state names for consistency",
			Example: `states:
  - name: validate_input  # Good
    # instead of: validateInput, ValidateInput`,
		})
	}

	// Suggest error handling states
	hasErrorHandling := false

	for _, state := range config.States {
		if state.OnError != "" {
			hasErrorHandling = true

			break
		}
	}

	if !hasErrorHandling && len(config.States) > 2 {
		suggestions = append(suggestions, Suggestion{
			Message: "Consider adding error handling to action states",
			Example: `states:
  - name: process_data
    type: action
    onError: handle_error  # Transition to error state on failure`,
		})
	}

	// Suggest using metadata for documentation
	hasMetadata := false

	for _, state := range config.States {
		if len(state.Metadata) > 0 {
			hasMetadata = true

			break
		}
	}

	if !hasMetadata && len(config.States) > 3 {
		suggestions = append(suggestions, Suggestion{
			Message: "Consider adding metadata to document complex states",
			Example: `states:
  - name: complex_state
    type: action
    metadata:
      description: "Processes user data and validates against schema"
      owner: "data-team"`,
		})
	}

	return suggestions
}

// containsUpperCase checks if a string contains uppercase characters.
func containsUpperCase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}

	return false
}

// HasErrors returns true if the result has any errors.
func (r ValidationResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if the result has any warnings.
func (r ValidationResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// String returns a human-readable summary of validation results.
func (r ValidationResult) String() string {
	if r.Valid {
		return "✓ Configuration is valid"
	}

	var msg string

	msg += fmt.Sprintf("✗ Configuration has %d error(s)\n", len(r.Errors))

	var msgSb240 strings.Builder
	for _, err := range r.Errors {
		msgSb240.WriteString(fmt.Sprintf("  [%s] %s", err.Code, err.Message))

		if err.Location.State != "" {
			msgSb240.WriteString(fmt.Sprintf(" (state: %s)", err.Location.State))
		}

		msgSb240.WriteString("\n")

		if err.Fix != nil {
			msgSb240.WriteString(fmt.Sprintf("    Fix: %s\n", err.Fix.Description))
		}
	}

	msg += msgSb240.String()

	if len(r.Warnings) > 0 {
		msg += fmt.Sprintf("\n⚠ %d warning(s):\n", len(r.Warnings))

		var msgSb253 strings.Builder
		for _, warn := range r.Warnings {
			msgSb253.WriteString(fmt.Sprintf("  [%s] %s\n", warn.Code, warn.Message))
		}

		msg += msgSb253.String()
	}

	if len(r.Suggestions) > 0 {
		msg += fmt.Sprintf("\n💡 %d suggestion(s) for improvement\n", len(r.Suggestions))
	}

	return msg
}
