package statemachine

import (
	"errors"
	"fmt"
)

// Predefined error types.
var (
	ErrStateNotFound           = errors.New("state not found")
	ErrTransitionNotFound      = errors.New("no valid transition found")
	ErrInvalidConfig           = errors.New("invalid configuration")
	ErrSamplingNotAvailable    = errors.New("sampling not available")
	ErrElicitationNotAvailable = errors.New("elicitation not available")
	ErrValidationDataNotFound  = errors.New("validation data not found in context")
	ErrActionFailed            = errors.New("action execution failed")
	ErrTimeout                 = errors.New("state execution timeout")

	// Configuration validation errors.
	ErrConfigNameRequired       = errors.New("config name is required")
	ErrInitialStateRequired     = errors.New("initial state is required")
	ErrFinalStateRequired       = errors.New("at least one final state is required")
	ErrStateRequired            = errors.New("at least one state is required")
	ErrInitialStateNotFound     = errors.New("initial state does not exist")
	ErrFinalStateNotFound       = errors.New("final state does not exist")
	ErrStateNameRequired        = errors.New("state name is required")
	ErrDuplicateStateName       = errors.New("duplicate state name")
	ErrStateTypeRequired        = errors.New("state type is required")
	ErrActionStateMissingAction = errors.New("action state must have at least one action")
	ErrActionTypeRequired       = errors.New("action type is required")
	ErrActionNameRequired       = errors.New("action name is required")
	ErrTransitionFromRequired   = errors.New("transition from state is required")
	ErrTransitionToRequired     = errors.New("transition to state is required")
	ErrTransitionFromNotFound   = errors.New("transition from state does not exist")
	ErrTransitionToNotFound     = errors.New("transition to state does not exist")

	// State type errors.
	ErrConditionalNotImplemented = errors.New("conditional state type not yet implemented")
	ErrUnknownStateType          = errors.New("unknown state type")

	// Action creation errors.
	ErrUnknownActionType            = errors.New("unknown action type")
	ErrSamplingUserRequired         = errors.New("sampling action requires 'user' parameter")
	ErrElicitationMessageRequired   = errors.New("elicitation action requires 'message' parameter")
	ErrElicitationSchemaRequired    = errors.New("elicitation action requires 'schema' parameter")
	ErrInvalidSchemaField           = errors.New("invalid schema field")
	ErrValidationMustBeProgrammatic = errors.New(
		"validation actions must be created programmatically with NewValidationAction",
	)
	ErrSequenceActionsRequired       = errors.New("sequence action requires 'actions' parameter")
	ErrInvalidActionConfig           = errors.New("invalid action config")
	ErrConditionalMustBeProgrammatic = errors.New(
		"conditional actions must be created programmatically with NewConditionalAction",
	)
	ErrRetryActionRequired                  = errors.New("retry action requires 'action' parameter")
	ErrSampleWithFallbackMustBeProgrammatic = errors.New(
		"sample_with_fallback must be created programmatically - see actions.SampleWithFallback",
	)
	ErrElicitFormMustBeProgrammatic = errors.New(
		"elicit_form must be created programmatically - see actions.ElicitForm",
	)
	ErrElicitConfirmationMustBeProgrammatic = errors.New(
		"elicit_confirmation must be created programmatically - see actions.ElicitConfirmation",
	)
	ErrValidateTransitionMustBeProgrammatic = errors.New(
		"validate_transition must be created programmatically - see actions.ValidateTransition",
	)
	ErrTryWithFallbackMustBeProgrammatic = errors.New(
		"try_with_fallback must be created programmatically - see actions.TryWithFallback",
	)
	ErrValidatedSequenceMustBeProgrammatic = errors.New(
		"validated_sequence must be created programmatically - see actions.ValidatedSequence",
	)
	ErrRetryWithBackoffMustBeProgrammatic = errors.New(
		"retry_with_backoff must be created programmatically - see actions.RetryWithBackoff",
	)

	// Expression errors.
	ErrInvalidExpression     = errors.New("invalid expression")
	ErrUnsupportedExpression = errors.New("unsupported expression")

	// Test errors (used in test files).
	ErrTestActionFailed  = errors.New("action failed")
	ErrTestAction2Failed = errors.New("action2 failed")
	ErrTestTemporary     = errors.New("temporary error")
	ErrTestPermanent     = errors.New("permanent error")
)

// StateError wraps an error with state context.
type StateError struct {
	State string
	Err   error
}

func (e *StateError) Error() string {
	return fmt.Sprintf("state %s: %v", e.State, e.Err)
}

func (e *StateError) Unwrap() error {
	return e.Err
}

// TransitionError wraps an error with transition context.
type TransitionError struct {
	From string
	To   string
	Err  error
}

func (e *TransitionError) Error() string {
	if e.To == "" {
		return fmt.Sprintf("transition from %s: %v", e.From, e.Err)
	}

	return fmt.Sprintf("transition %s -> %s: %v", e.From, e.To, e.Err)
}

func (e *TransitionError) Unwrap() error {
	return e.Err
}

// WrapStateError wraps an error with state context.
func WrapStateError(state string, err error) error {
	if err == nil {
		return nil
	}

	return &StateError{
		State: state,
		Err:   err,
	}
}

// WrapTransitionError wraps an error with transition context.
func WrapTransitionError(from, to string, err error) error {
	if err == nil {
		return nil
	}

	return &TransitionError{
		From: from,
		To:   to,
		Err:  err,
	}
}
