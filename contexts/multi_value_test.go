package contexts

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// testStringer is a test type that implements fmt.Stringer.
type testStringer struct {
	value string
}

func (ts testStringer) String() string {
	return ts.value
}

func TestWithMultipleValues(t *testing.T) {
	t.Parallel()

	t.Run("stores and retrieves multiple values", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{
			"userId":    "12345",
			"requestId": "abc-def",
			"count":     42,
		}

		multiCtx := WithMultipleValues(ctx, vals)

		assert.Equal(t, "12345", multiCtx.Value("userId"))
		assert.Equal(t, "abc-def", multiCtx.Value("requestId"))
		assert.Equal(t, 42, multiCtx.Value("count"))
	})

	t.Run("returns nil for missing key", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{"key1": "value1"}

		multiCtx := WithMultipleValues(ctx, vals)

		assert.Nil(t, multiCtx.Value("nonexistent"))
	})

	t.Run("works with empty map", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{}

		multiCtx := WithMultipleValues(ctx, vals)

		assert.NotNil(t, multiCtx)
		assert.Nil(t, multiCtx.Value("anyKey"))
	})

	t.Run("panics on nil parent context", func(t *testing.T) {
		t.Parallel()

		vals := map[string]any{"key": "value"}

		assert.Panics(t, func() {
			WithMultipleValues[string](nil, vals) //nolint:staticcheck // Testing nil context behavior
		})
	})

	t.Run("panics on nil vals map", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()

		assert.Panics(t, func() {
			WithMultipleValues[string](ctx, nil)
		})
	})

	t.Run("supports integer keys", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[int]any{
			1: "first",
			2: "second",
			3: "third",
		}

		multiCtx := WithMultipleValues(ctx, vals)

		assert.Equal(t, "first", multiCtx.Value(1))
		assert.Equal(t, "second", multiCtx.Value(2))
		assert.Equal(t, "third", multiCtx.Value(3))
	})

	t.Run("supports custom key types", func(t *testing.T) {
		t.Parallel()

		type customKey struct {
			id   int
			name string
		}

		ctx := t.Context()
		key1 := customKey{id: 1, name: "alpha"}
		key2 := customKey{id: 2, name: "beta"}

		vals := map[customKey]any{
			key1: "value1",
			key2: "value2",
		}

		multiCtx := WithMultipleValues(ctx, vals)

		assert.Equal(t, "value1", multiCtx.Value(key1))
		assert.Equal(t, "value2", multiCtx.Value(key2))
	})

	t.Run("supports various value types", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Name string
			Age  int
		}

		ctx := t.Context()
		ptr := &testStruct{Name: "Alice", Age: 30}

		vals := map[string]any{
			"string":  "hello",
			"int":     42,
			"bool":    true,
			"struct":  testStruct{Name: "Bob", Age: 25},
			"pointer": ptr,
			"nil":     nil,
		}

		multiCtx := WithMultipleValues(ctx, vals)

		assert.Equal(t, "hello", multiCtx.Value("string"))
		assert.Equal(t, 42, multiCtx.Value("int"))
		assert.Equal(t, true, multiCtx.Value("bool"))
		assert.Equal(t, testStruct{Name: "Bob", Age: 25}, multiCtx.Value("struct"))
		assert.Equal(t, ptr, multiCtx.Value("pointer"))
		assert.Nil(t, multiCtx.Value("nil"))
	})
}

