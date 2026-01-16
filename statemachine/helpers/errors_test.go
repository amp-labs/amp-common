//nolint:testifylint // Test file
package helpers

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test errors.
var (
	errTest      = errors.New("test")
	errTestError = errors.New("test error")
	errTestOrig  = errors.New("original error")
	errTestBase  = errors.New("base error")
)

// TestSamplingError_Error tests SamplingError error message formatting.
func TestSamplingError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *SamplingError
		expected string
	}{
		{
			name: "error without context",
			err: &SamplingError{
				Operation: "test_op",
				Err:       errTestError,
				Context:   nil,
			},
			expected: "sampling error in test_op: test error",
		},
		{
			name: "error with context",
			err: &SamplingError{
				Operation: "test_op",
				Err:       errTestError,
				Context: map[string]any{
					"key": "value",
				},
			},
			expected: "sampling error in test_op: test error (context: map[key:value])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestSamplingError_Unwrap tests error unwrapping for SamplingError.
func TestSamplingError_Unwrap(t *testing.T) {
	t.Parallel()

	originalErr := errTestOrig
	samplingErr := &SamplingError{
		Operation: "test_op",
		Err:       originalErr,
	}

	unwrapped := samplingErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

// TestElicitationError_Error tests ElicitationError error message formatting.
func TestElicitationError_Error(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      *ElicitationError
		expected string
	}{
		{
			name: "error without context",
			err: &ElicitationError{
				Operation: "test_op",
				Err:       errTestError,
				Context:   nil,
			},
			expected: "elicitation error in test_op: test error",
		},
		{
			name: "error with context",
			err: &ElicitationError{
				Operation: "test_op",
				Err:       errTestError,
				Context: map[string]any{
					"key": "value",
				},
			},
			expected: "elicitation error in test_op: test error (context: map[key:value])",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.err.Error()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestElicitationError_Unwrap tests error unwrapping for ElicitationError.
func TestElicitationError_Unwrap(t *testing.T) {
	t.Parallel()

	originalErr := errTestOrig
	elicitationErr := &ElicitationError{
		Operation: "test_op",
		Err:       originalErr,
	}

	unwrapped := elicitationErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

// TestWrapSamplingError tests SamplingError wrapping.
//
//nolint:dupl // Similar structure to TestWrapElicitationError but tests different wrapper function
func TestWrapSamplingError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation string
		err       error
		context   map[string]any
		expectNil bool
	}{
		{
			name:      "wrap non-nil error",
			operation: "test_op",
			err:       errTestError,
			context:   nil,
			expectNil: false,
		},
		{
			name:      "wrap nil error returns nil",
			operation: "test_op",
			err:       nil,
			context:   nil,
			expectNil: true,
		},
		{
			name:      "wrap with context",
			operation: "test_op",
			err:       errTestError,
			context:   map[string]any{"key": "value"},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := WrapSamplingError(tt.operation, tt.err, tt.context)

			if tt.expectNil {
				assert.NoError(t, result)
			} else {
				require.Error(t, result)
				assert.True(t, IsSamplingError(result))
				assert.ErrorIs(t, result, tt.err)
			}
		})
	}
}

// TestWrapElicitationError tests ElicitationError wrapping.
//
//nolint:dupl // Similar structure to TestWrapSamplingError but tests different wrapper function
func TestWrapElicitationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation string
		err       error
		context   map[string]any
		expectNil bool
	}{
		{
			name:      "wrap non-nil error",
			operation: "test_op",
			err:       errTestError,
			context:   nil,
			expectNil: false,
		},
		{
			name:      "wrap nil error returns nil",
			operation: "test_op",
			err:       nil,
			context:   nil,
			expectNil: true,
		},
		{
			name:      "wrap with context",
			operation: "test_op",
			err:       errTestError,
			context:   map[string]any{"key": "value"},
			expectNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := WrapElicitationError(tt.operation, tt.err, tt.context)

			if tt.expectNil {
				assert.NoError(t, result)
			} else {
				require.Error(t, result)
				assert.True(t, IsElicitationError(result))
				assert.ErrorIs(t, result, tt.err)
			}
		})
	}
}

