//go:build assertions_disabled

package assert

// True asserts that the given value is true.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func True(value bool, args ...any) {
	// Intentionally left blanks
}

// False asserts that the given value is false.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func False(value bool, args ...any) {
	// Intentionally left blanks
}

// Nil asserts that the given value is nil.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func Nil(value any, args ...any) {
	// Intentionally left blanks
}

// NotNil asserts that the given value is not nil.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func NotNil(value any, args ...any) {
	// Intentionally left blanks
}

// ContextHasValue asserts that the given context contains a value for the specified key.
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func ContextHasValue[Key any, Value any](ctx context.Context, key Key, args ...any) {
	// Intentionally left blanks
}

// ContextDoesNotHaveValue asserts that the given context does not contain a value for the specified key.
// If the assertion fails, it panics with a message.
// The optional args are passed to False and follow the same formatting rules.
func ContextDoesNotHaveValue[Key any, Value any](ctx context.Context, key Key, args ...any) {
	// Intentionally left blanks
}

// ContextIsAlive asserts that the given context is alive (not cancelled or done).
// If the assertion fails, it panics with a message.
// The optional args are passed to True and follow the same formatting rules.
func ContextIsAlive(ctx context.Context, args ...any) {
	// Intentionally left blanks
}

// ContextIsDead asserts that the given context is not alive (cancelled or done).
// If the assertion fails, it panics with a message.
// The optional args are passed to False and follow the same formatting rules.
func ContextIsDead(ctx context.Context, args ...any) {
	// Intentionally left blanks
}

// EmptySlice asserts that the given slice is empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func EmptySlice[T any](slice []T, args ...any) {
	// Intentionally left blanks
}

// NonEmptySlice asserts that the given slice is not empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func NonEmptySlice[T any](slice []T, args ...any) {
	// Intentionally left blanks
}

// EmptyMap asserts that the given map is empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func EmptyMap[K comparable, V any](m map[K]V, args ...any) {
	// Intentionally left blanks
}

// NonEmptyMap asserts that the given map is not empty.
// If the assertion fails, it panics with a message.
// The optional args can be used to provide a formatted panic message:
// - If the first arg is a string, it's used as a format string with remaining args.
// - Otherwise, all args are included in the panic message.
func NonEmptyMap[K comparable, V any](m map[K]V, args ...any) {
	// Intentionally left blanks
}
