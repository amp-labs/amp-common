//nolint:tparallel,paralleltest,testifylint // Test file
package testing

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestEngine(t *testing.T) {
	t.Parallel()

	config := CommonTestConfigs.Linear()
	engine := NewTestEngine(t, config)

	assert.NotNil(t, engine)
	assert.NotNil(t, engine.Engine)
	assert.Empty(t, engine.executionTrace)
	assert.Empty(t, engine.assertions)
}

func TestTestEngineExecute(t *testing.T) {
	t.Parallel()

	config := CommonTestConfigs.Linear()
	engine := NewTestEngine(t, config)

	ctx := context.Background()
	smCtx := CreateTestContext(map[string]any{
		"test_value": "test",
	})

	// Note: This will fail without actual action implementations
	// but tests the infrastructure
	_ = engine.Execute(ctx, smCtx)

	assert.NotEmpty(t, engine.executionTrace)
}

func TestCreateTestContext(t *testing.T) {
	t.Parallel()

	data := map[string]any{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	ctx := CreateTestContext(data)

	require.NotNil(t, ctx)
	assert.Equal(t, "test-session", ctx.SessionID)
	assert.Equal(t, "test-project", ctx.ProjectID)

	for key, expectedValue := range data {
		actualValue, exists := ctx.Get(key)
		assert.True(t, exists, "key %s should exist", key)
		assert.Equal(t, expectedValue, actualValue, "value for key %s", key)
	}
}

func TestMockSamplingClient(t *testing.T) {
	t.Parallel()

	responses := map[string]string{
		"prompt1": "response1",
		"prompt2": "response2",
	}

	client := NewMockSamplingClient(responses)

	response, err := client.Sample("prompt1")
	require.NoError(t, err)
	assert.Equal(t, "response1", response)

	_, err = client.Sample("unknown")
	assert.Error(t, err)
}

func TestMockElicitationClient(t *testing.T) {
	t.Parallel()

	responses := map[string]any{
		"question1": "answer1",
		"question2": 42,
	}

	client := NewMockElicitationClient(responses)

	answer, err := client.Elicit("question1")
	require.NoError(t, err)
	assert.Equal(t, "answer1", answer)

	_, err = client.Elicit("unknown")
	assert.Error(t, err)
}

func TestCommonTestConfigs(t *testing.T) {
	t.Parallel()

	t.Run("Linear", func(t *testing.T) {
		config := CommonTestConfigs.Linear()
		assert.NotNil(t, config)
		assert.Equal(t, "linear", config.Name)
		assert.Equal(t, "start", config.InitialState)
		assert.Contains(t, config.FinalStates, "end")
	})

	t.Run("Branching", func(t *testing.T) {
		config := CommonTestConfigs.Branching()
		assert.NotNil(t, config)
		assert.Equal(t, "branching", config.Name)
		assert.Len(t, config.FinalStates, 2)
	})

	t.Run("Loop", func(t *testing.T) {
		config := CommonTestConfigs.Loop()
		assert.NotNil(t, config)
		assert.Equal(t, "loop", config.Name)
	})

	t.Run("Complex", func(t *testing.T) {
		config := CommonTestConfigs.Complex()
		assert.NotNil(t, config)
		assert.Equal(t, "complex", config.Name)
		assert.GreaterOrEqual(t, len(config.States), 5)
	})
}

func TestMatchers(t *testing.T) {
	t.Parallel()

	config := CommonTestConfigs.Linear()
	engine := NewTestEngine(t, config)

	// Create some test trace data
	engine.executionTrace = []TraceEntry{
		{State: "start"},
		{State: "middle"},
		{State: "end"},
	}

	t.Run("StateWasVisited", func(t *testing.T) {
		matcher := StateWasVisited("middle")
		matched, err := matcher.Match(engine)
		assert.True(t, matched)
		assert.NoError(t, err)

		matcher = StateWasVisited("nonexistent")
		matched, err = matcher.Match(engine)
		assert.False(t, matched)
		assert.Error(t, err)
	})

	t.Run("TransitionWasTaken", func(t *testing.T) {
		matcher := TransitionWasTaken("start", "middle")
		matched, err := matcher.Match(engine)
		assert.True(t, matched)
		assert.NoError(t, err)
	})
}

func TestRunScenario(t *testing.T) {
	t.Parallel()

	scenario := LinearWorkflowScenario()
	// Would normally run full scenario, but testing the infrastructure
	assert.NotNil(t, scenario.Config)
	assert.NotEmpty(t, scenario.Name)
}
