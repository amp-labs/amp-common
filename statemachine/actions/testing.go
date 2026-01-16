package actions

import (
	"context"
	"testing"

	"github.com/amp-labs/amp-common/statemachine"
)

// MockContext creates a mock state machine context for testing.
func MockContext(sessionID, projectID string, data map[string]any) *statemachine.Context {
	ctx := statemachine.NewContext(sessionID, projectID)
	if data != nil {
		ctx.Merge(data)
	}

	return ctx
}

// AssertContextValue checks that a context value matches the expected value.
func AssertContextValue(t *testing.T, ctx *statemachine.Context, key string, expected any) {
	t.Helper()

	val, exists := ctx.Get(key)
	if !exists {
		t.Errorf("expected context to have key %q", key)

		return
	}

	if val != expected {
		t.Errorf("expected %q to be %v, got %v", key, expected, val)
	}
}

// AssertContextHasKey checks that a context has a specific key.
func AssertContextHasKey(t *testing.T, ctx *statemachine.Context, key string) {
	t.Helper()

	if _, exists := ctx.Get(key); !exists {
		t.Errorf("expected context to have key %q", key)
	}
}

// AssertContextMissingKey checks that a context does not have a specific key.
func AssertContextMissingKey(t *testing.T, ctx *statemachine.Context, key string) {
	t.Helper()

	if _, exists := ctx.Get(key); exists {
		t.Errorf("expected context to not have key %q", key)
	}
}

// AssertContextString checks that a context string value matches expected.
func AssertContextString(t *testing.T, ctx *statemachine.Context, key string, expected string) {
	t.Helper()

	val, ok := ctx.GetString(key)
	if !ok {
		t.Errorf("expected context to have string key %q", key)

		return
	}

	if val != expected {
		t.Errorf("expected %q to be %q, got %q", key, expected, val)
	}
}

// AssertContextBool checks that a context boolean value matches expected.
func AssertContextBool(t *testing.T, ctx *statemachine.Context, key string, expected bool) {
	t.Helper()

	val, ok := ctx.GetBool(key)
	if !ok {
		t.Errorf("expected context to have boolean key %q", key)

		return
	}

	if val != expected {
		t.Errorf("expected %q to be %v, got %v", key, expected, val)
	}
}

// AssertContextInt checks that a context integer value matches expected.
func AssertContextInt(t *testing.T, ctx *statemachine.Context, key string, expected int) {
	t.Helper()

	val, ok := ctx.GetInt(key)
	if !ok {
		t.Errorf("expected context to have integer key %q", key)

		return
	}

	if val != expected {
		t.Errorf("expected %q to be %d, got %d", key, expected, val)
	}
}

// MockAction creates a mock action for testing.
type MockAction struct {
	name      string
	ExecuteFn func(ctx context.Context, c *statemachine.Context) error
	Executed  bool
	CallCount int
}

// NewMockAction creates a new mock action.
func NewMockAction(fn func(ctx context.Context, c *statemachine.Context) error) *MockAction {
	return &MockAction{name: "mock_action", ExecuteFn: fn}
}

// NewSuccessAction creates a mock action that always succeeds.
func NewSuccessAction() *MockAction {
	return &MockAction{
		name: "success_action",
		ExecuteFn: func(ctx context.Context, c *statemachine.Context) error {
			c.Set("success", true)

			return nil
		},
	}
}

// NewFailureAction creates a mock action that always fails.
func NewFailureAction(err error) *MockAction {
	return &MockAction{
		name: "failure_action",
		ExecuteFn: func(ctx context.Context, c *statemachine.Context) error {
			return err
		},
	}
}

// NewSetValueAction creates a mock action that sets a value in context.
func NewSetValueAction(key string, value any) *MockAction {
	return &MockAction{
		name: "set_value_action",
		ExecuteFn: func(ctx context.Context, c *statemachine.Context) error {
			c.Set(key, value)

			return nil
		},
	}
}

// Name returns the name of the mock action.
func (m *MockAction) Name() string {
	return m.name
}

// Execute executes the mock action.
func (m *MockAction) Execute(ctx context.Context, c *statemachine.Context) error {
	m.Executed = true
	m.CallCount++

	if m.ExecuteFn != nil {
		return m.ExecuteFn(ctx, c)
	}

	return nil
}

// ActionTestCase represents a test case for an action.
type ActionTestCase struct {
	Name           string
	Action         statemachine.Action
	InitialContext map[string]any
	ExpectError    bool
	ExpectedKeys   []string
	Validate       func(t *testing.T, ctx *statemachine.Context)
}

// RunActionTestCases runs a series of action test cases.
func RunActionTestCases(t *testing.T, cases []ActionTestCase) {
	t.Helper()

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			ctx := MockContext("test-session", "test-project", testCase.InitialContext)
			err := testCase.Action.Execute(context.Background(), ctx)

			if testCase.ExpectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !testCase.ExpectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// Check expected keys
			for _, key := range testCase.ExpectedKeys {
				AssertContextHasKey(t, ctx, key)
			}

			// Run custom validation
			if testCase.Validate != nil {
				testCase.Validate(t, ctx)
			}
		})
	}
}
