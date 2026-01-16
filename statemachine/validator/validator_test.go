//nolint:varnamelen // Test file
package validator

import (
	"testing"

	"github.com/amp-labs/amp-common/statemachine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		config     *statemachine.Config
		wantValid  bool
		wantErrors []string // Error codes
	}{
		{
			name: "valid simple workflow",
			config: &statemachine.Config{
				Name:         "valid",
				InitialState: "start",
				FinalStates:  []string{"end"},
				States: []statemachine.StateConfig{
					{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "end", Type: "final"},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "start", To: "end", Condition: "always"},
				},
			},
			wantValid: true,
		},
		{
			name: "unreachable state",
			config: &statemachine.Config{
				Name:         "unreachable",
				InitialState: "start",
				FinalStates:  []string{"end"},
				States: []statemachine.StateConfig{
					{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "orphan", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "end", Type: "final"},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "start", To: "end", Condition: "always"},
				},
			},
			wantValid:  false,
			wantErrors: []string{"UNREACHABLE_STATE", "MISSING_TRANSITION"}, // unreachable states also have missing transitions
		},
		{
			name: "missing transition",
			config: &statemachine.Config{
				Name:         "missing_transition",
				InitialState: "start",
				FinalStates:  []string{"end"},
				States: []statemachine.StateConfig{
					{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "middle", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "end", Type: "final"},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "start", To: "middle", Condition: "always"},
					// Missing transition from middle to end
				},
			},
			wantValid:  false,
			wantErrors: []string{"MISSING_TRANSITION", "UNREACHABLE_STATE"},
		},
		{
			name: "duplicate transition",
			config: &statemachine.Config{
				Name:         "duplicate",
				InitialState: "start",
				FinalStates:  []string{"end"},
				States: []statemachine.StateConfig{
					{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "end", Type: "final"},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "start", To: "end", Condition: "always"},
					{From: "start", To: "end", Condition: "always"},
				},
			},
			wantValid:  false,
			wantErrors: []string{"DUPLICATE_TRANSITION"},
		},
		{
			name: "naming convention violation",
			config: &statemachine.Config{
				Name:         "naming",
				InitialState: "StartState",
				FinalStates:  []string{"EndState"},
				States: []statemachine.StateConfig{
					{Name: "StartState", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
					{Name: "EndState", Type: "final"},
				},
				Transitions: []statemachine.TransitionConfig{
					{From: "StartState", To: "EndState", Condition: "always"},
				},
			},
			wantValid:  true, // Naming violations are warnings, not errors
			wantErrors: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := Validate(tt.config)

			assert.Equal(t, tt.wantValid, result.Valid)

			if len(tt.wantErrors) > 0 {
				require.Len(t, result.Errors, len(tt.wantErrors),
					"expected %d errors, got %d", len(tt.wantErrors), len(result.Errors))

				errorCodes := make(map[string]bool)
				for _, err := range result.Errors {
					errorCodes[err.Code] = true
				}

				for _, wantCode := range tt.wantErrors {
					assert.True(t, errorCodes[wantCode],
						"expected error code %s not found", wantCode)
				}
			}
		})
	}
}

func TestUnreachableStateRule(t *testing.T) {
	t.Parallel()

	rule := &unreachableStateRule{}

	config := &statemachine.Config{
		Name:         "test",
		InitialState: "start",
		FinalStates:  []string{"end"},
		States: []statemachine.StateConfig{
			{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "reachable", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "unreachable", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "end", Type: "final"},
		},
		Transitions: []statemachine.TransitionConfig{
			{From: "start", To: "reachable", Condition: "always"},
			{From: "reachable", To: "end", Condition: "always"},
		},
	}

	result := rule.Check(config)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "UNREACHABLE_STATE", result.Errors[0].Code)
	assert.Contains(t, result.Errors[0].Message, "unreachable")
	assert.NotNil(t, result.Errors[0].Fix)
}