func TestMultiValueCtxValue(t *testing.T) {
	t.Parallel()

	t.Run("local values take precedence over parent", func(t *testing.T) {
		t.Parallel()

		parent := context.WithValue(t.Context(), "key", "parentValue") //nolint:staticcheck
		vals := map[string]any{"key": "localValue"}

		multiCtx := WithMultipleValues(parent, vals)

		assert.Equal(t, "localValue", multiCtx.Value("key"))
	})

	t.Run("falls back to parent context for missing keys", func(t *testing.T) {
		t.Parallel()

		parent := context.WithValue(t.Context(), "parentKey", "parentValue") //nolint:staticcheck
		vals := map[string]any{"localKey": "localValue"}

		multiCtx := WithMultipleValues(parent, vals)

		assert.Equal(t, "localValue", multiCtx.Value("localKey"))
		assert.Equal(t, "parentValue", multiCtx.Value("parentKey"))
	})

	t.Run("returns nil for keys not in local or parent", func(t *testing.T) {
		t.Parallel()

		parent := context.WithValue(t.Context(), "parentKey", "parentValue") //nolint:staticcheck
		vals := map[string]any{"localKey": "localValue"}

		multiCtx := WithMultipleValues(parent, vals)

		assert.Nil(t, multiCtx.Value("nonexistent"))
	})

	t.Run("handles type mismatch gracefully", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{"key": "value"}

		multiCtx := WithMultipleValues(ctx, vals)

		// Asking for an int key when we have string keys should return nil
		assert.Nil(t, multiCtx.Value(42))
	})

	t.Run("works with nested multi-value contexts", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals1 := map[string]any{"key1": "value1"}
		vals2 := map[string]any{"key2": "value2"}

		multiCtx1 := WithMultipleValues(ctx, vals1)
		multiCtx2 := WithMultipleValues(multiCtx1, vals2)

		assert.Equal(t, "value1", multiCtx2.Value("key1"))
		assert.Equal(t, "value2", multiCtx2.Value("key2"))
	})
}

func TestMultiValueCtxString(t *testing.T) {
	t.Parallel()

	t.Run("formats empty map correctly", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{}

		multiCtx := WithMultipleValues(ctx, vals)

		str := fmt.Sprintf("%v", multiCtx)
		assert.Contains(t, str, "WithMultipleValues()")
		assert.Contains(t, str, "Background")
	})

	t.Run("formats single value", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{"userId": "12345"}

		multiCtx := WithMultipleValues(ctx, vals)

		str := fmt.Sprintf("%v", multiCtx)
		assert.Contains(t, str, "WithMultipleValues(")
		assert.Contains(t, str, "userId=12345")
		assert.Contains(t, str, "Background")
	})

	t.Run("formats multiple values", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{
			"userId":    "12345",
			"requestId": "abc-def",
		}

		multiCtx := WithMultipleValues(ctx, vals)

		str := fmt.Sprintf("%v", multiCtx)
		assert.Contains(t, str, "WithMultipleValues(")
		assert.Contains(t, str, "userId=12345")
		assert.Contains(t, str, "requestId=abc-def")
		assert.Contains(t, str, "Background")
	})

	t.Run("includes parent context name", func(t *testing.T) {
		t.Parallel()

		parent := context.WithValue(t.Context(), "parentKey", "parentValue") //nolint:staticcheck
		vals := map[string]any{"localKey": "localValue"}

		multiCtx := WithMultipleValues(parent, vals)

		str := fmt.Sprintf("%v", multiCtx)
		assert.Contains(t, str, "WithMultipleValues(")
		assert.Contains(t, str, "localKey=localValue")
	})
}

func TestStringify(t *testing.T) {
	t.Parallel()

	t.Run("handles string values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "hello", stringify("hello"))
	})

	t.Run("handles nil values", func(t *testing.T) {
		t.Parallel()

		assert.Equal(t, "<nil>", stringify(nil))
	})

	t.Run("handles values with String method", func(t *testing.T) {
		t.Parallel()

		ts := testStringer{value: "custom-string"}
		assert.Equal(t, "custom-string", stringify(ts))
	})

	t.Run("handles types without String method", func(t *testing.T) {
		t.Parallel()

		result := stringify(42)
		assert.Equal(t, "int", result)

		result = stringify(true)
		assert.Equal(t, "bool", result)
	})

	t.Run("handles pointer types", func(t *testing.T) {
		t.Parallel()

		type testStruct struct {
			Value int
		}

		ptr := &testStruct{Value: 42}
		result := stringify(ptr)
		assert.Contains(t, result, "testStruct")
	})
}

func TestContextName(t *testing.T) {
	t.Parallel()

	t.Run("returns String for context with String method", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		name := contextName(ctx)
		assert.Contains(t, name, "Background")
	})

	t.Run("returns type name for context without String method", func(t *testing.T) {
		t.Parallel()

		type customCtx struct {
			context.Context //nolint:containedctx
		}

		ctx := &customCtx{Context: t.Context()}
		name := contextName(ctx)
		assert.Contains(t, name, "customCtx")
	})
}

