package envutil_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/amp-labs/amp-common/envutil"
)

var errInvalidVersion = errors.New("invalid version")

// ExampleInt demonstrates reading an integer environment variable with a default.
func ExampleInt() {
	ctx := context.Background()

	// Set environment variable
	_ = os.Setenv("PORT", "8080")

	defer func() { _ = os.Unsetenv("PORT") }()

	// Read with default
	port, _ := envutil.Int[int](ctx, "PORT", envutil.Default(3000)).Value()

	fmt.Printf("Port: %d\n", port)
	// Output: Port: 8080
}

// ExampleString demonstrates reading a string environment variable with validation.
func ExampleString() {
	ctx := context.Background()

	// Set environment variable
	_ = os.Setenv("API_VERSION", "v1")

	defer func() { _ = os.Unsetenv("API_VERSION") }()

	// Read with validation
	version, _ := envutil.String(ctx, "API_VERSION",
		envutil.Validate(func(s string) error {
			if s != "v1" && s != "v2" {
				return fmt.Errorf("%w: %s", errInvalidVersion, s)
			}

			return nil
		}),
	).Value()

	fmt.Printf("API Version: %s\n", version)
	// Output: API Version: v1
}

// ExampleDuration demonstrates reading a duration environment variable.
func ExampleDuration() {
	ctx := context.Background()

	// Set environment variable
	_ = os.Setenv("TIMEOUT", "30s")

	defer func() { _ = os.Unsetenv("TIMEOUT") }()

	// Read duration with default
	timeout, _ := envutil.Duration(ctx, "TIMEOUT", envutil.Default(10*time.Second)).Value()

	fmt.Printf("Timeout: %v\n", timeout)
	// Output: Timeout: 30s
}

// ExampleURL demonstrates reading a URL environment variable.
func ExampleURL() {
	ctx := context.Background()

	// Set environment variable
	_ = os.Setenv("API_ENDPOINT", "https://api.example.com")

	defer func() { _ = os.Unsetenv("API_ENDPOINT") }()

	// Read URL
	apiURL, _ := envutil.URL(ctx, "API_ENDPOINT").Value()

	fmt.Printf("API Host: %s\n", apiURL.Host)
	// Output: API Host: api.example.com
}

// ExampleFilePath demonstrates reading a file path environment variable.
func ExampleFilePath() {
	ctx := context.Background()

	// Create a temporary file
	tmpfile, _ := os.CreateTemp("", "example")

	defer func() { _ = os.Remove(tmpfile.Name()) }()

	_ = tmpfile.Close()

	// Set environment variable
	_ = os.Setenv("CONFIG_FILE", tmpfile.Name())

	defer func() { _ = os.Unsetenv("CONFIG_FILE") }()

	// Read file path
	configPath, _ := envutil.FilePath(ctx, "CONFIG_FILE").Value()

	fmt.Println("Config file path read")

	_ = configPath // Use configPath
	// Output: Config file path read
}

// ExampleCombine2 demonstrates combining two environment variable readers.
func ExampleCombine2() {
	ctx := context.Background()

	// Set environment variables
	_ = os.Setenv("HOST", "localhost")
	_ = os.Setenv("PORT", "8080")

	defer func() { _ = os.Unsetenv("HOST") }()
	defer func() { _ = os.Unsetenv("PORT") }()

	// Combine host and port readers
	combined := envutil.Combine2(
		envutil.String(ctx, "HOST", envutil.Default("0.0.0.0")),
		envutil.Int[int](ctx, "PORT", envutil.Default(3000)),
	)

	// Get the tuple value
	tuple, _ := combined.Value()
	host := tuple.First()
	port := tuple.Second()

	fmt.Printf("Address: %s:%d\n", host, port)
	// Output: Address: localhost:8080
}

// ExampleDefault demonstrates using a default value when an environment variable is missing.
func ExampleDefault() {
	ctx := context.Background()

	// Variable not set
	_ = os.Unsetenv("MAX_CONNECTIONS")

	// Read with default
	maxConns, _ := envutil.Int[int](ctx, "MAX_CONNECTIONS", envutil.Default(100)).Value()

	fmt.Printf("Max Connections: %d\n", maxConns)
	// Output: Max Connections: 100
}
