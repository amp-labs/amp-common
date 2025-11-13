package retry

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttempt_NoContext(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	attempt := Attempt(ctx)

	assert.Equal(t, uint(0), attempt, "should return 0 when no attempt in context")
}

func TestAttempt_WithContext(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	ctx = withAttempt(ctx, 5)

	attempt := Attempt(ctx)
	assert.Equal(t, uint(5), attempt)
}

func TestAttempt_InRetryLoop(t *testing.T) {
	t.Parallel()

	attempts := []uint{}
	err := Do(t.Context(), func(ctx context.Context) error {
		attempt := Attempt(ctx)
		attempts = append(attempts, attempt)

		if attempt < 3 {
			return errors.New("retry me") //nolint:err113 // Test error
		}

		return nil
	}, WithAttempts(5))

	require.NoError(t, err)
	assert.Equal(t, []uint{0, 1, 2, 3}, attempts, "attempts should be 0-indexed")
}

func TestWithAttempt(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	// First attempt
	ctx1 := withAttempt(ctx, 0)
	assert.Equal(t, uint(0), Attempt(ctx1))

	// Second attempt
	ctx2 := withAttempt(ctx, 1)
	assert.Equal(t, uint(1), Attempt(ctx2))

	// Contexts are independent
	assert.Equal(t, uint(0), Attempt(ctx1))
	assert.Equal(t, uint(1), Attempt(ctx2))
}
