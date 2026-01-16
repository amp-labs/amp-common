// Package actions provides composable actions for state machine workflows,
// including sampling, elicitation, validation, and error handling primitives.
package actions

import (
	"errors"
	"fmt"
	"slices"
	"time"
)

var (
	// ErrParameterNotFound is returned when a required parameter is not found.
	ErrParameterNotFound = errors.New("parameter not found")
	// ErrParameterTypeMismatch is returned when a parameter has an unexpected type.
	ErrParameterTypeMismatch = errors.New("parameter type mismatch")
	// ErrInvalidDurationFormat is returned when a duration parameter has invalid format.
	ErrInvalidDurationFormat = errors.New("invalid duration format")
	// ErrNoParametersSpecified is returned when at least one of a set of parameters must be specified.
	ErrNoParametersSpecified = errors.New("no parameters specified")
	// ErrInvalidEnumValue is returned when a value is not in the allowed set of values.
	ErrInvalidEnumValue = errors.New("invalid enum value")
)

// ParamExtractor provides utilities for extracting and validating action parameters.
type ParamExtractor struct {
	params map[string]any
}

// NewParamExtractor creates a new parameter extractor.
func NewParamExtractor(params map[string]any) *ParamExtractor {
	return &ParamExtractor{params: params}
}

// GetString extracts a string parameter.
func (p *ParamExtractor) GetString(key string, required bool) (string, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return "", fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return "", nil
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("parameter %q must be a string, got %T: %w", key, val, ErrParameterTypeMismatch)
	}

	return str, nil
}

// GetInt extracts an integer parameter.
func (p *ParamExtractor) GetInt(key string, required bool, defaultVal int) (int, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return 0, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return defaultVal, nil
	}

	// Handle both int and float64 (JSON unmarshalling)
	switch v := val.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("parameter %q must be an integer, got %T: %w", key, val, ErrParameterTypeMismatch)
	}
}

// GetBool extracts a boolean parameter.
func (p *ParamExtractor) GetBool(key string, required bool, defaultVal bool) (bool, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return false, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return defaultVal, nil
	}

	b, ok := val.(bool)
	if !ok {
		return false, fmt.Errorf("parameter %q must be a boolean, got %T: %w", key, val, ErrParameterTypeMismatch)
	}

	return b, nil
}

// GetFloat extracts a float64 parameter.
func (p *ParamExtractor) GetFloat(key string, required bool, defaultVal float64) (float64, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return 0, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return defaultVal, nil
	}

	// Handle both int and float64
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("parameter %q must be a number, got %T: %w", key, val, ErrParameterTypeMismatch)
	}
}

// GetDuration extracts a duration parameter (supports string or number of seconds).
func (p *ParamExtractor) GetDuration(key string, required bool, defaultVal time.Duration) (time.Duration, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return 0, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return defaultVal, nil
	}

	// Handle string duration (e.g., "1h30m")
	if str, ok := val.(string); ok {
		d, err := time.ParseDuration(str)
		if err != nil {
			return 0, fmt.Errorf("parameter %q: %w: %w", key, ErrInvalidDurationFormat, err)
		}

		return d, nil
	}

	// Handle number of seconds
	switch v := val.(type) {
	case float64:
		return time.Duration(v) * time.Second, nil
	case int:
		return time.Duration(v) * time.Second, nil
	default:
		return 0, fmt.Errorf(
			"parameter %q must be a duration string or number, got %T: %w",
			key, val, ErrParameterTypeMismatch,
		)
	}
}

// GetStringSlice extracts a string slice parameter.
func (p *ParamExtractor) GetStringSlice(key string, required bool) ([]string, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return nil, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return nil, nil
	}

	slice, ok := val.([]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an array, got %T: %w", key, val, ErrParameterTypeMismatch)
	}

	result := make([]string, len(slice))
	for i, item := range slice {
		str, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("parameter %q[%d] must be a string, got %T: %w", key, i, item, ErrParameterTypeMismatch)
		}

		result[i] = str
	}

	return result, nil
}

// GetMap extracts a map parameter.
func (p *ParamExtractor) GetMap(key string, required bool) (map[string]any, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return nil, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return nil, nil
	}

	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an object, got %T: %w", key, val, ErrParameterTypeMismatch)
	}

	return m, nil
}

// GetStringMap extracts a map[string]string parameter.
func (p *ParamExtractor) GetStringMap(key string, required bool) (map[string]string, error) {
	val, exists := p.params[key]
	if !exists {
		if required {
			return nil, fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}

		return nil, nil
	}

	m, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parameter %q must be an object, got %T: %w", key, val, ErrParameterTypeMismatch)
	}

	result := make(map[string]string)

	for k, v := range m {
		str, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("parameter %q[%s] must be a string, got %T: %w", key, k, v, ErrParameterTypeMismatch)
		}

		result[k] = str
	}

	return result, nil
}

// ValidateRequired checks that all required parameters are present.
func (p *ParamExtractor) ValidateRequired(keys ...string) error {
	for _, key := range keys {
		if _, exists := p.params[key]; !exists {
			return fmt.Errorf("required parameter %q: %w", key, ErrParameterNotFound)
		}
	}

	return nil
}

// ValidateOneOf checks that at least one of the specified parameters is present.
func (p *ParamExtractor) ValidateOneOf(keys ...string) error {
	for _, key := range keys {
		if _, exists := p.params[key]; exists {
			return nil
		}
	}

	return fmt.Errorf("at least one of %v: %w", keys, ErrNoParametersSpecified)
}

// WrapError wraps an error with action context.
func WrapError(actionName string, err error) error {
	if err == nil {
		return nil
	}

	return fmt.Errorf("[%s] %w", actionName, err)
}

// ValidateEnum validates that a string value is one of the allowed values.
func ValidateEnum(value string, allowed []string) error {
	if slices.Contains(allowed, value) {
		return nil
	}

	return fmt.Errorf("value %q not in allowed values %v: %w", value, allowed, ErrInvalidEnumValue)
}

// CoalesceString returns the first non-empty string.
func CoalesceString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}

// CoalesceInt returns the first non-zero integer.
func CoalesceInt(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}

	return 0
}

// CoalesceDuration returns the first non-zero duration.
func CoalesceDuration(values ...time.Duration) time.Duration {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}

	return 0
}
