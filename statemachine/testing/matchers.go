package testing

import (
	"errors"
	"fmt"
	"time"
)

// Matcher errors.
var (
	ErrNoExecutionTrace          = errors.New("no execution trace available")
	ErrExecutionCompletedNoError = errors.New("execution completed without error")
	ErrNoMatchersPassed          = errors.New("no matchers passed")
	ErrStateNotVisited           = errors.New("state was not visited")
	ErrTransitionNotTaken        = errors.New("transition was not taken")
	ErrContextKeyNotExist        = errors.New("context key does not exist")
	ErrContextValueMismatch      = errors.New("context value mismatch")
	ErrExecutionTooSlow          = errors.New("execution exceeded time limit")
)

// Matcher defines an assertion matcher interface.
type Matcher interface {
	Match(engine *TestEngine) (bool, error)
	Description() string
}

// StateWasVisited creates a matcher that checks if a state was visited.
func StateWasVisited(name string) Matcher {
	return &stateVisitedMatcher{stateName: name}
}

type stateVisitedMatcher struct {
	stateName string
}

func (m *stateVisitedMatcher) Match(engine *TestEngine) (bool, error) {
	for _, entry := range engine.executionTrace {
		if entry.State == m.stateName {
			return true, nil
		}
	}

	return false, fmt.Errorf("%w: '%s'", ErrStateNotVisited, m.stateName)
}

func (m *stateVisitedMatcher) Description() string {
	return fmt.Sprintf("state '%s' should be visited", m.stateName)
}

// TransitionWasTaken creates a matcher that checks if a transition occurred.
func TransitionWasTaken(from, to string) Matcher {
	return &transitionTakenMatcher{from: from, to: to}
}

type transitionTakenMatcher struct {
	from string
	to   string
}

func (m *transitionTakenMatcher) Match(engine *TestEngine) (bool, error) {
	for i := range len(engine.executionTrace) - 1 {
		if engine.executionTrace[i].State == m.from && engine.executionTrace[i+1].State == m.to {
			return true, nil
		}
	}

	return false, fmt.Errorf("%w: from '%s' to '%s'", ErrTransitionNotTaken, m.from, m.to)
}

func (m *transitionTakenMatcher) Description() string {
	return fmt.Sprintf("transition from '%s' to '%s' should be taken", m.from, m.to)
}

// ContextContains creates a matcher that checks context values.
func ContextContains(key string, value any) Matcher {
	return &contextContainsMatcher{key: key, value: value}
}

type contextContainsMatcher struct {
	key   string
	value any
}

func (m *contextContainsMatcher) Match(engine *TestEngine) (bool, error) {
	if len(engine.executionTrace) == 0 {
		return false, ErrNoExecutionTrace
	}

	lastEntry := engine.executionTrace[len(engine.executionTrace)-1]
	actual, exists := lastEntry.Context[m.key]

	if !exists {
		return false, fmt.Errorf("%w: '%s'", ErrContextKeyNotExist, m.key)
	}

	if actual != m.value {
		return false, fmt.Errorf("%w: context[%s] = %v, expected %v", ErrContextValueMismatch, m.key, actual, m.value)
	}

	return true, nil
}

func (m *contextContainsMatcher) Description() string {
	return fmt.Sprintf("context should contain %s = %v", m.key, m.value)
}

// ExecutionCompleted creates a matcher that checks if execution completed successfully.
func ExecutionCompleted() Matcher {
	return &executionCompletedMatcher{}
}

type executionCompletedMatcher struct{}

func (m *executionCompletedMatcher) Match(engine *TestEngine) (bool, error) {
	if len(engine.executionTrace) == 0 {
		return false, ErrNoExecutionTrace
	}

	lastEntry := engine.executionTrace[len(engine.executionTrace)-1]
	if lastEntry.Error != nil {
		return false, fmt.Errorf("execution failed with error: %w", lastEntry.Error)
	}

	return true, nil
}

func (m *executionCompletedMatcher) Description() string {
	return "execution should complete successfully"
}

// ExecutionFailed creates a matcher that checks if execution failed.
func ExecutionFailed() Matcher {
	return &executionFailedMatcher{}
}

type executionFailedMatcher struct{}

func (m *executionFailedMatcher) Match(engine *TestEngine) (bool, error) {
	if len(engine.executionTrace) == 0 {
		return false, ErrNoExecutionTrace
	}

	lastEntry := engine.executionTrace[len(engine.executionTrace)-1]
	if lastEntry.Error == nil {
		return false, ErrExecutionCompletedNoError
	}

	return true, nil
}

func (m *executionFailedMatcher) Description() string {
	return "execution should fail"
}

// ExecutionTookLessThan creates a matcher that checks execution duration.
func ExecutionTookLessThan(duration time.Duration) Matcher {
	return &executionDurationMatcher{maxDuration: duration}
}

type executionDurationMatcher struct {
	maxDuration time.Duration
}

func (m *executionDurationMatcher) Match(engine *TestEngine) (bool, error) {
	totalDuration := time.Duration(0)
	for _, entry := range engine.executionTrace {
		totalDuration += entry.Duration
	}

	if totalDuration > m.maxDuration {
		return false, fmt.Errorf("%w: took %s, max %s", ErrExecutionTooSlow, totalDuration, m.maxDuration)
	}

	return true, nil
}

func (m *executionDurationMatcher) Description() string {
	return fmt.Sprintf("execution should take less than %s", m.maxDuration)
}

// All creates a matcher that requires all sub-matchers to pass.
func All(matchers ...Matcher) Matcher {
	return &allMatcher{matchers: matchers}
}

type allMatcher struct {
	matchers []Matcher
}

func (m *allMatcher) Match(engine *TestEngine) (bool, error) {
	for _, matcher := range m.matchers {
		matched, err := matcher.Match(engine)
		if !matched || err != nil {
			return false, err
		}
	}

	return true, nil
}

func (m *allMatcher) Description() string {
	return "all matchers should pass"
}

// Any creates a matcher that requires at least one sub-matcher to pass.
func Any(matchers ...Matcher) Matcher {
	return &anyMatcher{matchers: matchers}
}

type anyMatcher struct {
	matchers []Matcher
}

func (m *anyMatcher) Match(engine *TestEngine) (bool, error) {
	for _, matcher := range m.matchers {
		matched, err := matcher.Match(engine)
		if matched && err == nil {
			return true, nil
		}
	}

	return false, ErrNoMatchersPassed
}

func (m *anyMatcher) Description() string {
	return "at least one matcher should pass"
}
