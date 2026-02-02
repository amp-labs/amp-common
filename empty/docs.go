// Package empty provides utilities for creating empty values of various types.
//
// This package is useful for scenarios where you need explicit empty/zero values,
// particularly in testing, placeholder values, or when working with APIs that
// distinguish between nil and empty collections.
//
// Example usage:
//
//	// Create empty collections
//	emptySlice := empty.Slice[string]()
//	emptyMap := empty.Map[string, int]()
//
//	// Use empty struct for signaling
//	done := make(chan empty.T)
//	done <- empty.V
//
//	// Work with pointers to empty collections
//	slicePtr := empty.SlictPtr[int]()
//	mapPtr := empty.MapPtr[string, bool]()
package empty
