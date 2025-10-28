// Package zero provides utilities for working with zero values of generic types.
package zero

import "reflect"

// Value returns the zero value for type T.
// This is useful when you need to explicitly obtain the zero value of a generic type parameter.
//
// Example:
//
//	var defaultInt = zero.Value[int]()        // returns 0
//	var defaultStr = zero.Value[string]()     // returns ""
//	var defaultPtr = zero.Value[*MyStruct]()  // returns nil
func Value[T any]() T {
	var zeroVal T

	return zeroVal
}

// IsZero reports whether value is the zero value for type T.
// It uses reflect.DeepEqual to perform a deep comparison between value and the zero value of T.
//
// Example:
//
//	zero.IsZero(0)              // returns true
//	zero.IsZero(42)             // returns false
//	zero.IsZero("")             // returns true
//	zero.IsZero("hello")        // returns false
//	zero.IsZero[*MyStruct](nil) // returns true
func IsZero[T any](value T) bool {
	var zeroVal T

	return reflect.DeepEqual(value, zeroVal)
}
