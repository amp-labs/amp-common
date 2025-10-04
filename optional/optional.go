// Package optional provides a type-safe Optional type for representing values that may or may not be present.
// It is designed to avoid nil-related panics by explicitly modeling the presence or absence of a value.
// An Optional is conceptually a set of size zero or one.
package optional

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
)

var errMissingValueField = errors.New("optional: missing 'value' field in JSON")

// Value represents a value that may or may not be present.
// It is a generic type that can hold any value of type T.
// Use Some(value) to create a Value with a value, or None() for an empty Value.
type Value[T any] struct {
	value T
	isSet bool
}

// Some creates a Value containing the given value.
func Some[T any](value T) Value[T] {
	return Value[T]{value: value, isSet: true}
}

// None creates an empty Value with no value.
func None[T any]() Value[T] {
	return Value[T]{isSet: false}
}

// All returns an iterator that yields the value if present, or yields nothing if empty.
// This allows using Value in Go's range loops.
func (o Value[T]) All() iter.Seq[T] {
	return func(yield func(T) bool) {
		if o.isSet {
			yield(o.value)
		}
	}
}

// ForEach applies the given function to the value if present.
// Does nothing if the Value is empty.
func (o Value[T]) ForEach(f func(T)) {
	for v := range o.All() {
		f(v)
	}
}

// NonEmpty returns true if the Value contains a value.
func (o Value[T]) NonEmpty() bool {
	return o.isSet
}

// Empty returns true if the Value does not contain a value.
func (o Value[T]) Empty() bool {
	return !o.isSet
}

// Get returns the value and a boolean indicating whether the value is present.
// This is the safe way to extract a value from a Value.
func (o Value[T]) Get() (T, bool) {
	return o.value, o.isSet
}

// GetOrPanic returns the value if present, or panics if empty.
// Use this only when you are certain the Value contains a value.
func (o Value[T]) GetOrPanic() T {
	if !o.isSet {
		panic("called GetOrPanic on None")
	}

	return o.value
}

// GetOrElse returns the value if present, or the provided default value if empty.
func (o Value[T]) GetOrElse(defaultValue T) T {
	if o.isSet {
		return o.value
	}

	return defaultValue
}

// GetOrElseFunc returns the value if present, or calls the provided function to get a default value if empty.
// This is useful when computing the default value is expensive.
func (o Value[T]) GetOrElseFunc(defaultFunc func() T) T {
	if o.isSet {
		return o.value
	}

	return defaultFunc()
}

// OrElse returns this Value if it contains a value, or the alternative Value if empty.
func (o Value[T]) OrElse(alternative Value[T]) Value[T] {
	if o.isSet {
		return o
	} else {
		return alternative
	}
}

// OrElseFunc returns this Value if it contains a value, or calls the provided function
// to get an alternative Value if empty. This is useful when computing the alternative is expensive.
func (o Value[T]) OrElseFunc(alternativeFunc func() Value[T]) Value[T] {
	if o.isSet {
		return o
	} else {
		return alternativeFunc()
	}
}

// Equals compares this Value with another using the provided equality function.
// Two Values are equal if both are empty, or both contain values that are equal according to the provided function.
func (o Value[T]) Equals(other Value[T], eq func(T, T) bool) bool {
	if o.isSet != other.isSet {
		return false
	}

	if !o.isSet && !other.isSet {
		return true
	}

	return eq(o.value, other.value)
}

// Filter returns this Value if it contains a value that satisfies the predicate, or None otherwise.
func (o Value[T]) Filter(predicate func(T) bool) Value[T] {
	if o.isSet && predicate(o.value) {
		return o
	}

	return None[T]()
}

// Size returns the size of this Value as a set: 1 if it contains a value, 0 if empty.
func (o Value[T]) Size() int {
	if o.isSet {
		return 1
	}

	return 0
}

// String returns a string representation of the Value.
// Returns "Some(value)" if present, or "None" if empty.
func (o Value[T]) String() string {
	if o.isSet {
		return fmt.Sprintf("Some(%v)", o.value)
	} else {
		return "None"
	}
}

// Map transforms the value inside the Value using the provided function.
// Returns Some(f(value)) if the Value contains a value, or None if empty.
func Map[T any, U any](o Value[T], f func(T) U) Value[U] {
	if o.isSet {
		return Some(f(o.value))
	} else {
		return None[U]()
	}
}

// FlatMap transforms the value inside the Value using the provided function that returns a Value.
// This is useful for chaining Value-returning operations without nesting.
// Returns f(value) if the Value contains a value, or None if empty.
func FlatMap[T any, U any](o Value[T], f func(T) Value[U]) Value[U] {
	if o.isSet {
		return f(o.value)
	} else {
		return None[U]()
	}
}

// MarshalJSON implements json.Marshaler.
// None is marshaled as null, Some(value) is marshaled as {"value": ...}.
func (o Value[T]) MarshalJSON() ([]byte, error) {
	if !o.isSet {
		return []byte("null"), nil
	}

	return json.Marshal(map[string]T{"value": o.value})
}

// UnmarshalJSON implements json.Unmarshaler.
// null is unmarshaled as None, {"value": ...} is unmarshaled as Some(value).
func (o *Value[T]) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		o.isSet = false

		var zero T
		o.value = zero

		return nil
	}

	var wrapper map[string]T
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return err
	}

	value, ok := wrapper["value"]
	if !ok {
		return errMissingValueField
	}

	o.value = value
	o.isSet = true

	return nil
}
