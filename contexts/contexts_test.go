package contexts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const testKeyConst = "testKey"

type contextKey string

func TestEnsureContext(t *testing.T) {
	t.Parallel()

	t.Run("returns first non-nil context", func(t *testing.T) {
		t.Parallel()

		ctx1 := t.Context()
		ctx2 := t.Context()

		result := EnsureContext(nil, nil, ctx1, ctx2)
		assert.Equal(t, ctx1, result)
	})

	t.Run("returns background context when all are nil", func(t *testing.T) {
		t.Parallel()

		result := EnsureContext(nil, nil, nil)
		assert.NotNil(t, result)
		assert.Equal(t, context.Background(), result) //nolint:usetesting
	})

	t.Run("handles empty input", func(t *testing.T) {
		t.Parallel()

		result := EnsureContext()
		assert.NotNil(t, result)
		assert.Equal(t, context.Background(), result) //nolint:usetesting
	})

	t.Run("returns single non-nil context", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey("key"), "value")
		result := EnsureContext(ctx)
		assert.Equal(t, ctx, result)
	})
}

func TestIsContextAlive(t *testing.T) {
	t.Parallel()

	t.Run("returns false for nil context", func(t *testing.T) {
		t.Parallel()
		// Note: Testing with nil context directly to verify nil handling
		assert.False(t, IsContextAlive(nil)) //nolint:staticcheck // Testing nil context behavior
	})

	t.Run("returns true for active context", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		assert.True(t, IsContextAlive(ctx))
	})

	t.Run("returns false for cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		assert.False(t, IsContextAlive(ctx))
	})

	t.Run("returns false for expired context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
		defer cancel()

		time.Sleep(10 * time.Millisecond)
		assert.False(t, IsContextAlive(ctx))
	})

	t.Run("returns true for context with future deadline", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Hour)
		defer cancel()

		assert.True(t, IsContextAlive(ctx))
	})
}

func TestWithValue(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves value with string key", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		key := testKeyConst
		value := "testValue"

		ctx = WithValue(ctx, key, value)
		assert.Equal(t, value, ctx.Value(key))
	})

	t.Run("creates background context when nil", func(t *testing.T) {
		t.Parallel()

		key := testKeyConst
		value := 42
		// Note: Testing with nil context directly to verify nil handling
		ctx := WithValue[string, int](nil, key, value) //nolint:staticcheck // Testing nil context behavior
		assert.NotNil(t, ctx)
		assert.Equal(t, value, ctx.Value(key))
	})

	t.Run("supports custom key types", func(t *testing.T) {
		t.Parallel()

		type customKey struct{ id int }

		key := customKey{id: 123}
		value := "customValue"

		ctx := WithValue(t.Context(), key, value)
		assert.Equal(t, value, ctx.Value(key))
	})

	t.Run("supports different value types", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		// Integer
		ctx = WithValue(ctx, "int", 42)
		assert.Equal(t, 42, ctx.Value("int"))

		// Struct
		type testStruct struct{ Name string }

		s := testStruct{Name: "test"}
		ctx = WithValue(ctx, "struct", s)
		assert.Equal(t, s, ctx.Value("struct"))

		// Pointer
		ptr := &testStruct{Name: "pointer"}
		ctx = WithValue(ctx, "pointer", ptr)
		assert.Equal(t, ptr, ctx.Value("pointer"))
	})
}

func TestGetValue(t *testing.T) {
	t.Parallel()

	t.Run("retrieves existing value with correct type", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		key := contextKey(testKeyConst)
		expectedValue := "testValue"

		ctx = context.WithValue(ctx, key, expectedValue)
		value, ok := GetValue[contextKey, string](ctx, key)

		assert.True(t, ok)
		assert.Equal(t, expectedValue, value)
	})

	t.Run("returns false for nil context", func(t *testing.T) {
		t.Parallel()
		// Note: Testing with nil context directly to verify nil handling
		value, ok := GetValue[string, string](nil, "key") //nolint:staticcheck // Testing nil context behavior

		assert.False(t, ok)
		assert.Equal(t, "", value)
	})

	t.Run("returns false for missing key", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		value, ok := GetValue[string, string](ctx, "nonexistent")

		assert.False(t, ok)
		assert.Equal(t, "", value)
	})

	t.Run("returns false for type mismatch", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey("key"), "stringValue")
		value, ok := GetValue[contextKey, int](ctx, contextKey("key"))

		assert.False(t, ok)
		assert.Equal(t, 0, value)
	})

	t.Run("handles integer values", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(t.Context(), contextKey("count"), 42)
		value, ok := GetValue[contextKey, int](ctx, contextKey("count"))

		assert.True(t, ok)
		assert.Equal(t, 42, value)
	})

	t.Run("handles struct values", func(t *testing.T) {
		t.Parallel()

		type user struct {
			Name string
			Age  int
		}

		expectedUser := user{Name: "Alice", Age: 30}
		ctx := context.WithValue(t.Context(), contextKey("user"), expectedUser)
		value, ok := GetValue[contextKey, user](ctx, contextKey("user"))

		assert.True(t, ok)
		assert.Equal(t, expectedUser, value)
	})

	t.Run("handles pointer values", func(t *testing.T) {
		t.Parallel()

		type data struct{ Value int }

		expected := &data{Value: 123}

		ctx := context.WithValue(t.Context(), contextKey("data"), expected)
		value, ok := GetValue[contextKey, *data](ctx, contextKey("data"))

		assert.True(t, ok)
		assert.Equal(t, expected, value)
	})

	t.Run("handles custom key types", func(t *testing.T) {
		t.Parallel()

		type contextKey string

		key := contextKey("myKey")
		expectedValue := "myValue"

		ctx := context.WithValue(t.Context(), key, expectedValue)
		value, ok := GetValue[contextKey, string](ctx, key)

		assert.True(t, ok)
		assert.Equal(t, expectedValue, value)
	})

	t.Run("returns zero value on failure", func(t *testing.T) {
		t.Parallel()

		type customStruct struct {
			Field string
		}

		ctx := t.Context()
		value, ok := GetValue[string, customStruct](ctx, "missing")

		assert.False(t, ok)
		assert.Equal(t, customStruct{}, value)
	})
}

func TestWithValueAndGetValueIntegration(t *testing.T) {
	t.Parallel()

	t.Run("round-trip with type safety", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		key := testKeyConst
		expectedValue := 42

		ctx = WithValue(ctx, key, expectedValue)
		value, ok := GetValue[string, int](ctx, key)

		assert.True(t, ok)
		assert.Equal(t, expectedValue, value)
	})

	t.Run("multiple values in same context", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		ctx = WithValue(ctx, "key1", "value1")
		ctx = WithValue(ctx, "key2", 123)
		ctx = WithValue(ctx, "key3", true)

		val1, ok1 := GetValue[string, string](ctx, "key1")
		val2, ok2 := GetValue[string, int](ctx, "key2")
		val3, ok3 := GetValue[string, bool](ctx, "key3")

		assert.True(t, ok1)
		assert.Equal(t, "value1", val1)
		assert.True(t, ok2)
		assert.Equal(t, 123, val2)
		assert.True(t, ok3)
		assert.True(t, val3)
	})
}
