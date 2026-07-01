package gpqinbox

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/amp-labs/amp-common/actor"
)

// item is a small weighted payload used to assert delivery order.
type item struct {
	id     int
	weight int
}

func weightOfItem(i item) int { return i.weight }

// drainInbox writes all values into the inbox, closes the input so the pump
// drains its queue, and returns the values in delivery order.
//
// Note the pump holds a single item out of the queue at a time (gpq has no
// peek), so the first value written is always delivered first regardless of its
// priority; every value after it is delivered in strict priority order.
func drainInbox[T any](t *testing.T, cfg Config, weightOf func(T) int, values ...T) []T {
	t.Helper()

	in, out, _, err := newInbox(context.Background(), cfg, weightOf)
	require.NoError(t, err)

	for _, v := range values {
		in <- v
	}

	close(in)

	got := make([]T, 0, len(values))
	for v := range out {
		got = append(got, v)
	}

	return got
}

func idsOf(items []item) []int {
	ids := make([]int, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.id)
	}

	return ids
}

func TestDefaultWeightToBucket(t *testing.T) {
	t.Parallel()

	const buckets = 3

	// Higher weight => lower (higher-priority) bucket; clamped to [0, buckets).
	assert.Equal(t, uint(2), DefaultWeightToBucket(0, buckets), "weight 0 => lowest bucket")
	assert.Equal(t, uint(1), DefaultWeightToBucket(1, buckets))
	assert.Equal(t, uint(0), DefaultWeightToBucket(2, buckets), "weight 2 => highest bucket")
	assert.Equal(t, uint(0), DefaultWeightToBucket(99, buckets), "over-large weight clamps to bucket 0")
	assert.Equal(t, uint(2), DefaultWeightToBucket(-5, buckets), "negative weight clamps to lowest bucket")
}

func TestNewInbox_PriorityOrderingAcrossBuckets(t *testing.T) {
	t.Parallel()

	// Buckets: weight 2 => bucket 0 (highest), weight 1 => bucket 1,
	// weight 0 => bucket 2 (lowest). The first value (id 0) is held out of the
	// queue and delivered first; the remainder come out in priority order, with
	// FIFO preserved inside each bucket.
	got := drainInbox(t, Config{Buckets: 3}, weightOfItem,
		item{id: 0, weight: 0}, // held, delivered first
		item{id: 1, weight: 2}, // bucket 0
		item{id: 2, weight: 1}, // bucket 1
		item{id: 3, weight: 2}, // bucket 0
		item{id: 4, weight: 0}, // bucket 2
	)

	// id 0 (held) first, then bucket 0 (ids 1,3 FIFO), then bucket 1 (id 2),
	// then bucket 2 (id 4).
	assert.Equal(t, []int{0, 1, 3, 2, 4}, idsOf(got))
}

func TestNewInbox_FIFOWithinBucket(t *testing.T) {
	t.Parallel()

	// All equal weight => same bucket => submission order is preserved.
	got := drainInbox(t, Config{Buckets: 3}, weightOfItem,
		item{id: 0, weight: 1},
		item{id: 1, weight: 1},
		item{id: 2, weight: 1},
		item{id: 3, weight: 1},
	)

	assert.Equal(t, []int{0, 1, 2, 3}, idsOf(got))
}

func TestNewInbox_BoundedBlocksWhenFull(t *testing.T) {
	t.Parallel()

	const maxSize = 2

	input, output, count, err := newInbox(
		context.Background(),
		Config{Buckets: 2, MaxSize: maxSize},
		func(i int) int { return i },
	)
	require.NoError(t, err)

	// Nothing reads from output, so these fill the bounded buffer (one held by
	// the pump, one in the queue).
	for _, v := range []int{1, 2} {
		input <- v
	}

	require.Eventually(t, func() bool {
		return count() == maxSize
	}, time.Second, time.Millisecond, "buffer should fill to its bound")

	// A further send must block until the consumer drains an item.
	sent := make(chan struct{})

	go func() {
		input <- 3

		close(sent)
	}()

	select {
	case <-sent:
		t.Fatal("send should block while the bounded buffer is full")
	case <-time.After(100 * time.Millisecond):
		// Expected: still blocked.
	}

	<-output // make room

	select {
	case <-sent:
		// Expected: the blocked send proceeds now that there is room.
	case <-time.After(time.Second):
		t.Fatal("send did not unblock after the buffer was drained")
	}
}

func TestNewInbox_ClosesOutputWhenInputClosed(t *testing.T) {
	t.Parallel()

	in, out, _, err := newInbox(context.Background(), Config{Buckets: 2}, func(i int) int { return i })
	require.NoError(t, err)

	close(in)

	select {
	case _, ok := <-out:
		assert.False(t, ok, "output channel should be closed and empty")
	case <-time.After(time.Second):
		t.Fatal("output channel was not closed after input closed")
	}
}

func TestNewInbox_ContextCancelClosesOutput(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	in, out, _, err := newInbox(ctx, Config{Buckets: 2}, func(i int) int { return i })
	require.NoError(t, err)

	for _, v := range []int{1, 2} {
		in <- v
	}

	cancel()

	// After cancellation the pump stops and closes the output. Any buffered
	// values may or may not have been delivered, so just drain until closed.
	done := make(chan struct{})

	go func() {
		defer close(done)

		for range out { //nolint:revive
		}
	}()

	select {
	case <-done:
		// Output channel closed as expected.
	case <-time.After(time.Second):
		t.Fatal("output channel was not closed after context cancellation")
	}
}

