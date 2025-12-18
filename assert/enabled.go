//go:build !assertions_disabled

package assert

import (
	"context"
	"fmt"

	"github.com/amp-labs/amp-common/contexts"
)

// True asserts that the given value is true.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func True(value bool, args ...any) {
	if value {
		return
	}

	if len(args) == 0 {
		panic("assertion failed")
	}

	first := args[0]
	remaining := args[1:]

	if firstStr, ok := first.(string); ok {
		panic(fmt.Sprintf(firstStr, remaining...))
	}

	panic(fmt.Sprintf("assertion failed: %v", args))
}

// False asserts that the given value is false.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func False(value bool, args ...any) {
	True(!value, args...)
}

// Nil asserts that the given value is nil.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func Nil(value any, args ...any) {
	True(value == nil, args...)
}

// NotNil asserts that the given value is not nil.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func NotNil(value any, args ...any) {
	True(value != nil, args...)
}

// ContextHasValue asserts that the given context contains a value for the specified key.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func ContextHasValue[Key any, Value any](ctx context.Context, key Key, args ...any) {
	_, found := contexts.GetValue[Key, Value](ctx, key)

	True(found, args...)
}

// ContextDoesNotHaveValue asserts that the given context does not contain a value for the specified key.
// If the assertion fails, it panics with a message.
// The optional args are passed to False and follow the same formatting rules.
func ContextDoesNotHaveValue[Key any, Value any](ctx context.Context, key Key, args ...any) {
	_, found := contexts.GetValue[Key, Value](ctx, key)

	False(found, args...)
}

// ContextIsAlive asserts that the given context is alive (not canceled or done).
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func ContextIsAlive(ctx context.Context, args ...any) {
	True(contexts.IsContextAlive(ctx), args...)
}

// ContextIsDead asserts that the given context is not alive (canceled or done).
// If the assertion fails, it panics with a message.
// The optional args are passed to False and follow the same formatting rules.
func ContextIsDead(ctx context.Context, args ...any) {
	False(contexts.IsContextAlive(ctx), args...)
}

// EmptySlice asserts that the given slice is empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func EmptySlice[T any](slice []T, args ...any) {
	if len(slice) == 0 {
		return
	}

	if len(args) == 0 {
		panic("assertion failed")
	}

	first := args[0]
	remaining := args[1:]

	if firstStr, ok := first.(string); ok {
		panic(fmt.Sprintf(firstStr, remaining...))
	}

	panic(fmt.Sprintf("assertion failed: %v", args))
}

// NonEmptySlice asserts that the given slice is not empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func NonEmptySlice[T any](slice []T, args ...any) {
	if len(slice) > 0 {
		return
	}

	if len(args) == 0 {
		panic("assertion failed")
	}

	first := args[0]
	remaining := args[1:]

	if firstStr, ok := first.(string); ok {
		panic(fmt.Sprintf(firstStr, remaining...))
	}

	panic(fmt.Sprintf("assertion failed: %v", args))
}

// EmptyMap asserts that the given map is empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func EmptyMap[K comparable, V any](m map[K]V, args ...any) {
	if len(m) == 0 {
		return
	}

	if len(args) == 0 {
		panic("assertion failed")
	}

	first := args[0]
	remaining := args[1:]

	if firstStr, ok := first.(string); ok {
		panic(fmt.Sprintf(firstStr, remaining...))
	}

	panic(fmt.Sprintf("assertion failed: %v", args))
}

// NonEmptyMap asserts that the given map is not empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func NonEmptyMap[K comparable, V any](m map[K]V, args ...any) {
	if len(m) > 0 {
		return
	}

	if len(args) == 0 {
		panic("assertion failed")
	}

	first := args[0]
	remaining := args[1:]

	if firstStr, ok := first.(string); ok {
		panic(fmt.Sprintf(firstStr, remaining...))
	}

	panic(fmt.Sprintf("assertion failed: %v", args))
}