func TestMissingTransitionRule(t *testing.T) {
	t.Parallel()

	rule := &missingTransitionRule{}

	config := &statemachine.Config{
		Name:         "test",
		InitialState: "start",
		FinalStates:  []string{"end"},
		States: []statemachine.StateConfig{
			{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "dead_end", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "end", Type: "final"},
		},
		Transitions: []statemachine.TransitionConfig{
			{From: "start", To: "dead_end", Condition: "always"},
		},
	}

	result := rule.Check(config)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "MISSING_TRANSITION", result.Errors[0].Code)
	assert.Contains(t, result.Errors[0].Message, "dead_end")
}

func TestDuplicateTransitionRule(t *testing.T) {
	t.Parallel()

	rule := &duplicateTransitionRule{}

	config := &statemachine.Config{
		Name:         "test",
		InitialState: "start",
		FinalStates:  []string{"end"},
		States: []statemachine.StateConfig{
			{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "end", Type: "final"},
		},
		Transitions: []statemachine.TransitionConfig{
			{From: "start", To: "end", Condition: "always"},
			{From: "start", To: "end", Condition: "always"},
			{From: "start", To: "end", Condition: "other"},
		},
	}

	result := rule.Check(config)

	require.Len(t, result.Errors, 1)
	assert.Equal(t, "DUPLICATE_TRANSITION", result.Errors[0].Code)
}

func TestNamingConventionRule(t *testing.T) {
	t.Parallel()

	rule := &namingConventionRule{}

	config := &statemachine.Config{
		Name:         "test",
		InitialState: "StartState",
		FinalStates:  []string{"end_state"},
		States: []statemachine.StateConfig{
			{Name: "StartState", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "middleState", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "end_state", Type: "final"},
		},
		Transitions: []statemachine.TransitionConfig{
			{From: "StartState", To: "middleState", Condition: "always"},
			{From: "middleState", To: "end_state", Condition: "always"},
		},
	}

	result := rule.Check(config)

	require.Len(t, result.Warnings, 2)
	assert.Equal(t, "NAMING_CONVENTION", result.Warnings[0].Code)
}

func TestCyclicTransitionRule(t *testing.T) {
	t.Parallel()

	rule := &cyclicTransitionRule{}

	config := &statemachine.Config{
		Name:         "test",
		InitialState: "start",
		FinalStates:  []string{"complete"},
		States: []statemachine.StateConfig{
			{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "loop", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
			{Name: "complete", Type: "final"},
		},
		Transitions: []statemachine.TransitionConfig{
			{From: "start", To: "loop", Condition: "always"},
			{From: "loop", To: "loop", Condition: "retry"},
			{From: "loop", To: "complete", Condition: "done"},
		},
	}

	result := rule.Check(config)

	// Should not report error because loop has path to final state
	assert.Empty(t, result.Errors)
}

func TestValidationResultString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result ValidationResult
		want   string
	}{
		{
			name: "valid result",
			result: ValidationResult{
				Valid: true,
			},
			want: "✓ Configuration is valid",
		},
		{
			name: "result with errors",
			result: ValidationResult{
				Valid: false,
				Errors: []ValidationError{
					{
						Code:     "TEST_ERROR",
						Message:  "Test error message",
						Location: Location{State: "test_state"},
					},
				},
			},
			want: "✗ Configuration has 1 error(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.result.String()
			assert.Contains(t, result, tt.want)
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Parallel()

	t.Run("isSnakeCase", func(t *testing.T) {
		t.Parallel()

		assert.True(t, isSnakeCase("valid_state"))
		assert.True(t, isSnakeCase("another_valid_state_name"))
		assert.False(t, isSnakeCase("InvalidState"))
		assert.False(t, isSnakeCase("invalid-state"))
		assert.False(t, isSnakeCase("invalid state"))
	})

	t.Run("toSnakeCase", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "start_state", toSnakeCase("StartState"))
		assert.Equal(t, "my_state_name", toSnakeCase("MyStateName"))
		assert.Equal(t, "valid_state", toSnakeCase("valid_state"))
		assert.Equal(t, "with_dash", toSnakeCase("with-dash"))
	})

	t.Run("containsUpperCase", func(t *testing.T) {
		t.Parallel()

		assert.True(t, containsUpperCase("StartState"))
		assert.True(t, containsUpperCase("A"))
		assert.False(t, containsUpperCase("valid_state"))
		assert.False(t, containsUpperCase("123"))
	})
}
