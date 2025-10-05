package contexts

import "context"

// EnsureContext will choose the first non-nil context passed in. If all values
// are nil, a new context will be created.
func EnsureContext(ctx ...context.Context) context.Context {
	for _, c := range ctx {
		if c != nil {
			return c
		}
	}

	return context.Background()
}

// IsContextAlive returns true if the context is not done.
func IsContextAlive(ctx context.Context) bool {
	if ctx == nil {
		return false
	}

	// This is non-blocking, so it will return immediately.
	select {
	case <-ctx.Done():
		return false
	default:
		return true
	}
}

// WithValue is a type-safe wrapper around context.WithValue that stores a value
// of type V with a key of type K. If ctx is nil, a new background context is created.
// This provides compile-time type safety for context values.
func WithValue[K any, V any](ctx context.Context, key K, value V) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, key, value)
}

// GetValue is a type-safe wrapper around context.Value that retrieves a value of type V
// using a key of type K. Returns the value and true if found and type matches, or the
// zero value of V and false otherwise. Returns false if ctx is nil.
func GetValue[K any, V any](ctx context.Context, key K) (V, bool) {
	var zero V

	if ctx == nil {
		return zero, false
	}

	val := ctx.Value(key)
	if val == nil {
		return zero, false
	}

	v, ok := val.(V)
	if !ok {
		return zero, false
	}

	return v, true
}
