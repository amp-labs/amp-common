package statemachine

import (
	"context"
	"testing"
)

// customAction is a test action type for testing custom action builders.
type customAction struct {
	name     string
	executed bool
}

func (c *customAction) Name() string {
	return c.name
}

func (c *customAction) Execute(ctx context.Context, smCtx *Context) error {
	c.executed = true
	smCtx.Set(c.name+"_executed", true)

	return nil
}

// customRetryAction is a test action type that fails once then succeeds.
type customRetryAction struct {
	name     string
	attempts int
}

func (c *customRetryAction) Name() string {
	return c.name
}

func (c *customRetryAction) Execute(ctx context.Context, smCtx *Context) error {
	c.attempts++

	// Fail on first attempt, succeed on subsequent attempts
	if c.attempts == 1 {
		return ErrTestTemporary
	}

	smCtx.Set(c.name+"_executed", true)
	smCtx.Set(c.name+"_attempts", c.attempts)

	return nil
}

// TestCustomActionBuilderInNestedActions verifies that custom action builders
// registered via RegisterActionBuilder work inside nested actions (sequence, retry).
func TestCustomActionBuilderInNestedActions(t *testing.T) {
	t.Parallel()

	// Create a builder with custom action builder
	builder := NewBuilder("test_custom_action")
	builder.WithInitialState("start")
	builder.WithFinalStates("complete")

	// Register custom action builder
	builder.RegisterActionBuilder("custom", func(_ *ActionFactory, name string, params map[string]any) (Action, error) {
		return &customAction{name: name}, nil
	})

	// Add a state using sequence with custom actions
	builder.AddState(StateConfig{
		Name: "start",
		Type: "action",
		Actions: []ActionConfig{
			{
				Type: "sequence",
				Name: "sequence_with_custom",
				Parameters: map[string]any{
					"actions": []any{
						map[string]any{
							"type":       "custom",
							"name":       "custom1",
							"parameters": map[string]any{},
						},
						map[string]any{
							"type":       "custom",
							"name":       "custom2",
							"parameters": map[string]any{},
						},
					},
				},
			},
		},
	})

	builder.AddState(StateConfig{
		Name: "complete",
		Type: "final",
	})

	builder.AddTransition(TransitionConfig{
		From:      "start",
		To:        "complete",
		Condition: "always",
	})

	// Build the engine
	engine, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build engine: %v", err)
	}

	// Execute the engine
	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err = engine.Execute(ctx, smCtx)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if smCtx.CurrentState != "complete" {
		t.Errorf("expected final state 'complete', got %s", smCtx.CurrentState)
	}
}

// TestCustomActionBuilderInRetry verifies that custom action builders work inside retry actions.
func TestCustomActionBuilderInRetry(t *testing.T) {
	t.Parallel()

	// Create a builder with custom action builder
	builder := NewBuilder("test_retry_custom")
	builder.WithInitialState("start")
	builder.WithFinalStates("complete")

	// Register custom action builder
	builder.RegisterActionBuilder(
		"custom_retry",
		func(_ *ActionFactory, name string, params map[string]any) (Action, error) {
			return &customRetryAction{name: name}, nil
		},
	)

	// Add a state using retry with custom action
	builder.AddState(StateConfig{
		Name: "start",
		Type: "action",
		Actions: []ActionConfig{
			{
				Type: "retry",
				Name: "retry_with_custom",
				Parameters: map[string]any{
					"maxRetries": 3,
					"backoffMs":  10,
					"action": map[string]any{
						"type":       "custom_retry",
						"name":       "custom_action",
						"parameters": map[string]any{},
					},
				},
			},
		},
	})

	builder.AddState(StateConfig{
		Name: "complete",
		Type: "final",
	})

	builder.AddTransition(TransitionConfig{
		From:      "start",
		To:        "complete",
		Condition: "always",
	})

	// Build the engine
	engine, err := builder.Build()
	if err != nil {
		t.Fatalf("failed to build engine: %v", err)
	}

	// Execute the engine
	ctx := context.Background()
	smCtx := NewContext("test-session", "test-project")

	err = engine.Execute(ctx, smCtx)
	if err != nil {
		t.Fatalf("execution failed: %v", err)
	}

	if smCtx.CurrentState != "complete" {
		t.Errorf("expected final state 'complete', got %s", smCtx.CurrentState)
	}
}
