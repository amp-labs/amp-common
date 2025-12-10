package retry

import (
	"fmt"
	"math"
	"sync"
	"time"
)

const (
	defaultBucketCount = 60 // Number of time buckets for rate tracking
)

// Budget implements a retry budget to prevent cascading failures and retry storms.
// It tracks the rate of initial calls and retried calls over time, and prevents
// retries when the system is overloaded.
//
// The budget uses two parameters:
//   - Rate: The minimum initial request rate (requests/second) before budget enforcement kicks in
//   - Ratio: The maximum allowed ratio of retries to initial requests (e.g., 0.1 = 10% retries)
//
// This implements the "retry budget" pattern described in the SRE book to prevent
// cascading failures where retries consume more resources than initial requests.
//
// Example:
//
//	budget := &retry.Budget{
//	    Rate:  10.0,  // Only enforce budget when > 10 requests/sec
//	    Ratio: 0.1,   // Allow up to 10% of requests to be retries
//	}
type Budget struct {
	// Rate is the minimum initial request rate (req/sec) before budget enforcement begins.
	Rate float64
	// Ratio is the maximum allowed ratio of retried requests to initial requests.
	Ratio float64

	mu           sync.Mutex
	initialCalls *movingRate
	retriedCalls *movingRate
}

// sendOK determines whether a retry attempt should be allowed based on the current
// retry budget. It prevents retry storms by limiting retries when the system is
// under heavy load.
//
// The function returns:
//   - true if the attempt is allowed (either initial call or budget permits retry)
//   - false if the retry budget is exhausted
//
// Initial calls are always allowed and count toward the budget. Retries are only
// allowed if:
//  1. The initial request rate is below the Rate threshold, OR
//  2. The ratio of retries to initial requests is below the Ratio threshold
func (b *Budget) sendOK(isRetry bool) bool {
	if b == nil {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.retriedCalls == nil {
		b.retriedCalls = newMovingRate()
	}

	if b.initialCalls == nil {
		b.initialCalls = newMovingRate()
	}

	currentTime := time.Now()

	// Initial calls are always allowed
	if !isRetry {
		b.initialCalls.Add(currentTime, 1)

		return true
	}

	// Check if retry budget is exhausted
	initialRate := b.initialCalls.Rate(currentTime)
	retriedRate := b.retriedCalls.Rate(currentTime)

	if initialRate > b.Rate &&
		retriedRate/initialRate > b.Ratio {
		return false
	}

	b.retriedCalls.Add(currentTime, 1)

	return true
}

// overload checks if the system is currently overloaded based on the retry budget.
// It returns true if the total request rate exceeds the Rate threshold AND the
// ratio of retries exceeds the Ratio threshold.
//
// This method is useful for monitoring and alerting, but is not currently used
// in the retry decision logic (sendOK is used instead).
func (b *Budget) overload(isRetry bool) bool {
	if b == nil {
		return true
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if b.retriedCalls == nil {
		b.retriedCalls = newMovingRate()
	}

	if b.initialCalls == nil {
		b.initialCalls = newMovingRate()
	}

	currentTime := time.Now()

	if isRetry {
		b.retriedCalls.Add(currentTime, 1)
	} else {
		b.initialCalls.Add(currentTime, 1)
	}

	initialRate := b.initialCalls.Rate(currentTime)
	retriedRate := b.retriedCalls.Rate(currentTime)
	totalRate := initialRate + retriedRate

	return totalRate > b.Rate && retriedRate/totalRate > b.Ratio
}

// timeRoundDown rounds a time down to the nearest multiple of the duration.
// This is used by movingRate to align timestamps to bucket boundaries.
//
// Example: timeRoundDown(10:15:37, 1min) = 10:15:00.
func timeRoundDown(t time.Time, d time.Duration) time.Time {
	rt := t.Round(d)
	if rt.After(t) {
		rt = rt.Add(-d)
	}

	return rt
}

// movingRate tracks the rate of events over a sliding time window using time buckets.
// It maintains a fixed number of time buckets and computes the rate by counting
// events in those buckets, accounting for partially-filled oldest and newest buckets.
//
// The implementation uses a circular buffer of counts, where each bucket represents
// a time interval (BucketLength). As time advances, old buckets are shifted out and
// new empty buckets are added.
type movingRate struct {
	// BucketLength is the duration of each time bucket (e.g., 1 second).
	BucketLength time.Duration
	// BucketNum is the number of buckets to maintain (e.g., 60 for 60-second window).
	BucketNum int

	counts     []int     // Count of events in each bucket
	lastUpdate time.Time // Timestamp of the last update
}

// newMovingRate creates a new movingRate with default settings:
// 60 buckets of 1 second each, giving a 60-second sliding window.
func newMovingRate() *movingRate {
	return &movingRate{
		BucketLength: time.Second,
		BucketNum:    defaultBucketCount,
	}
}

// Add increments the event count at time t by n. Events with timestamps before
// the last update are ignored (time cannot move backward in this data structure).
func (mr *movingRate) Add(t time.Time, n int) {
	if t.Before(mr.lastUpdate) {
		return
	}

	mr.forward(t)
	mr.counts[len(mr.counts)-1] += n
}

// Rate computes the current event rate (events per second) at time t.
// Returns NaN if the timestamp is before the last update.
func (mr *movingRate) Rate(t time.Time) float64 {
	if t.Before(mr.lastUpdate) {
		return math.NaN()
	}

	mr.forward(t)

	return mr.count() / mr.second()
}

// count returns the total count of events in the sliding window, accounting for
// partial buckets. When the history is not fully initialized (fewer buckets than
// BucketNum), it sums all available buckets. When fully initialized, it applies
// a fractional multiplier to the oldest bucket since it may be partially outside
// the window.
func (mr *movingRate) count() float64 {
	// History is not yet fully initialized - sum all buckets
	if len(mr.counts) <= mr.BucketNum {
		var s float64
		for _, c := range mr.counts {
			s += float64(c)
		}

		return s
	}

	// Calculate the fraction of the oldest bucket that falls within the window
	oldestFraction := 1.0 -
		float64(mr.lastUpdate.Sub(timeRoundDown(mr.lastUpdate, mr.BucketLength)))/
			float64(mr.BucketLength)

	// Apply fractional multiplier to oldest bucket, sum the rest fully
	s := oldestFraction * float64(mr.counts[0])
	for i := 1; i < len(mr.counts); i++ {
		s += float64(mr.counts[i])
	}

	return s
}

// second returns the time span in seconds covered by the current sliding window.
// When the history is not fully initialized, it calculates the actual time span.
// When fully initialized, it returns the configured window size (BucketNum * BucketLength).
func (mr *movingRate) second() float64 {
	if len(mr.counts) == 0 {
		return 0.0
	}

	// History is not yet fully initialized - calculate actual time span
	if len(mr.counts) <= mr.BucketNum {
		d := time.Duration(len(mr.counts)-1) * mr.BucketLength
		d += mr.lastUpdate.Sub(timeRoundDown(mr.lastUpdate, mr.BucketLength))

		return d.Seconds()
	}

	// Fully initialized - return fixed window size
	d := time.Duration(mr.BucketNum) * mr.BucketLength

	return d.Seconds()
}

// shift adds numBuckets new empty buckets to the end of the counts array and removes old buckets
// to maintain a maximum of BucketNum+1 buckets. The +1 allows for partial bucket handling
// at both ends of the window.
func (mr *movingRate) shift(numBuckets int) {
	if numBuckets > mr.BucketNum+1 {
		numBuckets = mr.BucketNum + 1
	}

	// Add numBuckets new empty buckets
	zero := make([]int, numBuckets)
	mr.counts = append(mr.counts, zero...)

	// We actually keep BucketNum+1 buckets -- the newest and oldest
	// buckets are partially evaluated so the window length stays constant.
	if del := len(mr.counts) - (mr.BucketNum + 1); del > 0 {
		mr.counts = mr.counts[del:]
	}

	// Update lastUpdate to the rounded time after shifting numBuckets buckets
	mr.lastUpdate = timeRoundDown(mr.lastUpdate, mr.BucketLength).Add(time.Duration(numBuckets) * mr.BucketLength)
}

// forward advances the moving window to the given time by shifting buckets as needed.
// If time moves backward, it's a no-op. If this is the first update, it initializes
// the first bucket.
func (mr *movingRate) forward(t time.Time) {
	defer func() {
		mr.lastUpdate = t
	}()

	// First update - initialize with a single bucket
	if mr.lastUpdate.IsZero() {
		mr.counts = []int{0}

		return
	}

	rt := timeRoundDown(t, mr.BucketLength)
	if !rt.After(mr.lastUpdate) {
		return
	}

	// Calculate how many bucket boundaries we've crossed
	n := int(rt.Sub(timeRoundDown(mr.lastUpdate, mr.BucketLength)) / mr.BucketLength)
	if n <= 0 {
		panic(fmt.Sprintf("assertion failure: n = %d, want >0; rt = %v, mr.lastUpdate = %v, mr.BucketLength = %v",
			n, rt, mr.lastUpdate, mr.BucketLength))
	}

	mr.shift(n)
}
