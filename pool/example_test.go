package pool_test

import (
	"fmt"
	"io"
	"time"

	"github.com/amp-labs/amp-common/pool"
)

// mockCloser is a simple closeable object for examples.
type mockCloser struct {
	id     int
	closed bool
}

func (m *mockCloser) Close() error {
	m.closed = true

	return nil
}

// ExampleNew demonstrates creating a basic pool.
func ExampleNew() {
	counter := 0

	// Create a pool with a factory function
	poolInstance := pool.New(func() (io.Closer, error) {
		counter++

		return &mockCloser{id: counter}, nil
	})

	defer func() { _ = poolInstance.Close() }()

	// Get an object from the pool
	obj, err := poolInstance.Get()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	// Return it to the pool
	poolInstance.Put(obj)

	fmt.Println("Pool created and used successfully")
	// Output: Pool created and used successfully
}

// ExamplePool_Get demonstrates getting objects from a pool.
func ExamplePool_Get() {
	// Create a pool
	poolInstance := pool.New(func() (io.Closer, error) {
		return &mockCloser{id: 1}, nil
	})

	defer func() { _ = poolInstance.Close() }()

	// Get an object
	obj, err := poolInstance.Get()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Printf("Got object from pool: %T\n", obj)
	// Output: Got object from pool: *pool_test.mockCloser
}

// ExamplePool_Put demonstrates returning objects to a pool.
func ExamplePool_Put() {
	// Create a pool
	poolInstance := pool.New(func() (io.Closer, error) {
		return &mockCloser{}, nil
	})

	defer func() { _ = poolInstance.Close() }()

	// Get an object
	obj, err := poolInstance.Get()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	// Use the object (simulated)

	// Return it to the pool for reuse
	poolInstance.Put(obj)

	fmt.Println("Object returned to pool")
	// Output: Object returned to pool
}

// ExamplePool_CloseIdle demonstrates cleaning up idle objects.
func ExamplePool_CloseIdle() {
	// Create a pool
	poolInstance := pool.New(func() (io.Closer, error) {
		return &mockCloser{}, nil
	})

	defer func() { _ = poolInstance.Close() }()

	// Get and return some objects
	obj1, _ := poolInstance.Get()
	obj2, _ := poolInstance.Get()
	poolInstance.Put(obj1)
	poolInstance.Put(obj2)

	// Wait for objects to become idle
	time.Sleep(100 * time.Millisecond)

	// Close objects idle for more than 50ms
	closed, err := poolInstance.CloseIdle(50 * time.Millisecond)
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	fmt.Printf("Closed %d idle objects\n", closed)
	// Output: Closed 2 idle objects
}

// ExampleWithName demonstrates creating a named pool for metrics.
func ExampleWithName() {
	// Create a named pool for Prometheus metrics
	poolInstance := pool.New(
		func() (io.Closer, error) {
			return &mockCloser{}, nil
		},
		pool.WithName[io.Closer]("database-connections"),
	)

	defer func() { _ = poolInstance.Close() }()

	// Use the pool
	obj, err := poolInstance.Get()
	if err != nil {
		fmt.Printf("Error: %v\n", err)

		return
	}

	poolInstance.Put(obj)

	fmt.Println("Named pool created")
	// Output: Named pool created
}
