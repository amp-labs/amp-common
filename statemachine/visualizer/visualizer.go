// Package visualizer generates visual diagrams from state machine configurations.
//
//nolint:gosec,varnamelen,noinlineerr // File paths from config; short names idiomatic; inline error checks
package visualizer

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/amp-labs/amp-common/statemachine"
	"gopkg.in/yaml.v3"
)

// Visualizer errors.
var (
	ErrConfigNil      = errors.New("config cannot be nil")
	ErrNoInitialState = errors.New("config must have an initial state")
)

// GenerateMermaid converts a Config to a Mermaid state diagram.
func GenerateMermaid(config *statemachine.Config) (string, error) {
	return GenerateMermaidWithOptions(config, DefaultOptions())
}

// GenerateMermaidFromFile loads a config from a file and generates a Mermaid diagram.
func GenerateMermaidFromFile(path string) (string, error) {
	config, err := loadConfig(path)
	if err != nil {
		return "", fmt.Errorf("failed to load config: %w", err)
	}

	return GenerateMermaid(config)
}

// GenerateMermaidWithOptions generates a Mermaid diagram with custom options.
func GenerateMermaidWithOptions(config *statemachine.Config, opts Options) (string, error) {
	if config == nil {
		return "", ErrConfigNil
	}

	if config.InitialState == "" {
		return "", ErrNoInitialState
	}

	var sb strings.Builder

	// Header
	sb.WriteString("```mermaid\n")
	sb.WriteString(fmt.Sprintf("stateDiagram-%s\n", opts.Direction))

	// Initial state marker
	sb.WriteString(fmt.Sprintf("    [*] --> %s\n", config.InitialState))

	// Build highlight map for quick lookup
	highlightMap := make(map[string]bool)
	for _, state := range opts.HighlightPath {
		highlightMap[state] = true
	}

	// Build final states map for quick lookup
	finalStatesMap := make(map[string]bool)
	for _, finalState := range config.FinalStates {
		finalStatesMap[finalState] = true
	}

	// Build transition map: from state -> list of transitions
	transitionMap := make(map[string][]statemachine.TransitionConfig)
	for _, transition := range config.Transitions {
		transitionMap[transition.From] = append(transitionMap[transition.From], transition)
	}

	// Process each state
	for _, state := range config.States {
		// State declaration with description if actions shown
		if opts.ShowActions && len(state.Actions) > 0 {
			actionNames := make([]string, len(state.Actions))
			for i, action := range state.Actions {
				actionNames[i] = action.Type
			}

			sb.WriteString(fmt.Sprintf("    %s: %s\\n[%s]\n",
				state.Name, state.Name, strings.Join(actionNames, ", ")))
		}

		isFinal := finalStatesMap[state.Name]

		// Apply styling based on state type and highlighting
		switch {
		case highlightMap[state.Name]:
			sb.WriteString(fmt.Sprintf("    class %s highlighted\n", state.Name))
		case isFinal:
			sb.WriteString(fmt.Sprintf("    class %s finalState\n", state.Name))
		case len(state.Actions) > 0:
			sb.WriteString(fmt.Sprintf("    class %s actionState\n", state.Name))
		}

		// Add transitions from this state
		transitions := transitionMap[state.Name]
		for _, transition := range transitions {
			transitionLabel := ""
			if opts.ShowConditions && transition.Condition != "" && transition.Condition != "always" {
				transitionLabel = ": " + transition.Condition
			}

			sb.WriteString(fmt.Sprintf("    %s --> %s%s\n",
				state.Name, transition.To, transitionLabel))
		}

		// Mark final states
		if isFinal {
			sb.WriteString(fmt.Sprintf("    %s --> [*]\n", state.Name))
		}
	}

	// Add class definitions based on theme
	sb.WriteString("\n")
	sb.WriteString("    classDef actionState fill:#e1f5ff,stroke:#01579b,stroke-width:2px\n")
	sb.WriteString("    classDef finalState fill:#c8e6c9,stroke:#2e7d32,stroke-width:2px\n")
	sb.WriteString("    classDef highlighted fill:#fff9c4,stroke:#f57f17,stroke-width:3px\n")

	sb.WriteString("```\n")

	return sb.String(), nil
}

// loadConfig loads a state machine config from a YAML file.
func loadConfig(path string) (*statemachine.Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var config statemachine.Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &config, nil
}
