package validate

import "context"

// Func wraps a validation function into a type that implements the HasValidate interface.
// This is useful when you have validation logic defined as a function and need to pass it
// to code that expects a HasValidate implementation.
//
// The wrapped function will be called when Validate() is invoked on the returned type.
// If the provided function is nil, Validate() will return nil (validation succeeds).
//
// Parameters:
//   - f: The validation function to wrap. Can be nil, in which case validation always succeeds.
//
// Returns:
//   - A HasValidate implementation that delegates to the provided function
//
// Example:
//
//	// Define validation logic as a function
//	validatePort := func() error {
//	    if port < 1 || port > 65535 {
//	        return fmt.Errorf("port %d is out of range", port)
//	    }
//	    return nil
//	}
//
//	// Wrap it to satisfy HasValidate interface
//	validator := validate.Func(validatePort)
//
//	// Now it can be used with validate.Validate
//	if err := validate.Validate(ctx, validator); err != nil {
//	    log.Fatal(err)
//	}
func Func(f func() error) HasValidate {
	return &validateFunc{
		validate: f,
	}
}

// FuncWithContext wraps a context-aware validation function into a type that implements
// the HasValidateWithContext interface. This is useful when you have validation logic
// that requires a context (for cancellation, timeouts, or accessing external resources)
// and need to pass it to code that expects a HasValidateWithContext implementation.
//
// The wrapped function will be called when Validate(ctx) is invoked on the returned type.
// If the provided function is nil, Validate() will return nil (validation succeeds).
//
// Parameters:
//   - f: The context-aware validation function to wrap. Can be nil, in which case validation always succeeds.
//
// Returns:
//   - A HasValidateWithContext implementation that delegates to the provided function
//
// Example:
//
//	// Define validation logic that needs context
//	validateConnection := func(ctx context.Context) error {
//	    select {
//	    case <-ctx.Done():
//	        return ctx.Err()
//	    default:
//	    }
//
//	    conn, err := db.PingContext(ctx)
//	    if err != nil {
//	        return fmt.Errorf("database connection failed: %w", err)
//	    }
//	    conn.Close()
//	    return nil
//	}
//
//	// Wrap it to satisfy HasValidateWithContext interface
//	validator := validate.FuncWithContext(validateConnection)
//
//	// Now it can be used with validate.Validate
//	if err := validate.Validate(ctx, validator); err != nil {
//	    log.Fatal(err)
//	}
func FuncWithContext(f func(ctx context.Context) error) HasValidateWithContext {
	return &validateFuncWithContext{
		validate: f,
	}
}

// validateFunc is an internal type that wraps a validation function to implement HasValidate.
// This type is returned by Func() and should not be used directly.
type validateFunc struct {
	validate func() error
}

// Compile-time assertion that validateFunc implements HasValidate.
var _ HasValidate = (*validateFunc)(nil)

// Validate executes the wrapped validation function and returns its result.
// If the wrapped function is nil, this method returns nil (validation succeeds).
// This implements the HasValidate interface.
func (v *validateFunc) Validate() error {
	if v.validate != nil {
		return v.validate()
	}

	return nil
}

// validateFuncWithContext is an internal type that wraps a context-aware validation function
// to implement HasValidateWithContext. This type is returned by FuncWithContext() and should
// not be used directly.
type validateFuncWithContext struct {
	validate func(ctx context.Context) error
}

// Compile-time assertion that validateFuncWithContext implements HasValidateWithContext.
var _ HasValidateWithContext = (*validateFuncWithContext)(nil)

// Validate executes the wrapped context-aware validation function and returns its result.
// If the wrapped function is nil, this method returns nil (validation succeeds).
// This implements the HasValidateWithContext interface.
func (v *validateFuncWithContext) Validate(ctx context.Context) error {
	if v.validate != nil {
		return v.validate(ctx)
	}

	return nil
}
