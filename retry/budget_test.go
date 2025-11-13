package retry

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBudget_NilBudget(t *testing.T) {
	t.Parallel()

	var budget *Budget

	assert.True(t, budget.sendOK(false), "nil budget should always allow")
	assert.True(t, budget.sendOK(true), "nil budget should always allow")
}

func TestBudget_InitialCallsAlwaysAllowed(t *testing.T) {
	t.Parallel()

	budget := &Budget{
		Rate:  10.0,
		Ratio: 0.1,
	}

	// Initial calls should always be allowed
	for range 100 {
		assert.True(t, budget.sendOK(false), "initial calls should always be allowed")
	}
}

func TestBudget_RetriesLimitedWhenOverloaded(t *testing.T) {
	t.Parallel()

	budget := &Budget{
		Rate:  5.0, // Allow retries when rate > 5 req/sec
		Ratio: 0.1, // Allow 10% retries
	}

	now := time.Now()

	// Send many initial calls in a short time to exceed rate
	for range 20 {
		budget.initialCalls = &movingRate{
			BucketLength: time.Second,
			BucketNum:    60,
			counts:       []int{20},
			lastUpdate:   now,
		}
	}

	// Send retries - some should be blocked
	allowed := 0

	for range 10 {
		if budget.sendOK(true) {
			allowed++
		}
	}

	// Not all retries should be allowed when budget is exhausted
	assert.Less(t, allowed, 10, "some retries should be blocked when budget exhausted")
}