// TestIsUserDeclined_Various tests IsUserDeclined with various error types.
func TestIsUserDeclined_Various(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "direct user declined error",
			err:      ErrUserDeclinedAction,
			expected: true,
		},
		{
			name:     "wrapped in sampling error",
			err:      WrapSamplingError("test", ErrUserDeclinedAction, nil),
			expected: true,
		},
		{
			name:     "wrapped in elicitation error",
			err:      WrapElicitationError("test", ErrUserDeclinedAction, nil),
			expected: true,
		},
		{
			name:     "different error",
			err:      ErrValidationFailed,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsUserDeclined(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsCapabilityMissing_Various tests IsCapabilityMissing with various error types.
func TestIsCapabilityMissing_Various(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "sampling unavailable",
			err:      ErrSamplingUnavailable,
			expected: true,
		},
		{
			name:     "elicitation unavailable",
			err:      ErrElicitationUnavailable,
			expected: true,
		},
		{
			name:     "wrapped sampling unavailable",
			err:      WrapSamplingError("test", ErrSamplingUnavailable, nil),
			expected: true,
		},
		{
			name:     "wrapped elicitation unavailable",
			err:      WrapElicitationError("test", ErrElicitationUnavailable, nil),
			expected: true,
		},
		{
			name:     "different error",
			err:      ErrValidationFailed,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsCapabilityMissing(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsSamplingError tests SamplingError type detection.
func TestIsSamplingError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "sampling error",
			err:      WrapSamplingError("test", errTest, nil),
			expected: true,
		},
		{
			name:     "elicitation error",
			err:      WrapElicitationError("test", errTest, nil),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errTest,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsSamplingError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestIsElicitationError tests ElicitationError type detection.
func TestIsElicitationError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "elicitation error",
			err:      WrapElicitationError("test", errTest, nil),
			expected: true,
		},
		{
			name:     "sampling error",
			err:      WrapSamplingError("test", errTest, nil),
			expected: false,
		},
		{
			name:     "generic error",
			err:      errTest,
			expected: false,
		},
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := IsElicitationError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestHandleGracefulDegradation tests graceful degradation with various error types.
func TestHandleGracefulDegradation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		err            error
		fallback       any
		expectedResult any
		expectError    bool
	}{
		{
			name:           "nil error returns nil",
			err:            nil,
			fallback:       "fallback",
			expectedResult: nil,
			expectError:    false,
		},
		{
			name:           "capability missing uses fallback",
			err:            ErrSamplingUnavailable,
			fallback:       "fallback value",
			expectedResult: "fallback value",
			expectError:    false,
		},
		{
			name:           "user declined uses fallback",
			err:            ErrUserDeclinedAction,
			fallback:       "fallback value",
			expectedResult: "fallback value",
			expectError:    false,
		},
		{
			name:           "other error propagates",
			err:            ErrValidationFailed,
			fallback:       "fallback value",
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := HandleGracefulDegradation(tt.err, tt.fallback)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

// TestSentinelErrors tests that sentinel errors are defined.
func TestSentinelErrors(t *testing.T) {
	t.Parallel()

	require.Error(t, ErrSamplingUnavailable)
	require.Error(t, ErrElicitationUnavailable)
	assert.Error(t, ErrUserDeclinedAction)
	assert.Error(t, ErrValidationFailed)
	assert.Error(t, ErrJSONParseFailed)
	assert.Error(t, ErrCapabilityCheckFailed)
	assert.Error(t, ErrFallbackExecutionFailed)
	assert.Error(t, ErrUnsupportedFallbackType)
}

// TestErrorChaining tests that errors can be chained and unwrapped correctly.
func TestErrorChaining(t *testing.T) {
	t.Parallel()

	baseErr := errTestBase
	wrappedErr := WrapSamplingError("operation", baseErr, nil)
	doubleWrapped := WrapElicitationError("elicitation", wrappedErr, nil)

	// Should be able to unwrap all the way to base error
	assert.ErrorIs(t, doubleWrapped, baseErr)
	assert.ErrorIs(t, doubleWrapped, wrappedErr)
}

// TestErrorContextPreservation tests that context is preserved through wrapping.
func TestErrorContextPreservation(t *testing.T) {
	t.Parallel()

	context := map[string]any{
		"key1": "value1",
		"key2": 42,
	}

	err := WrapSamplingError("test_op", errTest, context)

	samplingErr := &SamplingError{}
	ok := errors.As(err, &samplingErr)
	require.True(t, ok)
	assert.Equal(t, context, samplingErr.Context)
	assert.Equal(t, "test_op", samplingErr.Operation)
}

// TestMultipleErrorWrapping tests wrapping errors multiple times.
func TestMultipleErrorWrapping(t *testing.T) {
	t.Parallel()

	baseErr := errTestBase

	// Wrap as sampling error
	samplingErr := WrapSamplingError("sampling_op", baseErr, map[string]any{"sampling": true})

	// Wrap again as elicitation error
	elicitationErr := WrapElicitationError("elicitation_op", samplingErr, map[string]any{"elicitation": true})

	// Both type checks should work via unwrapping
	assert.True(t, IsSamplingError(elicitationErr))
	assert.True(t, IsElicitationError(elicitationErr))

	// Original error should still be accessible
	assert.ErrorIs(t, elicitationErr, baseErr)
}
