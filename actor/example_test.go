package actor_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/amp-labs/amp-common/actor"
)

var errUnknownCommand = errors.New("unknown command")

// ExampleNew demonstrates creating a basic actor.
func ExampleNew() {
	ctx := context.Background()

	// Create an actor that processes string requests and returns string responses
	myActor := actor.New(func(ref *actor.Ref[string, string]) actor.Processor[string, string] {
		return actor.SimpleProcessor(func(msg string) (string, error) {
			return "Processed: " + msg, nil
		})
	})

	// Start the actor (Run returns a Ref)
	ref := myActor.Run(ctx, "processor", 10)
	defer ref.Stop()

	// Send a request
	result, err := ref.RequestCtx(ctx, "Hello")
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Println(result)
	// Output: Processed: Hello
}

// ExampleRef_RequestCtx demonstrates synchronous request-response with an actor.
func ExampleRef_RequestCtx() {
	ctx := context.Background()

	// Create an actor that doubles numbers
	doubler := actor.New(func(ref *actor.Ref[int, int]) actor.Processor[int, int] {
		return actor.SimpleProcessor(func(num int) (int, error) {
			return num * 2, nil
		})
	})

	// Start the actor and get reference
	ref := doubler.Run(ctx, "doubler", 10)
	defer ref.Stop()

	// Send synchronous request
	result, err := ref.RequestCtx(ctx, 21)
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Printf("Result: %d\n", result)
	// Output: Result: 42
}

// ExampleRef_SendCtx demonstrates asynchronous message sending to an actor.
func ExampleRef_SendCtx() {
	ctx := context.Background()

	// Create an actor that logs messages
	logger := actor.New(func(ref *actor.Ref[string, struct{}]) actor.Processor[string, struct{}] {
		return actor.SimpleProcessor(func(msg string) (struct{}, error) {
			fmt.Printf("Logged: %s\n", msg)

			return struct{}{}, nil
		})
	})

	// Start the actor and get reference
	ref := logger.Run(ctx, "logger", 10)
	defer ref.Stop()

	// Send async message (no response expected)
	ref.SendCtx(ctx, "Hello, Actor!")

	// Give actor time to process
	time.Sleep(20 * time.Millisecond)

	// Output: Logged: Hello, Actor!
}

// ExampleActor demonstrates a complete actor workflow with state.
func ExampleActor() {
	ctx := context.Background()

	// Create a counter actor with internal state
	counter := actor.New(func(ref *actor.Ref[string, int]) actor.Processor[string, int] {
		count := 0

		return actor.SimpleProcessor(func(cmd string) (int, error) {
			switch cmd {
			case "increment":
				count++

				return count, nil
			case "get":
				return count, nil
			default:
				return 0, fmt.Errorf("%w: %s", errUnknownCommand, cmd)
			}
		})
	})

	// Start the actor and get reference
	ref := counter.Run(ctx, "counter", 10)
	defer ref.Stop()

	// Increment counter
	val1, _ := ref.RequestCtx(ctx, "increment")
	fmt.Printf("After increment: %d\n", val1)

	val2, _ := ref.RequestCtx(ctx, "increment")
	fmt.Printf("After increment: %d\n", val2)

	val3, _ := ref.RequestCtx(ctx, "get")
	fmt.Printf("Current value: %d\n", val3)

	// Output:
	// After increment: 1
	// After increment: 2
	// Current value: 2
}
