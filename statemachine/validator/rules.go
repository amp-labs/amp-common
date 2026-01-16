//nolint:lll,mnd // Long validation messages; arithmetic for case conversion
package validator

import (
	"fmt"

	"github.com/amp-labs/amp-common/statemachine"
)

// Severity defines the severity level of a validation issue.
type Severity int

const (
	SeverityError Severity = iota
	SeverityWarning
	SeverityInfo
)

// RuleResult contains both errors and warnings from a rule check.
type RuleResult struct {
	Errors   []ValidationError
	Warnings []ValidationWarning
}

// Rule defines a validation rule that can check a config for specific issues.
type Rule interface {
	Name() string
	Severity() Severity
	Check(config *statemachine.Config) RuleResult
}

// DefaultRules returns the standard set of validation rules.
func DefaultRules() []Rule {
	return []Rule{
		&unreachableStateRule{},
		&missingTransitionRule{},
		&duplicateTransitionRule{},
		&namingConventionRule{},
		&cyclicTransitionRule{},
	}
}

// RegisteredRules stores custom validation rules.
var RegisteredRules []Rule

// RegisterRule adds a custom validation rule.
func RegisterRule(rule Rule) {
	RegisteredRules = append(RegisteredRules, rule)
}

// unreachableStateRule checks for states that cannot be reached from the initial state.
type unreachableStateRule struct{}

func (r *unreachableStateRule) Name() string {
	return "UnreachableState"
}

func (r *unreachableStateRule) Severity() Severity {
	return SeverityError
}

func (r *unreachableStateRule) Check(config *statemachine.Config) RuleResult {
	var errors []ValidationError

	// Find all reachable states using BFS
	reachable := make(map[string]bool)
	reachable[config.InitialState] = true

	queue := []string{config.InitialState}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, transition := range config.Transitions {
			if transition.From == current && !reachable[transition.To] {
				reachable[transition.To] = true
				queue = append(queue, transition.To)
			}
		}
	}

	// Check each state for reachability
	for _, state := range config.States {
		if !reachable[state.Name] && state.Name != config.InitialState {
			errors = append(errors, ValidationError{
				Code:     "UNREACHABLE_STATE",
				Message:  fmt.Sprintf("State '%s' cannot be reached from initial state '%s'", state.Name, config.InitialState),
				Location: Location{State: state.Name},
				Fix: &Fix{
					Description: fmt.Sprintf("Add a transition to '%s' or remove the state", state.Name),
					Apply:       nil, // Would implement actual fix function
				},
			})
		}
	}

	return RuleResult{Errors: errors}
}

// missingTransitionRule checks for non-final states without outgoing transitions.
type missingTransitionRule struct{}

func (r *missingTransitionRule) Name() string {
	return "MissingTransition"
}

func (r *missingTransitionRule) Severity() Severity {
	return SeverityError
}

func (r *missingTransitionRule) Check(config *statemachine.Config) RuleResult {
	var errors []ValidationError

	// Build map of final states
	finalStates := make(map[string]bool)
	for _, finalState := range config.FinalStates {
		finalStates[finalState] = true
	}

	// Build map of states with outgoing transitions
	hasOutgoing := make(map[string]bool)
	for _, transition := range config.Transitions {
		hasOutgoing[transition.From] = true
	}

	// Check each non-final state has outgoing transitions
	for _, state := range config.States {
		if !finalStates[state.Name] && !hasOutgoing[state.Name] {
			errors = append(errors, ValidationError{
				Code:     "MISSING_TRANSITION",
				Message:  fmt.Sprintf("Non-final state '%s' has no outgoing transitions", state.Name),
				Location: Location{State: state.Name},
				Fix: &Fix{
					Description: "Add a transition or mark as final state",
					Apply:       nil,
				},
			})
		}
	}

	return RuleResult{Errors: errors}
}

// duplicateTransitionRule checks for duplicate transitions with same from/to/condition.
type duplicateTransitionRule struct{}

func (r *duplicateTransitionRule) Name() string {
	return "DuplicateTransition"
}

func (r *duplicateTransitionRule) Severity() Severity {
	return SeverityError
}