func TestWithMultipleValuesIntegration(t *testing.T) {
	t.Parallel()

	t.Run("works with context cancellation", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		vals := map[string]any{"key": "value"}

		multiCtx := WithMultipleValues(ctx, vals)

		// Should be able to retrieve value before cancellation
		assert.Equal(t, "value", multiCtx.Value("key"))

		cancel()

		// Should still be able to retrieve value after cancellation
		assert.Equal(t, "value", multiCtx.Value("key"))

		// But context should be canceled
		select {
		case <-multiCtx.Done():
			// Expected
		default:
			t.Error("expected context to be canceled")
		}
	})

	t.Run("preserves cancellation from parent", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		vals := map[string]any{"key": "value"}

		multiCtx := WithMultipleValues(ctx, vals)

		cancel()

		assert.Error(t, multiCtx.Err())
	})

	t.Run("complex nested context chain", func(t *testing.T) {
		t.Parallel()

		// Build a complex context chain
		ctx := t.Context()
		ctx = context.WithValue(ctx, "base", "baseValue") //nolint:staticcheck

		vals1 := map[string]any{"layer1": "value1"}
		ctx = WithMultipleValues(ctx, vals1)

		ctx = context.WithValue(ctx, "middle", "middleValue") //nolint:staticcheck

		vals2 := map[string]any{"layer2": "value2"}
		ctx = WithMultipleValues(ctx, vals2)

		// All values should be accessible
		assert.Equal(t, "baseValue", ctx.Value("base"))
		assert.Equal(t, "value1", ctx.Value("layer1"))
		assert.Equal(t, "middleValue", ctx.Value("middle"))
		assert.Equal(t, "value2", ctx.Value("layer2"))
	})

	t.Run("string representation with nested contexts", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals1 := map[string]any{"outer": "value1"}
		ctx = WithMultipleValues(ctx, vals1)

		vals2 := map[string]any{"inner": "value2"}
		ctx = WithMultipleValues(ctx, vals2)

		str := fmt.Sprintf("%v", ctx)

		// Should contain both WithMultipleValues calls
		assert.Equal(t, 2, strings.Count(str, "WithMultipleValues"))
		assert.Contains(t, str, "inner=value2")
	})
}

func TestWithMultipleValuesConcurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent reads are safe", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		vals := map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		}

		multiCtx := WithMultipleValues(ctx, vals)

		// Spawn multiple goroutines reading concurrently
		done := make(chan bool)

		for range 10 {
			go func() {
				for range 100 {
					_ = multiCtx.Value("key1")
					_ = multiCtx.Value("key2")
					_ = multiCtx.Value("key3")
				}
				done <- true
			}()
		}

		// Wait for all goroutines to complete
		for range 10 {
			<-done
		}
	})
}

func BenchmarkWithMultipleValues(b *testing.B) {
	ctx := b.Context()
	vals := map[string]any{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
		"key4": "value4",
		"key5": "value5",
	}

	b.ResetTimer()

	for range b.N {
		multiCtx := WithMultipleValues(ctx, vals)
		_ = multiCtx.Value("key3")
	}
}

func BenchmarkWithMultipleValuesVsChainedWithValue(b *testing.B) {
	ctx := b.Context()

	b.Run("WithMultipleValues", func(b *testing.B) {
		vals := map[string]any{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		}

		b.ResetTimer()

		for range b.N {
			multiCtx := WithMultipleValues(ctx, vals)
			_ = multiCtx.Value("key3")
		}
	})

	b.Run("ChainedWithValue", func(b *testing.B) {
		b.ResetTimer()

		for range b.N {
			chainedCtx := context.WithValue(ctx, "key1", "value1")       //nolint:staticcheck
			chainedCtx = context.WithValue(chainedCtx, "key2", "value2") //nolint:staticcheck
			chainedCtx = context.WithValue(chainedCtx, "key3", "value3") //nolint:staticcheck
			chainedCtx = context.WithValue(chainedCtx, "key4", "value4") //nolint:staticcheck
			chainedCtx = context.WithValue(chainedCtx, "key5", "value5") //nolint:staticcheck
			_ = chainedCtx.Value("key3")
		}
	})
}
