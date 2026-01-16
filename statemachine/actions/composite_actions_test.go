package actions

import (
	"context"
	"testing"
	"time"

	"github.com/amp-labs/amp-common/statemachine"
)

func TestTryWithFallback(t *testing.T) {
	t.Parallel()

	primaryAction := NewFailureAction(ErrPrimaryFailed)
	fallbackAction := NewSuccessAction()

	action := NewTryWithFallback("test_try", primaryAction, fallbackAction, true)

	ctx := MockContext("test-session", "test-project", nil)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	AssertContextString(t, ctx, "test_try_source", "fallback")
}

func TestValidatedSequence(t *testing.T) {
	t.Parallel()

	action := NewValidatedSequence(
		"test_sequence",
		[]statemachine.Action{
			NewSetValueAction("step1", "done"),
			NewSetValueAction("step2", "done"),
		},
		nil,
		false,
	)

	ctx := MockContext("test-session", "test-project", nil)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	AssertContextInt(t, ctx, "test_sequence_completed_steps", 2)
	AssertContextInt(t, ctx, "test_sequence_failed_at", -1)
}

func TestConditionalBranch(t *testing.T) {
	t.Parallel()

	action := NewConditionalBranch(
		"test_branch",
		[]Branch{
			{
				Condition: func(c *statemachine.Context) bool {
					val, _ := c.GetString("provider")

					return val == "salesforce"
				},
				Action: NewSetValueAction("provider_type", "salesforce"),
			},
			{
				Condition: func(c *statemachine.Context) bool {
					val, _ := c.GetString("provider")

					return val == "hubspot"
				},
				Action: NewSetValueAction("provider_type", "hubspot"),
			},
		},
		NewSetValueAction("provider_type", "unknown"),
	)

	// Test first branch
	ctx := MockContext("test-session", "test-project", map[string]any{
		"provider": "salesforce",
	})

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	AssertContextInt(t, ctx, "test_branch_branch_taken", 0)

	// Test default branch
	ctx2 := MockContext("test-session", "test-project", map[string]any{
		"provider": "other",
	})

	err = action.Execute(context.Background(), ctx2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	AssertContextInt(t, ctx2, "test_branch_branch_taken", -1)
}

func TestRetryWithBackoff(t *testing.T) {
	t.Parallel()

	callCount := 0
	mockAction := &MockAction{
		name: "mock_action",
		ExecuteFn: func(ctx context.Context, c *statemachine.Context) error {
			callCount++
			if callCount < 3 {
				return ErrTemporaryError
			}

			return nil
		},
	}

	action := NewRetryWithBackoff(
		"test_retry",
		mockAction,
		5,
		10*time.Millisecond,
		0,
		0,
		nil,
	)

	ctx := MockContext("test-session", "test-project", nil)

	err := action.Execute(context.Background(), ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	AssertContextInt(t, ctx, "test_retry_attempts", 3)
	AssertContextBool(t, ctx, "test_retry_succeeded", true)
}
