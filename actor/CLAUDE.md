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
