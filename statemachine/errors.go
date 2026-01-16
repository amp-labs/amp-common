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

	// ErrConfigNameRequired indicates that a configuration name is required.
	ErrConfigNameRequired = errors.New("config name is required")
	// ErrInitialStateRequired indicates that an initial state is required.
	ErrInitialStateRequired = errors.New("initial state is required")
	// ErrFinalStateRequired indicates that at least one final state is required.
	ErrFinalStateRequired = errors.New("at least one final state is required")
	// ErrStateRequired indicates that at least one state is required.
	ErrStateRequired = errors.New("at least one state is required")
	// ErrInitialStateNotFound indicates that the initial state does not exist.
	ErrInitialStateNotFound = errors.New("initial state does not exist")
	// ErrFinalStateNotFound indicates that a final state does not exist.
	ErrFinalStateNotFound = errors.New("final state does not exist")
	// ErrStateNameRequired indicates that a state name is required.
	ErrStateNameRequired = errors.New("state name is required")
	// ErrDuplicateStateName indicates that a duplicate state name was found.
	ErrDuplicateStateName = errors.New("duplicate state name")
	// ErrStateTypeRequired indicates that a state type is required.
	ErrStateTypeRequired = errors.New("state type is required")
	// ErrActionStateMissingAction indicates that an action state must have at least one action.
	ErrActionStateMissingAction = errors.New("action state must have at least one action")
	// ErrActionTypeRequired indicates that an action type is required.
	ErrActionTypeRequired = errors.New("action type is required")
	// ErrActionNameRequired indicates that an action name is required.
	ErrActionNameRequired = errors.New("action name is required")
	// ErrTransitionFromRequired indicates that a transition from state is required.
	ErrTransitionFromRequired = errors.New("transition from state is required")
	// ErrTransitionToRequired indicates that a transition to state is required.
	ErrTransitionToRequired = errors.New("transition to state is required")
	// ErrTransitionFromNotFound indicates that a transition from state does not exist.
	ErrTransitionFromNotFound = errors.New("transition from state does not exist")
	// ErrTransitionToNotFound indicates that a transition to state does not exist.
	ErrTransitionToNotFound = errors.New("transition to state does not exist")
	// ErrNoConfigLoader indicates that no config loader is registered.
	ErrNoConfigLoader = errors.New("no config loader registered; use SetConfigLoader() or provide a file path")

	// ErrConditionalNotImplemented indicates that conditional state types are not yet implemented.
	ErrConditionalNotImplemented = errors.New("conditional state type not yet implemented")
	// ErrUnknownStateType indicates that an unknown state type was encountered.
	ErrUnknownStateType = errors.New("unknown state type")

	// ErrUnknownActionType indicates that an unknown action type was encountered.
	ErrUnknownActionType = errors.New("unknown action type")
	// ErrInvalidActionFormat indicates that an action has an invalid format.
	ErrInvalidActionFormat = errors.New("invalid action format")
	// ErrSamplingUserRequired indicates that a sampling action requires a 'user' parameter.
	ErrSamplingUserRequired = errors.New("sampling action requires 'user' parameter")
	// ErrElicitationMessageRequired indicates that an elicitation action requires a 'message' parameter.
	ErrElicitationMessageRequired = errors.New("elicitation action requires 'message' parameter")
	// ErrElicitationSchemaRequired indicates that an elicitation action requires a 'schema' parameter.
	ErrElicitationSchemaRequired = errors.New("elicitation action requires 'schema' parameter")
	// ErrInvalidSchemaField indicates that a schema field is invalid.
	ErrInvalidSchemaField = errors.New("invalid schema field")
	// ErrValidationMustBeProgrammatic indicates that validation actions must be created programmatically.
	ErrValidationMustBeProgrammatic = errors.New(
		"validation actions must be created programmatically with NewValidationAction",
	)
	// ErrSequenceActionsRequired indicates that a sequence action requires an 'actions' parameter.
	ErrSequenceActionsRequired = errors.New("sequence action requires 'actions' parameter")
	// ErrInvalidActionConfig indicates that an action config is invalid.
	ErrInvalidActionConfig = errors.New("invalid action config")
	// ErrConditionalMustBeProgrammatic indicates that conditional actions must be created programmatically.
	ErrConditionalMustBeProgrammatic = errors.New(
		"conditional actions must be created programmatically with NewConditionalAction",
	)
	// ErrRetryActionRequired indicates that a retry action requires an 'action' parameter.
	ErrRetryActionRequired = errors.New("retry action requires 'action' parameter")
	// ErrSampleWithFallbackMustBeProgrammatic indicates that sample_with_fallback must be created programmatically.
	ErrSampleWithFallbackMustBeProgrammatic = errors.New(
		"sample_with_fallback must be created programmatically - see actions.SampleWithFallback",
	)
	// ErrElicitFormMustBeProgrammatic indicates that elicit_form must be created programmatically.
	ErrElicitFormMustBeProgrammatic = errors.New(
		"elicit_form must be created programmatically - see actions.ElicitForm",
	)
	// ErrElicitConfirmationMustBeProgrammatic indicates that elicit_confirmation must be created programmatically.
	ErrElicitConfirmationMustBeProgrammatic = errors.New(
		"elicit_confirmation must be created programmatically - see actions.ElicitConfirmation",
	)
	// ErrValidateTransitionMustBeProgrammatic indicates that validate_transition must be created programmatically.
	ErrValidateTransitionMustBeProgrammatic = errors.New(
		"validate_transition must be created programmatically - see actions.ValidateTransition",
	)
	// ErrTryWithFallbackMustBeProgrammatic indicates that try_with_fallback must be created programmatically.
	ErrTryWithFallbackMustBeProgrammatic = errors.New(
		"try_with_fallback must be created programmatically - see actions.TryWithFallback",
	)
	// ErrValidatedSequenceMustBeProgrammatic indicates that validated_sequence must be created programmatically.
	ErrValidatedSequenceMustBeProgrammatic = errors.New(
		"validated_sequence must be created programmatically - see actions.ValidatedSequence",
	)
	// ErrRetryWithBackoffMustBeProgrammatic indicates that retry_with_backoff must be created programmatically.
	ErrRetryWithBackoffMustBeProgrammatic = errors.New(
		"retry_with_backoff must be created programmatically - see actions.RetryWithBackoff",
	)

	// ErrInvalidExpression indicates that an expression is invalid.
	ErrInvalidExpression = errors.New("invalid expression")
	// ErrUnsupportedExpression indicates that an expression is unsupported.
	ErrUnsupportedExpression = errors.New("unsupported expression")

	// ErrTestActionFailed is used in test files to indicate that an action failed.
	ErrTestActionFailed = errors.New("action failed")
	// ErrTestAction2Failed is used in test files to indicate that action2 failed.
	ErrTestAction2Failed = errors.New("action2 failed")
	// ErrTestTemporary is used in test files to indicate a temporary error.
	ErrTestTemporary = errors.New("temporary error")
	// ErrTestPermanent is used in test files to indicate a permanent error.
	ErrTestPermanent = errors.New("permanent error")
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
