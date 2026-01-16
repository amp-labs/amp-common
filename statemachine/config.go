package statemachine

import (
	"fmt"
	"io/fs"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// StateTypeConditional represents a conditional state type.
	StateTypeConditional = "conditional"
)

// ConfigLoader is an interface for loading configurations by name.
// Applications can implement this to provide embedded or custom config loading.
type ConfigLoader interface {
	LoadByName(name string) ([]byte, error)
	ListAvailable() []string
}

var (
	// defaultConfigLoader is the global config loader used by LoadConfig.
	// Applications can set this to provide embedded configs.
	defaultConfigLoader ConfigLoader
)

// SetConfigLoader sets the default config loader for name-based loading.
// This allows applications to provide embedded configs or custom loading logic.
func SetConfigLoader(loader ConfigLoader) {
	defaultConfigLoader = loader
}

// Config defines the structure of a state machine configuration.
type Config struct {
	Name         string             `json:"name"         yaml:"name"`
	InitialState string             `json:"initialState" yaml:"initialState"`
	FinalStates  []string           `json:"finalStates"  yaml:"finalStates"`
	States       []StateConfig      `json:"states"       yaml:"states"`
	Transitions  []TransitionConfig `json:"transitions"  yaml:"transitions"`
}

// StateConfig defines the configuration for a state.
type StateConfig struct {
	Name     string         `json:"name"     yaml:"name"`
	Type     string         `json:"type"     yaml:"type"` // "action", "composite", "conditional", "final"
	Actions  []ActionConfig `json:"actions"  yaml:"actions"`
	OnError  string         `json:"onError"  yaml:"onError"`
	Timeout  string         `json:"timeout"  yaml:"timeout"`
	Metadata map[string]any `json:"metadata" yaml:"metadata"`
}

// ActionConfig defines the configuration for an action.
// Type can be: "sampling", "elicitation", "validation", "sequence", "conditional".
type ActionConfig struct {
	Type       string         `json:"type"       yaml:"type"`
	Name       string         `json:"name"       yaml:"name"`
	Parameters map[string]any `json:"parameters" yaml:"parameters"`
}

// TransitionConfig defines the configuration for a transition.
type TransitionConfig struct {
	From      string `json:"from"      yaml:"from"`
	To        string `json:"to"        yaml:"to"`
	Condition string `json:"condition" yaml:"condition"` // expression or "always"
	Priority  int    `json:"priority"  yaml:"priority"`
}

