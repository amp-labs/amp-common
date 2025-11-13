package retry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAbort_StopsRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	originalErr := errors.New("validation failed") //nolint:err113 // Test error

	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return Abort(originalErr)
	}, WithAttempts(10))

	require.Error(t, err)
	assert.Equal(t, 1, callCount, "should not retry after Abort")
	// Error should be unwrappable to the original error
	assert.ErrorIs(t, err, originalErr)
}

func TestAbort_Unwrap(t *testing.T) {
	t.Parallel()

	originalErr := errors.New("base error") //nolint:err113 // Test error
	abortErr := Abort(originalErr)

	// Should be able to unwrap to get original error
	assert.ErrorIs(t, abortErr, originalErr)
}

func TestPermanentError_Temporary(t *testing.T) {
	t.Parallel()

	err := Abort(errors.New("test")) //nolint:err113 // Test error
	retryErr, ok := err.(interface{ Temporary() bool })

	assert.True(t, ok, "should implement Error interface")
	assert.False(t, retryErr.Temporary(), "should not be temporary")
}

func TestPermanentError_UnwrapChain(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error") //nolint:err113 // Test error
	wrappedErr := Abort(baseErr)

	// Test that errors.Is works with the chain
	require.ErrorIs(t, wrappedErr, baseErr)

	// Test that errors.Unwrap works
	unwrapped := errors.Unwrap(wrappedErr)
	assert.Equal(t, baseErr, unwrapped)
}

func TestErrExhausted(t *testing.T) {
	t.Parallel()

	require.Error(t, ErrExhausted)
	assert.Equal(t, "retry budget exhausted", ErrExhausted.Error())
}

func TestTemporaryError_AllowsRetries(t *testing.T) {
	t.Parallel()

	callCount := 0
	tempErr := errors.New("temporary error") //nolint:err113 // Test error

	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++
		if callCount < 3 {
			return tempErr
		}

		return nil
	}, WithAttempts(5))

	require.NoError(t, err)
	assert.Equal(t, 3, callCount, "should retry temporary errors")
}

func TestAbort_WithWrappedError(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("base error")                     //nolint:err113 // Test error
	wrappedErr := errors.New("wrapped: " + baseErr.Error()) //nolint:err113 // Test error
	abortErr := Abort(wrappedErr)

	callCount := 0
	err := Do(t.Context(), func(ctx context.Context) error {
		callCount++

		return abortErr
	}, WithAttempts(10))

	require.Error(t, err)
	assert.Equal(t, 1, callCount)
	// Error should be unwrappable to the wrapped error
	assert.ErrorIs(t, err, wrappedErr)
}