func (r *duplicateTransitionRule) Check(config *statemachine.Config) RuleResult {
	var errors []ValidationError

	seen := make(map[string]bool)

	for i, transition := range config.Transitions {
		key := fmt.Sprintf("%s->%s:%s", transition.From, transition.To, transition.Condition)
		if seen[key] {
			errors = append(errors, ValidationError{
				Code:    "DUPLICATE_TRANSITION",
				Message: fmt.Sprintf("Duplicate transition from '%s' to '%s' with condition '%s'", transition.From, transition.To, transition.Condition),
				Location: Location{
					State: transition.From,
					Line:  i + 1,
				},
				Fix: &Fix{
					Description: "Remove duplicate transition",
					Apply:       nil,
				},
			})
		}

		seen[key] = true
	}

	return RuleResult{Errors: errors}
}

// namingConventionRule warns about naming convention violations.
type namingConventionRule struct{}

func (r *namingConventionRule) Name() string {
	return "NamingConvention"
}

func (r *namingConventionRule) Severity() Severity {
	return SeverityWarning
}

func (r *namingConventionRule) Check(config *statemachine.Config) RuleResult {
	var warnings []ValidationWarning

	// Check state names are snake_case
	for _, state := range config.States {
		if !isSnakeCase(state.Name) {
			warnings = append(warnings, ValidationWarning{
				Code:     "NAMING_CONVENTION",
				Message:  fmt.Sprintf("State '%s' should use snake_case naming (suggested: '%s')", state.Name, toSnakeCase(state.Name)),
				Location: Location{State: state.Name},
			})
		}
	}

	return RuleResult{Warnings: warnings}
}

// cyclicTransitionRule detects potential infinite loops without exit conditions.
type cyclicTransitionRule struct{}

func (r *cyclicTransitionRule) Name() string {
	return "CyclicTransition"
}

func (r *cyclicTransitionRule) Severity() Severity {
	return SeverityWarning
}

func (r *cyclicTransitionRule) Check(config *statemachine.Config) RuleResult {
	var warnings []ValidationWarning

	// Build adjacency list
	graph := make(map[string][]string)
	for _, transition := range config.Transitions {
		graph[transition.From] = append(graph[transition.From], transition.To)
	}

	// Detect cycles using DFS
	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var dfs func(string) bool

	dfs = func(state string) bool {
		visited[state] = true
		recStack[state] = true

		for _, next := range graph[state] {
			if !visited[next] {
				if dfs(next) {
					return true
				}
			} else if recStack[next] {
				// Cycle detected - check if it has exit condition
				hasFinalExit := false

				for _, finalState := range config.FinalStates {
					if reachableFrom(state, finalState, graph, make(map[string]bool)) {
						hasFinalExit = true

						break
					}
				}

				if !hasFinalExit {
					warnings = append(warnings, ValidationWarning{
						Code:     "POTENTIAL_INFINITE_LOOP",
						Message:  fmt.Sprintf("Cycle detected involving state '%s' with no clear exit to final state", state),
						Location: Location{State: state},
					})
				}

				return true
			}
		}

		recStack[state] = false

		return false
	}

	for _, state := range config.States {
		if !visited[state.Name] {
			dfs(state.Name)
		}
	}

	return RuleResult{Warnings: warnings}
}

// Helper functions

func isSnakeCase(s string) bool {
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			return false
		}

		if r == '-' || r == ' ' {
			return false
		}
	}

	return true
}

func toSnakeCase(s string) string {
	var result []rune

	for i, r := range s {
		switch {
		case r >= 'A' && r <= 'Z':
			if i > 0 {
				result = append(result, '_')
			}

			result = append(result, r+32) // Convert to lowercase
		case r == '-' || r == ' ':
			result = append(result, '_')
		default:
			result = append(result, r)
		}
	}

	return string(result)
}

func reachableFrom(from, to string, graph map[string][]string, visited map[string]bool) bool {
	if from == to {
		return true
	}

	if visited[from] {
		return false
	}

	visited[from] = true
	for _, next := range graph[from] {
		if reachableFrom(next, to, graph, visited) {
			return true
		}
	}

	return false
}
