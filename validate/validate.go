// Package validate provides a unified validation framework for types that implement validation interfaces.
// It supports both context-aware and context-free validation patterns, allowing callers to validate
// arbitrary values in a type-safe manner without knowing their specific validation requirements.
package validate

import (
	"context"
	"fmt"

	"github.com/amp-labs/amp-common/contexts"
	"github.com/amp-labs/amp-common/errors"
	"github.com/amp-labs/amp-common/logger"
	"github.com/amp-labs/amp-common/utils"
)

// contextKey is a custom type for context keys used within this package.
// Using a custom type instead of a plain string prevents collisions with context keys
// from other packages, following the best practice described in the context package documentation.
// This ensures that our context values don't accidentally conflict with keys used elsewhere.
type contextKey string

// wantProblemErrorsKey is the context key for storing the problem errors preference.
// When this flag is set to true via WithWantProblemErrors, validation errors should be
// formatted as RFC-7807/RFC-9457 Problem Details (using the problem package) instead of plain errors.
// This allows HTTP handlers to return structured error responses with proper status codes,
// remediation guidance, and machine-readable error details.
const wantProblemErrorsKey contextKey = "wantProblemErrors"

// WithWantProblemErrors returns a new context with the problem errors preference set.
// This configuration flag controls whether validation errors should be formatted as
// RFC-7807/RFC-9457 Problem Details for HTTP responses.
//
// When wantProblemErrors is true:
//   - Validation errors should be wrapped with the problem package
//   - HTTP handlers can return structured JSON responses with status codes, details, and remediation
//   - Error responses follow the RFC-7807/RFC-9457 standard format
//
// When wantProblemErrors is false (default):
//   - Validation errors are returned as plain Go errors
//   - Suitable for non-HTTP contexts or when simple error messages are preferred
//
// This flag is typically set at the HTTP handler level and flows down through the call stack
// via context propagation, allowing validation logic to remain agnostic of the transport layer
// while still producing appropriate error formats for the caller.
//
// Example:
//
//	func HandleCreateUser(c *fiber.Ctx) error {
//	    ctx := validate.WithWantProblemErrors(c.Context(), true)
//	    if err := validate.Validate(ctx, req); err != nil {
//	        // err can be formatted as a problem.Problem for HTTP response
//	        return problem.FromError(err)
//	    }
//	    // ... handle request
//	}
func WithWantProblemErrors(ctx context.Context, wantProblemErrors bool) context.Context {
	return contexts.WithValue[contextKey, bool](ctx, wantProblemErrorsKey, wantProblemErrors)
}

// WantProblemErrors retrieves the problem errors preference from the context.
// Returns true if the caller wants validation errors formatted as RFC-7807/RFC-9457 Problem Details,
// false otherwise.
//
// If the preference has not been explicitly set via WithWantProblemErrors, this function
// returns false (the default), indicating that plain Go errors should be used.
//
// This function is typically called from within a type's Validate() implementation to determine
// the appropriate error format for the current execution context.
//
// Example:
//
//	type CreateUserRequest struct {
//	    Email string
//	}
//
//	func (r CreateUserRequest) Validate(ctx context.Context) error {
//	    if r.Email == "" {
//	        if validate.WantProblemErrors(ctx) {
//	            // Format as RFC-7807/RFC-9457 problem detail
//	            return problem.BadRequest(ctx, problem.Detail("email is required"))
//	        }
//	        // Return plain error
//	        return fmt.Errorf("email is required")
//	    }
//	    return nil
//	}
func WantProblemErrors(ctx context.Context) bool {
	value, found := contexts.GetValue[contextKey, bool](ctx, wantProblemErrorsKey)
	if found {
		return value
	}

	return false
}

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
		return fmt.Errorf("%w: %w", errors.ErrValidation, err)
	}

	return nil
}

// validateInternal performs the actual validation logic by type-asserting the value
// against the validation interfaces. This is separated from Validate to avoid wrapping
// errors multiple times if validation is called recursively.
//
// The function handles three cases:
//  1. Nil or nil-like values (nil pointers, nil interfaces, etc.) - returns nil (validation passes)
//  2. Values implementing HasValidateWithContext - calls context-aware validation
//  3. Values implementing HasValidate - calls context-free validation
//  4. Values implementing neither interface - returns nil (validation passes)
//
// Note: If a value implements both interfaces, HasValidate takes precedence over HasValidateWithContext
// due to Go's type switch evaluation order.
func validateInternal(ctx context.Context, value any) error {
	if utils.IsNilish(value) {
		return nil
	}

	switch v := value.(type) {
	case HasValidate:
		return v.Validate()
	case HasValidateWithContext:
		return v.Validate(ctx)
	default:
		logger.Get(ctx).Warn("Validate called on unsupported type",
			"type", fmt.Sprintf("%T", v))

		return nil
	}
}
