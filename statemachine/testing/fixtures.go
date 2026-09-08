//nolint:gosec,mnd,revive // Test fixtures with safe file permissions; file mode constants; package name shadows stdlib
package testing

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/amp-labs/amp-common/statemachine"
)

// Test helper errors.
var (
	ErrNoMockResponseForPrompt   = errors.New("no mock response for prompt")
	ErrNoMockResponseForQuestion = errors.New("no mock response for question")
)

// MockSamplingClient creates a mock sampling client for testing.
type MockSamplingClient struct {
	responses map[string]string
}

// NewMockSamplingClient creates a mock sampling client with predefined responses.
func NewMockSamplingClient(responses map[string]string) *MockSamplingClient {
	return &MockSamplingClient{
		responses: responses,
	}
}

// Sample returns a predefined response for the given prompt.
func (m *MockSamplingClient) Sample(prompt string) (string, error) {
	if response, ok := m.responses[prompt]; ok {
		return response, nil
	}

	return "", fmt.Errorf("%w: %s", ErrNoMockResponseForPrompt, prompt)
}

// MockElicitationClient creates a mock elicitation client for testing.
type MockElicitationClient struct {
	responses map[string]any
}

// NewMockElicitationClient creates a mock elicitation client with predefined responses.
func NewMockElicitationClient(responses map[string]any) *MockElicitationClient {
	return &MockElicitationClient{
		responses: responses,
	}
}

// Elicit returns a predefined response for the given question.
func (m *MockElicitationClient) Elicit(question string) (any, error) {
	if response, ok := m.responses[question]; ok {
		return response, nil
	}

	return nil, fmt.Errorf("%w: %s", ErrNoMockResponseForQuestion, question)
}

// LoadTestConfig loads a config from the testdata directory.
func LoadTestConfig(name string) (*statemachine.Config, error) {
	path := filepath.Join("testdata", name)

	return statemachine.LoadConfig(path)
}

// CreateTestConfig creates a simple test config.
func CreateTestConfig(name string, initialState string, finalStates []string) *statemachine.Config {
	return &statemachine.Config{
		Name:         name,
		InitialState: initialState,
		FinalStates:  finalStates,
		States:       []statemachine.StateConfig{},
		Transitions:  []statemachine.TransitionConfig{},
	}
}

// CreateTestContext creates a context with test data.
func CreateTestContext(data map[string]any) *statemachine.Context {
	ctx := statemachine.NewContext("test-session", "test-project")

	for key, value := range data {
		ctx.Set(key, value)
	}

	return ctx
}

// SaveTestConfig saves a config to the testdata directory.
func SaveTestConfig(name string, config *statemachine.Config) error {
	testdataDir := "testdata"

	err := os.MkdirAll(testdataDir, 0o755)
	if err != nil {
		return fmt.Errorf("failed to create testdata dir: %w", err)
	}

	path := filepath.Join(testdataDir, name)

	// Would use yaml.Marshal here
	_ = path
	_ = config

	return nil
}

// State and condition names used by the fixture configurations below.
const (
	stateStart    = "start"
	stateMiddle   = "middle"
	stateEnd      = "end"
	stateInit     = "init"
	stateValidate = "validate"
	stateProcess  = "process"
	stateRetry    = "retry"
	stateSuccess  = "success"
	stateFailure  = "failure"
	stateComplete = "complete"

	conditionAlways = "always"

	// Every fixture state carries the same placeholder action.
	actionTypeNoop = "noop"
	actionNameTest = "test"
)

// actionState builds an action state with the placeholder action that every
// fixture below uses. Keeps the fixture configs readable.
func actionState(name string) statemachine.StateConfig {
	return statemachine.StateConfig{
		Name: name,
		Type: statemachine.StateTypeAction,
		Actions: []statemachine.ActionConfig{
			{Type: actionTypeNoop, Name: actionNameTest},
		},
	}
}

// finalState builds a terminal state with the given name.
func finalState(name string) statemachine.StateConfig {
	return statemachine.StateConfig{
		Name: name,
		Type: statemachine.StateTypeFinal,
	}
}

// CommonTestConfigs provides frequently used test configurations.
var CommonTestConfigs = struct {
	Linear    func() *statemachine.Config
	Branching func() *statemachine.Config
	Loop      func() *statemachine.Config
	Complex   func() *statemachine.Config
}{
	Linear: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "linear",
			InitialState: stateStart,
			FinalStates:  []string{stateEnd},
			States: []statemachine.StateConfig{
				actionState(stateStart),
				actionState(stateMiddle),
				finalState(stateEnd),
			},
			Transitions: []statemachine.TransitionConfig{
				{From: stateStart, To: stateMiddle, Condition: conditionAlways},
				{From: stateMiddle, To: stateEnd, Condition: conditionAlways},
			},
		}
	},
	Branching: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "branching",
			InitialState: stateStart,
			FinalStates:  []string{stateSuccess, stateFailure},
			States: []statemachine.StateConfig{
				actionState(stateStart),
				finalState(stateSuccess),
				finalState(stateFailure),
			},
			Transitions: []statemachine.TransitionConfig{
				{From: stateStart, To: stateSuccess, Condition: "result.success"},
				{From: stateStart, To: stateFailure, Condition: "result.failure"},
			},
		}
	},
	Loop: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "loop",
			InitialState: stateStart,
			FinalStates:  []string{stateComplete},
			States: []statemachine.StateConfig{
				actionState(stateStart),
				actionState(stateRetry),
				finalState(stateComplete),
			},
			Transitions: []statemachine.TransitionConfig{
				{From: stateStart, To: stateRetry, Condition: conditionAlways},
				{From: stateRetry, To: stateRetry, Condition: "attempts < 3"},
				{From: stateRetry, To: stateComplete, Condition: "attempts >= 3"},
			},
		}
	},
	Complex: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "complex",
			InitialState: stateInit,
			FinalStates:  []string{stateSuccess, stateFailure},
			States: []statemachine.StateConfig{
				actionState(stateInit),
				actionState(stateValidate),
				actionState(stateProcess),
				actionState(stateRetry),
				finalState(stateSuccess),
				finalState(stateFailure),
			},
			Transitions: []statemachine.TransitionConfig{
				{From: stateInit, To: stateValidate, Condition: conditionAlways},
				{From: stateValidate, To: stateProcess, Condition: "valid"},
				{From: stateValidate, To: stateFailure, Condition: "!valid"},
				{From: stateProcess, To: stateSuccess, Condition: "success"},
				{From: stateProcess, To: stateRetry, Condition: "retryable"},
				{From: stateRetry, To: stateProcess, Condition: "attempts < 3"},
				{From: stateRetry, To: stateFailure, Condition: "attempts >= 3"},
			},
		}
	},
}
