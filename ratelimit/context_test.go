package ratelimit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

type fakeLimiter struct{}

func (*fakeLimiter) Reserve(context.Context, int) (Reservation, error) {
	return nil, nil //nolint:nilnil
}

func TestRateLimiterContext(t *testing.T) {
	t.Parallel()

	var typedNil *fakeLimiter

	lim := &fakeLimiter{}
	outer := WithRateLimiter(context.Background(), lim)

	tests := []struct {
		name string
		ctx  context.Context //nolint:containedctx
		want Limiter
	}{
		{name: "stored limiter", ctx: outer, want: lim},
		{name: "nothing stored", ctx: context.Background()},
		{name: "nil context", ctx: nil},
		{name: "nil limiter", ctx: WithRateLimiter(context.Background(), nil)},
		{name: "typed nil limiter", ctx: WithRateLimiter(context.Background(), typedNil)},
		{name: "nil hides outer", ctx: WithRateLimiter(outer, nil)},
		{name: "typed nil hides outer", ctx: WithRateLimiter(outer, typedNil)},
		{name: "nil parent context", ctx: WithRateLimiter(nil, lim), want: lim}, //nolint:staticcheck
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got, ok := GetRateLimiter(testCase.ctx)

			if testCase.want == nil {
				assert.False(t, ok)
				assert.Nil(t, got)
				assert.False(t, HasRateLimiter(testCase.ctx))

				return
			}

			assert.True(t, ok)
			assert.Same(t, testCase.want, got)
			assert.True(t, HasRateLimiter(testCase.ctx))
		})
	}
}
