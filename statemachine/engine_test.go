package statemachine

import (
	"context"
	"errors"
	"testing"
)

// mockAction is a simple action for testing.
type mockAction struct {
	name     string
	executed bool
	err      error
}

func (m *mockAction) Name() string {
	return m.name
}

func (m *mockAction) Execute(ctx context.Context, smCtx *Context) error {
	m.executed = true
	if m.err != nil {
		return m.err
	}

	smCtx.Set(m.name+"_executed", true)

	return nil
}

func TestEngineExecution(t *testing.T) {
	t.Parallel()

	// Create simple state machine: start -> middle -> end
	action1 := &mockAction{name: "action1"}
	action2 := &mockAction{name: "action2"}

	// Build engine manually since we can't easily inject mock actions via config
	engine := &Engine{
		states:       make(map[string]State),
		transitions:  []Transition{},
		initialState: "start",
		finalStates:  []string{"end"},
	}

	engine.RegisterState(NewActionState("start", action1, "middle"))
	engine.RegisterState(NewActionState("middle", action2, "end"))
	engine.RegisterState(NewFinalState("end"))

	engine.RegisterTransition(NewSimpleTransition("start", "middle"))
	engine.RegisterTransition(NewSimpleTransition("middle", "end"))

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := engine.Execute(ctx, smCtx)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if !action1.executed {
		t.Error("action1 was not executed")
	}

	if !action2.executed {
		t.Error("action2 was not executed")
	}

	if smCtx.CurrentState != "end" {
		t.Errorf("expected final state 'end', got %s", smCtx.CurrentState)
	}
}

func TestEngineWithConditionalTransitions(t *testing.T) {
	t.Parallel()

	action := &mockAction{name: "set_value"}

	engine := &Engine{
		states:       make(map[string]State),
		transitions:  []Transition{},
		initialState: "start",
		finalStates:  []string{"success", "failure"},
	}

	engine.RegisterState(NewActionState("start", action, ""))
	engine.RegisterState(NewFinalState("success"))
	engine.RegisterState(NewFinalState("failure"))

	// Conditional transition based on context
	engine.RegisterTransition(NewConditionalTransition("start", "success",
		func(ctx context.Context, smCtx *Context) (bool, error) {
			val, ok := smCtx.GetBool("set_value_executed")

			return ok && val, nil
		}))

	engine.RegisterTransition(NewConditionalTransition("start", "failure",
		func(ctx context.Context, smCtx *Context) (bool, error) {
			val, ok := smCtx.GetBool("set_value_executed")

			return !ok || !val, nil
		}))

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := engine.Execute(ctx, smCtx)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if smCtx.CurrentState != "success" {
		t.Errorf("expected final state 'success', got %s", smCtx.CurrentState)
	}
}

func TestEngineErrorHandling(t *testing.T) {
	t.Parallel()

	expectedErr := ErrTestActionFailed
	action := &mockAction{name: "failing_action", err: expectedErr}

	engine := &Engine{
		states:       make(map[string]State),
		transitions:  []Transition{},
		initialState: "start",
		finalStates:  []string{"end"},
	}

	engine.RegisterState(NewActionState("start", action, "end"))
	engine.RegisterState(NewFinalState("end"))

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := engine.Execute(ctx, smCtx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var stateErr *StateError
	if !errors.As(err, &stateErr) {
		t.Errorf("expected StateError, got %T", err)
	}

	if stateErr.State != "start" {
		t.Errorf("expected error from state 'start', got %s", stateErr.State)
	}
}

func TestEngineStateNotFound(t *testing.T) {
	t.Parallel()

	engine := &Engine{
		states:       make(map[string]State),
		transitions:  []Transition{},
		initialState: "nonexistent",
		finalStates:  []string{"end"},
	}

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := engine.Execute(ctx, smCtx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrStateNotFound) {
		t.Errorf("expected ErrStateNotFound, got %v", err)
	}
}

func TestEngineTransitionNotFound(t *testing.T) {
	t.Parallel()

	action := &mockAction{name: "action"}

	engine := &Engine{
		states:       make(map[string]State),
		transitions:  []Transition{},
		initialState: "start",
		finalStates:  []string{"end"},
	}

	engine.RegisterState(NewActionState("start", action, "middle"))
	engine.RegisterState(NewFinalState("end"))
	// Note: No transition from start to anywhere

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := engine.Execute(ctx, smCtx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !errors.Is(err, ErrTransitionNotFound) {
		t.Errorf("expected ErrTransitionNotFound, got %v", err)
	}
}
