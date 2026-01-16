// Package validator provides validation and auto-fixing for state machine configurations.
package validator

import (
	"errors"
	"fmt"
	"slices"

	"github.com/amp-labs/amp-common/statemachine"
)

var (
	// ErrTransitionExists is returned when attempting to add a transition that already exists.
	ErrTransitionExists = errors.New("transition already exists")
	// ErrStateNotFound is returned when attempting to remove a state that doesn't exist.
	ErrStateNotFound = errors.New("state not found")
	// ErrDuplicateNotFound is returned when attempting to remove a duplicate that doesn't exist.
	ErrDuplicateNotFound = errors.New("duplicate not found")
	// ErrStateAlreadyExists is returned when attempting to rename to an existing state name.
	ErrStateAlreadyExists = errors.New("state already exists")
	// ErrAlreadyFinalState is returned when attempting to mark a state as final that already is.
	ErrAlreadyFinalState = errors.New("already a final state")
)

// Fix represents an automatic fix for a validation error.
type Fix struct {
	Description string
	Apply       func(config *statemachine.Config) error
}

// AddMissingTransition creates a fix that adds a transition between states.
func AddMissingTransition(from, to string) *Fix {
	return &Fix{
		Description: fmt.Sprintf("Add transition from '%s' to '%s'", from, to),
		Apply: func(config *statemachine.Config) error {
			// Check if transition already exists
			for _, t := range config.Transitions {
				if t.From == from && t.To == to {
					return ErrTransitionExists
				}
			}

			// Add the transition
			config.Transitions = append(config.Transitions, statemachine.TransitionConfig{
				From:      from,
				To:        to,
				Condition: "always",
				Priority:  0,
			})

			return nil
		},
	}
}

// RemoveUnreachableState creates a fix that removes an unreachable state.
func RemoveUnreachableState(stateName string) *Fix {
	return &Fix{
		Description: fmt.Sprintf("Remove unreachable state '%s'", stateName),
		Apply: func(config *statemachine.Config) error {
			// Find and remove the state
			newStates := make([]statemachine.StateConfig, 0, len(config.States))
			found := false

			for _, state := range config.States {
				if state.Name != stateName {
					newStates = append(newStates, state)
				} else {
					found = true
				}
			}

			if !found {
				return fmt.Errorf("%w: '%s'", ErrStateNotFound, stateName)
			}

			config.States = newStates

			// Also remove any transitions involving this state
			newTransitions := make([]statemachine.TransitionConfig, 0, len(config.Transitions))
			for _, t := range config.Transitions {
				if t.From != stateName && t.To != stateName {
					newTransitions = append(newTransitions, t)
				}
			}

			config.Transitions = newTransitions

			return nil
		},
	}
}

// RenameState creates a fix that renames a state.
func RenameState(oldName, newName string) *Fix {
	return &Fix{
		Description: fmt.Sprintf("Rename state from '%s' to '%s'", oldName, newName),
		Apply: func(config *statemachine.Config) error {
			// Check new name doesn't already exist
			for _, state := range config.States {
				if state.Name == newName {
					return fmt.Errorf("%w: '%s'", ErrStateAlreadyExists, newName)
				}
			}

			// Rename in states
			found := false

			for i, state := range config.States {
				if state.Name == oldName {
					config.States[i].Name = newName
					found = true

					break
				}
			}

			if !found {
				return fmt.Errorf("%w: '%s'", ErrStateNotFound, oldName)
			}

			// Update initial state if needed
			if config.InitialState == oldName {
				config.InitialState = newName
			}

			// Update final states if needed
			for i, finalState := range config.FinalStates {
				if finalState == oldName {
					config.FinalStates[i] = newName
				}
			}

			// Update transitions
			for i, t := range config.Transitions {
				if t.From == oldName {
					config.Transitions[i].From = newName
				}

				if t.To == oldName {
					config.Transitions[i].To = newName
				}
			}

			// Update error handlers
			for i, state := range config.States {
				if state.OnError == oldName {
					config.States[i].OnError = newName
				}
			}

			return nil
		},
	}
}

// MarkAsFinalState creates a fix that marks a state as final.
func MarkAsFinalState(stateName string) *Fix {
	return &Fix{
		Description: fmt.Sprintf("Mark '%s' as a final state", stateName),
		Apply: func(config *statemachine.Config) error {
			// Check if already a final state
			if slices.Contains(config.FinalStates, stateName) {
				return fmt.Errorf("%w: '%s'", ErrAlreadyFinalState, stateName)
			}

			// Check state exists
			found := false

			for _, state := range config.States {
				if state.Name == stateName {
					found = true

					break
				}
			}

			if !found {
				return fmt.Errorf("%w: '%s'", ErrStateNotFound, stateName)
			}

			// Add to final states
			config.FinalStates = append(config.FinalStates, stateName)

			return nil
		},
	}
}

// RemoveDuplicateTransition creates a fix that removes a duplicate transition.
func RemoveDuplicateTransition(from, to, condition string) *Fix {
	return &Fix{
		Description: fmt.Sprintf("Remove duplicate transition from '%s' to '%s'", from, to),
		Apply: func(config *statemachine.Config) error {
			newTransitions := make([]statemachine.TransitionConfig, 0, len(config.Transitions))
			found := false
			firstOccurrence := true

			for _, t := range config.Transitions {
				matchesFrom := t.From == from
				matchesTo := t.To == to
				matchesCondition := t.Condition == condition

				if matchesFrom && matchesTo && matchesCondition {
					if firstOccurrence {
						// Keep first occurrence
						newTransitions = append(newTransitions, t)
						firstOccurrence = false
					} else {
						// Skip duplicates
						found = true
					}
				} else {
					newTransitions = append(newTransitions, t)
				}
			}

			if !found {
				return ErrDuplicateNotFound
			}

			config.Transitions = newTransitions

			return nil
		},
	}
}

// ApplyFixes applies a list of fixes to a config.
func ApplyFixes(config *statemachine.Config, fixes []*Fix) error {
	for _, fix := range fixes {
		if fix != nil && fix.Apply != nil {
			err := fix.Apply(config)
			if err != nil {
				return fmt.Errorf("failed to apply fix '%s': %w", fix.Description, err)
			}
		}
	}

	return nil
}