func TestNewInbox_EscalationDeliversEverything(t *testing.T) {
	t.Parallel()

	// Escalation is intra-bucket and timing dependent; this is a smoke test that
	// enabling it (and driving Prioritize on a ticker) does not break delivery.
	cfg := Config{
		Buckets:         2,
		Escalate:        true,
		EscalationRate:  time.Millisecond,
		PrioritizeEvery: 2 * time.Millisecond,
	}

	input, output, _, err := newInbox(context.Background(), cfg, func(i int) int { return i })
	require.NoError(t, err)

	const total = 20

	seen := make(map[int]struct{}, total)

	done := make(chan struct{})

	go func() {
		defer close(done)

		for v := range output {
			seen[v] = struct{}{}
		}
	}()

	for i := range total {
		input <- i
		// Let a few prioritize ticks elapse mid-stream.
		time.Sleep(time.Millisecond)
	}

	close(input)
	<-done

	assert.Len(t, seen, total, "every submitted value should be delivered exactly once")
}

// recordingActor builds an actor that appends each processed request to a
// shared slice and returns a reference plus an accessor for the recorded order.
func recordingActor[T any](t *testing.T, ctx context.Context, cfg Config) (*actor.Ref[T, empty], func() []T) {
	t.Helper()

	var (
		mu        sync.Mutex //nolint:varnamelen // conventional mutex name
		processed []T
	)

	act := actor.New(func(_ *actor.Ref[T, empty]) actor.Processor[T, empty] {
		return actor.SimpleProcessor(func(req T) (empty, error) {
			mu.Lock()
			defer mu.Unlock()

			processed = append(processed, req)

			return empty{}, nil
		})
	})

	ref, err := Run(ctx, act, "recorder", cfg)
	require.NoError(t, err)

	return ref, func() []T {
		mu.Lock()
		defer mu.Unlock()

		out := make([]T, len(processed))
		copy(out, processed)

		return out
	}
}

type empty struct{}

func TestRun_RequestResponse(t *testing.T) {
	t.Parallel()

	act := actor.New(func(_ *actor.Ref[int, int]) actor.Processor[int, int] {
		return actor.SimpleProcessor(func(req int) (int, error) {
			return req * 2, nil
		})
	})

	ref, err := Run(t.Context(), act, "doubler", Config{Buckets: 3})
	require.NoError(t, err)

	defer ref.Stop()

	result, err := ref.RequestCtxWithWeight(t.Context(), 21, 2)
	require.NoError(t, err)
	assert.Equal(t, 42, result)
}

func TestRun_StopDrainsBufferedMessages(t *testing.T) {
	t.Parallel()

	ref, recorded := recordingActor[int](t, t.Context(), Config{Buckets: 3})

	const total = 50

	for i := range total {
		ref.SendWithWeight(i, i%3)
	}

	// Stop closes the inbox; buffered messages drain and are processed before
	// the run loop exits.
	ref.Stop()
	ref.Wait()

	assert.Len(t, recorded(), total, "all buffered messages should be processed on graceful stop")
}

func TestRun_HigherWeightProcessedFirstUnderBacklog(t *testing.T) {
	t.Parallel()

	// Hold the actor busy on a gate while a backlog accumulates, then release it
	// and check that higher-weight messages were processed ahead of lower-weight
	// ones. The single item the inbox holds out of the queue (the first backlog
	// message) is exempt from the priority ordering, so we assert on the tail.
	var (
		mu        sync.Mutex //nolint:varnamelen // conventional mutex name
		processed []int
	)

	gate := make(chan struct{})
	firstSeen := make(chan struct{})

	var once sync.Once

	act := actor.New(func(_ *actor.Ref[int, empty]) actor.Processor[int, empty] {
		return actor.SimpleProcessor(func(req int) (empty, error) {
			once.Do(func() {
				// Signal that the actor has picked up the primer, then block so a
				// backlog builds up behind it.
				close(firstSeen)
				<-gate
			})

			mu.Lock()
			defer mu.Unlock()

			processed = append(processed, req)

			return empty{}, nil
		})
	})

	ref, err := Run(t.Context(), act, "backlog", Config{Buckets: 4})
	require.NoError(t, err)

	defer ref.Stop()

	// Primer occupies the actor and gets held open until we close the gate.
	ref.SendWithWeight(-1, 0)
	<-firstSeen

	// Backlog: interleave low and high weights. weight 3 => bucket 0 (highest),
	// weight 0 => bucket 3 (lowest).
	backlog := []struct{ val, weight int }{
		{val: 10, weight: 0},
		{val: 11, weight: 0},
		{val: 20, weight: 3},
		{val: 21, weight: 3},
		{val: 12, weight: 0},
		{val: 22, weight: 3},
	}
	for _, b := range backlog {
		ref.SendWithWeight(b.val, b.weight)
	}

	close(gate)

	// Wait until the whole backlog (primer + 6) has been processed.
	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()

		return len(processed) == len(backlog)+1
	}, 2*time.Second, time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	got := append([]int(nil), processed...)

	// processed[0] is the primer (-1). processed[1] is the single held backlog
	// item (10, the first backlog message), exempt from priority ordering.
	// Everything after that must have all high-weight (20/21/22) values ahead of
	// the remaining low-weight (11/12) values.
	require.Equal(t, -1, got[0])
	require.Equal(t, 10, got[1])

	tail := got[2:]
	highSeen := 0

	for _, v := range tail {
		switch v {
		case 20, 21, 22:
			highSeen++
		case 11, 12:
			assert.Equal(t, 3, highSeen, "all high-weight messages must be processed before low-weight ones (offender %d)", v)
		}
	}
}
