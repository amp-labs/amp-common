//nolint:ireturn
package envutil

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

var (
	// ErrBadEnvVar is returned when an environment variable cannot be parsed.
	ErrBadEnvVar = errors.New("error parsing environment variable")
	// ErrEnvVarMissing is returned when a required environment variable is not set.
	ErrEnvVarMissing = errors.New("missing environment variable")
)

// Reader represents a parsed environment variable value with error handling.
// It provides a fluent API for working with environment variables, including
// type conversions, defaults, validation, and graceful error handling.
type Reader[A any] struct {
	key     string
	present bool
	err     error

	value A
}

// Key returns the environment variable name.
func (e Reader[A]) Key() string {
	return e.key
}

// Value returns the parsed value or an error if missing or invalid.
func (e Reader[A]) Value() (A, error) { //nolint:ireturn
	if e.err != nil {
		return e.value, fmt.Errorf("%w %s: %w (given value is %v)", ErrBadEnvVar, e.key, e.err, e.value)
	}

	if !e.present {
		return e.value, fmt.Errorf("%w %s", ErrEnvVarMissing, e.key)
	}

	return e.value, e.err
}

// ValueOrPanic returns the value or panics if missing or invalid.
func (e Reader[A]) ValueOrPanic() A { //nolint:ireturn
	value, err := e.Value()
	if err != nil {
		panic(err)
	}

	return value
}

// ValueOrFatal returns the value or exits the program (os.Exit(1)) if missing or invalid.
func (e Reader[A]) ValueOrFatal() A { //nolint:ireturn
	value, err := e.Value()
	if err != nil {
		slog.Error("error reading environment variable", "key", e.key, "error", err)
		os.Exit(1)
	}

	return value
}

// ValueOrElseFunc returns the value or calls f() if missing or invalid.
// Useful when the fallback value is expensive to compute.
func (e Reader[A]) ValueOrElseFunc(f func() A) A { //nolint:ireturn
	if e.present && e.err == nil {
		return e.value
	}

	return f()
}

// ValueOrElseFuncErr returns the value or calls f() if missing or invalid.
// Like ValueOrElseFunc but allows the fallback function to return an error.
func (e Reader[A]) ValueOrElseFuncErr(f func() (A, error)) (A, error) { //nolint:ireturn
	if e.present && e.err == nil {
		return e.value, nil
	}

	return f()
}

// ValueOrElse returns the value or a default if missing or invalid.
// Logs a warning if there was a parsing error.
func (e Reader[A]) ValueOrElse(v A) A { //nolint:ireturn
	if e.present && e.err == nil {
		return e.value
	}

	if e.err != nil {
		slog.Warn("error reading environment variable, using fallback value",
			"key", e.key, "value", e.value, "error", e.err, "fallback", v)
	}

	return v
}

// DoWithValue calls f with the value if present and valid, otherwise does nothing.
func (e Reader[A]) DoWithValue(f func(A)) {
	if e.present && e.err == nil {
		f(e.value)
	}
}

// HasValue returns true if the variable is present and valid.
func (e Reader[A]) HasValue() bool {
	return e.present && e.err == nil
}

// HasError returns true if a parsing error occurred.
func (e Reader[A]) HasError() bool {
	return e.err != nil
}

// String returns a string representation of the Reader for debugging.
func (e Reader[A]) String() string {
	if e.present && e.err == nil {
		return fmt.Sprintf("%s=%v", e.key, e.value)
	}

	if e.err != nil {
		return fmt.Sprintf("%s=<error: %v>", e.key, e.err)
	}

	return e.key + "=<not set>"
}

// Error returns the parsing error, if any.
func (e Reader[A]) Error() error {
	return e.err
}

// WithErrorIfMissing returns a new Reader with err if the value is missing.
// If the value is present or already has an error, returns the Reader unchanged.
func (e Reader[A]) WithErrorIfMissing(err error) Reader[A] { //nolint:ireturn
	if e.present || e.err != nil {
		return e
	}

	return Reader[A]{
		key:     e.key,
		present: false,
		err:     err,
	}
}

// WithDefault returns a new Reader with v as the value if missing.
// If the value is present, returns the Reader unchanged.
func (e Reader[A]) WithDefault(v A) Reader[A] { //nolint:ireturn
	if e.present {
		return e
	}

	return Reader[A]{
		key:     e.key,
		present: true,
		err:     e.err,
		value:   v,
	}
}

// WithFallback returns v if this Reader has no value, otherwise returns this Reader.
func (e Reader[A]) WithFallback(v Reader[A]) Reader[A] { //nolint:ireturn
	if e.present {
		return e
	}

	return v
}

// Map transforms the value using f, preserving the same type.
// Convenience wrapper around the package-level Map function.
func (e Reader[A]) Map(f func(A) (A, error)) Reader[A] { //nolint:ireturn
	return Map(e, f)
}

// Map transforms a Reader's value from type A to type B using function f.
// Returns a new Reader with the transformed value, preserving errors and missing state.
func Map[A any, B any](env Reader[A], f func(A) (B, error)) Reader[B] {
	if !env.present || env.err != nil {
		return Reader[B]{
			key:     env.key,
			present: env.present,
			err:     env.err,
		}
	}

	val, err := f(env.value)
	// Special logic for unsetting a value.
	if err != nil {
		if errors.Is(err, errUnsetValue) {
			return Reader[B]{
				key: env.key,
			}
		}
	}

	return Reader[B]{
		present: true,
		key:     env.key,
		err:     err,
		value:   val,
	}
}
