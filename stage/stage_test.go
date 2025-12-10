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
			testStage := lazy.NewCtx[Stage](getRunningStage)

			assert.Equal(t, tt.expected, testStage.Get(t.Context()))
		})
	}
}

func TestGetRunningStageInvalidValue(t *testing.T) {
	// Set invalid environment variable
	t.Setenv("RUNNING_ENV", "invalid-stage")

	// Reset the lazy value for this test
	testStage := lazy.NewCtx[Stage](getRunningStage)

	// Should default to Test when running in test environment
	assert.Equal(t, Test, testStage.Get(t.Context()))
}

func TestGetRunningStageNoEnv(t *testing.T) {
	t.Parallel()

	// Ensure RUNNING_ENV is not set
	_ = os.Unsetenv("RUNNING_ENV")

	// Reset the lazy value for this test
	testStage := lazy.NewCtx[Stage](getRunningStage)

	// Should default to Test when running in test environment (test.v flag exists)
	assert.Equal(t, Test, testStage.Get(t.Context()))
}

func TestIsLocal(t *testing.T) {
	t.Setenv("RUNNING_ENV", "local")

	// Reset the lazy value
	runningStage = lazy.NewCtx[Stage](getRunningStage)

	assert.True(t, IsLocal(t.Context()))
	assert.False(t, IsDev(t.Context()))
	assert.False(t, IsStaging(t.Context()))
	assert.False(t, IsProd(t.Context()))
	assert.False(t, IsTest(t.Context()))
	assert.False(t, IsUnknown(t.Context()))
}

func TestIsDev(t *testing.T) {
	t.Setenv("RUNNING_ENV", "dev")

	// Reset the lazy value
	runningStage = lazy.NewCtx[Stage](getRunningStage)

	assert.False(t, IsLocal(t.Context()))
	assert.True(t, IsDev(t.Context()))
	assert.False(t, IsStaging(t.Context()))
	assert.False(t, IsProd(t.Context()))
	assert.False(t, IsTest(t.Context()))
	assert.False(t, IsUnknown(t.Context()))
}

func TestIsStaging(t *testing.T) {
	t.Setenv("RUNNING_ENV", "staging")

	// Reset the lazy value
	runningStage = lazy.NewCtx[Stage](getRunningStage)

	assert.False(t, IsLocal(t.Context()))
	assert.False(t, IsDev(t.Context()))
	assert.True(t, IsStaging(t.Context()))
	assert.False(t, IsProd(t.Context()))
	assert.False(t, IsTest(t.Context()))
	assert.False(t, IsUnknown(t.Context()))
}

func TestIsProd(t *testing.T) {
	t.Setenv("RUNNING_ENV", "prod")

	// Reset the lazy value
	runningStage = lazy.NewCtx[Stage](getRunningStage)

	assert.False(t, IsLocal(t.Context()))
	assert.False(t, IsDev(t.Context()))
	assert.False(t, IsStaging(t.Context()))
	assert.True(t, IsProd(t.Context()))
	assert.False(t, IsTest(t.Context()))
	assert.False(t, IsUnknown(t.Context()))
}

func TestIsTest(t *testing.T) {
	t.Setenv("RUNNING_ENV", "test")

	// Reset the lazy value
	runningStage = lazy.NewCtx[Stage](getRunningStage)

	assert.False(t, IsLocal(t.Context()))
	assert.False(t, IsDev(t.Context()))
	assert.False(t, IsStaging(t.Context()))
	assert.False(t, IsProd(t.Context()))
	assert.True(t, IsTest(t.Context()))
	assert.False(t, IsUnknown(t.Context()))
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
			runningStage = lazy.NewCtx[Stage](getRunningStage)

			assert.Equal(t, tt.expected, Current(t.Context()))
		})
	}
}

func TestErrUnrecognizedStage(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unrecognized stage", ErrUnrecognizedStage.Error())
}

func TestWithStage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		stage    Stage
		expected Stage
	}{
		{"WithStageLocal", Local, Local},
		{"WithStageTest", Test, Test},
		{"WithStageDev", Dev, Dev},
		{"WithStageStaging", Staging, Staging},
		{"WithStageProd", Prod, Prod},
		{"WithStageUnknown", Unknown, Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := WithStage(t.Context(), tt.stage)
			assert.Equal(t, tt.expected, Current(ctx))
		})
	}
}

func TestWithStageOverridesEnvironment(t *testing.T) {
	// Set environment to prod
	t.Setenv("RUNNING_ENV", "prod")

	// Override with context to local
	ctx := WithStage(t.Context(), Local)

	// Context override should take precedence
	assert.Equal(t, Local, Current(ctx))
	assert.True(t, IsLocal(ctx))
	assert.False(t, IsProd(ctx))
}

func TestWithStageForAllIsChecks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		stage         Stage
		expectLocal   bool
		expectDev     bool
		expectStaging bool
		expectProd    bool
		expectTest    bool
		expectUnknown bool
	}{
		{
			name:          "Local",
			stage:         Local,
			expectLocal:   true,
			expectDev:     false,
			expectStaging: false,
			expectProd:    false,
			expectTest:    false,
			expectUnknown: false,
		},
		{
			name:          "Dev",
			stage:         Dev,
			expectLocal:   false,
			expectDev:     true,
			expectStaging: false,
			expectProd:    false,
			expectTest:    false,
			expectUnknown: false,
		},
		{
			name:          "Staging",
			stage:         Staging,
			expectLocal:   false,
			expectDev:     false,
			expectStaging: true,
			expectProd:    false,
			expectTest:    false,
			expectUnknown: false,
		},
		{
			name:          "Prod",
			stage:         Prod,
			expectLocal:   false,
			expectDev:     false,
			expectStaging: false,
			expectProd:    true,
			expectTest:    false,
			expectUnknown: false,
		},
		{
			name:          "Test",
			stage:         Test,
			expectLocal:   false,
			expectDev:     false,
			expectStaging: false,
			expectProd:    false,
			expectTest:    true,
			expectUnknown: false,
		},
		{
			name:          "Unknown",
			stage:         Unknown,
			expectLocal:   false,
			expectDev:     false,
			expectStaging: false,
			expectProd:    false,
			expectTest:    false,
			expectUnknown: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := WithStage(t.Context(), tt.stage)

			assert.Equal(t, tt.expectLocal, IsLocal(ctx))
			assert.Equal(t, tt.expectDev, IsDev(ctx))
			assert.Equal(t, tt.expectStaging, IsStaging(ctx))
			assert.Equal(t, tt.expectProd, IsProd(ctx))
			assert.Equal(t, tt.expectTest, IsTest(ctx))
			assert.Equal(t, tt.expectUnknown, IsUnknown(ctx))
		})
	}
}

func TestWithStageNested(t *testing.T) {
	t.Parallel()

	// Create a context with Prod
	ctx1 := WithStage(t.Context(), Prod)
	assert.Equal(t, Prod, Current(ctx1))

	// Create a nested context with Dev
	ctx2 := WithStage(ctx1, Dev)
	assert.Equal(t, Dev, Current(ctx2))

	// Original context should still be Prod
	assert.Equal(t, Prod, Current(ctx1))
}
