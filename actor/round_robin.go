package actor

import (
	"errors"
	"sync/atomic"
)

var ErrNoActors = errors.New("no actors provided")

// roundRobinCounter returns a function that generates a round-robin counter
// for a given number of actors. It uses atomic operations to ensure
// thread-safe access to the counter variable. The counter will wrap around
// to zero when it reaches the maximum number of actors or overflows.
func roundRobinCounter(n int) func() int {
	var counter int32

	return func() int {
		for {
			current := atomic.LoadInt32(&counter)
			next := current + 1

			// Reset to zero if we're at an overflow boundary
			if next < 0 || next >= int32(n) { //nolint:gosec
				next = 0
			}

			// Ensure thread-safety by using CompareAndSwap.
			// If we failed to update the counter, retry by looping.
			if atomic.CompareAndSwapInt32(&counter, current, next) {
				return int(next)
			}
		}
	}
}

// roundRobinProcessor creates a round-robin processor that distributes incoming messages to a list of actors.
func roundRobinProcessor[Request, Response any](
	actors []*Ref[Request, Response],
) func(ref *Ref[Request, Response]) Processor[Request, Response] {
	return func(ref *Ref[Request, Response]) Processor[Request, Response] {
		counter := roundRobinCounter(len(actors))

		return NewProcessor(func(msg Message[Request, Response]) {
			// Each time a new message comes in, we advance the counter
			actor := actors[counter()]
			actor.Publish(msg)
		})
	}
}

// RoundRobin creates a new actor that uses a round-robin strategy to distribute
// incoming messages to a list of actors. It takes a variable number of actor references
// and returns a new actor that can be used to process requests. If no actors are provided,
// it returns an error.
func RoundRobin[Request, Response any](
	actors ...*Ref[Request, Response],
) (*Actor[Request, Response], error) {
	if len(actors) == 0 {
		return nil, ErrNoActors
	}

	return New[Request, Response](roundRobinProcessor(actors)), nil
}
