package statemachine

import "context"

// State represents a single state in the state machine.
type State interface {
	Name() string
	Execute(ctx context.Context, smCtx *Context) (TransitionResult, error)
}

// Transition represents a state transition rule.
type Transition interface {
	From() string
	To() string
	Condition(ctx context.Context, smCtx *Context) (bool, error)
}

// Action represents a composable unit of work.
type Action interface {
	Execute(ctx context.Context, smCtx *Context) error
	Name() string
}

// TransitionResult indicates the outcome of state execution.
type TransitionResult struct {
	NextState string
	Data      map[string]any
	Complete  bool
}
