// Package channels provides utilities for working with Go channels,
// including channel creation with flexible sizing and safe channel closing.
package channels

import (
	"context"
	"errors"
	"runtime/debug"

	"github.com/amp-labs/amp-common/utils"
)

// Create creates a channel with the specified size and returns a send-only channel,
// a receive-only channel, and a function to get the current queue length.
//
// The size parameter determines the channel type:
//   - size < 0: creates an infinite buffering channel (via InfiniteChan)
//   - size == 0: creates an unbuffered channel
//   - size > 0: creates a buffered channel with the specified capacity
//
// Returns:
//   - chan<- T: send-only channel for writing values
//   - <-chan T: receive-only channel for reading values
//   - func() int: function that returns the current number of items in the channel
func Create[T any](size int) (chan<- T, <-chan T, func() int) {
	switch {
	case size < 0:
		return InfiniteChan[T]()
	case size == 0:
		c := make(chan T)

		return c, c, func() int {
			return len(c)
		}
	default:
		c := make(chan T, size)

		return c, c, func() int {
			return len(c)
		}
	}
}

// CloseChannelIgnorePanic closes a channel like normal.
// However, if the channel has already been closed,
// it will suppress the resulting panic.
func CloseChannelIgnorePanic[T any](ch chan<- T) {
	if ch == nil {
		return
	}

	defer func() {
		// Recover from panic if the channel is already closed
		_ = recover()
	}()

	close(ch)
}

// SendCatchPanic sends a message to a channel and recovers from any panic that occurs.
// This is useful when sending to a channel that might be closed, avoiding a panic.
//
// Returns nil if the send succeeds or if ch is nil.
// Returns an error if a panic occurs during the send (e.g., sending on a closed channel).
func SendCatchPanic[T any](ch chan<- T, msg T) (err error) {
	if ch == nil {
		return nil
	}

	defer func() {
		if e := recover(); e != nil {
			err = utils.GetPanicRecoveryError(e, debug.Stack())
		}
	}()

	ch <- msg

	return nil
}

// SendContextCatchPanic sends a message to a channel with context support and recovers from any panic.
// It respects context cancellation and will return ctx.Err() if the context is canceled before the send.
//
// If ctx is nil, it falls back to SendCatchPanic behavior.
// Returns nil if the send succeeds or if channel is nil.
// Returns ctx.Err() if the context is canceled.
// Returns an error if a panic occurs during the send (e.g., sending on a closed channel).
func SendContextCatchPanic[T any](ctx context.Context, channel chan<- T, msg T) (err error) {
	if ctx == nil {
		return SendCatchPanic(channel, msg)
	}

	if channel == nil {
		return nil
	}

	defer func() {
		if e := recover(); e != nil {
			err2 := utils.GetPanicRecoveryError(e, debug.Stack())

			err = errors.Join(err2, err)
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case channel <- msg:
	}

	return nil
}

// InfiniteChan creates a channel with infinite buffering.
// It returns a send-only channel and a receive-only channel.
// The send-only channel can be used to send values without blocking.
// The receive-only channel can be used to receive values in the order they were sent.
//
// Note: Use with caution as it can lead to high memory usage if the sender outpaces
// the receiver. It's recommended to monitor the size of the internal queue if used in
// a long-running process.
func InfiniteChan[A any]() (chan<- A, <-chan A, func() int) {
	// Create input and output channels
	inputCh := make(chan A)
	outputCh := make(chan A)

	// Internal queue to store values between receives and sends
	var inputQueue []A

	// Start a goroutine to manage the buffering between input and output
	go func() {
		// outCh returns the output channel only when there's data to send
		// Returns nil when queue is empty to disable this select case
		outCh := func() chan A {
			if len(inputQueue) == 0 {
				return nil
			}

			return outputCh
		}

		// curVal returns the first value in the queue, or zero value if empty
		curVal := func() A {
			if len(inputQueue) == 0 {
				var zero A

				return zero
			}

			return inputQueue[0]
		}

		// Continue until queue is drained and input channel is closed
		for len(inputQueue) > 0 || inputCh != nil {
			select {
			// Receive from input channel and add to queue
			case v, ok := <-inputCh:
				if !ok {
					// Input closed, set to nil to disable this case
					inputCh = nil
				} else {
					// Append received value to queue
					inputQueue = append(inputQueue, v)
				}
			// Send first queued value to output channel
			case outCh() <- curVal():
				// Remove sent value from queue
				inputQueue = inputQueue[1:]
			}
		}

		// Close output channel when all values are sent
		close(outputCh)
	}()

	return inputCh, outputCh, func() int {
		return len(inputCh)
	}
}
