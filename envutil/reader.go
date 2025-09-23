//nolint:ireturn
package envutil

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
)

var (
	ErrBadEnvVar     = errors.New("error parsing environment variable")
	ErrEnvVarMissing = errors.New("missing environment variable")
)

// Reader is a type that represents a value read from an environment variable.
// It is used to provide a more ergonomic way to handle environment variables.
// It is a wrapper around the value, and it provides a way to handle errors and
// missing values, as well as transformations.
type Reader[A any] struct {
	key     string
	present bool
	err     error

	value A
}

// Key returns the key of the environment variable.
func (e Reader[A]) Key() string {
	return e.key
}

// Value returns the value of the environment variable, or an error if the value
// is missing or if there was an error parsing it.
func (e Reader[A]) Value() (A, error) { //nolint:ireturn
	if e.err != nil {
		return e.value, fmt.Errorf("%w %s: %w (given value is %v)", ErrBadEnvVar, e.key, e.err, e.value)
	}

	if !e.present {
		return e.value, fmt.Errorf("%w %s", ErrEnvVarMissing, e.key)
	}

	return e.value, e.err
}

// ValueOrPanic returns the value of the environment variable, or panics if the
// value is missing or if there was an error parsing it.
func (e Reader[A]) ValueOrPanic() A { //nolint:ireturn
	value, err := e.Value()
	if err != nil {
		panic(err)
	}

	return value
}

// ValueOrFatal returns the value of the environment variable, or exits the
// program if the value is missing or if there was an error parsing it.
func (e Reader[A]) ValueOrFatal() A { //nolint:ireturn
	value, err := e.Value()
	if err != nil {
		slog.Error("error reading environment variable", "key", e.key, "error", err)
		os.Exit(1)
	}

	return value
}

// ValueOrElseFunc returns the value of the environment variable, or the result
// of the given function if the value is missing or if there was an error parsing it.
// It's useful if the fallback value is expensive to compute.
func (e Reader[A]) ValueOrElseFunc(f func() A) A { //nolint:ireturn
	if e.present && e.err == nil {
		return e.value
	}

	return f()
}

// ValueOrElseFuncErr returns the value of the environment variable, or the result
// of the given function if the value is missing or if there was an error parsing it.
// It's useful if the fallback value is expensive to compute and may return an error.
func (e Reader[A]) ValueOrElseFuncErr(f func() (A, error)) (A, error) { //nolint:ireturn
	if e.present && e.err == nil {
		return e.value, nil
	}

	return f()
}

// ValueOrElse returns the value of the environment variable, or a default value
// if the value is missing or if there was an error parsing it.
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

// DoWithValue calls the given function with the value of the environment variable
// if the value is present and there was no error reading it.
func (e Reader[A]) DoWithValue(f func(A)) {
	if e.present && e.err == nil {
		f(e.value)
	}
}

// HasValue returns true if the environment variable was set, and false otherwise.
func (e Reader[A]) HasValue() bool {
	return e.present && e.err == nil
}

// HasError returns true if an error occurred when reading the environment variable.
func (e Reader[A]) HasError() bool {
	return e.err != nil
}

// String returns a string representation of the Reader.
func (e Reader[A]) String() string {
	if e.present && e.err == nil {
		return fmt.Sprintf("%s=%v", e.key, e.value)
	}

	if e.err != nil {
		return fmt.Sprintf("%s=<error: %v>", e.key, e.err)
	}

	return e.key + "=<not set>"
}

// Error returns the error that occurred when reading the environment variable, if any.
func (e Reader[A]) Error() error {
	return e.err
}

// WithErrorIfMissing returns a new Reader with the given error if the original
// Reader has no value. If the original Reader has a value, it is returned as is.
// Also if the original reader already has an error, that error is returned as is.
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

// WithDefault returns a new Reader with the given default value if the original
// Reader has no value. If the original Reader has a value, it is returned as is.
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

// WithFallback returns a new Reader with the given fallback Reader if the original
// Reader has no value. If the original Reader has a value, it is returned as is.
func (e Reader[A]) WithFallback(v Reader[A]) Reader[A] { //nolint:ireturn
	if e.present {
		return e
	}

	return v
}

// Map returns a new Reader with the value transformed by the given function.
// Less flexible than Map (type is restricted), but slightly more convenient.
func (e Reader[A]) Map(f func(A) (A, error)) Reader[A] { //nolint:ireturn
	return Map(e, f)
}

// Map returns a new Reader with the value transformed by the given function.
// This can translate types, so it is more flexible than Reader.Map.
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
