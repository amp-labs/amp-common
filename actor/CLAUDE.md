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
ref := myActor.RunPriority(ctx, "my-actor")  // unbounded, no depth arg

// Higher Weight is processed first; equal Weight keeps FIFO (submission) order.
ref.SendWithWeight("urgent", 10)
ref.SendWithWeight("normal", 1)
ref.RequestWithWeight("urgent", 10)
ref.RequestCtxWithWeight(ctx, "urgent", 10)
ref.Publish(actor.Message[string, int]{Request: "urgent", Weight: 10})
```

- Weight only matters when messages accumulate (consumer slower than producers).
- The priority inbox is **unbounded** (heap, like `channels.InfiniteChan`) — submits never block on a full mailbox, so there is no backpressure. Use `Run` when you need bounded FIFO delivery.
- `Stop()` drains queued messages in priority order; canceling `ctx` discards them.
- `Weight` is ignored by `Run`-started (FIFO) actors.

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
