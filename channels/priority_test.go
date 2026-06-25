package channels

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// drainPriority writes all values into the priority pump, closes the input so the
// pump drains its heap, and returns the values in delivery order.
func drainPriority[T any](less func(a, b T) bool, values ...T) []T {
	in, out, _ := CreatePriority(context.Background(), 0, less)

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

func TestCreatePriority_DeliversHighestFirst(t *testing.T) {
	t.Parallel()

	got := drainPriority(func(a, b int) bool { return a > b }, 2, 5, 1, 3, 4)

	assert.Equal(t, []int{5, 4, 3, 2, 1}, got)
}

func TestCreatePriority_DeliversLowestFirst(t *testing.T) {
	t.Parallel()

	got := drainPriority(func(a, b int) bool { return a < b }, 2, 5, 1, 3, 4)

	assert.Equal(t, []int{1, 2, 3, 4, 5}, got)
}

func TestCreatePriority_EqualPriorityKeepsFIFO(t *testing.T) {
	t.Parallel()

	type item struct {
		prio int
		id   int
	}

	// All equal priority: delivery must follow submission order by id.
	got := drainPriority(
		func(a, b item) bool { return a.prio > b.prio },
		item{prio: 7, id: 1},
		item{prio: 7, id: 2},
		item{prio: 7, id: 3},
		item{prio: 7, id: 4},
	)

	ids := make([]int, 0, len(got))
	for _, it := range got {
		ids = append(ids, it.id)
	}

	assert.Equal(t, []int{1, 2, 3, 4}, ids)
}

func TestCreatePriority_MixedPriorityStableWithinWeight(t *testing.T) {
	t.Parallel()

	type item struct {
		prio int
		id   int
	}

	// Two priority tiers; within each tier submission order must be preserved.
	got := drainPriority(
		func(a, b item) bool { return a.prio > b.prio },
		item{prio: 1, id: 1},
		item{prio: 2, id: 2},
		item{prio: 1, id: 3},
		item{prio: 2, id: 4},
		item{prio: 1, id: 5},
	)

	ids := make([]int, 0, len(got))
	for _, it := range got {
		ids = append(ids, it.id)
	}

	// prio 2 tier first (ids 2,4 in order), then prio 1 tier (ids 1,3,5 in order).
	assert.Equal(t, []int{2, 4, 1, 3, 5}, ids)
}

func TestCreatePriority_BoundedBlocksWhenFull(t *testing.T) {
	t.Parallel()

	const maxSize = 2

	in, out, count := CreatePriority(context.Background(), maxSize, func(a, b int) bool { return a > b })

	// Nothing reads from out, so these fill the bounded buffer.
	for _, v := range []int{1, 2} {
		in <- v
	}

	require.Eventually(t, func() bool {
		return count() == maxSize
	}, time.Second, time.Millisecond, "buffer should fill to its bound")

	// A further send must block until the consumer drains an item.
	sent := make(chan struct{})

	go func() {
		in <- 3

		close(sent)
	}()

	select {
	case <-sent:
		t.Fatal("send should block while the bounded buffer is full")
	case <-time.After(100 * time.Millisecond):
		// Expected: still blocked.
	}

	<-out // make room

	select {
	case <-sent:
		// Expected: the blocked send proceeds now that there is room.
	case <-time.After(time.Second):
		t.Fatal("send did not unblock after the buffer was drained")
	}
}

func TestCreatePriority_ClosesOutputWhenInputClosed(t *testing.T) {
	t.Parallel()

	in, out, _ := CreatePriority(context.Background(), 0, func(a, b int) bool { return a > b })

	close(in)

	select {
	case _, ok := <-out:
		assert.False(t, ok, "output channel should be closed and empty")
	case <-time.After(time.Second):
		t.Fatal("output channel was not closed after input closed")
	}
}

func TestCreatePriority_ContextCancelClosesOutput(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	in, out, _ := CreatePriority(ctx, 0, func(a, b int) bool { return a > b })

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

func TestCreatePriority_LengthFunctionReportsBuffered(t *testing.T) {
	t.Parallel()

	in, out, count := CreatePriority(context.Background(), 0, func(a, b int) bool { return a > b })

	const total = 6

	// Nothing reads from out, so every submitted value accumulates in the heap.
	for i := range total {
		in <- i
	}

	require.Eventually(t, func() bool {
		return count() == total
	}, time.Second, time.Millisecond, "all submitted values should be buffered")

	close(in)

	for range out { //nolint:revive
	}

	assert.Equal(t, 0, count(), "buffer should be empty after draining")
}
