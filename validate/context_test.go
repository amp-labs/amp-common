package validate

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonErrors "github.com/amp-labs/amp-common/errors"
	"github.com/stretchr/testify/assert"
)

func TestWithWantProblemErrors_SetTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithWantProblemErrors(ctx, true)

	result := WantProblemErrors(ctx)
	assert.True(t, result)
}

func TestWithWantProblemErrors_SetFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithWantProblemErrors(ctx, false)

	result := WantProblemErrors(ctx)
	assert.False(t, result)
}

func TestWantProblemErrors_DefaultFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	result := WantProblemErrors(ctx)
	assert.False(t, result, "default should be false")
}

func TestWantProblemErrors_Multiple(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Set to true
	ctx = WithWantProblemErrors(ctx, true)
	assert.True(t, WantProblemErrors(ctx))

	// Override to false
	ctx = WithWantProblemErrors(ctx, false)
	assert.False(t, WantProblemErrors(ctx))

	// Override back to true
	ctx = WithWantProblemErrors(ctx, true)
	assert.True(t, WantProblemErrors(ctx))
}

func TestWithWrappedError_SetTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithWrappedError(ctx, true)

	result := wantWrappedErrors(ctx)
	assert.True(t, result)
}

func TestWithWrappedError_SetFalse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithWrappedError(ctx, false)

	result := wantWrappedErrors(ctx)
	assert.False(t, result)
}

func TestWantWrappedErrors_DefaultTrue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	result := wantWrappedErrors(ctx)
	assert.True(t, result, "default should be true")
}

func TestWantWrappedErrors_Multiple(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Default is true
	assert.True(t, wantWrappedErrors(ctx))

	// Set to false
	ctx = WithWrappedError(ctx, false)
	assert.False(t, wantWrappedErrors(ctx))

	// Override back to true
	ctx = WithWrappedError(ctx, true)
	assert.True(t, wantWrappedErrors(ctx))
}

func TestContextFlags_Independent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Set both flags independently
	ctx = WithWantProblemErrors(ctx, true)
	ctx = WithWrappedError(ctx, false)

	assert.True(t, WantProblemErrors(ctx))
	assert.False(t, wantWrappedErrors(ctx))

	// Flip them
	ctx = WithWantProblemErrors(ctx, false)
	ctx = WithWrappedError(ctx, true)

	assert.False(t, WantProblemErrors(ctx))
	assert.True(t, wantWrappedErrors(ctx))
}

func TestContextFlags_Propagation(t *testing.T) {
	t.Parallel()

	// Parent context with flags set
	parentCtx := context.Background()
	parentCtx = WithWantProblemErrors(parentCtx, true)
	parentCtx = WithWrappedError(parentCtx, false)

	// Child context should inherit values
	childCtx := context.WithValue(parentCtx, "key", "value")

	assert.True(t, WantProblemErrors(childCtx))
	assert.False(t, wantWrappedErrors(childCtx))
}

func TestContextFlags_Isolation(t *testing.T) {
	t.Parallel()

	// Create two independent contexts
	ctx1 := context.Background()
	ctx1 = WithWantProblemErrors(ctx1, true)

	ctx2 := context.Background()
	ctx2 = WithWantProblemErrors(ctx2, false)

	// They should be independent
	assert.True(t, WantProblemErrors(ctx1))
	assert.False(t, WantProblemErrors(ctx2))
}

func TestContextFlags_EmptyContext(t *testing.T) {
	t.Parallel()

	// Test with a fresh empty context to verify default behavior
	ctx := context.Background()

	// Verify defaults without any flags set
	assert.False(t, WantProblemErrors(ctx), "default WantProblemErrors should be false")
	assert.True(t, wantWrappedErrors(ctx), "default wantWrappedErrors should be true")
}

// Benchmark tests
func BenchmarkWithWantProblemErrors(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = WithWantProblemErrors(ctx, true)
	}
}

func BenchmarkWantProblemErrors(b *testing.B) {
	ctx := context.Background()
	ctx = WithWantProblemErrors(ctx, true)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = WantProblemErrors(ctx)
	}
}

func BenchmarkWithWrappedError(b *testing.B) {
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = WithWrappedError(ctx, false)
	}
}

func BenchmarkWantWrappedErrors(b *testing.B) {
	ctx := context.Background()
	ctx = WithWrappedError(ctx, false)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = wantWrappedErrors(ctx)
	}
}

// Example type for context examples
type exampleValidType struct {
	Value string
}

func (e exampleValidType) Validate() error {
	if e.Value == "" {
		return fmt.Errorf("value is required")
	}

	return nil
}

// Example tests
func ExampleWithWantProblemErrors() {
	ctx := context.Background()

	// Enable problem error formatting for HTTP handlers
	ctx = WithWantProblemErrors(ctx, true)

	// Check if the caller wants problem-formatted errors
	if WantProblemErrors(ctx) {
		fmt.Println("problem errors enabled")
	} else {
		fmt.Println("problem errors disabled")
	}

	// Output:
	// problem errors enabled
}

func ExampleWithWantProblemErrors_default() {
	ctx := context.Background()

	// By default, problem errors are disabled
	if WantProblemErrors(ctx) {
		fmt.Println("problem errors enabled")
	} else {
		fmt.Println("problem errors disabled")
	}

	// Output:
	// problem errors disabled
}

func ExampleWithWrappedError() {
	ctx := context.Background()

	// Disable error wrapping for performance-critical code
	ctx = WithWrappedError(ctx, false)

	// Perform validation - errors won't be wrapped with ErrValidation
	invalidType := exampleValidType{Value: ""}
	err := Validate(ctx, invalidType)

	if err != nil {
		fmt.Println("validation failed:", err)
	}

	// Output:
	// validation failed: value is required
}

func ExampleWithWrappedError_enabled() {
	ctx := context.Background()

	// Enable error wrapping (this is the default)
	ctx = WithWrappedError(ctx, true)

	// Perform validation - errors will be wrapped with ErrValidation
	invalidType := exampleValidType{Value: ""}
	err := Validate(ctx, invalidType)

	if err != nil {
		// Error is wrapped, so it contains ErrValidation
		fmt.Println("validation failed")
		fmt.Println("is validation error:", errors.Is(err, commonErrors.ErrValidation))
	}

	// Output:
	// validation failed
	// is validation error: true
}