// LoadConfig loads a state machine configuration by path or name.
// Supports two modes:
//   - Path mode: Pass a file path (containing '/', '\', or ending in '.yaml') to load from filesystem
//     Example: LoadConfig("examples/simple.yaml"), LoadConfig("testdata/config.yaml")
//   - Name mode: Pass a bare name to load via the registered ConfigLoader
//     Example: LoadConfig("guided_setup")
//
// For name mode to work, you must call SetConfigLoader() first with an implementation.
func LoadConfig(pathOrName string) (*Config, error) {
	var (
		data []byte
		err  error
	)

	// Path detection: if input contains path separators or .yaml extension, treat as path
	isPath := strings.Contains(pathOrName, "/") ||
		strings.Contains(pathOrName, `\`) ||
		strings.HasSuffix(strings.ToLower(pathOrName), ".yaml")

	if isPath {
		// Direct filesystem read for arbitrary paths (tests, CLI tools, examples)
		data, err = os.ReadFile(pathOrName) //nolint:gosec // Intentional path-based loading
		if err != nil {
			// Fail fast - don't fallback for explicit paths
			return nil, fmt.Errorf("failed to read config file %q: %w", pathOrName, err)
		}

		return LoadConfigFromBytes(data)
	}

	// Bare name mode: use registered config loader
	if defaultConfigLoader == nil {
		return nil, fmt.Errorf("no config loader registered; use SetConfigLoader() or provide a file path")
	}

	data, err = defaultConfigLoader.LoadByName(pathOrName)
	if err != nil {
		available := defaultConfigLoader.ListAvailable()
		return nil, fmt.Errorf("failed to load config %q (available: %v): %w", pathOrName, available, err)
	}

	return LoadConfigFromBytes(data)
}

// LoadConfigFromBytes loads a state machine configuration from YAML bytes.
func LoadConfigFromBytes(data []byte) (*Config, error) {
	var config Config

	err := yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	err = config.Validate()
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// LoadConfigFromFS loads a configuration from an embedded filesystem.
// This is a convenience function for loading from embed.FS.
func LoadConfigFromFS(fsys fs.FS, path string) (*Config, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config from FS: %w", err)
	}

	return LoadConfigFromBytes(data)
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Name == "" {
		return ErrConfigNameRequired
	}

	if c.InitialState == "" {
		return ErrInitialStateRequired
	}

	if len(c.FinalStates) == 0 {
		return ErrFinalStateRequired
	}

	if len(c.States) == 0 {
		return ErrStateRequired
	}

	// Validate that initial state exists
	if !c.stateExists(c.InitialState) {
		return fmt.Errorf("%w: %s", ErrInitialStateNotFound, c.InitialState)
	}

	// Validate that final states exist
	for _, finalState := range c.FinalStates {
		if !c.stateExists(finalState) {
			return fmt.Errorf("%w: %s", ErrFinalStateNotFound, finalState)
		}
	}

	// Validate states
	stateNames := make(map[string]bool)

	for _, state := range c.States {
		if state.Name == "" {
			return ErrStateNameRequired
		}

		if stateNames[state.Name] {
			return fmt.Errorf("%w: %s", ErrDuplicateStateName, state.Name)
		}

		stateNames[state.Name] = true

		if state.Type == "" {
			return fmt.Errorf("state %s: %w", state.Name, ErrStateTypeRequired)
		}

		// Fail fast for state types that must be programmatic
		if state.Type == StateTypeConditional {
			return fmt.Errorf("state %s: %w", state.Name, ErrConditionalNotImplemented)
		}

		// Validate action states have actions
		if state.Type == "action" && len(state.Actions) == 0 {
			return fmt.Errorf("state %s: %w", state.Name, ErrActionStateMissingAction)
		}

		// Validate actions
		for i, action := range state.Actions {
			if action.Type == "" {
				return fmt.Errorf("state %s, action %d: %w", state.Name, i, ErrActionTypeRequired)
			}

			if action.Name == "" {
				return fmt.Errorf("state %s, action %d: %w", state.Name, i, ErrActionNameRequired)
			}

			// Fail fast for action types that must be programmatic
			if action.Type == "validation" {
				return fmt.Errorf("state %s, action %d: %w", state.Name, i, ErrValidationMustBeProgrammatic)
			}

			if action.Type == StateTypeConditional {
				return fmt.Errorf("state %s, action %d: %w", state.Name, i, ErrConditionalMustBeProgrammatic)
			}
		}
	}

	// Validate transitions
	for i, transition := range c.Transitions {
		if transition.From == "" {
			return fmt.Errorf("transition %d: %w", i, ErrTransitionFromRequired)
		}

		if transition.To == "" {
			return fmt.Errorf("transition %d: %w", i, ErrTransitionToRequired)
		}

		if !c.stateExists(transition.From) {
			return fmt.Errorf("transition %d: %w: %s", i, ErrTransitionFromNotFound, transition.From)
		}

		if !c.stateExists(transition.To) {
			return fmt.Errorf("transition %d: %w: %s", i, ErrTransitionToNotFound, transition.To)
		}
	}

	// Validate reachability (all states should be reachable from initial state)
	// This is a simple check - could be more sophisticated
	// In production, could log warnings for unreachable states
	_ = c.findReachableStates()

	return nil
}

// stateExists checks if a state with the given name exists.
func (c *Config) stateExists(name string) bool {
	for _, state := range c.States {
		if state.Name == name {
			return true
		}
	}

	return false
}

// findReachableStates finds all states reachable from the initial state.
func (c *Config) findReachableStates() map[string]bool {
	reachable := make(map[string]bool)
	reachable[c.InitialState] = true

	// Simple BFS
	queue := []string{c.InitialState}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, transition := range c.Transitions {
			if transition.From == current && !reachable[transition.To] {
				reachable[transition.To] = true
				queue = append(queue, transition.To)
			}
		}
	}

	return reachable
}
