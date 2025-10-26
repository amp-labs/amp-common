# future

[![Go Reference](https://pkg.go.dev/badge/github.com/amp-labs/amp-common/future.svg)](https://pkg.go.dev/github.com/amp-labs/amp-common/future)

A Go library for type-safe asynchronous programming with Futures and Promises, featuring automatic panic recovery, context support, and functional composition.

## Table of Contents

- [Purpose](#purpose)
- [Core Concepts](#core-concepts)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Basic Usage](#basic-usage)
- [Advanced Usage](#advanced-usage)
- [Callbacks](#callbacks)
- [Transformations](#transformations)
- [Combining Futures](#combining-futures)
- [Cancellation](#cancellation)
- [Best Practices](#best-practices)
- [Error Handling](#error-handling)
- [Troubleshooting](#troubleshooting)
- [API Reference](#api-reference)
- [Examples](#examples)
- [Thread Safety](#thread-safety)
- [Performance Considerations](#performance-considerations)

## Purpose

The `future` package provides a Future/Promise implementation for asynchronous programming in Go. It solves common challenges in async code:

- **Type-Safe Async**: Generic futures with compile-time type checking
- **Panic Recovery**: Automatically catches panics and converts them to errors with stack traces
- **Context Support**: Full context integration for cancellation and timeouts
- **Immutability**: Promises can only be completed once (first completion wins); results are immutable
- **Functional Composition**: Map, FlatMap, and Combine for building complex async workflows
- **Callback System**: OnSuccess, OnError, OnResult for reactive programming

## Core Concepts

### Future (Read-Only Side)

A `Future[T]` represents the eventual result of an asynchronous computation. It's the "consumer" side:

```go
type Future[T any] struct {
    // Provides read-only access to async results
}
```

**Key Features:**

- Read-only access to results (cannot be completed)
- Thread-safe concurrent access
- Memoized results (computed once, cached, reused for all subsequent Await() calls)
- Multiple waiters supported

### Promise (Write-Only Side)

A `Promise[T]` is used to complete a Future. It's the "producer" side:

```go
type Promise[T any] struct {
    // Provides write-only access for completing the future
}
```

**Key Features:**

- Write-once semantics (first completion wins)
- Thread-safe concurrent completion
- Automatically unblocks all waiters

### Separation of Concerns

The Future/Promise split prevents consumers from accidentally completing futures:

```go
future, promise := future.New[int]()
// Pass future to consumers (they can only read)
// Keep promise for producers (they can only write)
```

### Panic Recovery

All async operations automatically recover from panics:

```go
fut := future.Go(func() (int, error) {
    panic("something went wrong") // Recovered automatically
})
result, err := fut.Await()
// err contains: "recovered from panic: something went wrong" + stack trace
```

### Context Cancellation

Context-aware operations support cancellation and timeouts:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

fut := future.GoContext(ctx, fetchData)
result, err := fut.AwaitContext(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // Timeout occurred
}
```

## Installation

```bash
go get github.com/amp-labs/amp-common/future
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/amp-labs/amp-common/future"
)

func main() {
    // Create an async computation
    fut := future.Go(func() (string, error) {
        time.Sleep(100 * time.Millisecond)
        return "Hello, Future!", nil
    })

    // Wait for the result
    result, err := fut.Await()
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Result: %s\n", result)
}
```

## Basic Usage

### Creating Futures

**Using Go() - Most Common**

```go
// Simplest way - launches goroutine automatically
fut := future.Go(func() (User, error) {
    return db.FetchUser(userID)
})

result, err := fut.Await()
```

**Using GoContext() - With Cancellation**

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

fut := future.GoContext(ctx, func(ctx context.Context) (Data, error) {
    return fetchDataWithContext(ctx)
})

result, err := fut.AwaitContext(ctx)
```

**Using New() - Manual Control**

```go
// Full control over execution
fut, promise := future.New[int]()

go func() {
    result := expensiveComputation()
    promise.Success(result)
}()

value, err := fut.Await()
```

### Awaiting Results

**Blocking Await**

```go
fut := future.Go(fetchData)
result, err := fut.Await() // Blocks until complete
```

**Context-Aware Await**

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

fut := future.Go(slowOperation)
result, err := fut.AwaitContext(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

**Multiple Awaits (Idempotent)**

```go
fut := future.Go(computation)

// All calls return the same memoized result
result1, _ := fut.Await()
result2, _ := fut.Await()
result3, _ := fut.Await()
// result1 == result2 == result3
```

### Completing Promises

**Success**

```go
_, promise := future.New[string]()
promise.Success("completed")
```

**Failure**

```go
_, promise := future.New[User]()
promise.Failure(errors.New("fetch failed"))
```

**Complete (Go-Style)**

```go
_, promise := future.New[Data]()
data, err := someFunction()
promise.Complete(data, err) // Handles both success and error
```

### Fire-and-Forget Operations

**Async() - Simplest Async Execution**

For operations where you don't need to wait for the result or handle errors explicitly, use `Async()`:

```go
// Launch background work without blocking
future.Async(func() {
    updateCache()
    sendAnalytics()
    cleanupTempFiles()
})

// Continues immediately - no need to await
log.Println("Background work started")
```

**Use cases:**

- Logging and analytics
- Cache updates
- Non-critical background tasks
- Fire-and-forget operations

**AsyncContext() - With Cancellation Support**

For background operations that should respect cancellation, use `AsyncContext()`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Launch background work that respects context
future.AsyncContext(ctx, func(ctx context.Context) {
    if err := syncDataWithContext(ctx); err != nil {
        // Error is logged automatically
        return
    }

    select {
    case <-ctx.Done():
        log.Println("Sync canceled")
        return
    default:
        log.Println("Sync completed")
    }
})

// Continues immediately
```

**Use cases:**

- Background sync operations
- Non-blocking cleanup with timeouts
- Async operations that should respect cancellation
- Fire-and-forget with graceful shutdown

**AsyncWithError() - With Error Return**

For background operations that may fail and you want errors logged, use `AsyncWithError()`:

```go
// Launch background work that can return errors
future.AsyncWithError(func() error {
    if err := updateCache(); err != nil {
        return fmt.Errorf("cache update failed: %w", err)
    }

    if err := sendAnalytics(); err != nil {
        return fmt.Errorf("analytics failed: %w", err)
    }

    return nil
})

// Continues immediately - errors are logged automatically
log.Println("Background work started")
```

**Use cases:**

- Background operations that may fail
- Non-critical tasks where you want error visibility via logs
- Cache updates or cleanup that should log failures
- Fire-and-forget operations that need error tracking

**AsyncContextWithError() - With Context and Error Return**

For background operations that should respect cancellation and may fail, use `AsyncContextWithError()`:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// Launch background work that respects context and can return errors
future.AsyncContextWithError(ctx, func(ctx context.Context) error {
    if err := syncDataWithContext(ctx); err != nil {
        return fmt.Errorf("sync failed: %w", err)
    }

    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        log.Println("Sync completed")
        return nil
    }
})

// Continues immediately - errors are logged automatically
```

**Use cases:**

- Background sync operations that may fail
- Non-blocking cleanup with timeouts and error handling
- Async operations that should respect cancellation and log errors
- Fire-and-forget with graceful shutdown and error tracking

**Key Features:**

- **Non-blocking**: Returns immediately without waiting for completion
- **No boilerplate**: No need to create Future, await, or handle results manually
- **Automatic error logging**: Panics and errors are caught and logged automatically
- **Thread-safe**: Safe to call from multiple goroutines
- **Context support**: AsyncContext respects cancellation and deadlines

**Comparison with Go()**

```go
// ❌ Overkill: Using Go() when you don't need the result
fut := future.Go(func() (struct{}, error) {
    logEvent(event)
    return struct{}{}, nil
})
// Result is never used

// ✅ Better: Use Async for fire-and-forget
future.Async(func() {
    logEvent(event)
})

// ❌ Overkill: Using Go() when you need to return errors but don't need the result
fut := future.Go(func() (struct{}, error) {
    if err := updateCache(); err != nil {
        return struct{}{}, err
    }
    return struct{}{}, nil
})
// Result and error are never checked

// ✅ Better: Use AsyncWithError for fire-and-forget with error logging
future.AsyncWithError(func() error {
    return updateCache()
})

// ❌ Overkill: Using GoContext() for background work
fut := future.GoContext(ctx, func(ctx context.Context) (struct{}, error) {
    syncData(ctx)
    return struct{}{}, nil
})
// Result is never awaited

// ✅ Better: Use AsyncContext for fire-and-forget with context
future.AsyncContext(ctx, func(ctx context.Context) {
    syncData(ctx)
})

// ✅ Or use AsyncContextWithError if you need error logging
future.AsyncContextWithError(ctx, func(ctx context.Context) error {
    return syncData(ctx)
})
```

**Important Notes:**

- Errors and panics are logged but not propagated (fire-and-forget)
- `AsyncWithError` and `AsyncContextWithError` log errors returned by the function
- If you need the result or explicit error handling, use `Go()` or `GoContext()` instead
- Operations continue running even after the function returns (truly async)
- For critical operations where you must handle errors, use `Go()` + `Await()` or callbacks

## Advanced Usage

### Functional Transformations

**Map - Transform Success Values**

```go
// Fetch user ID, then transform to User
idFuture := future.Go(getUserId)
userFuture := future.Map(idFuture, func(id int) (User, error) {
    return fetchUser(id)
})

user, err := userFuture.Await()
```

**MapContext - With Context Support**

```go
ctx := context.Background() // In production, use req.Context() or similar

idFuture := future.Go(getUserId)
userFuture := future.MapContext(ctx, idFuture,
    func(ctx context.Context, id int) (User, error) {
        return fetchUserWithContext(ctx, id)
    })

user, err := userFuture.AwaitContext(ctx)
```

**FlatMap - Chain Async Operations**

```go
// Fetch user, then fetch their posts (both async)
userFuture := future.Go(fetchUser)
postsFuture := future.FlatMap(userFuture, func(user User) *future.Future[[]Post] {
    return future.Go(func() ([]Post, error) {
        return fetchPosts(user.ID)
    })
})

posts, err := postsFuture.Await()
```

### Combining Multiple Futures

**Combine - Wait for All (Short-Circuit on Error)**

```go
// ✅ Correct: All futures must return the same type for Combine
fut1 := future.Go(fetchUser1)  // Future[User]
fut2 := future.Go(fetchUser2)  // Future[User]
fut3 := future.Go(fetchUser3)  // Future[User]

combined := future.Combine(fut1, fut2, fut3)
users, err := combined.Await()  // []User
if err != nil {
    // One of the futures failed
}

firstUser := users[0]
secondUser := users[1]
thirdUser := users[2]

// ❌ Won't compile: Mixed types
userFut := future.Go(fetchUser)   // Future[User]
postFut := future.Go(fetchPost)   // Future[Post]
combined := future.Combine(userFut, postFut)  // Compile error!

// ✅ For different types, await separately or use a struct wrapper
type UserAndPosts struct {
    User  User
    Posts []Post
}
wrapper := future.Go(func() (UserAndPosts, error) {
    user, err1 := userFut.Await()
    posts, err2 := postFut.Await()
    if err1 != nil {
        return UserAndPosts{}, err1
    }
    if err2 != nil {
        return UserAndPosts{}, err2
    }
    return UserAndPosts{User: user, Posts: posts}, nil
})
```

**CombineNoShortCircuit - Collect All Errors**

```go
futs := make([]*future.Future[Result], len(tasks))
for i, task := range tasks {
    futs[i] = future.Go(task)
}

combined := future.CombineNoShortCircuit(futs...)
results, err := combined.Await()
if err != nil {
    // err contains ALL errors joined together
    // results still contains partial data
}
```

### Custom Executors

Implement custom execution strategies for advanced use cases like rate limiting or worker pools:

```go
// Example: Rate-limited executor
// go get golang.org/x/time/rate
import "golang.org/x/time/rate"

type RateLimitedExecutor[T any] struct {
    limiter *rate.Limiter
}

func NewRateLimitedExecutor[T any](rps int) *RateLimitedExecutor[T] {
    return &RateLimitedExecutor[T]{
        limiter: rate.NewLimiter(rate.Limit(rps), rps),
    }
}

func (e *RateLimitedExecutor[T]) Go(promise *future.Promise[T],
    callback func() (T, error)) {
    go func() {
        // Wait for rate limiter before executing
        _ = e.limiter.Wait(context.Background())
        promise.Complete(callback())
    }()
}

func (e *RateLimitedExecutor[T]) GoContext(ctx context.Context,
    promise *future.Promise[T], callback func(context.Context) (T, error)) {
    go func() {
        // Respect both rate limit and context cancellation
        if err := e.limiter.Wait(ctx); err != nil {
            promise.Failure(err)
            return
        }
        promise.Complete(callback(ctx))
    }()
}

// Use the rate-limited executor
exec := NewRateLimitedExecutor[APIResponse](10) // 10 requests per second
fut := future.GoWithExecutor(exec, func() (APIResponse, error) {
    return callAPI()
})
```

### Converting to Channels

**Basic Channel Conversion**

```go
fut1 := future.Go(operation1)
fut2 := future.Go(operation2)

// ToChannel returns a buffered channel (size 1) that receives exactly one result
// The buffer ensures the goroutine won't block even if no one is reading yet
select {
case result := <-fut1.ToChannel():
    if result.Error != nil {
        log.Printf("Operation 1 failed: %v", result.Error)
    }
case result := <-fut2.ToChannel():
    if result.Error != nil {
        log.Printf("Operation 2 failed: %v", result.Error)
    }
}
```

**Context-Aware Channel Conversion**

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

fut := future.GoContext(ctx, fetchData)

select {
case result := <-fut.ToChannelContext(ctx):
    if errors.Is(result.Error, context.DeadlineExceeded) {
        log.Printf("Operation timed out")
    }
case <-ctx.Done():
    log.Printf("Context canceled externally")
}
```

### When to Use Futures vs Channels

**Use Futures when:**

- You need a single eventual value (one result, one error)
- You want functional composition (Map, FlatMap, Combine)
- You need automatic panic recovery with stack traces
- You want callback-based reactive programming
- You prefer immutable, write-once semantics

**Use Channels when:**

- Streaming multiple values over time
- Implementing producer-consumer patterns
- Using select for complex coordination
- You need fine-grained control over send/receive timing
- Integrating with existing channel-based code

**Example - Future for single value:**

```go
fut := future.Go(fetchUser)
user, err := fut.Await()
```

**Example - Channel for streaming:**

```go
ch := make(chan Update, 10)
go streamUpdates(ch)
for update := range ch {
    process(update)
}
```

### When NOT to Use Futures

**Avoid Futures for:**

- **Simple synchronous operations** - Just call the function directly
- **Very fast operations** (< 1μs) - Goroutine overhead dominates (map lookups, simple arithmetic)
- **Operations needing mid-execution cancellation** - Use context + channels for fine-grained control
- **Streaming multiple values** - Use channels instead
- **Fire-and-forget operations** - Use `future.Async()` for automatic error logging, or plain goroutines if you don't need error handling

**Examples of what NOT to do:**

```go
// ❌ Overkill: Simple computation doesn't need async
fut := future.Go(func() (int, error) {
    return x + y, nil
})
result, _ := fut.Await()

// ✅ Just compute it directly
result := x + y

// ❌ Overkill: Map lookup is too fast for goroutine overhead
fut := future.Go(func() (string, error) {
    return userCache[userID], nil
})

// ✅ Direct access
name := userCache[userID]

// ❌ Wrong tool: Need to stream multiple values
fut := future.Go(func() ([]Update, error) {
    // Collect all updates first...
})

// ✅ Use channels for streaming
ch := make(chan Update, 10)
go streamUpdates(ch)
for update := range ch {
    process(update)
}

// ❌ Unnecessary: Fire-and-forget doesn't need Future
fut := future.Go(func() (interface{}, error) {
    logToAnalytics(event)
    return nil, nil
})

// ✅ Use Async for fire-and-forget with automatic error logging
future.Async(func() {
    logToAnalytics(event)
})

// ✅ Or AsyncWithError if the operation can fail and you want errors logged
future.AsyncWithError(func() error {
    return logToAnalyticsWithError(event)
})

// ✅ Or just use a goroutine if you don't need error logging
go logToAnalytics(event)
```

## Callbacks

### OnSuccess - React to Successful Completion

```go
fut := future.Go(fetchUser)

fut.OnSuccess(func(user User) {
    log.Printf("Successfully fetched user: %s", user.Name)
    metrics.RecordSuccess()
})

// Continues execution...
```

### OnError - React to Errors

```go
fut := future.Go(fetchUser)

fut.OnError(func(err error) {
    log.Printf("Failed to fetch user: %v", err)
    metrics.RecordError()
    alerting.NotifyOnCall()
})
```

### OnResult - Handle Both Cases

```go
fut := future.Go(fetchUser)

fut.OnResult(func(result try.Try[User]) {
    if result.Error != nil {
        log.Printf("Failed: %v", result.Error)
    } else {
        log.Printf("Success: %s", result.Value.Name)
    }
})
```

### Context-Aware Callbacks

```go
ctx := context.Background() // In production, use req.Context() or similar

fut := future.GoContext(ctx, fetchUser)

fut.OnSuccessContext(ctx, func(ctx context.Context, user User) {
    // Callback receives context for DB calls, HTTP requests, etc.
    if err := db.SaveUserContext(ctx, user); err != nil {
        log.ErrorContext(ctx, "Failed to save user", "error", err)
    }
})

fut.OnErrorContext(ctx, func(ctx context.Context, err error) {
    log.ErrorContext(ctx, "Fetch failed", "error", err)
    metrics.RecordError()
})
```

### Method Chaining

```go
future.Go(fetchUser).
    OnSuccess(func(user User) {
        log.Printf("Fetched: %s", user.Name)
    }).
    OnError(func(err error) {
        log.Printf("Error: %v", err)
    }).
    OnResult(func(result try.Try[User]) {
        metrics.RecordCompletion()
    })
```

### Callback Guarantees

- **Invoked exactly once** per callback registration
- **Run in separate goroutines** (non-blocking)
- **Panic-safe** (panics are recovered and do not crash or propagate)
- **No error propagation** (callbacks cannot affect the Future's result)
- **Thread-safe** (can be registered from any goroutine)
- **Immediate if already complete** (registered after completion)

**Note on panic handling:** If a callback panics, the panic is recovered and does not affect the Future's result or crash the program. The Future's value and error remain unchanged. The panic is caught by the goroutine's recover mechanism but does not propagate to other callbacks or the main Future.

## Transformations

The package provides powerful transformation functions for composing async operations:

### Map Functions

| Function | Description | Context Support |
|----------|-------------|-----------------|
| `Map` | Transform Future[A] → Future[B] | ✗ |
| `MapContext` | Transform with context | ✓ |
| `MapWithExecutor` | Transform with custom executor | ✗ |
| `MapContextWithExecutor` | Transform with both | ✓ |

### FlatMap Functions

| Function | Description | Context Support |
|----------|-------------|-----------------|
| `FlatMap` | Chain async operations (Future[A] → Future[Future[B]] → Future[B]) | ✗ |
| `FlatMapContext` | Chain with context | ✓ |
| `FlatMapWithExecutor` | Chain with custom executor | ✗ |
| `FlatMapContextWithExecutor` | Chain with both | ✓ |

### Combine Functions

| Function | Description | Short-Circuit | Context |
|----------|-------------|---------------|---------|
| `Combine` | Wait for all futures | ✓ (on error) | ✗ |
| `CombineContext` | Wait for all with context | ✓ (on error) | ✓ |
| `CombineNoShortCircuit` | Wait for all, collect errors | ✗ | ✗ |
| `CombineContextNoShortCircuit` | Wait for all, collect errors, with context | ✗ (except ctx cancel) | ✓ |

**Note:** All transformation functions:

- Automatically propagate errors
- Support panic recovery
- Are type-safe with generics
- Can be chained together

## Combining Futures

### Parallel Execution Pattern

```go
// Launch all futures first (they run concurrently)
// Note: All futures must return the same type for Combine
fut1 := future.Go(fetchData1)  // Future[Data]
fut2 := future.Go(fetchData2)  // Future[Data]
fut3 := future.Go(fetchData3)  // Future[Data]

// Then combine and wait
combined := future.Combine(fut1, fut2, fut3)
results, err := combined.Await()  // []Data

// All three operations ran in parallel
// Total time ≈ max(time1, time2, time3), not sum
```

### Error Handling Strategies

**Fail-Fast (Default)**

```go
combined := future.Combine(fut1, fut2, fut3)
results, err := combined.Await()
// Returns immediately on first error
// Remaining futures continue in background
```

**Collect All Errors**

```go
combined := future.CombineNoShortCircuit(fut1, fut2, fut3)
results, err := combined.Await()
// Waits for ALL futures to complete
// err contains all errors joined with errors.Join()
```

### Context-Aware Combining

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

fut1 := future.GoContext(ctx, fetchUser)
fut2 := future.GoContext(ctx, fetchPosts)

combined := future.CombineContext(ctx, fut1, fut2)
results, err := combined.AwaitContext(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // Timeout occurred while waiting
}
```

## Cancellation

### Manual Cancellation with Cancel()

The `Cancel()` method allows you to explicitly cancel a future and trigger its cancellation callback:

```go
fut, promise := future.New[Data](func() {
    // Cleanup function called when Cancel() is invoked
    log.Println("Future cancelled, cleaning up resources")
})

go func() {
    time.Sleep(10 * time.Second)
    promise.Success(data)
}()

// Cancel after 1 second
time.Sleep(1 * time.Second)
fut.Cancel() // Triggers cleanup callback
```

### Cancel() vs Context Cancellation

**Use Cancel() when:**

- You need to explicitly abort a specific future
- You want to trigger cleanup callbacks registered with `New()`
- You're managing futures without contexts

**Use Context Cancellation when:**

- You need coordinated cancellation across multiple operations
- You want timeout-based cancellation
- You're propagating cancellation through call chains

```go
// Context cancellation (preferred for most cases)
ctx, cancel := context.WithCancel(context.Background())
fut := future.GoContext(ctx, fetchData)
cancel() // Cancels the underlying operation

// Manual cancellation (for specific future control)
fut, _ := future.New[Data](cleanupFunc)
fut.Cancel() // Triggers cleanup callback
```

### Cancellation Behavior

- **Cancel() is idempotent**: Multiple calls have no additional effect
- **Callbacks still execute**: OnSuccess/OnError callbacks run with cancellation error
- **Goroutines may continue**: Cancel() doesn't forcefully stop the underlying goroutine
- **Cleanup functions run once**: Cancellation callbacks registered with `New()` execute exactly once

**Important:** Cancellation is cooperative. The underlying operation must respect context cancellation to actually stop execution.

## Best Practices

### 1. Always Use Context for User-Facing Operations

```go
// Good: respects timeouts and cancellation in HTTP handlers
ctx, cancel := context.WithTimeout(req.Context(), 10*time.Second)
defer cancel()

fut := future.GoContext(ctx, func(ctx context.Context) (Data, error) {
    return fetchDataWithContext(ctx)
})

// Bad: no timeout, could hang forever
fut := future.Go(func() (Data, error) {
    return fetchDataWithoutContext()
})
```

### 2. Launch Futures Before Combining

```go
// Good: Futures launched first, easier to debug and inspect
fut1 := future.Go(op1)
fut2 := future.Go(op2)
fut3 := future.Go(op3)
combined := future.Combine(fut1, fut2, fut3)

// Less clear: Inline creation (still concurrent, but harder to debug)
combined := future.Combine(
    future.Go(op1),
    future.Go(op2),
    future.Go(op3),
)
// Note: Both approaches run concurrently, but the first allows
// you to inspect individual futures before combining
```

### 3. Use Callbacks for Side Effects

```go
// Good: Non-blocking side effects
future.Go(fetchUser).OnSuccess(func(user User) {
    cache.Set(user.ID, user)
    metrics.RecordSuccess()
})

// Avoid: Blocking just for side effects
user, err := future.Go(fetchUser).Await()
if err == nil {
    cache.Set(user.ID, user)
}
```

### 4. Choose the Right Combination Strategy

```go
// Use Combine when you need all to succeed
combined := future.Combine(fut1, fut2, fut3)
// Fails fast on first error

// Use CombineNoShortCircuit when you want partial results
combined := future.CombineNoShortCircuit(fut1, fut2, fut3)
// Collects all errors and results
```

### 5. Handle Panics Gracefully

```go
// The package handles panics automatically
fut := future.Go(func() (int, error) {
    // Risky operation - panics are caught automatically
    return riskyOperation()
})

_, err := fut.Await()
if err != nil {
    // Panic has been recovered and converted to error
    // Stack trace included for debugging
    log.Printf("Operation failed: %v", err)
}

// Don't try to recover panics yourself - it's handled
```

### 6. Reuse Executors for Custom Behavior

```go
// Create executor once
exec := &MyRateLimitedExecutor[Data]{
    rateLimit: 10, // 10 operations per second
}

// Reuse for multiple operations
fut1 := future.GoWithExecutor(exec, op1)
fut2 := future.GoWithExecutor(exec, op2)
fut3 := future.GoWithExecutor(exec, op3)
// All operations respect the same rate limit
```

### 7. Use Type-Safe Error Creation

```go
// Good: Use NewError for pre-failed futures
func fetchUser(id string) *future.Future[User] {
    if id == "" {
        return future.NewError[User](errors.New("id cannot be empty"))
    }
    return future.Go(func() (User, error) {
        return db.GetUser(id)
    })
}

// Maintains consistent async interface
```

## Error Handling

### Error Propagation

Errors automatically flow through transformations:

```go
fut1 := future.Go(func() (int, error) {
    return 0, errors.New("source error")
})

fut2 := future.Map(fut1, func(val int) (string, error) {
    // This is never called
    return "transformed", nil
})

_, err := fut2.Await()
// err == "source error" (propagated through Map)
```

### Panic Recovery

Panics are converted to errors with full stack traces:

```go
fut := future.Go(func() (int, error) {
    panic("unexpected panic")
})

_, err := fut.Await()
// err contains:
// - "recovered from panic: unexpected panic"
// - Full stack trace
// - File and line number
```

### Context Errors

Context cancellation and timeouts are treated as errors:

```go
ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
defer cancel()

fut := future.GoContext(ctx, slowOperation)
_, err := fut.AwaitContext(ctx)
// err == context.DeadlineExceeded
```

### Multiple Errors

When combining futures, errors are joined:

```go
fut1 := future.Go(func() (int, error) {
    return 0, errors.New("error 1")
})

fut2 := future.Go(func() (int, error) {
    return 0, errors.New("error 2")
})

combined := future.CombineNoShortCircuit(fut1, fut2)
_, err := combined.Await()
// err contains both errors joined with errors.Join()
```

### Edge Cases

**Nil Future:**

```go
var fut *future.Future[int]
result, err := fut.Await() // ⚠️ Panics - always create with New() or Go()
```

**Never-Completed Promise:**

```go
fut, promise := future.New[int]()
// Never call promise.Complete(), promise.Success(), or promise.Failure()
result, err := fut.Await() // 🚨 BLOCKS FOREVER - production deadlock risk!
                           // This will leak goroutines and hang your application
```

**Critical:** Never-completed futures cause goroutine leaks and potential deadlocks. In production code, ALWAYS use `AwaitContext` with appropriate timeouts to prevent indefinite blocking.

**Best Practice:** Always use context-aware operations for user-facing code:

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

fut := future.GoContext(ctx, operation)
result, err := fut.AwaitContext(ctx) // ✅ Will timeout instead of blocking forever
```

### Error Identity

Errors preserve their identity through transformations and support `errors.Is`/`errors.As`:

```go
var ErrNotFound = errors.New("not found")

fut1 := future.Go(func() (int, error) {
    return 0, ErrNotFound
})

fut2 := future.Map(fut1, func(val int) (string, error) {
    return "transformed", nil
})

_, err := fut2.Await()
errors.Is(err, ErrNotFound) // ✓ true - error identity preserved
```

## Troubleshooting

### Issue: Goroutine Leak / Program Hangs

**Symptoms:**

- Program doesn't exit cleanly
- Number of goroutines keeps increasing
- Application appears to hang indefinitely

**Common Causes:**

- Future created but never completed (promise.Complete/Success/Failure never called)
- Await() called without context timeout
- Context canceled but operation doesn't respect cancellation

**Solutions:**

```go
// ❌ Bad: No timeout protection
fut, promise := future.New[Data]()
go someOperation(promise)
result, _ := fut.Await() // May block forever

// ✅ Good: Always use context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

fut := future.GoContext(ctx, func(ctx context.Context) (Data, error) {
    return someOperation(ctx)
})
result, err := fut.AwaitContext(ctx)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

### Issue: Panic Not Captured

**Symptoms:**

- Application crashes with panic instead of returning error
- Stack trace shows panic in callback code

**Common Cause:**

- Panic occurs in a callback (OnSuccess/OnError/OnResult), not in the Future's main operation

**Explanation:**
Futures automatically recover panics in the main operation, but callbacks run in separate goroutines and their panics are recovered but not propagated to the Future's result.

**Solution:**

```go
// Callbacks must handle their own panics if needed
fut.OnSuccess(func(user User) {
    defer func() {
        if r := recover(); r != nil {
            log.Printf("Callback panicked: %v", r)
        }
    }()

    riskyCallbackOperation(user)
})
```

### Issue: Context Canceled But Operation Continues

**Symptoms:**

- Context is canceled but underlying work still runs
- Resources not cleaned up when expected
- Operations complete even after timeout

**Common Cause:**

- Context cancellation is cooperative - the underlying operation must actively check for cancellation

**Solution:**

```go
// ❌ Bad: Doesn't check context
fut := future.GoContext(ctx, func(ctx context.Context) (Data, error) {
    // This will run to completion even if context is canceled
    return longRunningOperation()
})

// ✅ Good: Check context regularly
fut := future.GoContext(ctx, func(ctx context.Context) (Data, error) {
    select {
    case <-ctx.Done():
        return Data{}, ctx.Err()
    default:
    }

    // Use context-aware functions
    return longRunningOperationWithContext(ctx)
})
```

### Issue: "All Goroutines Are Asleep" Deadlock

**Symptoms:**

- Fatal error: all goroutines are asleep - deadlock!
- Program exits with runtime error

**Common Causes:**

- Awaiting a future that will never complete
- Circular dependencies between futures
- All futures waiting on each other

**Solution:**

```go
// ❌ Bad: Circular dependency
fut1, promise1 := future.New[int]()
fut2, promise2 := future.New[int]()

go func() {
    val, _ := fut2.Await()
    promise1.Success(val + 1)
}()

go func() {
    val, _ := fut1.Await()
    promise2.Success(val + 1)
}()

// Both futures wait on each other - deadlock!

// ✅ Good: Break the cycle
fut1 := future.Go(func() (int, error) {
    return computeValue1(), nil
})

fut2 := future.FlatMap(fut1, func(val1 int) *future.Future[int] {
    return future.Go(func() (int, error) {
        return computeValue2(val1), nil
    })
})
```

### Issue: Memory Leak with Long-Running Futures

**Symptoms:**

- Memory usage increases over time
- Futures are created but results never accessed
- Goroutines accumulate in "running" state

**Common Cause:**

- Creating futures without awaiting them (orphaned futures)
- Large result sets cached in memory

**Solution:**

```go
// ❌ Bad: Creates future but never awaits it
for _, item := range items {
    future.Go(func() (Result, error) {
        return process(item)
    })
    // Future is created but orphaned!
}

// ✅ Good: Await or use callbacks
futures := make([]*future.Future[Result], len(items))
for i, item := range items {
    futures[i] = future.Go(func() (Result, error) {
        return process(item)
    })
}
combined := future.Combine(futures...)
results, err := combined.Await()

// ✅ Alternative: Use callbacks for side effects
for _, item := range items {
    future.Go(func() (Result, error) {
        return process(item)
    }).OnSuccess(func(result Result) {
        handleResult(result)
    })
}
```

### Issue: Race Condition with Shared State

**Symptoms:**

- Inconsistent results
- Data corruption
- Random failures in concurrent tests

**Common Cause:**

- Multiple futures accessing shared state without synchronization

**Solution:**

```go
// ❌ Bad: Race condition on shared map
results := make(map[string]int)
futures := []*future.Future[int]{...}

for _, fut := range futures {
    fut.OnSuccess(func(val int) {
        results["key"] = val // Race condition!
    })
}

// ✅ Good: Use mutex or channels for synchronization
var mu sync.Mutex
results := make(map[string]int)

for _, fut := range futures {
    fut.OnSuccess(func(val int) {
        mu.Lock()
        defer mu.Unlock()
        results["key"] = val
    })
}

// ✅ Better: Collect results via Combine
combined := future.Combine(futures...)
values, err := combined.Await()
// Process values without races
```

## API Reference

### Core Functions

#### Creating Futures

```go
// Manual control
func New[T any](cancel ...func()) (*Future[T], *Promise[T])

// Automatic execution
func Go[T any](fn func() (T, error)) *Future[T]
func GoContext[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T]

// With custom executor
func GoWithExecutor[T any](exec Executor[T], fn func() (T, error)) *Future[T]
func GoContextWithExecutor[T any](ctx context.Context, exec Executor[T],
    fn func(context.Context) (T, error)) *Future[T]

// Pre-failed future
func NewError[T any](err error) *Future[T]

// Fire-and-forget (no result tracking)
func Async(fn func())
func AsyncContext(ctx context.Context, fn func(context.Context))
func AsyncWithError(fn func() error)
func AsyncContextWithError(ctx context.Context, fn func(context.Context) error)
```

#### Awaiting Results

```go
// Future methods
func (f *Future[T]) Await() (T, error)
func (f *Future[T]) AwaitContext(ctx context.Context) (T, error)
```

#### Completing Promises

```go
// Promise methods
func (p *Promise[T]) Success(value T)
func (p *Promise[T]) Failure(err error)
func (p *Promise[T]) Complete(value T, err error)
```

### Callbacks

```go
// Future methods
func (f *Future[T]) OnSuccess(callback func(T)) *Future[T]
func (f *Future[T]) OnError(callback func(error)) *Future[T]
func (f *Future[T]) OnResult(callback func(try.Try[T])) *Future[T]

// Context-aware versions
func (f *Future[T]) OnSuccessContext(ctx context.Context,
    callback func(context.Context, T)) *Future[T]
func (f *Future[T]) OnErrorContext(ctx context.Context,
    callback func(context.Context, error)) *Future[T]
func (f *Future[T]) OnResultContext(ctx context.Context,
    callback func(context.Context, try.Try[T])) *Future[T]
```

### Transformations

```go
// Map: one-to-one transformation
func Map[A, B any](fut *Future[A], fn func(A) (B, error)) *Future[B]
func MapContext[A, B any](ctx context.Context, fut *Future[A],
    fn func(context.Context, A) (B, error)) *Future[B]
func MapWithExecutor[A, B any](fut *Future[A], exec Executor[B],
    fn func(A) (B, error)) *Future[B]
func MapContextWithExecutor[A, B any](ctx context.Context, fut *Future[A],
    exec Executor[B], fn func(context.Context, A) (B, error)) *Future[B]

// FlatMap: chain async operations
func FlatMap[A, B any](fut *Future[A], fn func(A) *Future[B]) *Future[B]
func FlatMapContext[A, B any](ctx context.Context, fut *Future[A],
    fn func(A) *Future[B]) *Future[B]
func FlatMapWithExecutor[A, B any](fut *Future[A], exec Executor[B],
    fn func(A) *Future[B]) *Future[B]
func FlatMapContextWithExecutor[A, B any](ctx context.Context, fut *Future[A],
    exec Executor[B], fn func(A) *Future[B]) *Future[B]
```

### Combining Futures

```go
// Short-circuit on first error
func Combine[T any](futures ...*Future[T]) *Future[[]T]
func CombineContext[T any](ctx context.Context, futures ...*Future[T]) *Future[[]T]
func CombineWithExecutor[T any](exec Executor[[]T], futures ...*Future[T]) *Future[[]T]
func CombineContextWithExecutor[T any](ctx context.Context, exec Executor[[]T],
    futures ...*Future[T]) *Future[[]T]

// No short-circuit (collect all errors)
func CombineNoShortCircuit[T any](futures ...*Future[T]) *Future[[]T]
func CombineContextNoShortCircuit[T any](ctx context.Context,
    futures ...*Future[T]) *Future[[]T]
func CombineNoShortCircuitWithExecutor[T any](exec Executor[[]T],
    futures ...*Future[T]) *Future[[]T]
func CombineContextNoShortCircuitWithExecutor[T any](ctx context.Context,
    exec Executor[[]T], futures ...*Future[T]) *Future[[]T]
```

### Channel Conversion

```go
// Future methods
func (f *Future[T]) ToChannel() <-chan try.Try[T]
func (f *Future[T]) ToChannelContext(ctx context.Context) <-chan try.Try[T]
```

### Cancellation

```go
// Future method
func (f *Future[T]) Cancel()
```

### Executor Interface

```go
type Executor[T any] interface {
    Go(promise *Promise[T], callback func() (T, error))
    GoContext(ctx context.Context, promise *Promise[T],
        callback func(context.Context) (T, error))
}

// Default implementation
type DefaultGoExecutor[T any] struct{}
func NewDefaultExecutor[T any]() Executor[T]
```

## Examples

**Note:** The following examples use Go 1.22+ syntax where loop variables are automatically captured per-iteration. If you're using Go < 1.22, create a local copy of the loop variable inside the loop body (e.g., `url := url`) to avoid capturing issues.

### Example 1: Parallel HTTP Requests

```go
urls := []string{
    "https://api.example.com/users/1",
    "https://api.example.com/users/2",
    "https://api.example.com/users/3",
}

// Launch all requests concurrently
// Note: Go 1.22+ captures loop variables per-iteration automatically
futures := make([]*future.Future[*http.Response], len(urls))
for i, url := range urls {
    futures[i] = future.Go(func() (*http.Response, error) {
        return http.Get(url) // 'url' correctly captured per-iteration
    })
}

// Wait for all to complete
combined := future.Combine(futures...)
responses, err := combined.Await()
if err != nil {
    log.Printf("One or more requests failed: %v", err)
    return
}

for i, resp := range responses {
    log.Printf("Response %d: %d", i, resp.StatusCode)
}
```

### Example 2: Chaining Async Operations

```go
ctx := context.Background() // In production, use req.Context() or similar

// Fetch user ID
userIDFuture := future.GoContext(ctx, func(ctx context.Context) (int, error) {
    return getUserID(ctx)
})

// Fetch user details (depends on user ID)
userFuture := future.FlatMapContext(ctx, userIDFuture,
    func(userID int) *future.Future[User] {
        return future.GoContext(ctx, func(ctx context.Context) (User, error) {
            return fetchUser(ctx, userID)
        })
    })

// Fetch user's posts (depends on user)
postsFuture := future.FlatMapContext(ctx, userFuture,
    func(user User) *future.Future[[]Post] {
        return future.GoContext(ctx, func(ctx context.Context) ([]Post, error) {
            return fetchPosts(ctx, user.ID)
        })
    })

posts, err := postsFuture.AwaitContext(ctx)
```

### Example 3: Timeout and Retry

```go
func fetchWithRetry(ctx context.Context, maxAttempts int) (*Data, error) {
    for attempt := 1; attempt <= maxAttempts; attempt++ {
        timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)

        fut := future.GoContext(timeoutCtx, fetchData)
        result, err := fut.AwaitContext(timeoutCtx)
        cancel() // Clean up immediately to prevent context leak

        if err == nil {
            return &result, nil
        }

        if !errors.Is(err, context.DeadlineExceeded) {
            return nil, err // Non-timeout error
        }

        if attempt < maxAttempts {
            log.Printf("Attempt %d timed out, retrying...", attempt)
            time.Sleep(time.Second * time.Duration(attempt))
        }
    }

    return nil, errors.New("all attempts failed")
}
```

### Example 4: Callback-Based Caching

```go
func fetchUserWithCache(userID int) *future.Future[User] {
    // Check cache first
    if cached, ok := cache.Get(userID); ok {
        return future.Go(func() (User, error) {
            return cached.(User), nil
        })
    }

    // Fetch from DB
    fut := future.Go(func() (User, error) {
        return db.FetchUser(userID)
    })

    // Cache successful results asynchronously
    fut.OnSuccess(func(user User) {
        cache.Set(userID, user)
    })

    return fut
}
```

### Example 5: Fan-Out/Fan-In Pattern

```go
func processItems(ctx context.Context, items []Item) ([]Result, error) { // ctx typically from req.Context()
    // Fan-out: launch concurrent processing
    // Note: Go 1.22+ captures loop variables per-iteration automatically
    futures := make([]*future.Future[Result], len(items))
    for i, item := range items {
        futures[i] = future.GoContext(ctx, func(ctx context.Context) (Result, error) {
            return processItem(ctx, item) // 'item' correctly captured per-iteration
        })
    }

    // Fan-in: collect all results
    combined := future.CombineContext(ctx, futures...)
    return combined.AwaitContext(ctx)
}
```

### Example 6: Progress Tracking with Callbacks

```go
func processWithProgress(items []Item) ([]Result, error) {
    var processed atomic.Int32
    total := len(items)

    // Note: Go 1.22+ captures loop variables per-iteration automatically
    futures := make([]*future.Future[Result], total)
    for i, item := range items {
        fut := future.Go(func() (Result, error) {
            return processItem(item) // 'item' correctly captured per-iteration
        })

        // Track progress
        fut.OnResult(func(result try.Try[Result]) {
            count := processed.Add(1)
            percentage := (float64(count) / float64(total)) * 100
            log.Printf("Progress: %.1f%% (%d/%d)", percentage, count, total)
        })

        futures[i] = fut
    }

    combined := future.Combine(futures...)
    return combined.Await()
}
```

## Thread Safety

All operations in this package are thread-safe:

- **Futures** can be awaited by multiple goroutines simultaneously
- **Promises** can be completed from any goroutine (first completion wins)
- **Callbacks** can be registered concurrently with fulfillment
- **Transformations** are safe for concurrent use
- Result **memoization** uses sync.Once for thread-safe caching
- Channel **operations** use mutexes to prevent race conditions

## Performance Considerations

1. **Goroutine Overhead**: Each `Go()` call spawns a goroutine (~2KB stack allocation).
   - For very fast operations (< 1μs, e.g., simple arithmetic, map lookups, or struct field access), direct execution may be faster
   - For high-volume operations (>1000/sec, e.g., processing message queues or handling frequent API requests), consider a custom executor with worker pool
   - Example: `GoWithExecutor(poolExecutor, fastOp)` for rate limiting or resource pooling
   - Trade-off: Goroutines enable true concurrency but add ~2-3μs overhead per spawn

2. **Memory Usage**: Futures store results in memory. Large result sets should be handled carefully.

3. **Callback Execution**: Callbacks run in separate goroutines, adding overhead. Use for async side effects only.

4. **Executor Reuse**: Create custom executors once and reuse them to avoid allocation overhead.

5. **Context Propagation**: Context-aware operations have slight overhead but enable proper cancellation.

## License

This package is part of amp-common and follows the same license.
