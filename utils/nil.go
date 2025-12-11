package utils //nolint:revive // utils is an appropriate package name for utility functions

import "reflect"

// IsNilish returns true if the value is a literal nil
// or if it points to something with a nil value.
func IsNilish(val any) bool {
	if val == nil {
		return true
	}

	valOf := reflect.ValueOf(val)

	switch valOf.Kind() { //nolint:exhaustive
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer,
		reflect.UnsafePointer, reflect.Interface, reflect.Slice:
		return valOf.IsNil()
	}

	return false
}
