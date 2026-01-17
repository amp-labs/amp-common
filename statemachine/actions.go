// Package statemachine provides a declarative state machine framework for orchestrating complex workflows.
package statemachine

import (
	"context"
)

// BaseAction provides common functionality for actions.
type BaseAction struct {
	name string
}

func (a *BaseAction) Name() string {
	return a.name
}

// NoopAction is a no-operation action for testing workflows.
type NoopAction struct {
	BaseAction
}

// NewNoopAction creates a new noop action.
func NewNoopAction(name string) *NoopAction {
	return &NoopAction{
		BaseAction: BaseAction{name: name},
	}
}

// Execute does nothing and always succeeds.
func (a *NoopAction) Execute(ctx context.Context, smCtx *Context) error {
	return nil
}

// ValidationAction performs validation with optional sampling for feedback.
type ValidationAction struct {
	BaseAction

	validator func(ctx context.Context, smCtx *Context) (bool, string, error)
}

// NewValidationAction creates a new validation action.
func NewValidationAction(
	name string,
	validator func(ctx context.Context, smCtx *Context) (bool, string, error),
) *ValidationAction {
	return &ValidationAction{
		BaseAction: BaseAction{name: name},
		validator:  validator,
	}
}

func (a *ValidationAction) Execute(ctx context.Context, smCtx *Context) error {
	valid, feedback, err := a.validator(ctx, smCtx)
	if err != nil {
		return err
	}

	smCtx.Set(a.name+"_valid", valid)
	smCtx.Set(a.name+"_feedback", feedback)

	return nil
}
