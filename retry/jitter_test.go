package retry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWithoutJitter(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	result := WithoutJitter.jitter(delay)

	assert.Equal(t, delay, result, "WithoutJitter should return exact delay")
}

func TestWithoutJitter_MultipleCallsConsistent(t *testing.T) {
	t.Parallel()

	delay := 500 * time.Millisecond

	for range 100 {
		result := WithoutJitter.jitter(delay)
		assert.Equal(t, delay, result, "WithoutJitter should always return exact delay")
	}
}

func TestFullJitter(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	results := make(map[time.Duration]bool)

	// Run multiple times to check randomness
	for range 100 {
		result := FullJitter.jitter(delay)
		results[result] = true

		// FullJitter should return between 0 and delay
		assert.GreaterOrEqual(t, result, time.Duration(0))
		assert.LessOrEqual(t, result, delay)
	}

	// Should have multiple different values (randomness check)
	assert.Greater(t, len(results), 10, "FullJitter should produce varied results")
}

func TestEqualJitter(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	sum := time.Duration(0)
	iterations := 1000

	for range iterations {
		result := EqualJitter.jitter(delay)
		sum += result

		// EqualJitter should return between delay/2 and delay
		assert.GreaterOrEqual(t, result, delay/2)
		assert.LessOrEqual(t, result, delay)
	}

	// Average should be around 75% of delay (halfway between 50% and 100%)
	average := sum / time.Duration(iterations)
	expected := (3 * delay) / 4 // 75%

	// Allow 10% variance due to randomness
	lowerBound := expected - (expected / 10)
	upperBound := expected + (expected / 10)

	assert.GreaterOrEqual(t, average, lowerBound)
	assert.LessOrEqual(t, average, upperBound)
}

func TestJitter_ZeroDelay(t *testing.T) {
	t.Parallel()

	delay := time.Duration(0)

	tests := []struct {
		name   string
		jitter Jitter
	}{
		{"WithoutJitter", WithoutJitter},
		{"FullJitter", FullJitter},
		{"EqualJitter", EqualJitter},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.jitter.jitter(delay)
			assert.Equal(t, time.Duration(0), result)
		})
	}
}

func TestJitter_NegativeValue(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	negativeJitter := Jitter(-0.5)

	result := negativeJitter.jitter(delay)
	assert.Equal(t, delay, result, "negative jitter should act like WithoutJitter")
}

func TestJitter_CustomValue(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	customJitter := Jitter(0.25) // 25% random, 75% deterministic

	sum := time.Duration(0)
	iterations := 1000

	for range iterations {
		result := customJitter.jitter(delay)
		sum += result

		// Should be between 75% and 100% of delay
		assert.GreaterOrEqual(t, result, (3*delay)/4)
		assert.LessOrEqual(t, result, delay)
	}

	// Average should be around 87.5% of delay
	average := sum / time.Duration(iterations)
	expected := (7 * delay) / 8 // 87.5%

	// Allow 10% variance
	lowerBound := expected - (expected / 10)
	upperBound := expected + (expected / 10)

	assert.GreaterOrEqual(t, average, lowerBound)
	assert.LessOrEqual(t, average, upperBound)
}

func TestJitter_ExactlyOne(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	oneJitter := Jitter(1.0)

	// Jitter value of exactly 1.0 should behave like FullJitter
	for range 100 {
		result := oneJitter.jitter(delay)
		assert.GreaterOrEqual(t, result, time.Duration(0))
		assert.LessOrEqual(t, result, delay)
	}
}

func TestJitter_ExactlyZero(t *testing.T) {
	t.Parallel()

	delay := 1 * time.Second
	zeroJitter := Jitter(0.0)

	// Jitter value of exactly 0.0 still applies some randomization
	// It's effectively random(0, delay), same as other positive jitter values
	result := zeroJitter.jitter(delay)
	assert.GreaterOrEqual(t, result, time.Duration(0))
	assert.LessOrEqual(t, result, delay)
}
