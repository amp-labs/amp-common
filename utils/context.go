package utils

import "context"

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

func AddContextValue[K any, V any](ctx context.Context, key K, value V) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}

	return context.WithValue(ctx, key, value)
}

func ExtractContextValue[K any, V any](ctx context.Context, key K) (V, bool) {
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
