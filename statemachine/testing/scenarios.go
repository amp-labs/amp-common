package testing

import (
	"context"
	"testing"

	"github.com/amp-labs/amp-common/statemachine"
)

// TestScenario represents a complete test scenario for a state machine.
type TestScenario struct {
	Name           string
	Config         *statemachine.Config
	InitialContext map[string]any
	MockResponses  map[string]any
	Assertions     []Assertion
}

// RunScenario executes a test scenario and validates results.
func RunScenario(t *testing.T, scenario TestScenario) {
	t.Helper()
	t.Run(scenario.Name, func(t *testing.T) {
		// Create test engine
		engine := NewTestEngine(t, scenario.Config)

		// Set up context
		ctx := context.Background()
		smCtx := CreateTestContext(scenario.InitialContext)

		// Execute
		err := engine.Execute(ctx, smCtx)

		// Validate based on scenario expectations
		if err != nil && len(scenario.Assertions) > 0 {
			t.Fatalf("Execution failed: %v", err)
		}

		// Run custom assertions
		for _, assertion := range scenario.Assertions {
			if !assertion.Passed {
				t.Errorf("Assertion failed: %s - %v", assertion.Name, assertion.Error)
			}
		}
	})
}

// LinearWorkflowScenario creates a scenario for testing linear workflows.
func LinearWorkflowScenario() TestScenario {
	return TestScenario{
		Name:           "Linear Workflow",
		Config:         CommonTestConfigs.Linear(),
		InitialContext: map[string]any{},
		Assertions:     []Assertion{},
	}
}

// BranchingWorkflowScenario creates a scenario for testing branching logic.
func BranchingWorkflowScenario() TestScenario {
	return TestScenario{
		Name:   "Branching Workflow",
		Config: CommonTestConfigs.Branching(),
		InitialContext: map[string]any{
			"result.success": true,
		},
		Assertions: []Assertion{},
	}
}

// ErrorRecoveryScenario creates a scenario for testing error recovery.
func ErrorRecoveryScenario() TestScenario {
	return TestScenario{
		Name:   "Error Recovery",
		Config: CommonTestConfigs.Complex(),
		InitialContext: map[string]any{
			"force_error": true,
		},
		Assertions: []Assertion{},
	}
}

// RetryScenario creates a scenario for testing retry logic.
func RetryScenario() TestScenario {
	return TestScenario{
		Name:   "Retry Logic",
		Config: CommonTestConfigs.Loop(),
		InitialContext: map[string]any{
			"attempts": 0,
		},
		Assertions: []Assertion{},
	}
}
