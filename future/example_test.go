package future_test

import (
	"context"
	"fmt"
	"time"

	"github.com/amp-labs/amp-common/future"
)

// ExampleGo demonstrates basic future creation and awaiting.
func ExampleGo() {
	// Create an async computation
	fut := future.Go(func() (string, error) {
		return "Hello, Future!", nil
	})

	// Wait for the result
	result, err := fut.Await()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Println(result)
	// Output: Hello, Future!
}

// ExampleGoContext demonstrates context-aware future with timeout.
func ExampleGoContext() {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Create a context-aware future
	fut := future.GoContext(ctx, func(ctx context.Context) (int, error) {
		return 42, nil
	})

	// Await with context
	result, err := fut.AwaitContext(ctx)
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Printf("Result: %d\n", result)
	// Output: Result: 42
}

// ExampleNew demonstrates manual future/promise creation.
func ExampleNew() {
	// Create a future/promise pair
	fut, promise := future.New[int]()

	// Launch async work
	go func() {
		// Simulate work
		time.Sleep(10 * time.Millisecond)
		promise.Success(100)
	}()

	// Wait for completion
	result, err := fut.Await()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Printf("Result: %d\n", result)
	// Output: Result: 100
}

// ExampleMap demonstrates transforming future values.
func ExampleMap() {
	// Create a future that returns an integer
	intFuture := future.Go(func() (int, error) {
		return 42, nil
	})

	// Transform the integer to a string
	stringFuture := future.Map(intFuture, func(value int) (string, error) {
		return fmt.Sprintf("The answer is %d", value), nil
	})

	// Get the transformed result
	result, err := stringFuture.Await()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Println(result)
	// Output: The answer is 42
}

// ExampleCombine demonstrates waiting for multiple futures.
func ExampleCombine() {
	// Launch multiple async operations
	fut1 := future.Go(func() (int, error) { return 1, nil })
	fut2 := future.Go(func() (int, error) { return 2, nil })
	fut3 := future.Go(func() (int, error) { return 3, nil })

	// Combine all futures
	combined := future.Combine(fut1, fut2, fut3)

	// Wait for all to complete
	results, err := combined.Await()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	// Calculate sum
	sum := 0
	for _, val := range results {
		sum += val
	}

	fmt.Printf("Sum: %d\n", sum)
	// Output: Sum: 6
}

// ExampleAsync demonstrates fire-and-forget async operations.
func ExampleAsync() {
	// Launch background work without waiting
	future.Async(func() {
		fmt.Println("Background work completed")
	})

	// Give async operation time to complete for example output
	time.Sleep(50 * time.Millisecond)

	// Output: Background work completed
}

// ExampleFuture_OnSuccess demonstrates callback-based success handling.
func ExampleFuture_OnSuccess() {
	fut := future.Go(func() (string, error) {
		return "Success!", nil
	})

	// Register success callback
	fut.OnSuccess(func(value string) {
		fmt.Printf("Callback received: %s\n", value)
	})

	// Wait for completion to ensure callback runs
	_, _ = fut.Await()

	time.Sleep(10 * time.Millisecond) // Give callback time to execute

	// Output: Callback received: Success!
}

// ExampleFuture_ToChannel demonstrates converting a future to a channel.
func ExampleFuture_ToChannel() {
	fut := future.Go(func() (int, error) {
		return 42, nil
	})

	// Convert to channel
	ch := fut.ToChannel()

	// Read from channel
	result := <-ch
	if result.Error != nil {
		fmt.Printf("Error: %v\n", result.Error)

		return
	}

	fmt.Printf("Value: %d\n", result.Value)
	// Output: Value: 42
}
