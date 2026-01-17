package validate

import (
	"context"
	"errors"
	"fmt"
	"testing"

	commonErrors "github.com/amp-labs/amp-common/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Static errors for test types.
var (
	errValueRequired      = errors.New("value is required")
	errPortMustBePositive = errors.New("port must be positive")
	errUserIDRequired     = errors.New("userID is required")
)

// Test types implementing HasValidate interface.
type validType struct {
	Value string
}

func (v validType) Validate() error {
	if v.Value == "" {
		return errValueRequired
	}

	return nil
}

type alwaysValidType struct{}

func (a alwaysValidType) Validate() error {
	return nil
}

// Test types implementing HasValidateWithContext interface.
type contextValidType struct {
	Value string
}

func (c contextValidType) Validate(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if c.Value == "" {
		return errValueRequired
	}

	return nil
}

type contextAlwaysValidType struct{}

func (c contextAlwaysValidType) Validate(ctx context.Context) error {
	return nil
}

// Test type that panics during validation.
type panicType struct{}

func (p panicType) Validate() error {
	panic("validation panic")
}

type contextPanicType struct{}

func (c contextPanicType) Validate(ctx context.Context) error {
	panic("context validation panic")
}

// Test type that doesn't implement any validation interface.
type nonValidatableType struct {
	Value string
}

func TestValidate_HasValidate_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := validType{Value: "test"}

	err := Validate(ctx, value)
	assert.NoError(t, err)
}

func TestValidate_HasValidate_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := validType{Value: ""}

	err := Validate(ctx, value)
	require.Error(t, err)
	require.ErrorIs(t, err, commonErrors.ErrValidation)
	assert.Contains(t, err.Error(), "value is required")
}

func TestValidate_HasValidateWithContext_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := contextValidType{Value: "test"}

	err := Validate(ctx, value)
	assert.NoError(t, err)
}

func TestValidate_HasValidateWithContext_Error(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := contextValidType{Value: ""}

	err := Validate(ctx, value)
	require.Error(t, err)
	require.ErrorIs(t, err, commonErrors.ErrValidation)
	assert.Contains(t, err.Error(), "value is required")
}

func TestValidate_HasValidateWithContext_ContextCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel context before validation

	value := contextValidType{Value: "test"}

	err := Validate(ctx, value)
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestValidate_NoInterface_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := nonValidatableType{Value: "test"}

	err := Validate(ctx, value)
	assert.NoError(t, err)
}

func TestValidate_NilValue_Success(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	err := Validate(ctx, nil)
	assert.NoError(t, err)
}

func TestValidate_PanicRecovery_HasValidate(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := panicType{}

	err := Validate(ctx, value)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
	assert.Contains(t, err.Error(), "validation panic")
}

func TestValidate_PanicRecovery_HasValidateWithContext(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	value := contextPanicType{}

	err := Validate(ctx, value)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "panic")
	assert.Contains(t, err.Error(), "context validation panic")
}

func TestValidate_ErrorWrapping_Enabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithWrappedError(ctx, true)

	value := validType{Value: ""}

	err := Validate(ctx, value)
	require.Error(t, err)
	assert.ErrorIs(t, err, commonErrors.ErrValidation)
}

func TestValidate_ErrorWrapping_Disabled(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ctx = WithWrappedError(ctx, false)

	value := validType{Value: ""}

	err := Validate(ctx, value)
	require.Error(t, err)
	// When wrapping is disabled, ErrValidation should not be in the error chain
	require.NotErrorIs(t, err, commonErrors.ErrValidation)
	assert.Contains(t, err.Error(), "value is required")
}

func TestValidate_MultipleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     any
		wantError bool
	}{
		{
			name:      "valid HasValidate",
			value:     validType{Value: "test"},
			wantError: false,
		},
		{
			name:      "invalid HasValidate",
			value:     validType{Value: ""},
			wantError: true,
		},
		{
			name:      "valid HasValidateWithContext",
			value:     contextValidType{Value: "test"},
			wantError: false,
		},
		{
			name:      "invalid HasValidateWithContext",
			value:     contextValidType{Value: ""},
			wantError: true,
		},
		{
			name:      "non-validatable type",
			value:     nonValidatableType{Value: "test"},
			wantError: false,
		},
		{
			name:      "nil value",
			value:     nil,
			wantError: false,
		},
		{
			name:      "string type",
			value:     "test",
			wantError: false,
		},
		{
			name:      "int type",
			value:     42,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := Validate(ctx, tt.value)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidate_PointerTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     any
		wantError bool
	}{
		{
			name:      "pointer to valid type",
			value:     &validType{Value: "test"},
			wantError: false,
		},
		{
			name:      "pointer to invalid type",
			value:     &validType{Value: ""},
			wantError: true,
		},
		{
			name:      "pointer to context valid type",
			value:     &contextValidType{Value: "test"},
			wantError: false,
		},
		{
			name:      "pointer to context invalid type",
			value:     &contextValidType{Value: ""},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := Validate(ctx, tt.value)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInternal_TypeChecking(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		value        any
		wantError    bool
		errorMessage string
	}{
		{
			name:      "implements HasValidate - valid",
			value:     alwaysValidType{},
			wantError: false,
		},
		{
			name:         "implements HasValidate - invalid",
			value:        validType{Value: ""},
			wantError:    true,
			errorMessage: "value is required",
		},
		{
			name:      "implements HasValidateWithContext - valid",
			value:     contextAlwaysValidType{},
			wantError: false,
		},
		{
			name:         "implements HasValidateWithContext - invalid",
			value:        contextValidType{Value: ""},
			wantError:    true,
			errorMessage: "value is required",
		},
		{
			name:      "implements neither interface",
			value:     nonValidatableType{Value: "test"},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			err := validateInternal(ctx, tt.value)

			if tt.wantError {
				require.Error(t, err)

				if tt.errorMessage != "" {
					assert.Contains(t, err.Error(), tt.errorMessage)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Benchmark tests.
func BenchmarkValidate_HasValidate(b *testing.B) {
	ctx := context.Background()
	value := validType{Value: "test"}

	b.ResetTimer()

	for b.Loop() {
		_ = Validate(ctx, value)
	}
}

func BenchmarkValidate_HasValidateWithContext(b *testing.B) {
	ctx := context.Background()
	value := contextValidType{Value: "test"}

	b.ResetTimer()

	for b.Loop() {
		_ = Validate(ctx, value)
	}
}

func BenchmarkValidate_NoInterface(b *testing.B) {
	ctx := context.Background()
	value := nonValidatableType{Value: "test"}

	b.ResetTimer()

	for b.Loop() {
		_ = Validate(ctx, value)
	}
}

// Example type for documentation.
type exampleConfig struct {
	Port int
}

func (c exampleConfig) Validate() error {
	if c.Port <= 0 {
		return errPortMustBePositive
	}

	return nil
}

type exampleRequest struct {
	UserID string
}

func (r exampleRequest) Validate(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	if r.UserID == "" {
		return errUserIDRequired
	}

	return nil
}

func ExampleValidate() {
	ctx := context.Background()
	cfg := exampleConfig{Port: 8080}

	err := Validate(ctx, cfg)
	if err != nil {
		fmt.Println("validation failed:", err)

		return
	}

	fmt.Println("validation succeeded")
	// Output: validation succeeded
}

func ExampleValidate_withContext() {
	ctx := context.Background()
	req := exampleRequest{UserID: "user-123"}

	err := Validate(ctx, req)
	if err != nil {
		fmt.Println("validation failed:", err)

		return
	}

	fmt.Println("validation succeeded")
	// Output: validation succeeded
}
