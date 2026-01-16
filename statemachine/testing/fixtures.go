// Package testing provides testing utilities for state machine workflows.
//
//nolint:gosec,mnd // Test fixtures with safe file permissions; file mode constants
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
			InitialState: "start",
			FinalStates:  []string{"end"},
			States: []statemachine.StateConfig{
				{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "middle", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "end", Type: "final"},
			},
			Transitions: []statemachine.TransitionConfig{
				{From: "start", To: "middle", Condition: "always"},
				{From: "middle", To: "end", Condition: "always"},
			},
		}
	},
	Branching: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "branching",
			InitialState: "start",
			FinalStates:  []string{"success", "failure"},
			States: []statemachine.StateConfig{
				{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "success", Type: "final"},
				{Name: "failure", Type: "final"},
			},
			Transitions: []statemachine.TransitionConfig{
				{From: "start", To: "success", Condition: "result.success"},
				{From: "start", To: "failure", Condition: "result.failure"},
			},
		}
	},
	Loop: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "loop",
			InitialState: "start",
			FinalStates:  []string{"complete"},
			States: []statemachine.StateConfig{
				{Name: "start", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "retry", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "complete", Type: "final"},
			},
			Transitions: []statemachine.TransitionConfig{
				{From: "start", To: "retry", Condition: "always"},
				{From: "retry", To: "retry", Condition: "attempts < 3"},
				{From: "retry", To: "complete", Condition: "attempts >= 3"},
			},
		}
	},
	Complex: func() *statemachine.Config {
		return &statemachine.Config{
			Name:         "complex",
			InitialState: "init",
			FinalStates:  []string{"success", "failure"},
			States: []statemachine.StateConfig{
				{Name: "init", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "validate", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "process", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "retry", Type: "action", Actions: []statemachine.ActionConfig{{Type: "noop", Name: "test"}}},
				{Name: "success", Type: "final"},
				{Name: "failure", Type: "final"},
			},
			Transitions: []statemachine.TransitionConfig{
				{From: "init", To: "validate", Condition: "always"},
				{From: "validate", To: "process", Condition: "valid"},
				{From: "validate", To: "failure", Condition: "!valid"},
				{From: "process", To: "success", Condition: "success"},
				{From: "process", To: "retry", Condition: "retryable"},
				{From: "retry", To: "process", Condition: "attempts < 3"},
				{From: "retry", To: "failure", Condition: "attempts >= 3"},
			},
		}
	},
}
