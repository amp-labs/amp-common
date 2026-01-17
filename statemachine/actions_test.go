package statemachine

import (
	"context"
	"testing"
	"time"
)

func TestSequenceAction(t *testing.T) {
	t.Parallel()

	action1 := &mockAction{name: "action1"}
	action2 := &mockAction{name: "action2"}
	action3 := &mockAction{name: "action3"}

	sequence := NewSequenceAction("sequence", action1, action2, action3)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := sequence.Execute(ctx, smCtx)
	if err != nil {
		t.Fatalf("sequence execution failed: %v", err)
	}

	if !action1.executed {
		t.Error("action1 was not executed")
	}

	if !action2.executed {
		t.Error("action2 was not executed")
	}

	if !action3.executed {
		t.Error("action3 was not executed")
	}
}

func TestSequenceActionStopsOnError(t *testing.T) {
	t.Parallel()

	action1 := &mockAction{name: "action1"}
	action2 := &mockAction{name: "action2", err: ErrTestAction2Failed}
	action3 := &mockAction{name: "action3"}

	sequence := NewSequenceAction("sequence", action1, action2, action3)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := sequence.Execute(ctx, smCtx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if !action1.executed {
		t.Error("action1 should have been executed")
	}

	if !action2.executed {
		t.Error("action2 should have been executed (even though it failed)")
	}

	if action3.executed {
		t.Error("action3 should not have been executed after action2 failed")
	}
}

func TestConditionalAction(t *testing.T) {
	t.Parallel()

	thenAction := &mockAction{name: "then"}
	elseAction := &mockAction{name: "else"}

	tests := []struct {
		name          string
		condition     bool
		shouldRunThen bool
		shouldRunElse bool
	}{
		{"true condition", true, true, false},
		{"false condition", false, false, true},
	}

	for _, tt := range tests { //nolint:varnamelen // tt is standard Go test idiom
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			thenAction.executed = false
			elseAction.executed = false

			conditional := NewConditionalAction("conditional",
				func(ctx context.Context, smCtx *Context) (bool, error) {
					return tt.condition, nil
				},
				thenAction,
				elseAction,
			)

			ctx := context.Background()
			smCtx := NewContext("test-session", "test-project")

			err := conditional.Execute(ctx, smCtx)
			if err != nil {
				t.Fatalf("conditional execution failed: %v", err)
			}

			if thenAction.executed != tt.shouldRunThen {
				t.Errorf("then action executed=%v, want=%v", thenAction.executed, tt.shouldRunThen)
			}

			if elseAction.executed != tt.shouldRunElse {
				t.Errorf("else action executed=%v, want=%v", elseAction.executed, tt.shouldRunElse)
			}
		})
	}
}

func TestRetryAction(t *testing.T) {
	t.Parallel()

	// Action that fails twice then succeeds
	attempts := 0
	action := &mockAction{name: "retry_action"}
	action.err = ErrTestTemporary

	// Override Execute to simulate intermittent failure
	originalExecute := func(ctx context.Context, smCtx *Context) error {
		attempts++
		if attempts < 3 {
			return action.err
		}

		action.executed = true
		smCtx.Set(action.name+"_executed", true)

		return nil
	}

	// Create a wrapper that uses the custom logic
	wrappedAction := &customMockAction{
		name:        "retry_action",
		executeFunc: originalExecute,
	}

	retry := NewRetryAction("retry", wrappedAction, 5, 10*time.Millisecond)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := retry.Execute(ctx, smCtx)
	if err != nil {
		t.Fatalf("retry execution failed: %v", err)
	}

	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

func TestRetryActionExhaustsRetries(t *testing.T) {
	t.Parallel()

	action := &mockAction{name: "always_fails", err: ErrTestPermanent}

	retry := NewRetryAction("retry", action, 3, 1*time.Millisecond)

	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err := retry.Execute(ctx, smCtx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	// Should have tried 3 times
	if !action.executed {
		t.Error("action should have been attempted")
	}
}

func TestValidationAction(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name             string
		valid            bool
		feedback         string
		expectedValid    bool
		expectedFeedback string
	}{
		{"valid data", true, "looks good", true, "looks good"},
		{"invalid data", false, "field required", false, "field required"},
	}

	for _, tt := range tests { //nolint:varnamelen // tt is standard Go test idiom
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			validation := NewValidationAction("validate",
				func(ctx context.Context, smCtx *Context) (bool, string, error) {
					return tt.valid, tt.feedback, nil
				},
			)

			ctx := context.Background()
			smCtx := NewContext("test-session", "test-project")

			err := validation.Execute(ctx, smCtx)
			if err != nil {
				t.Fatalf("validation execution failed: %v", err)
			}

			valid, ok := smCtx.GetBool("validate_valid")
			if !ok {
				t.Fatal("validate_valid not set in context")
			}

			if valid != tt.expectedValid {
				t.Errorf("expected valid=%v, got %v", tt.expectedValid, valid)
			}

			feedback, ok := smCtx.GetString("validate_feedback")
			if !ok {
				t.Fatal("validate_feedback not set in context")
			}

			if feedback != tt.expectedFeedback {
				t.Errorf("expected feedback=%s, got %s", tt.expectedFeedback, feedback)
			}
		})
	}
}

// customMockAction allows custom execute logic.
type customMockAction struct {
	name        string
	executeFunc func(ctx context.Context, smCtx *Context) error
}

func (m *customMockAction) Name() string {
	return m.name
}

func (m *customMockAction) Execute(ctx context.Context, smCtx *Context) error {
	return m.executeFunc(ctx, smCtx)
}
