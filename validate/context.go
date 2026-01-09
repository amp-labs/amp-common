package validate

import (
	"context"

	"github.com/amp-labs/amp-common/contexts"
)

// contextKey is a custom type for context keys used within this package.
// Using a custom type instead of a plain string prevents collisions with context keys
// from other packages, following the best practice described in the context package documentation.
// This ensures that our context values don't accidentally conflict with keys used elsewhere.
type contextKey string

const (
	// wantProblemErrorsKey is the context key for storing the problem errors preference.
	// When this flag is set to true via WithWantProblemErrors, validation errors should be
	// formatted as RFC-7807/RFC-9457 Problem Details (using the problem package) instead of plain errors.
	// This allows HTTP handlers to return structured error responses with proper status codes,
	// remediation guidance, and machine-readable error details.
	wantProblemErrorsKey contextKey = "wantProblemErrors"

	// wantWrappedErrorsKey is the context key for storing the wrapped errors preference.
	// When this flag is set to true (the default) via WithWrappedError, validation errors should be
	// wrapped with additional context using fmt.Errorf with %w. When false, errors are returned
	// directly without wrapping, which can be useful for performance-critical paths or when
	// the error context is already sufficient.
	wantWrappedErrorsKey contextKey = "wantWrappedErrors"
)

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

// WithWrappedError returns a new context with the wrapped errors preference set.
// This configuration flag controls whether validation errors should be wrapped with
// additional context information using fmt.Errorf with %w format verb.
//
// When wantWrapped is true (the default if not explicitly set):
//   - Validation errors should be wrapped with contextual information
//   - Error chains provide detailed traces for debugging
//   - Stack traces and error context flow through the call hierarchy
//   - Suitable for most use cases where debuggability is important
//
// When wantWrapped is false:
//   - Validation errors are returned directly without additional wrapping
//   - Reduces allocation overhead in performance-critical paths
//   - Useful when error messages are already descriptive enough
//   - Simplifies error handling when deep error chains aren't needed
//
// This flag is typically set at the service or handler level and propagates through
// the call stack via context, allowing low-level validation code to adapt its error
// handling strategy based on the caller's requirements.
//
// Example:
//
//	func ValidateAtScale(ctx context.Context, items []Item) error {
//	    // Disable wrapping for performance in high-throughput validation
//	    ctx = validate.WithWrappedError(ctx, false)
//	    for _, item := range items {
//	        if err := item.Validate(ctx); err != nil {
//	            return err // No wrapping overhead
//	        }
//	    }
//	    return nil
//	}
func WithWrappedError(ctx context.Context, wantWrapped bool) context.Context {
	return contexts.WithValue[contextKey, bool](ctx, wantWrappedErrorsKey, wantWrapped)
}

// wantWrappedErrors retrieves the wrapped errors preference from the context.
// Returns true if the caller wants validation errors wrapped with additional context,
// false otherwise.
//
// Unlike WantProblemErrors which defaults to false, this function defaults to true,
// meaning errors are wrapped by default unless explicitly disabled via WithWrappedError.
// This default reflects the principle that error context is generally beneficial for
// debugging and should only be disabled when there's a specific reason (e.g., performance).
//
// This is an unexported helper function used internally by the validate package to
// determine the appropriate error handling strategy. Package consumers should use
// WithWrappedError to configure this behavior rather than calling this function directly.
//
// Example usage within the validate package:
//
//	func (v *Validator) validate(ctx context.Context, value any) error {
//	    if err := checkValue(value); err != nil {
//	        if wantWrappedErrors(ctx) {
//	            return fmt.Errorf("validation failed for %T: %w", value, err)
//	        }
//	        return err
//	    }
//	    return nil
//	}
func wantWrappedErrors(ctx context.Context) bool {
	value, found := contexts.GetValue[contextKey, bool](ctx, wantWrappedErrorsKey)
	if found {
		return value
	}

	return true
}
