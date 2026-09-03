package region

import (
	"context"
	"os"
	"testing"

	"github.com/amp-labs/amp-common/lazy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetCurrentRegion replaces the package-level cached region with a fresh
// lazy value so the test observes the current REGION environment variable,
// and restores a fresh value again on cleanup so later tests are not affected.
func resetCurrentRegion(t *testing.T) {
	t.Helper()

	currentRegion = lazy.NewCtx[Region](getRegion)

	t.Cleanup(func() {
		currentRegion = lazy.NewCtx[Region](getRegion)
	})
}

func TestRegionConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		region   Region
		expected string
	}{
		{"Unknown", Unknown, "unknown"},
		{"Us", Us, "us"},
		{"Eu", Eu, "eu"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, string(tt.region))
			assert.Equal(t, tt.expected, tt.region.String())
		})
	}
}

func TestRegionValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		region   Region
		expected bool
	}{
		{"Us", Us, true},
		{"Eu", Eu, true},
		{"Unknown", Unknown, false},
		{"Empty", Region(""), false},
		{"Unrecognized", Region("apac"), false},
		{"WrongCase", Region("US"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.region.Valid())
		})
	}
}

func TestRegionOrElse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		region   Region
		dfl      Region
		expected Region
	}{
		{"ValidKeepsSelf", Us, Eu, Us},
		{"ValidKeepsSelfEu", Eu, Us, Eu},
		{"UnknownUsesDefault", Unknown, Us, Us},
		{"EmptyUsesDefault", Region(""), Eu, Eu},
		{"UnrecognizedUsesDefault", Region("apac"), Us, Us},
		// The default is returned unchecked, even when it is not itself valid.
		{"InvalidDefaultPassesThrough", Unknown, Unknown, Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.expected, tt.region.OrElse(tt.dfl))
		})
	}
}

func TestGetRegionWithEnv(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected Region
	}{
		{"Us", "us", Us},
		{"Eu", "eu", Eu},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REGION", tt.envValue)

			// Use a fresh lazy value so the cached package-level value is not consulted.
			testRegion := lazy.NewCtx[Region](getRegion)

			assert.Equal(t, tt.expected, testRegion.Get(t.Context()))
		})
	}
}

func TestGetRegionInvalidValue(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
	}{
		{"Unrecognized", "invalid-region"},
		{"WrongCase", "US"},
		{"Whitespace", " us"},
		{"Empty", ""},
		// "unknown" is the fallback, never an accepted configuration value.
		{"ExplicitUnknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REGION", tt.envValue)

			testRegion := lazy.NewCtx[Region](getRegion)

			assert.Equal(t, Unknown, testRegion.Get(t.Context()))
		})
	}
}

func TestGetRegionNoEnv(t *testing.T) {
	// t.Setenv restores the original value on cleanup; unsetting afterwards
	// guarantees REGION is absent for the duration of this test only.
	t.Setenv("REGION", "")
	require.NoError(t, os.Unsetenv("REGION"))

	testRegion := lazy.NewCtx[Region](getRegion)

	// Unlike stage, there is no test-mode fallback: an unset REGION is Unknown.
	assert.Equal(t, Unknown, testRegion.Get(t.Context()))
}

func TestCurrent(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected Region
	}{
		{"Us", "us", Us},
		{"Eu", "eu", Eu},
		{"Invalid", "mars", Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("REGION", tt.envValue)
			resetCurrentRegion(t)

			assert.Equal(t, tt.expected, Current(t.Context()))
		})
	}
}

func TestCurrentNoEnv(t *testing.T) {
	t.Setenv("REGION", "")
	require.NoError(t, os.Unsetenv("REGION"))
	resetCurrentRegion(t)

	assert.Equal(t, Unknown, Current(t.Context()))
	assert.False(t, Current(t.Context()).Valid())
	assert.Equal(t, Us, Current(t.Context()).OrElse(Us))
}

func TestCurrentIsCached(t *testing.T) {
	t.Setenv("REGION", "us")
	resetCurrentRegion(t)

	assert.Equal(t, Us, Current(t.Context()))

	// The region is determined once; later changes to the environment are not seen.
	t.Setenv("REGION", "eu")
	assert.Equal(t, Us, Current(t.Context()))

	// A context override still takes precedence over the cached value.
	assert.Equal(t, Eu, Current(WithRegion(t.Context(), Eu)))
}

func TestCurrentNilContext(t *testing.T) {
	t.Setenv("REGION", "eu")
	resetCurrentRegion(t)

	// A nil context skips the override lookup and falls through to the
	// environment-derived value without panicking.
	assert.Equal(t, Eu, Current(nil)) //nolint:staticcheck // Deliberately exercising the nil-context path.
}

func TestErrUnrecognizedRegion(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "unrecognized region", ErrUnrecognizedRegion.Error())
}

func TestWithRegion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		region   Region
		expected Region
	}{
		{"Us", Us, Us},
		{"Eu", Eu, Eu},
		{"Unknown", Unknown, Unknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := WithRegion(t.Context(), tt.region)
			assert.Equal(t, tt.expected, Current(ctx))
		})
	}
}

func TestWithRegionInvalidValuePassesThrough(t *testing.T) {
	t.Parallel()

	// A context override is returned as-is, even when it is not a valid region.
	// Callers that need a usable region should apply OrElse.
	ctx := WithRegion(t.Context(), Region("mars"))

	assert.Equal(t, Region("mars"), Current(ctx))
	assert.False(t, Current(ctx).Valid())
	assert.Equal(t, Us, Current(ctx).OrElse(Us))
}

func TestWithRegionOverridesEnvironment(t *testing.T) {
	t.Setenv("REGION", "us")
	resetCurrentRegion(t)

	ctx := WithRegion(t.Context(), Eu)

	// Context override should take precedence over the environment.
	assert.Equal(t, Eu, Current(ctx))
	// The un-overridden context still sees the environment value.
	assert.Equal(t, Us, Current(t.Context()))
}

func TestWithRegionNested(t *testing.T) {
	t.Parallel()

	ctx1 := WithRegion(t.Context(), Us)
	assert.Equal(t, Us, Current(ctx1))

	ctx2 := WithRegion(ctx1, Eu)
	assert.Equal(t, Eu, Current(ctx2))

	// The outer context is unaffected by the nested override.
	assert.Equal(t, Us, Current(ctx1))
}

func TestSetRegion(t *testing.T) {
	t.Parallel()

	var (
		gotKey   any
		gotValue any
		calls    int
	)

	setter := func(key any, value any) {
		gotKey = key
		gotValue = value
		calls++
	}

	SetRegion(Eu, setter)

	assert.Equal(t, 1, calls)
	assert.Equal(t, regionContextKey, gotKey)
	assert.Equal(t, Eu, gotValue)

	// Storing the captured pair in a context must be equivalent to WithRegion,
	// which is what makes SetRegion usable with context builders.
	ctx := context.WithValue(t.Context(), gotKey, gotValue)
	assert.Equal(t, Eu, Current(ctx))
}

func TestSetRegionNilSetter(t *testing.T) {
	t.Parallel()

	assert.NotPanics(t, func() {
		SetRegion(Us, nil)
	})
}
