package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestExpBackoff_Delay(t *testing.T) {
	t.Parallel()

	backoff := ExpBackoff{
		Base:   100 * time.Millisecond,
		Max:    2 * time.Second,
		Factor: 2.0,
	}

	tests := []struct {
		name     string
		attempt  uint
		expected time.Duration
	}{
		{"first attempt", 0, 100 * time.Millisecond},
		{"second attempt", 1, 200 * time.Millisecond},
		{"third attempt", 2, 400 * time.Millisecond},
		{"fourth attempt", 3, 800 * time.Millisecond},
		{"fifth attempt", 4, 1600 * time.Millisecond},
		{"sixth attempt (hits max)", 5, 2 * time.Second},
		{"seventh attempt (capped)", 6, 2 * time.Second},
		{"tenth attempt (still capped)", 10, 2 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			delay := backoff.Delay(tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestExpBackoff_DifferentFactors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		factor   float64
		attempt  uint
		expected time.Duration
	}{
		{"factor 1.5", 1.5, 3, 337500 * time.Microsecond}, // 100ms * 1.5^3 = 337.5ms
		{"factor 3.0", 3.0, 2, 900 * time.Millisecond},
		{"factor 1.0 (no growth)", 1.0, 5, 100 * time.Millisecond},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			backoff := ExpBackoff{
				Base:   100 * time.Millisecond,
				Max:    10 * time.Second,
				Factor: tt.factor,
			}
			delay := backoff.Delay(tt.attempt)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestExpBackoff_MinimumIsBase(t *testing.T) {
	t.Parallel()

	backoff := ExpBackoff{
		Base:   500 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 2.0,
	}

	// First attempt should always return at least Base
	delay := backoff.Delay(0)
	assert.Equal(t, 500*time.Millisecond, delay)
}

func TestExpBackoff_MaximumCap(t *testing.T) {
	t.Parallel()

	backoff := ExpBackoff{
		Base:   100 * time.Millisecond,
		Max:    1 * time.Second,
		Factor: 10.0, // Very aggressive growth
	}

	// Should quickly hit the max
	delay := backoff.Delay(3)
	assert.Equal(t, 1*time.Second, delay)

	// Should stay at max
	delay = backoff.Delay(10)
	assert.Equal(t, 1*time.Second, delay)
}

func TestExpBackoff_ZeroFactor(t *testing.T) {
	t.Parallel()

	backoff := ExpBackoff{
		Base:   100 * time.Millisecond,
		Max:    10 * time.Second,
		Factor: 0.0,
	}

	// With factor 0, all attempts should return Base
	for i := range uint(10) {
		delay := backoff.Delay(i)
		assert.Equal(t, 100*time.Millisecond, delay)
	}
}