func TestTimeRoundDown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		time     time.Time
		duration time.Duration
		expected time.Time
	}{
		{
			name:     "round down to second",
			time:     time.Date(2024, 1, 1, 12, 30, 45, 500000000, time.UTC),
			duration: time.Second,
			expected: time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
		},
		{
			name:     "round down to minute",
			time:     time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC),
			duration: time.Minute,
			expected: time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
		},
		{
			name:     "already aligned",
			time:     time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
			duration: time.Minute,
			expected: time.Date(2024, 1, 1, 12, 30, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := timeRoundDown(tt.time, tt.duration)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMovingRate_NewMovingRate(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()

	assert.Equal(t, time.Second, mr.BucketLength)
	assert.Equal(t, 60, mr.BucketNum)
	assert.Nil(t, mr.counts)
	assert.True(t, mr.lastUpdate.IsZero())
}

func TestMovingRate_AddAndRate(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	now := time.Now()

	// Add events
	mr.Add(now, 10)
	mr.Add(now.Add(500*time.Millisecond), 5)
	mr.Add(now.Add(1*time.Second), 3)

	// Calculate rate
	rate := mr.Rate(now.Add(1 * time.Second))

	// Rate should be total events / time span
	assert.Greater(t, rate, 0.0)
	assert.Less(t, rate, 20.0) // Should be less than 18 events/sec
}

func TestMovingRate_BackwardTimeIgnored(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	now := time.Now()

	mr.Add(now, 10)
	count1 := mr.count()

	// Try to add event in the past - should be ignored
	mr.Add(now.Add(-1*time.Second), 5)
	count2 := mr.count()

	assert.InDelta(t, count1, count2, 0.01, "events in the past should be ignored")
}

func TestMovingRate_Forward(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	now := time.Now()

	// Initialize with first update
	mr.forward(now)
	assert.Len(t, mr.counts, 1)
	assert.Equal(t, now, mr.lastUpdate)

	// Forward by 5 seconds
	mr.forward(now.Add(5 * time.Second))
	assert.Greater(t, len(mr.counts), 1, "should have added buckets")
}

func TestMovingRate_Shift(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	mr.counts = []int{1, 2, 3, 4, 5}
	mr.lastUpdate = time.Now()

	// Shift forward
	mr.shift(3)

	// Should have added 3 new buckets
	assert.GreaterOrEqual(t, len(mr.counts), 5)

	// Should maintain max of BucketNum+1 buckets
	assert.LessOrEqual(t, len(mr.counts), mr.BucketNum+1)
}

func TestMovingRate_Count_NotFullyInitialized(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	mr.counts = []int{10, 20, 30}

	count := mr.count()
	assert.InDelta(t, 60.0, count, 0.01, "should sum all buckets when not fully initialized")
}

func TestMovingRate_Count_FullyInitialized(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	mr.BucketNum = 5
	mr.BucketLength = time.Second
	mr.lastUpdate = time.Date(2024, 1, 1, 12, 0, 0, 500000000, time.UTC)

	// Create 6 buckets (BucketNum + 1)
	mr.counts = []int{10, 20, 30, 40, 50, 60}

	count := mr.count()

	// Should apply fractional multiplier to oldest bucket
	assert.Greater(t, count, 200.0) // More than sum of buckets 2-6
	assert.Less(t, count, 210.0)    // But less than full sum
}

func TestMovingRate_Second_NotFullyInitialized(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	mr.BucketLength = time.Second
	mr.lastUpdate = time.Date(2024, 1, 1, 12, 0, 3, 500000000, time.UTC)
	mr.counts = []int{10, 20, 30, 40}

	seconds := mr.second()

	// Should calculate actual time span
	assert.Greater(t, seconds, 3.0)
	assert.Less(t, seconds, 4.0)
}

func TestMovingRate_Second_FullyInitialized(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	mr.BucketNum = 60
	mr.BucketLength = time.Second
	mr.counts = make([]int, 61) // BucketNum + 1

	seconds := mr.second()

	assert.InDelta(t, 60.0, seconds, 0.01, "should return fixed window size when fully initialized")
}

func TestMovingRate_RateNaN_BackwardTime(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	now := time.Now()

	mr.Add(now, 10)

	// Try to get rate for time in the past
	rate := mr.Rate(now.Add(-1 * time.Second))
	assert.True(t, math.IsNaN(rate), "rate should be NaN for backward time")
}

func TestMovingRate_LargeTimeGap(t *testing.T) {
	t.Parallel()

	mr := newMovingRate()
	now := time.Now()

	mr.Add(now, 10)

	// Add event far in the future (more than BucketNum buckets away)
	mr.Add(now.Add(120*time.Second), 5)

	// Old data should be completely shifted out
	rate := mr.Rate(now.Add(120 * time.Second))
	assert.Greater(t, rate, 0.0)
	assert.Less(t, rate, 1.0, "rate should be low after long gap")
}

func TestBudget_Overload(t *testing.T) {
	t.Parallel()

	budget := &Budget{
		Rate:  0.5,
		Ratio: 0.2, // 20% retries allowed
	}

	now := time.Now()

	budget.initialCalls = &movingRate{
		BucketLength: time.Second,
		BucketNum:    60,
		lastUpdate:   now,
	}

	// Simulate high load: many initial calls
	for range 50 {
		budget.initialCalls.counts = append(budget.initialCalls.counts, 1)
	}

	for range 10 {
		budget.initialCalls.counts = append(budget.initialCalls.counts, 0)
	}

	budget.retriedCalls = &movingRate{
		BucketLength: time.Second,
		BucketNum:    60,
		lastUpdate:   now,
	}

	// Simulate many retries
	for range 15 {
		budget.retriedCalls.counts = append(budget.retriedCalls.counts, 1)
	}

	for range 45 {
		budget.retriedCalls.counts = append(budget.retriedCalls.counts, 0)
	}

	// System should be overloaded (rate > 10 and retry ratio > 20%)
	overloaded := budget.overload(false)
	assert.True(t, overloaded)
}

func TestBudget_NotOverload_LowRate(t *testing.T) {
	t.Parallel()

	budget := &Budget{
		Rate:  100.0, // High threshold
		Ratio: 0.1,
	}

	now := time.Now()

	// Simulate low load
	budget.initialCalls = &movingRate{
		BucketLength: time.Second,
		BucketNum:    60,
		counts:       []int{10},
		lastUpdate:   now,
	}

	budget.retriedCalls = &movingRate{
		BucketLength: time.Second,
		BucketNum:    60,
		counts:       []int{2},
		lastUpdate:   now,
	}

	overloaded := budget.overload(false)
	assert.False(t, overloaded, "should not be overloaded at low rate")
}
