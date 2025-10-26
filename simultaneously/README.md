# simultaneously

[![Go Reference](https://pkg.go.dev/badge/github.com/amp-labs/amp-common/simultaneously.svg)](https://pkg.go.dev/github.com/amp-labs/amp-common/simultaneously)

A Go library for safe, controlled parallel execution with automatic panic recovery, context cancellation, and error handling.

## Table of Contents

- [Purpose](#purpose)
- [Core Concepts](#core-concepts)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Basic Usage](#basic-usage)
- [Advanced Usage](#advanced-usage)
- [Data Transformations](#data-transformations)
- [Best Practices](#best-practices)
- [Error Handling](#error-handling)
- [API Reference](#api-reference)

## Purpose

The `simultaneously` package provides utilities for running functions concurrently with controlled parallelism. It solves common challenges in concurrent programming:

- **Controlled Concurrency**: Limit how many goroutines run at once to prevent resource exhaustion
- **Panic Recovery**: Automatically recover from panics and convert them to errors
- **Context Cancellation**: Stop all work when one function fails or context is canceled
- **Error Aggregation**: Collect and return errors from parallel operations
- **Type-Safe Transformations**: Generic functions for transforming slices, maps, and sets in parallel

## Core Concepts

### Executor

The `Executor` interface manages concurrent execution with configurable concurrency limits:

```go
type Executor interface {
    // Execute a function asynchronously with context support
    GoContext(ctx context.Context, fn func(context.Context) error, done func(error))

    // Execute a function asynchronously (convenience wrapper)
    Go(fn func(context.Context) error, done func(error))

    // Shut down the executor and wait for completion
    Close() error
}
```

**Key Features:**
- Semaphore-based concurrency control
- Thread-safe execution management
- Graceful shutdown with completion tracking
- Reusable across multiple batches of work

### Concurrency Control

The `maxConcurrent` parameter controls how many operations run simultaneously:

- `maxConcurrent > 0`: Limit to N concurrent operations
- `maxConcurrent <= 0`: Unlimited concurrency (bounded only by goroutine scheduler)
- Automatically capped at number of items to process

### Panic Recovery

All functions automatically recover from panics and convert them to errors:

```go
err := simultaneously.Do(2,
    func(ctx context.Context) error {
        panic("something went wrong") // Recovered and returned as error
    },
)
// err contains: "recovered from panic: something went wrong" + stack trace
```

### Context Cancellation

When any function returns an error, the shared context is canceled to stop remaining work:

```go
err := simultaneously.DoCtx(ctx, 3,
    func(ctx context.Context) error {
        return errors.New("failed") // Triggers cancellation
    },
    func(ctx context.Context) error {
        // This should check ctx.Done() and exit early
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            // Do work...
        }
    },
)
```

## Installation

```bash
go get github.com/amp-labs/amp-common/simultaneously
```

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/amp-labs/amp-common/simultaneously"
)

func main() {
    // Run 3 tasks with max 2 concurrent
    err := simultaneously.Do(2,
        func(ctx context.Context) error {
            fmt.Println("Task 1")
            time.Sleep(100 * time.Millisecond)
            return nil
        },
        func(ctx context.Context) error {
            fmt.Println("Task 2")
            time.Sleep(100 * time.Millisecond)
            return nil
        },
        func(ctx context.Context) error {
            fmt.Println("Task 3")
            time.Sleep(100 * time.Millisecond)
            return nil
        },
    )

    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Basic Usage

### Running Multiple Functions

Use `Do` or `DoCtx` to run multiple functions in parallel:

```go
// Without context (uses context.Background)
err := simultaneously.Do(maxConcurrent,
    func(ctx context.Context) error {
        // Task 1
        return nil
    },
    func(ctx context.Context) error {
        // Task 2
        return nil
    },
)

// With context for cancellation/timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

err = simultaneously.DoCtx(ctx, maxConcurrent,
    func(ctx context.Context) error {
        // Task 1
        return nil
    },
    func(ctx context.Context) error {
        // Task 2
        return nil
    },
)
```

### Transforming Slices

Transform slice elements in parallel while preserving order:

```go
numbers := []int{1, 2, 3, 4, 5}

// Double each number in parallel (max 2 concurrent)
doubled, err := simultaneously.MapSlice(2, numbers,
    func(ctx context.Context, n int) (int, error) {
        return n * 2, nil
    },
)
// doubled = [2, 4, 6, 8, 10]
```

### Transforming Maps

Transform map entries in parallel:

```go
input := map[string]int{
    "a": 1,
    "b": 2,
    "c": 3,
}

// Convert to map[int]string in parallel
output, err := simultaneously.MapGoMap(2, input,
    func(ctx context.Context, k string, v int) (int, string, error) {
        return v, strings.ToUpper(k), nil
    },
)
// output = map[int]string{1: "A", 2: "B", 3: "C"}
```

## Advanced Usage

### Reusing Executors

For multiple batches of work, reuse an executor to avoid creation overhead:

```go
// Create executor once
exec := simultaneously.NewDefaultExecutor(3) // max 3 concurrent
defer exec.Close()

// Process multiple batches
batch1 := []int{1, 2, 3, 4, 5}
batch2 := []int{6, 7, 8, 9, 10}

result1, err := simultaneously.MapSliceWithExecutor(exec, batch1,
    func(ctx context.Context, n int) (int, error) {
        return n * 2, nil
    },
)
if err != nil {
    return err
}

result2, err := simultaneously.MapSliceWithExecutor(exec, batch2,
    func(ctx context.Context, n int) (int, error) {
        return n * 2, nil
    },
)
if err != nil {
    return err
}
```

### Custom Executor Implementation

Implement the `Executor` interface for custom behavior:

```go
type CustomExecutor struct {
    // Your custom fields
}

func (e *CustomExecutor) GoContext(ctx context.Context, fn func(context.Context) error, done func(error)) {
    // Your custom execution logic
}

func (e *CustomExecutor) Go(fn func(context.Context) error, done func(error)) {
    e.GoContext(context.Background(), fn, done)
}

func (e *CustomExecutor) Close() error {
    // Your custom cleanup logic
    return nil
}

// Use your custom executor
exec := &CustomExecutor{}
err := simultaneously.DoWithExecutor(exec, tasks...)
```

### Flat Mapping

Expand each input into multiple outputs (flattening):

```go
// FlatMapSlice - expand strings into characters
words := []string{"hello", "world"}
chars, err := simultaneously.FlatMapSlice(2, words,
    func(ctx context.Context, word string) ([]rune, error) {
        return []rune(word), nil
    },
)
// chars = ['h', 'e', 'l', 'l', 'o', 'w', 'o', 'r', 'l', 'd']

// FlatMapGoMap - expand each entry into multiple entries
input := map[string]int{"a": 2, "b": 3}
output, err := simultaneously.FlatMapGoMap(2, input,
    func(ctx context.Context, k string, v int) (map[string]int, error) {
        result := make(map[string]int)
        for i := 0; i < v; i++ {
            result[fmt.Sprintf("%s%d", k, i)] = i
        }
        return result, nil
    },
)
// output = map[string]int{"a0": 0, "a1": 1, "b0": 0, "b1": 1, "b2": 2}
```

## Data Transformations

The package provides parallel transformation functions for various data structures:

### Slices

| Function | Description | Order Preserved |
|----------|-------------|-----------------|
| `MapSlice` | Transform each element | ✓ |
| `FlatMapSlice` | Transform and flatten | ✓ |

### Go Maps (standard `map[K]V`)

| Function | Description | Order Preserved |
|----------|-------------|-----------------|
| `MapGoMap` | Transform key-value pairs | ✗ |
| `FlatMapGoMap` | Transform and flatten | ✗ |

### amp-common Maps

| Function | Description | Order Preserved |
|----------|-------------|-----------------|
| `MapMap` | Transform amp-common Map | ✗ |
| `FlatMapMap` | Transform and flatten | ✗ |
| `MapOrderedMap` | Transform with order | ✓ |
| `FlatMapOrderedMap` | Transform and flatten with order | ✓ |

### amp-common Sets

| Function | Description | Order Preserved |
|----------|-------------|-----------------|
| `MapSet` | Transform set elements | ✗ |
| `FlatMapSet` | Transform and flatten | ✗ |
| `MapOrderedSet` | Transform with order | ✓ |
| `FlatMapOrderedSet` | Transform and flatten with order | ✓ |

**Note:** All functions have:
- Base version (uses `context.Background()`)
- `Ctx` version (accepts custom context)
- `WithExecutor` version (uses custom executor)

## Best Practices

### 1. Always Check Context Cancellation

Long-running functions should periodically check if the context is canceled:

```go
err := simultaneously.DoCtx(ctx, 2,
    func(ctx context.Context) error {
        for i := 0; i < 1000; i++ {
            // Check cancellation periodically
            select {
            case <-ctx.Done():
                return ctx.Err()
            default:
            }

            // Do work
            processItem(i)
        }
        return nil
    },
)
```

### 2. Use Appropriate Concurrency Limits

Choose `maxConcurrent` based on your workload:

```go
// CPU-bound work: limit to number of CPUs
cpuBound := runtime.NumCPU()
err := simultaneously.Do(cpuBound, tasks...)

// I/O-bound work: higher concurrency is ok
ioBound := 50
err := simultaneously.Do(ioBound, tasks...)

// Unlimited: use 0 or negative value (use with caution)
err := simultaneously.Do(0, tasks...)
```

### 3. Reuse Executors for Multiple Batches

When processing multiple batches, reuse the executor:

```go
exec := simultaneously.NewDefaultExecutor(10)
defer exec.Close()

for _, batch := range batches {
    results, err := simultaneously.MapSliceWithExecutor(exec, batch, transform)
    if err != nil {
        return err
    }
    // Process results...
}
```

### 4. Set Timeouts for Operations

Use context timeouts to prevent hanging:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := simultaneously.DoCtx(ctx, 5, tasks...)
if errors.Is(err, context.DeadlineExceeded) {
    // Handle timeout
}
```

### 5. Handle Panics Gracefully

The package recovers panics automatically and converts them to errors with stack traces:

```go
err := simultaneously.Do(2,
    func(ctx context.Context) error {
        // Risky operation - panics are recovered automatically
        // and converted to errors with stack traces
        return riskyOperation(ctx)
    },
)

if err != nil {
    // Panic has already been recovered and logged with stack trace
    // Just handle the error normally
    log.Printf("Operation failed: %v", err)
}
```

You don't need to recover panics yourself - the library handles this for you.

## Error Handling

### Error Propagation

The first error encountered stops all remaining work:

```go
err := simultaneously.Do(3,
    func(ctx context.Context) error {
        time.Sleep(100 * time.Millisecond)
        return errors.New("task 1 failed") // This error is returned
    },
    func(ctx context.Context) error {
        time.Sleep(200 * time.Millisecond)
        return errors.New("task 2 failed") // May not execute
    },
)
// err = "task 1 failed" (first error wins)
```

### Multiple Errors

When multiple errors occur simultaneously, they are combined:

```go
err := simultaneously.Do(3,
    func(ctx context.Context) error {
        return errors.New("error 1")
    },
    func(ctx context.Context) error {
        return errors.New("error 2")
    },
)
// err contains both errors joined with errors.Join
```

### Panic Recovery

Panics are converted to errors with stack traces:

```go
err := simultaneously.Do(1,
    func(ctx context.Context) error {
        panic("unexpected panic")
    },
)
// err contains:
// - "recovered from panic: unexpected panic"
// - Full stack trace
// - File and line number where panic occurred
```

### Context Errors

Context cancellation and deadline errors are propagated:

```go
ctx, cancel := context.WithCancel(context.Background())
cancel() // Cancel immediately

err := simultaneously.DoCtx(ctx, 2, tasks...)
// err = context.Canceled

ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Nanosecond)
defer cancel2()

err = simultaneously.DoCtx(ctx2, 2, longRunningTasks...)
// err = context.DeadlineExceeded
```

## API Reference

### Core Functions

#### Do / DoCtx

Run multiple functions in parallel with controlled concurrency:

```go
func Do(maxConcurrent int, funcs ...func(ctx context.Context) error) error
func DoCtx(ctx context.Context, maxConcurrent int, funcs ...func(ctx context.Context) error) error
```

#### DoWithExecutor / DoCtxWithExecutor

Run functions using a custom executor:

```go
func DoWithExecutor(exec Executor, funcs ...func(ctx context.Context) error) error
func DoCtxWithExecutor(ctx context.Context, exec Executor, funcs ...func(ctx context.Context) error) error
```

### Executor

#### NewDefaultExecutor

Create a new executor with concurrency limit:

```go
func NewDefaultExecutor(maxConcurrent int) Executor
```

### Slice Transformations

Transform slices in parallel while preserving order:

```go
// Map: one-to-one transformation
func MapSlice[In, Out any](maxConcurrent int, values []In,
    transform func(ctx context.Context, value In) (Out, error)) ([]Out, error)

func MapSliceCtx[In, Out any](ctx context.Context, maxConcurrent int, values []In,
    transform func(ctx context.Context, value In) (Out, error)) ([]Out, error)

func MapSliceWithExecutor[In, Out any](exec Executor, values []In,
    transform func(ctx context.Context, value In) (Out, error)) ([]Out, error)

// FlatMap: one-to-many transformation with flattening
func FlatMapSlice[In, Out any](maxConcurrent int, values []In,
    transform func(ctx context.Context, value In) ([]Out, error)) ([]Out, error)

func FlatMapSliceCtx[In, Out any](ctx context.Context, maxConcurrent int, values []In,
    transform func(ctx context.Context, value In) ([]Out, error)) ([]Out, error)

func FlatMapSliceWithExecutor[In, Out any](exec Executor, values []In,
    transform func(ctx context.Context, value In) ([]Out, error)) ([]Out, error)
```

### Go Map Transformations

Transform standard Go maps in parallel:

```go
// Map: transform key-value pairs
func MapGoMap[InK comparable, InV, OutK comparable, OutV any](
    maxConcurrent int, input map[InK]InV,
    transform func(ctx context.Context, key InK, val InV) (OutK, OutV, error),
) (map[OutK]OutV, error)

func MapGoMapCtx[InK comparable, InV, OutK comparable, OutV any](
    ctx context.Context, maxConcurrent int, input map[InK]InV,
    transform func(ctx context.Context, key InK, val InV) (OutK, OutV, error),
) (map[OutK]OutV, error)

func MapGoMapWithExecutor[InK comparable, InV, OutK comparable, OutV any](
    exec Executor, input map[InK]InV,
    transform func(ctx context.Context, key InK, val InV) (OutK, OutV, error),
) (map[OutK]OutV, error)

// FlatMap: expand entries into multiple entries
func FlatMapGoMap[InK comparable, InV, OutK comparable, OutV any](
    maxConcurrent int, input map[InK]InV,
    transform func(ctx context.Context, key InK, val InV) (map[OutK]OutV, error),
) (map[OutK]OutV, error)

func FlatMapGoMapCtx[InK comparable, InV, OutK comparable, OutV any](
    ctx context.Context, maxConcurrent int, input map[InK]InV,
    transform func(ctx context.Context, key InK, val InV) (map[OutK]OutV, error),
) (map[OutK]OutV, error)

func FlatMapGoMapWithExecutor[InK comparable, InV, OutK comparable, OutV any](
    exec Executor, input map[InK]InV,
    transform func(ctx context.Context, key InK, val InV) (map[OutK]OutV, error),
) (map[OutK]OutV, error)
```

### amp-common Data Structures

The package also provides parallel transformations for amp-common `Map`, `OrderedMap`, `Set`, and `OrderedSet` types. Each follows the same pattern with base, `Ctx`, and `WithExecutor` variants.

See the [full API documentation](https://pkg.go.dev/github.com/amp-labs/amp-common/simultaneously) for details.

## Examples

### Example 1: Parallel HTTP Requests

```go
urls := []string{
    "https://api.example.com/users/1",
    "https://api.example.com/users/2",
    "https://api.example.com/users/3",
}

responses, err := simultaneously.MapSlice(5, urls,
    func(ctx context.Context, url string) (*http.Response, error) {
        req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
        if err != nil {
            return nil, err
        }
        return http.DefaultClient.Do(req)
    },
)
```

### Example 2: Parallel File Processing

```go
files := []string{"file1.txt", "file2.txt", "file3.txt"}

err := simultaneously.Do(2,
    func(ctx context.Context) error {
        return processFile(ctx, "file1.txt")
    },
    func(ctx context.Context) error {
        return processFile(ctx, "file2.txt")
    },
    func(ctx context.Context) error {
        return processFile(ctx, "file3.txt")
    },
)
```

### Example 3: Batch Processing with Retry

```go
exec := simultaneously.NewDefaultExecutor(10)
defer exec.Close()

for attempt := 0; attempt < 3; attempt++ {
    err := simultaneously.DoCtxWithExecutor(ctx, exec, tasks...)
    if err == nil {
        break // Success
    }

    if attempt < 2 {
        time.Sleep(time.Second * time.Duration(attempt+1))
    }
}
```

### Example 4: Processing with Progress Tracking

```go
var processed atomic.Int32
total := len(items)

results, err := simultaneously.MapSlice(10, items,
    func(ctx context.Context, item Item) (Result, error) {
        result, err := process(item)
        if err != nil {
            return Result{}, err
        }

        count := processed.Add(1)
        if count%100 == 0 {
            log.Printf("Processed %d/%d items", count, total)
        }

        return result, nil
    },
)
```

## Thread Safety

All functions in this package are thread-safe:

- Output collections (maps, sets) use mutexes for concurrent writes
- Slice outputs preserve order using indexed writes with mutex protection
- Executors use atomic operations and channels for synchronization
- Context cancellation is handled with `sync.Once` to ensure single execution

## Performance Considerations

1. **Concurrency Overhead**: Each concurrent operation has overhead. For very fast operations, sequential execution may be faster.

2. **Memory Usage**: Higher concurrency means more goroutines and memory. Monitor memory usage and adjust `maxConcurrent` accordingly.

3. **CPU vs I/O Bound**:
   - CPU-bound: Set `maxConcurrent` to `runtime.NumCPU()`
   - I/O-bound: Higher concurrency (10-100) is usually fine

4. **Executor Reuse**: Creating executors has overhead. Reuse them when processing multiple batches.

## License

This package is part of amp-common and follows the same license.
