package validate

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/amp-labs/amp-common/contexts"
	commonErrors "github.com/amp-labs/amp-common/errors"
	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/utils"
)

// HasValidate defines the interface for types that can validate themselves without requiring a context.
// Types implementing this interface should return an error if validation fails, or nil if the value is valid.
// This is the simpler validation interface for types that don't need contextual information during validation.
type HasValidate interface {
	// Validate checks the validity of the implementing type and returns an error if validation fails.
	// This method should be idempotent and safe to call multiple times.
	Validate() error
}

// HasValidateWithContext defines the interface for types that require a context during validation.
// This interface is useful when validation needs to access external resources, respect cancellation,
// or requires deadline/timeout handling. Types implementing this interface should return an error
// if validation fails, or nil if the value is valid.
type HasValidateWithContext interface {
	// Validate checks the validity of the implementing type using the provided context and returns an error
	// if validation fails. The context can be used for cancellation, timeout handling, or passing
	// request-scoped values. This method should respect context cancellation and return promptly if
	// ctx.Done() is signaled.
	Validate(ctx context.Context) error
}

// Validate performs validation on a value by checking if it implements either HasValidate or HasValidateWithContext.
// If the value implements HasValidateWithContext, it calls Validate(ctx) with the provided context.
// If the value implements HasValidate, it calls Validate() without context.
// If the value implements neither interface or is nil, validation succeeds.
//
// The function automatically wraps any validation errors with errors.ErrValidation, making it easy to
// identify validation failures using errors.Is(err, errors.ErrValidation).
//
// Parameters:
//   - ctx: The context to pass to context-aware validators. Used for cancellation, deadlines, and carrying values.
//   - value: The value to validate. Can be any type including nil.
//
// Returns:
//   - nil if validation succeeds or the value doesn't implement a validation interface
//   - An error wrapped with errors.ErrValidation if validation fails
//
// Example:
//
//	type Config struct {
//	    Port int
//	}
//
//	func (c Config) Validate() error {
//	    if c.Port <= 0 {
//	        return fmt.Errorf("port must be positive")
//	    }
//	    return nil
//	}
//
//	cfg := Config{Port: -1}
//	if err := validate.Validate(ctx, cfg); err != nil {
//	    // err will be wrapped with errors.ErrValidation
//	    log.Fatal(err)
//	}
func Validate(ctx context.Context, value any) error {
	//nolint:contextcheck // EnsureContext preserves context inheritance
	err := validateInternal(contexts.EnsureContext(ctx), value)
	if err != nil {
		if !wantWrappedErrors(ctx) {
			return err
		}

		return fmt.Errorf("%w for %T: %w", commonErrors.ErrValidation, value, err)
	}

	return nil
}

// validateInternal performs the actual validation logic by type-asserting the value
// against the validation interfaces. This is separated from Validate to avoid wrapping
// errors multiple times if validation is called recursively.
//
// The function handles three cases:
//  1. Values implementing HasValidateWithContext - calls context-aware validation
//  2. Values implementing HasValidate - calls context-free validation
//  3. Values implementing neither interface - logs a warning and returns nil
//
// Panic recovery: This function includes panic recovery logic to ensure that panics
// within validation methods don't crash the application. If a Validate() method panics:
//   - The panic is caught and converted to an error using utils.GetPanicRecoveryError
//   - The error includes the panic value and stack trace for debugging
//   - If validation already returned an error, the panic error is joined with it
//   - This makes validation failures safe even when validation logic has bugs
//
// Metrics: The function records Prometheus metrics for each validation attempt:
//   - validationsTotal: Counter tracking validation calls by type capability and success
//   - validationTime: Histogram tracking validation duration by type and success
//
// Note: Only validations for types implementing validation interfaces are timed.
// Types that don't implement validation interfaces are counted but not timed.
func validateInternal(ctx context.Context, value any) (errOut error) {
	defer func() {
		if err := recover(); err != nil {
			panicErr := utils.GetPanicRecoveryError(err, debug.Stack())

			if errOut == nil {
				errOut = panicErr
			} else {
				errOut = errors.Join(errOut, panicErr)
			}
		}
	}()

	typeName := fmt.Sprintf("%T", value)

	switch val := value.(type) {
	case HasValidate:
		start := time.Now()
		err := val.Validate()
		end := time.Now()

		if err != nil {
			validationsTotal.WithLabelValues("true", "true").Inc()

			validationTime.WithLabelValues(typeName, "true").
				Observe(float64(end.Sub(start).Milliseconds()))

			return err
		}

		validationsTotal.WithLabelValues("true", "false").Inc()

		validationTime.WithLabelValues(typeName, "false").
			Observe(float64(end.Sub(start).Milliseconds()))
	case HasValidateWithContext:
		start := time.Now()
		err := val.Validate(ctx)
		end := time.Now()

		if err != nil {
			validationsTotal.WithLabelValues("true", "true").Inc()

			validationTime.WithLabelValues(typeName, "true").
				Observe(float64(end.Sub(start).Milliseconds()))

			return err
		}

		validationsTotal.WithLabelValues("true", "false").Inc()

		validationTime.WithLabelValues(typeName, "false").
			Observe(float64(end.Sub(start).Milliseconds()))
	default:
		validationsTotal.WithLabelValues("false", "false").Inc()

		logger.Get(ctx).Warn("Validate called on unsupported type",
			"type", typeName)
	}

	return nil
}
