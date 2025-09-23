package utils

import (
	"reflect"
	"runtime"
)

// GetFunctionName returns the name of the function passed as an argument.
// If the argument is nil, it returns "<nil>". If the argument is not a function,
// it will return "<not a function>".
func GetFunctionName(f any) string {
	if IsNilish(f) {
		return "<nil>"
	}

	funcPtr := runtime.FuncForPC(reflect.ValueOf(f).Pointer())

	if funcPtr == nil {
		return "<not a function>"
	}

	return funcPtr.Name()
}
