package channels

import (
	"container/heap"
	"context"
	"sync/atomic"
)

// CreatePriority creates an unbounded, priority-ordered channel pump, mirroring
// the shape of Create and InfiniteChan: it returns a send-only channel, a
// receive-only channel, and a function reporting the number of buffered items.
//
// Values written to the send channel are buffered in an internal heap and
// delivered on the receive channel in priority order, as defined by less:
// less(a, b) reports whether a should be delivered before b. Values of equal
// priority — those for which neither less(a, b) nor less(b, a) holds — are
// delivered in FIFO order of submission.
//
// Like InfiniteChan, the buffer is unbounded, so the send channel never blocks
// on capacity and no backpressure is applied. Priority only has a visible effect
// when values accumulate faster than the consumer drains them; if the receiver
// keeps up with arrivals, delivery order closely tracks arrival order.
//
// Lifecycle:
//   - Closing the send channel drains all buffered values to the receive channel
//     in priority order, then closes the receive channel.
//   - Canceling ctx stops processing immediately, closes the receive channel,
//     and discards any values still buffered.
//
// The design follows github.com/brunoga/prioritychannel, adapted to this
// package's (send, recv, len) convention and extended with a FIFO tie-break for
// equal-priority values. less must be non-nil.
func CreatePriority[T any](ctx context.Context, less func(a, b T) bool) (chan<- T, <-chan T, func() int) {
	input := make(chan T)
	output := make(chan T)

	queue := &priorityHeap[T]{less: less}

	// pending is updated only by the pump goroutine and read by the returned
	// length function, so it must be accessed atomically.
	var (
		pending atomic.Int64
		seq     uint64
	)

	go func() {
		defer close(output)

		inputClosed := false

		for {
			// outChan is nil (disabled in the select) unless the heap has an
			// item ready to deliver; top holds that highest-priority item.
			var (
				outChan chan<- T
				top     T
			)

			if queue.Len() > 0 {
				top = queue.items[0].value
				outChan = output
			}

			// inChan is nil once the input is closed, disabling the receive
			// case so the loop drains the heap and then exits.
			inChan := input
			if inputClosed {
				inChan = nil
			}

			if inputClosed && queue.Len() == 0 {
				return
			}

			select {
			case <-ctx.Done():
				return
			case value, ok := <-inChan:
				if !ok {
					inputClosed = true

					continue
				}

				seq++

				heap.Push(queue, priorityEntry[T]{value: value, seq: seq})
				pending.Store(int64(queue.Len()))
			case outChan <- top:
				heap.Pop(queue)
				pending.Store(int64(queue.Len()))
			}
		}
	}()

	count := func() int {
		return int(pending.Load())
	}

	return input, output, count
}

// priorityEntry wraps a buffered value with a monotonic sequence number so that
// equal-priority values can be ordered FIFO by submission.
type priorityEntry[T any] struct {
	value T
	seq   uint64
}

// priorityHeap implements heap.Interface over priorityEntry values. It is owned
// exclusively by the pump goroutine in CreatePriority, so its methods take no
// locks.
type priorityHeap[T any] struct {
	items []priorityEntry[T]
	less  func(a, b T) bool
}

func (h *priorityHeap[T]) Len() int {
	return len(h.items)
}

func (h *priorityHeap[T]) Less(i, j int) bool {
	a, b := h.items[i], h.items[j]

	switch {
	case h.less(a.value, b.value):
		return true
	case h.less(b.value, a.value):
		return false
	default:
		// Equal priority: preserve submission order.
		return a.seq < b.seq
	}
}

func (h *priorityHeap[T]) Swap(i, j int) {
	h.items[i], h.items[j] = h.items[j], h.items[i]
}

func (h *priorityHeap[T]) Push(x any) {
	entry, _ := x.(priorityEntry[T])
	h.items = append(h.items, entry)
}

func (h *priorityHeap[T]) Pop() any {
	old := h.items
	n := len(old)
	entry := old[n-1]
	old[n-1] = priorityEntry[T]{} // release the value for GC
	h.items = old[:n-1]

	return entry
}
