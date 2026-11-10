# Package: ratelimit

Rate limiter contract: reserve tokens before a call, wait, then report how the
call went. The package enforces no limit itself.

## Usage

```go
// A rate-limited call
res, err := limiter.Reserve(ctx, 1)
if err != nil {
    return err // the limiter itself failed, or the input was bad
}

if !res.OK() {
    return errShed // denied: don't make the call, and don't call Done
}

if _, delay := res.Delay(); delay > 0 {
    select {
    case <-time.After(delay):
    case <-ctx.Done():
        res.Done(ratelimit.Canceled) // never made the call: return the tokens
        return ctx.Err()
    }
}

rsp, err := client.Do(req)
switch {
case err != nil:
    res.Done(ratelimit.Failed)
    return err
case rsp.StatusCode == http.StatusTooManyRequests:
    res.Done(ratelimit.Throttled, ratelimit.ThrottleParams{Reason: "429"})
default:
    res.Done(ratelimit.OK)
}

// Carrying a Limiter in an ordinary context
ctx = ratelimit.WithRateLimiter(ctx, limiter)

// ... or in a mutable one, such as Fiber's
ratelimit.SetRateLimiter(limiter, c.Context().SetUserValue)

// Anywhere downstream
if lim, ok := ratelimit.GetRateLimiter(ctx); ok {
    // lim is non-nil and safe to call
}

// Building a Limiter from functions
allowAll := ratelimit.NewLimiterBuilder().Build() // every call allowed, no delay

reserve := func(context.Context, int) (ratelimit.Reservation, error) {
    return ratelimit.NewReservationBuilder().
        SetDelay(func() (string, time.Duration) {
            return "test", time.Second
        }).
        SetDone(func(s ratelimit.Status, _ ...ratelimit.ThrottleParams) {
            got = append(got, s)
        }).
        Build(), nil
}

scripted := ratelimit.NewLimiterBuilder().SetReserve(reserve).Build()
```

## Common Patterns

- `Limiter.Reserve(ctx, tokens)` returns a `Reservation` immediately; the
  caller does any waiting
- `Reservation.OK()` false means the limiter denied the call (load shedding or
  short-circuiting). A denial is not an error: `Reserve` errors only when the
  limiter itself fails or the input is bad
- `Reservation.Delay()` returns a reason (may be blank) and how long to wait
- `Reservation.Done(status, ...ThrottleParams)` reports the outcome:
  - `OK`: the call succeeded
  - `Throttled`: 429 or equivalent; pass `ThrottleParams` (`Reason`, optional `Delay`)
  - `Failed`: the call failed for some other reason
  - `Canceled`: the call was never made; the limiter returns the tokens
  - `None`: the zero value, meaning nobody set a status
- `LimiterBuilder` / `ReservationBuilder`: zero values are ready to use
  (`NewLimiterBuilder` / `NewReservationBuilder` return them). Unset functions
  default to `OK` true, no delay, and a no-op `Done`. `Build` copies the
  functions, so a builder can be reused
- `NewReservationBuilder`'s doc says connector code should only call it in
  tests; the server side decides reservation details
- `HasRateLimiter(ctx)` is shorthand for the `ok` of `GetRateLimiter(ctx)`

## Gotchas

- **Don't call `Done` when `OK()` is false.** When `OK()` is true but the call
  is never made (canceled during the wait, for example), call `Done(Canceled)`,
  or the tokens aren't returned
- **Typed nils count as absent.** `GetRateLimiter` returns `nil, false` for a
  nil `*T` stored as a `Limiter`, so a `true` result is always safe to call.
  Storing a nil or typed-nil `Limiter` hides one stored further up the context
- **Fiber:** after `SetRateLimiter(lim, c.Context().SetUserValue)` (or a closure
  calling `c.Locals`), read it back with `GetRateLimiter(c.Context())`.
  `c.UserContext()` is a separate context and doesn't see it. Verified with
  fiber v2.52.15 and fasthttp v1.74.0, where `RequestCtx.Value` returns user
  values for any key type
- `SetRateLimiter` silently does nothing if `f` is nil
- `WithRateLimiter(nil, l)` derives from `context.Background()`;
  `GetRateLimiter(nil)` returns false

## Testing

`context_test.go` is a table-driven test of the context helpers (stored,
missing, nil context, nil and typed-nil limiters, hiding an outer limiter).
It defines `fakeLimiter`; reuse it rather than adding another. For a scripted
`Limiter` or `Reservation`, use the builders as in Usage.

## Related

- `contexts` - `WithValue` / `GetValue` back the context helpers
- `utils.IsNilish` - the typed-nil check in `GetRateLimiter`
- `Status` and `Reservation` match the server repo's `shared/limiter`
  definitions exactly; `ThrottleParams` here has no provider-specific fields
