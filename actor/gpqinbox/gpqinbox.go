// Package gpqinbox provides an opt-in, priority-ordered inbox for actors backed
// by github.com/JustinTimperio/gpq. It is a DISTINCT code path: it does not
// change actor.Run (FIFO) or actor.RunPriority (single heap). Reach for it only
// when you specifically want gpq's bucketed priority model with time-based
// escalation.
//
// # Why a separate package
//
// gpq transitively depends on badgerDB (an embedded LSM database) and its
// dependency tree. Keeping the bridge here confines that weight to callers who
// opt in — importing the core actor package does not pull badger into a binary
// that only uses Run or RunPriority.
//
// # Model
//
// gpq distributes messages across a fixed number of priority buckets. Bucket 0
// is served first; within a bucket, delivery is FIFO. A message's Weight is
// mapped to a bucket by Config.WeightToBucket (higher Weight => higher-priority,
// lower-numbered bucket).
//
// # Escalation and its limitation
//
// gpq's "escalation" ages a waiting item toward the front OF ITS OWN BUCKET on
// each Prioritize tick. It never promotes an item into a higher-priority
// bucket. Therefore escalation mitigates unfairness WITHIN a priority level but
// does not prevent cross-tier starvation: a bucket that continuously receives
// traffic can still keep lower buckets from ever being served, because Dequeue
// always drains the lowest-numbered non-empty bucket first. If cross-tier
// starvation is the concern, an aging comparator (effective priority as a
// function of wait time) is the better tool.
//
// # Timeouts are intentionally not exposed
//
// gpq supports per-item timeouts, but a timed-out item is dropped silently
// inside Prioritize with no callback. For actors that would strand any
// Request/RequestCtx caller waiting on the message's response channel, so this
// bridge does not surface timeouts.
package gpqinbox

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/JustinTimperio/gpq"
	"github.com/JustinTimperio/gpq/schema"

	"github.com/amp-labs/amp-common/actor"
)

const (
	defaultBuckets         = 2
	defaultPrioritizeEvery = 250 * time.Millisecond
)

// Config configures a gpq-backed actor inbox.
type Config struct {
	// Buckets is the number of gpq priority levels. Bucket 0 is highest
	// priority. Must be >= 1; defaults to 2 when <= 0.
	Buckets int

	// MaxSize bounds the number of buffered (queued but not yet delivered)
	// messages. When <= 0 the inbox is unbounded and submits never block on a
	// full mailbox. When > 0, once that many messages are buffered, submits
	// block until the actor drains one, applying backpressure (SendCtx /
	// RequestCtx honor their context while blocked).
	MaxSize int

	// Escalate enables gpq's intra-bucket escalation (see package docs). When
	// false the inbox is a plain bucketed priority queue with FIFO order within
	// each bucket and no Prioritize ticker runs.
	Escalate bool

	// EscalationRate is how long an item must wait before it becomes eligible to
	// move one slot toward the front of its bucket. Only used when Escalate is
	// true; defaults to PrioritizeEvery when <= 0.
	EscalationRate time.Duration

	// PrioritizeEvery is how often the inbox calls gpq.Prioritize to apply
	// escalation. Only used when Escalate is true; defaults to 250ms when <= 0.
	// Prioritize is a stop-the-world pass over the queue, so avoid very small
	// values on large queues.
	PrioritizeEvery time.Duration

	// WeightToBucket maps a message's Weight to a gpq bucket in [0, Buckets).
	// Higher Weight should map to a lower-numbered (higher-priority) bucket. When
	// nil, DefaultWeightToBucket is used.
	WeightToBucket func(weight, buckets int) uint
}

// DefaultWeightToBucket maps Weight to a bucket in [0, buckets) as
// clamp(buckets-1-weight, 0, buckets-1). Weight 0 (the actor default) lands in
// the lowest-priority bucket and larger weights climb toward bucket 0; weights
// at or above buckets-1 map to bucket 0 and negative weights to the lowest
// bucket.
func DefaultWeightToBucket(weight, buckets int) uint {
	bucket := min(max((buckets-1)-weight, 0), buckets-1)

	return uint(bucket) //nolint:gosec // bucket is clamped to [0, buckets), so never negative
}

func (c Config) withDefaults() Config {
	if c.Buckets <= 0 {
		c.Buckets = defaultBuckets
	}

	if c.PrioritizeEvery <= 0 {
		c.PrioritizeEvery = defaultPrioritizeEvery
	}

	if c.EscalationRate <= 0 {
		c.EscalationRate = c.PrioritizeEvery
	}

	if c.WeightToBucket == nil {
		c.WeightToBucket = DefaultWeightToBucket
	}

	return c
}

