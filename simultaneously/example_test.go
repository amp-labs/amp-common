package simultaneously_test

import (
	"context"
	"fmt"
	"time"

	"github.com/amp-labs/amp-common/simultaneously"
)

// ExampleDo demonstrates parallel execution with concurrency limit.
func ExampleDo() {
	// Run up to 2 operations concurrently
	err := simultaneously.Do(2,
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)

			return nil
		},
		func(ctx context.Context) error {
			time.Sleep(10 * time.Millisecond)

			return nil
		},
	)
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Println("All tasks complete")
	// Output:
	// All tasks complete
}

// ExampleMapSlice demonstrates transforming a slice in parallel.
func ExampleMapSlice() {
	numbers := []int{1, 2, 3, 4, 5}

	// Double each number in parallel (max 3 concurrent operations)
	results, err := simultaneously.MapSlice(3, numbers, func(ctx context.Context, n int) (int, error) {
		return n * 2, nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	// Results maintain original order
	fmt.Printf("Doubled: %v\n", results)
	// Output: Doubled: [2 4 6 8 10]
}

// ExampleMapGoMap demonstrates transforming a map in parallel.
func ExampleMapGoMap() {
	input := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
	}

	// Square each value in parallel
	results, err := simultaneously.MapGoMap(2, input, func(ctx context.Context, key string, val int) (string, int, error) {
		return key, val * val, nil
	})
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	// Sum results
	sum := 0
	for _, v := range results {
		sum += v
	}

	fmt.Printf("Sum of squares: %d\n", sum)
	// Output: Sum of squares: 14
}

// ExampleNewDefaultExecutor demonstrates reusing an executor for multiple batches.
func ExampleNewDefaultExecutor() {
	// Create a reusable executor with max 3 concurrent operations
	exec := simultaneously.NewDefaultExecutor(3)

	defer func() { _ = exec.Close() }()

	completed := 0

	// Submit first batch
	exec.Go(func(ctx context.Context) error {
		completed++

		return nil
	}, func(err error) {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	})

	exec.Go(func(ctx context.Context) error {
		completed++

		return nil
	}, func(err error) {
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	})

	// Wait for completion
	_ = exec.Close()

	fmt.Printf("Completed %d tasks\n", completed)
	// Output: Completed 2 tasks
}
