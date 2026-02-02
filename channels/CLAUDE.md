# Package: channels

Channel creation and safe channel operations.

## Usage

```go
// Create flexible channels
send, recv, lenFunc := channels.Create[int](10)  // buffered
send, recv, lenFunc := channels.Create[int](0)   // unbuffered
send, recv, lenFunc := channels.Create[int](-1)  // infinite buffer

// Safe channel closing
channels.CloseChannelIgnorePanic(ch)  // Won't panic if already closed

// Safe sending with panic recovery
err := channels.SendCatchPanic(ch, value)
err := channels.SendContextCatchPanic(ctx, ch, value)
```

## Common Patterns

- `Create()` with size < 0 creates infinite buffering channel
- Use `CloseChannelIgnorePanic()` in cleanup code
- `SendCatchPanic()` for sending to potentially closed channels

## Gotchas

- Negative size creates infinite buffer (via InfiniteChan)
- Send functions recover from panics and return errors