// Run starts the actor with a gpq-backed priority inbox and returns a reference
// for sending messages. Messages submitted with a higher Weight are placed in a
// higher-priority bucket (see Config.WeightToBucket) and processed first; within
// a bucket, delivery is FIFO. Use actor.SendWithWeight / RequestWithWeight (or
// set Message.Weight for Publish) to choose the weight.
//
// Lifecycle matches Run/RunPriority: Stop drains and processes buffered messages
// in priority order before the run loop exits, while canceling ctx stops
// immediately and discards buffered messages. The same ctx must be used to stop
// the inbox, so pass the actor's context here.
//
// Run returns an error only if the underlying gpq queue cannot be constructed.
func Run[Request, Response any](
	ctx context.Context,
	a *actor.Actor[Request, Response],
	name string,
	cfg Config,
) (*actor.Ref[Request, Response], error) {
	w, r, count, err := newInbox(ctx, cfg, func(m actor.Message[Request, Response]) int {
		return m.Weight
	})
	if err != nil {
		return nil, err
	}

	return a.RunWithInbox(ctx, name, w, r, count, true), nil
}

// newInbox builds the (send, recv, count) triple backed by a gpq queue. weightOf
// extracts the priority weight from a queued value. The returned channels and
// count function have the same contract as channels.CreatePriority: closing the
// send channel drains the queue in priority order and then closes the receive
// channel; canceling ctx closes the receive channel and discards buffered
// values.
func newInbox[T any](
	ctx context.Context,
	cfg Config,
	weightOf func(T) int,
) (chan<- T, <-chan T, func() int, error) {
	cfg = cfg.withDefaults()

	// DiskCacheEnabled is deliberately left false: the queue is in-memory only.
	// Enabling it would gob-encode each value, which fails for actor messages
	// that carry a response channel.
	_, queue, err := gpq.NewGPQ[T](schema.GPQOptions{
		MaxPriority: uint(cfg.Buckets), //nolint:gosec // Buckets is defaulted to a small positive value
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("gpqinbox: create gpq queue: %w", err)
	}

	input := make(chan T)
	output := make(chan T)

	enqueueOpts := schema.EnqueueOptions{
		ShouldEscalate: cfg.Escalate,
		EscalationRate: cfg.EscalationRate,
	}

	// pending is written only by the pump goroutine and read by the returned
	// count function, so it is accessed atomically.
	var pending atomic.Int64

	go func() {
		defer close(output)
		defer queue.Close()

		// The escalation ticker only runs when escalation is enabled; otherwise
		// prioritizeC stays nil and its select case is never taken.
		var prioritizeC <-chan time.Time

		if cfg.Escalate {
			ticker := time.NewTicker(cfg.PrioritizeEvery)
			defer ticker.Stop()

			prioritizeC = ticker.C
		}

		inputClosed := false

		// gpq's Dequeue is a non-blocking poll that removes the item, and gpq has
		// no peek. To offer the next item on a channel we must hold it out of the
		// queue: haveItem/top is that single in-flight item. Consequence: an item
		// pulled into top is committed to being delivered next even if a
		// higher-priority item arrives just after — a one-item priority inversion,
		// harmless for starvation behavior.
		var (
			haveItem bool
			top      T
		)

		for {
			if !haveItem {
				item, derr := queue.Dequeue()
				if derr == nil {
					top = item.Data
					haveItem = true
				}
			}

			buffered := int(queue.ItemsInQueue()) //nolint:gosec // queue length fits an int in any realistic inbox
			if haveItem {
				buffered++
			}

			pending.Store(int64(buffered))

			// Exit once the producer side is closed and everything has drained.
			if inputClosed && !haveItem {
				return
			}

			// outChan is nil (disabled) until we hold an item to deliver.
			var outChan chan<- T
			if haveItem {
				outChan = output
			}

			// inChan is nil (disabled) once input is closed — so the loop drains
			// and exits — or the bounded buffer is full — so producers block on
			// send, applying backpressure.
			inChan := input
			if inputClosed || (cfg.MaxSize > 0 && buffered >= cfg.MaxSize) {
				inChan = nil
			}

			select {
			case <-ctx.Done():
				return
			case value, ok := <-inChan:
				if !ok {
					inputClosed = true

					continue
				}

				bucket := cfg.WeightToBucket(weightOf(value), cfg.Buckets)

				eerr := queue.Enqueue(schema.NewItem(bucket, value, enqueueOpts))
				if eerr != nil {
					// Buckets are pre-created and bucket is clamped to [0,Buckets),
					// so Enqueue cannot fail in practice. Drop and log defensively
					// rather than blocking the pump.
					slog.Error("gpqinbox: enqueue failed", "error", eerr)
				}
			case outChan <- top:
				haveItem = false
			case <-prioritizeC:
				_, _, _ = queue.Prioritize()
			}
		}
	}()

	count := func() int {
		return int(pending.Load())
	}

	return input, output, count, nil
}
