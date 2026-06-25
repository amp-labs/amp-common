# Package: channels

Channel creation and safe channel operations.

## Usage

```go
// Create flexible channels
send, recv, lenFunc := channels.Create[int](10)  // buffered
send, recv, lenFunc := channels.Create[int](0)   // unbuffered
send, recv, lenFunc := channels.Create[int](-1)  // infinite buffer

// Priority-ordered, unbounded pump (same (send, recv, len) shape).
// less(a, b) reports whether a is delivered before b; equal priority is FIFO.
send, recv, lenFunc := channels.CreatePriority[int](ctx, func(a, b int) bool {
    return a > b // higher value delivered first
})

// Safe channel closing
channels.CloseChannelIgnorePanic(ch)  // Won't panic if already closed

// Safe sending with panic recovery
err := channels.SendCatchPanic(ch, value)
err := channels.SendContextCatchPanic(ctx, ch, value)
```

## Common Patterns

- `Create()` with size < 0 creates infinite buffering channel
- `CreatePriority()` reorders buffered items by a `less` func (heap-backed, unbounded); priority only shows when items accumulate
- Use `CloseChannelIgnorePanic()` in cleanup code
- `SendCatchPanic()` for sending to potentially closed channels

## Gotchas

- Negative size creates infinite buffer (via InfiniteChan)
- Send functions recover from panics and return errors
