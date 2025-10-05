package stage

import (
	"os"
	"testing"

	"github.com/amp-labs/amp-common/lazy"
	"github.com/stretchr/testify/assert"
)

func TestStageConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stage    Stage
		expected string
	}{
		{"Unknown", Unknown, "unknown"},
		{"Local", Local, "local"},
		{"Test", Test, "test"},
		{"Dev", Dev, "dev"},
		{"Staging", Staging, "staging"},
		{"Prod", Prod, "prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, string(tt.stage))
		})
	}
}

func TestGetRunningStageWithEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected Stage
	}{
		{"Local", "local", Local},
		{"Test", "test", Test},
		{"Dev", "dev", Dev},
		{"Staging", "staging", Staging},
		{"Prod", "prod", Prod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			t.Setenv("RUNNING_ENV", tt.envValue)

			// Reset the lazy value for this test
			testStage := lazy.New[Stage](getRunningStage)

			assert.Equal(t, tt.expected, testStage.Get())
		})
	}
}

func TestGetRunningStageInvalidValue(t *testing.T) {
	// Set invalid environment variable
	t.Setenv("RUNNING_ENV", "invalid-stage")

	// Reset the lazy value for this test
	testStage := lazy.New[Stage](getRunningStage)

	// Should default to Test when running in test environment
	assert.Equal(t, Test, testStage.Get())
}

func TestGetRunningStageNoEnv(t *testing.T) {
	t.Parallel()

	// Ensure RUNNING_ENV is not set
	os.Unsetenv("RUNNING_ENV")

	// Reset the lazy value for this test
	testStage := lazy.New[Stage](getRunningStage)

	// Should default to Test when running in test environment (test.v flag exists)
	assert.Equal(t, Test, testStage.Get())
}

func TestIsLocal(t *testing.T) {
	t.Setenv("RUNNING_ENV", "local")

	// Reset the lazy value
	runningStage = lazy.New[Stage](getRunningStage)

	assert.True(t, IsLocal())
	assert.False(t, IsDev())
	assert.False(t, IsStaging())
	assert.False(t, IsProd())
	assert.False(t, IsTest())
	assert.False(t, IsUnknown())
}

func TestIsDev(t *testing.T) {
	t.Setenv("RUNNING_ENV", "dev")

	// Reset the lazy value
	runningStage = lazy.New[Stage](getRunningStage)

	assert.False(t, IsLocal())
	assert.True(t, IsDev())
	assert.False(t, IsStaging())
	assert.False(t, IsProd())
	assert.False(t, IsTest())
	assert.False(t, IsUnknown())
}

func TestIsStaging(t *testing.T) {
	t.Setenv("RUNNING_ENV", "staging")

	// Reset the lazy value
	runningStage = lazy.New[Stage](getRunningStage)

	assert.False(t, IsLocal())
	assert.False(t, IsDev())
	assert.True(t, IsStaging())
	assert.False(t, IsProd())
	assert.False(t, IsTest())
	assert.False(t, IsUnknown())
}

func TestIsProd(t *testing.T) {
	t.Setenv("RUNNING_ENV", "prod")

	// Reset the lazy value
	runningStage = lazy.New[Stage](getRunningStage)

	assert.False(t, IsLocal())
	assert.False(t, IsDev())
	assert.False(t, IsStaging())
	assert.True(t, IsProd())
	assert.False(t, IsTest())
	assert.False(t, IsUnknown())
}

func TestIsTest(t *testing.T) {
	t.Setenv("RUNNING_ENV", "test")

	// Reset the lazy value
	runningStage = lazy.New[Stage](getRunningStage)

	assert.False(t, IsLocal())
	assert.False(t, IsDev())
	assert.False(t, IsStaging())
	assert.False(t, IsProd())
	assert.True(t, IsTest())
	assert.False(t, IsUnknown())
}

func TestCurrent(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected Stage
	}{
		{"Local", "local", Local},
		{"Test", "test", Test},
		{"Dev", "dev", Dev},
		{"Staging", "staging", Staging},
		{"Prod", "prod", Prod},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("RUNNING_ENV", tt.envValue)

			// Reset the lazy value for this test
			runningStage = lazy.New[Stage](getRunningStage)

			assert.Equal(t, tt.expected, Current())
		})
	}
}

func TestErrUnrecognizedStage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unrecognized stage", ErrUnrecognizedStage.Error())
}
