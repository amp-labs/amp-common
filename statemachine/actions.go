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

// Execute does nothing and always succeeds.
func (a *NoopAction) Execute(ctx context.Context, smCtx *Context) error {
	return nil
}

// NewNoopAction creates a new noop action.
func NewNoopAction(name string) *NoopAction {
	return &NoopAction{
		BaseAction: BaseAction{name: name},
	}
}
