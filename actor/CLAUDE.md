# Package: actor

Actor model implementation with message passing, sequential processing, and panic recovery.

## Usage

```go
import "github.com/amp-labs/amp-common/actor"

// Define processor
type MyProcessor struct{}
func (p *MyProcessor) Process(ctx context.Context, msg string) (int, error) {
    return len(msg), nil
}

// Create and run actor
myActor := actor.New(func(ref *actor.Ref[string, int]) actor.Processor[string, int] {
    return &MyProcessor{}
})
ref := myActor.Run(ctx, "my-actor", 100)  // 100 = mailbox size

// Send messages
ref.Send("hello")  // Fire and forget
response, err := ref.Request(ctx, "hello")  // Wait for response
ref.Publish(ctx, "broadcast")  // Non-blocking send
```

### Priority inbox

```go
// RunPriority gives the actor a heap-backed inbox instead of a FIFO channel.
ref := myActor.RunPriority(ctx, "my-actor", 0)    // 0 / negative = unbounded
ref := myActor.RunPriority(ctx, "my-actor", 1000) // bounded: blocks at 1000 queued

// Higher Weight is processed first; equal Weight keeps FIFO (submission) order.
ref.SendWithWeight("urgent", 10)
ref.SendWithWeight("normal", 1)
ref.RequestWithWeight("urgent", 10)
ref.RequestCtxWithWeight(ctx, "urgent", 10)
ref.Publish(actor.Message[string, int]{Request: "urgent", Weight: 10})
```

- Weight only matters when messages accumulate (consumer slower than producers).
- `maxSize <= 0`: inbox is **unbounded** (heap, like `channels.InfiniteChan`) — submits never block on a full mailbox.
- `maxSize > 0`: inbox is **bounded** — once that many messages are queued, submits block until the actor drains one, applying backpressure. `SendCtx`/`RequestCtx` honor their context while blocked.
- `Stop()` drains queued messages in priority order; canceling `ctx` discards them.
- `Weight` is ignored by `Run`-started (FIFO) actors.

### gpq priority inbox (opt-in, subpackage)

A third, distinct inbox lives in `actor/gpqinbox`, backed by
[`JustinTimperio/gpq`](https://github.com/JustinTimperio/gpq). It is a **separate
package on purpose**: gpq transitively pulls in badgerDB, so keeping it out of the
core `actor` package means only callers who opt in pay that dependency weight.
`Run` and `RunPriority` are unchanged.

```go
import "github.com/amp-labs/amp-common/actor/gpqinbox"

ref, err := gpqinbox.Run(ctx, myActor, "my-actor", gpqinbox.Config{
    Buckets:         4,                // priority levels; bucket 0 served first
    MaxSize:         0,                // 0/neg = unbounded, >0 = bounded backpressure
    Escalate:        true,             // enable intra-bucket aging
    EscalationRate:  time.Second,      // wait before an item ages one slot
    PrioritizeEvery: 250 * time.Millisecond, // how often Prioritize runs
})
// Send with weight exactly as with RunPriority; Weight maps to a bucket via
// Config.WeightToBucket (higher Weight => higher-priority, lower-numbered bucket).
ref.SendWithWeight("urgent", 3)
```

- The `actor` package exposes `RunWithInbox` (the shared engine behind `Run`/`RunPriority`)
  so alternative inboxes can plug in without importing their deps into core `actor`.
- **Escalation is intra-bucket only.** gpq ages a waiting item toward the front of
  *its own bucket*; it never promotes an item to a higher bucket, and `Dequeue`
  always drains the lowest-numbered non-empty bucket first. So a continuously busy
  high-priority bucket **can still starve lower buckets** — escalation fixes
  fairness *within* a tier, not cross-tier starvation. Use an aging comparator if
  cross-tier starvation is the concern.
- gpq **timeouts are intentionally not exposed** — a timed-out item is dropped
  silently, which would strand `Request`/`RequestCtx` callers waiting on a response.
- The pump holds one item out of the queue at a time (gpq has no peek), so the
  first buffered message is delivered before priority ordering kicks in (a
  one-item inversion, harmless for starvation behavior).
- Lifecycle matches `RunPriority`: `Stop()` drains in priority order; canceling
  `ctx` discards buffered messages.

## Common Patterns

- Actors process messages sequentially (one at a time)
- Mailbox: buffered channel for incoming messages
- `Send()` - Fire and forget
- `Request()` / `RequestCtx()` - Wait for response
- `Publish()` / `PublishCtx()` - Non-blocking send
- Automatic panic recovery with error notification
- Prometheus metrics for monitoring

## Gotchas

- Messages processed sequentially (no parallelism within actor)
- Dead actors return ErrDeadActor
- Panics converted to ErrActorPanic
- Mailbox size controls backpressure

## Related

- `channels` - Channel utilities
