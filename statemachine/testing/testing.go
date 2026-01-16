// Package testing provides testing utilities for state machine workflows.
//
//nolint:err113,varnamelen // Test engine uses dynamic errors; short names idiomatic
package testing

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/statemachine"
	"github.com/stretchr/testify/require"
)

// TestEngine wraps Engine with testing utilities.
type TestEngine struct {
	*statemachine.Engine

	t              *testing.T
	executionTrace []TraceEntry
	assertions     []Assertion
}

// TraceEntry records a single step in execution.
type TraceEntry struct {
	Timestamp time.Time
	State     string
	Action    string
	Duration  time.Duration
	Error     error
	Context   map[string]any // Snapshot of context
}

// Assertion represents a test assertion.
type Assertion struct {
	Name   string
	Passed bool
	Error  error
}

// NewTestEngine creates a test engine for a config with default factory.
func NewTestEngine(t *testing.T, config *statemachine.Config) *TestEngine {
	t.Helper()

	return NewTestEngineWithFactory(t, config, nil)
}

// NewTestEngineWithFactory creates a test engine for a config with a custom factory.
func NewTestEngineWithFactory(
	t *testing.T, config *statemachine.Config, factory *statemachine.ActionFactory,
) *TestEngine {
	t.Helper()

	engine, err := statemachine.NewEngine(config, factory)
	require.NoError(t, err, "failed to create engine")

	te := &TestEngine{
		Engine:         engine,
		t:              t,
		executionTrace: make([]TraceEntry, 0),
		assertions:     make([]Assertion, 0),
	}

	// Add execution hook to record state/action trace
	engine.AddExecutionHook(func(ctx context.Context, actionName string, stateName string, phase string, err error) {
		switch phase {
		case "start":
			// Record state entry
			te.executionTrace = append(te.executionTrace, TraceEntry{
				Timestamp: time.Now(),
				State:     stateName,
				Action:    actionName,
				Duration:  0, // Will be updated on "end" phase
				Error:     nil,
				Context:   nil, // Context snapshot added on "end" phase
			})
		case "end":
			// Update the last trace entry with duration and error
			if len(te.executionTrace) > 0 {
				lastIdx := len(te.executionTrace) - 1
				lastEntry := &te.executionTrace[lastIdx]
				lastEntry.Duration = time.Since(lastEntry.Timestamp)
				lastEntry.Error = err
			}
		}
	})

	return te
}

// Execute runs the state machine and records execution trace.
func (te *TestEngine) Execute(ctx context.Context, smCtx *statemachine.Context) error {
	te.t.Helper()

	// Execute with hooks recording each state transition
	err := te.Engine.Execute(ctx, smCtx)

	return err
}

// AssertStateVisited checks if a state was visited during execution.
func (te *TestEngine) AssertStateVisited(stateName string) {
	te.t.Helper()

	visited := false

	for _, entry := range te.executionTrace {
		if entry.State == stateName {
			visited = true

			break
		}
	}

	assertion := Assertion{
		Name:   fmt.Sprintf("State '%s' was visited", stateName),
		Passed: visited,
	}

	if !visited {
		assertion.Error = fmt.Errorf("%w: '%s'", ErrStateNotVisited, stateName)
	}

	te.assertions = append(te.assertions, assertion)
	require.True(te.t, visited, "state '%s' should have been visited", stateName)
}

// AssertTransitionTaken checks if a specific transition occurred.
func (te *TestEngine) AssertTransitionTaken(from, to string) {
	te.t.Helper()

	taken := false

	for i := range len(te.executionTrace) - 1 {
		if te.executionTrace[i].State == from && te.executionTrace[i+1].State == to {
			taken = true

			break
		}
	}

	assertion := Assertion{
		Name:   fmt.Sprintf("Transition from '%s' to '%s' was taken", from, to),
		Passed: taken,
	}

	if !taken {
		assertion.Error = fmt.Errorf("%w: from '%s' to '%s'", ErrTransitionNotTaken, from, to)
	}

	te.assertions = append(te.assertions, assertion)
	require.True(te.t, taken, "transition from '%s' to '%s' should have been taken", from, to)
}

// AssertFinalState checks the final state matches expected.
func (te *TestEngine) AssertFinalState(expected string) {
	te.t.Helper()

	if len(te.executionTrace) == 0 {
		te.t.Fatal("no execution trace recorded")
	}

	lastEntry := te.executionTrace[len(te.executionTrace)-1]
	actual := lastEntry.State

	assertion := Assertion{
		Name:   fmt.Sprintf("Final state is '%s'", expected),
		Passed: actual == expected,
	}

	if actual != expected {
		assertion.Error = fmt.Errorf("expected final state '%s', got '%s'", expected, actual) //nolint:err113
	}

	te.assertions = append(te.assertions, assertion)
	require.Equal(te.t, expected, actual, "final state should be '%s'", expected)
}

// AssertContextValue checks context value at end of execution.
func (te *TestEngine) AssertContextValue(key string, expected any) {
	te.t.Helper()

	if len(te.executionTrace) == 0 {
		te.t.Fatal("no execution trace recorded")
	}

	lastEntry := te.executionTrace[len(te.executionTrace)-1]
	actual, exists := lastEntry.Context[key]

	assertion := Assertion{
		Name:   fmt.Sprintf("Context[%s] = %v", key, expected),
		Passed: exists && actual == expected,
	}

	if !exists {
		assertion.Error = fmt.Errorf("%w: '%s'", ErrContextKeyNotExist, key)
	} else if actual != expected {
		assertion.Error = fmt.Errorf("%w: expected context[%s] = %v, got %v", ErrContextValueMismatch, key, expected, actual)
	}

	te.assertions = append(te.assertions, assertion)
	require.True(te.t, exists, "context should have key '%s'", key)
	require.Equal(te.t, expected, actual, "context[%s] should equal %v", key, expected)
}

// AssertExecutionTime checks total execution time.
func (te *TestEngine) AssertExecutionTime(maxDuration time.Duration) {
	te.t.Helper()

	if len(te.executionTrace) == 0 {
		te.t.Fatal("no execution trace recorded")
	}

	totalDuration := time.Duration(0)
	for _, entry := range te.executionTrace {
		totalDuration += entry.Duration
	}

	assertion := Assertion{
		Name:   fmt.Sprintf("Execution time < %s", maxDuration),
		Passed: totalDuration <= maxDuration,
	}

	if totalDuration > maxDuration {
		assertion.Error = fmt.Errorf("%w: took %s, max %s", ErrExecutionTooSlow, totalDuration, maxDuration)
	}

	te.assertions = append(te.assertions, assertion)
	require.LessOrEqual(te.t, totalDuration, maxDuration,
		"execution should take less than %s, took %s", maxDuration, totalDuration)
}

// GetTrace returns the execution trace for inspection.
func (te *TestEngine) GetTrace() []TraceEntry {
	return te.executionTrace
}

// GetAssertions returns all assertions made.
func (te *TestEngine) GetAssertions() []Assertion {
	return te.assertions
}
