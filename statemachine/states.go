package statemachine

import "context"

// ActionState executes a single action.
type ActionState struct {
	name      string
	action    Action
	nextState string
}

// NewActionState creates a new action state.
func NewActionState(name string, action Action, next string) *ActionState {
	return &ActionState{
		name:      name,
		action:    action,
		nextState: next,
	}
}

func (s *ActionState) Name() string {
	return s.name
}

func (s *ActionState) Execute(ctx context.Context, smCtx *Context) (TransitionResult, error) {
	err := s.action.Execute(ctx, smCtx)
	if err != nil {
		return TransitionResult{}, err
	}

	return TransitionResult{
		NextState: s.nextState,
		Data:      map[string]any{},
		Complete:  false,
	}, nil
}

// ConditionalState branches based on context data.
type ConditionalState struct {
	name      string
	condition func(ctx context.Context, smCtx *Context) (string, error)
}

// NewConditionalState creates a new conditional state.
func NewConditionalState(
	name string,
	cond func(ctx context.Context, smCtx *Context) (string, error),
) *ConditionalState {
	return &ConditionalState{
		name:      name,
		condition: cond,
	}
}

func (s *ConditionalState) Name() string {
	return s.name
}

func (s *ConditionalState) Execute(ctx context.Context, smCtx *Context) (TransitionResult, error) {
	nextState, err := s.condition(ctx, smCtx)
	if err != nil {
		return TransitionResult{}, err
	}

	return TransitionResult{
		NextState: nextState,
		Complete:  false,
	}, nil
}

// FinalState marks completion.
type FinalState struct {
	name string
}

// NewFinalState creates a new final state.
func NewFinalState(name string) *FinalState {
	return &FinalState{
		name: name,
	}
}

func (s *FinalState) Name() string {
	return s.name
}

func (s *FinalState) Execute(ctx context.Context, smCtx *Context) (TransitionResult, error) {
	return TransitionResult{
		NextState: s.name,
		Complete:  true,
	}, nil
}
