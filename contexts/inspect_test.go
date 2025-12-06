package contexts

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInspectContext(t *testing.T) {
	t.Parallel()

	t.Run("returns nil for nil context", func(t *testing.T) {
		t.Parallel()

		result := InspectContext(nil) //nolint:staticcheck
		assert.Nil(t, result)
	})

	t.Run("inspects background context", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background() //nolint:usetesting
		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "context", result.Package)
		assert.Equal(t, "backgroundCtx", result.Struct)
		assert.Empty(t, result.Parents)
		assert.Empty(t, result.Fields)
	})

	t.Run("inspects TODO context", func(t *testing.T) {
		t.Parallel()

		ctx := context.TODO() //nolint:usetesting
		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "context", result.Package)
		assert.Equal(t, "todoCtx", result.Struct)
		assert.Empty(t, result.Parents)
		assert.Empty(t, result.Fields)
	})

	t.Run("inspects context with single value", func(t *testing.T) {
		t.Parallel()

		ctx := context.WithValue(context.Background(), "key", "value") //nolint:staticcheck,usetesting
		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "context", result.Package)
		assert.Equal(t, "valueCtx", result.Struct)

		// Should have a parent (Background context)
		require.Len(t, result.Parents, 1)
		assert.Equal(t, "backgroundCtx", result.Parents[0].Struct)

		// Should have fields (key and val)
		require.NotEmpty(t, result.Fields)

		hasKey := false
		hasVal := false

		for _, field := range result.Fields {
			if field.Name == "key" {
				hasKey = true

				assert.Equal(t, "key", field.Value)
			}

			if field.Name == "val" {
				hasVal = true

				assert.Equal(t, "value", field.Value)
			}
		}

		assert.True(t, hasKey, "should have key field")
		assert.True(t, hasVal, "should have val field")
	})

	t.Run("inspects context with multiple values", func(t *testing.T) {
		t.Parallel()

		ctx := t.Context()
		ctx = context.WithValue(ctx, "key1", "value1") //nolint:staticcheck
		ctx = context.WithValue(ctx, "key2", 42)       //nolint:staticcheck
		ctx = context.WithValue(ctx, "key3", true)     //nolint:staticcheck

		result := InspectContext(ctx)

		require.NotNil(t, result)

		// Should have nested structure with multiple parents
		current := result
		depth := 0

		for current != nil && depth < 10 {
			depth++

			if len(current.Parents) > 0 {
				current = current.Parents[0]
			} else {
				break
			}
		}

		// Should have at least 3 levels (3 WithValue calls + Background)
		assert.GreaterOrEqual(t, depth, 3)
	})

	t.Run("inspects cancelled context", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "context", result.Package)
		assert.Equal(t, "cancelCtx", result.Struct)

		// Should have a parent (Background context)
		require.NotEmpty(t, result.Parents)
	})

	t.Run("inspects context with deadline", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithDeadline(t.Context(), time.Now().Add(1*time.Hour))
		defer cancel()

		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "context", result.Package)
		assert.Equal(t, "timerCtx", result.Struct)

		// Should have a parent
		// Fields may or may not be present depending on internal implementation
		// Just verify we got the structure
		require.NotEmpty(t, result.Parents)
	})

	t.Run("inspects context with timeout", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithTimeout(t.Context(), 1*time.Hour)
		defer cancel()

		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "context", result.Package)
		assert.Equal(t, "timerCtx", result.Struct)

		// Should have a parent
		// Fields may or may not be present depending on internal implementation
		require.NotEmpty(t, result.Parents)
	})

	t.Run("inspects complex nested context", func(t *testing.T) {
		t.Parallel()

		// Create a complex context chain
		ctx := t.Context()
		ctx = context.WithValue(ctx, "user", "alice") //nolint:staticcheck

		ctx, cancel := context.WithTimeout(ctx, 1*time.Hour)
		defer cancel()

		ctx = context.WithValue(ctx, "request_id", "12345") //nolint:staticcheck

		result := InspectContext(ctx)

		require.NotNil(t, result)

		// Walk the tree and verify it has multiple levels
		var nodeCount int
		var countNodes func(*ContextNode)
		countNodes = func(node *ContextNode) {
			if node == nil {
				return
			}

			nodeCount++

			for _, parent := range node.Parents {
				countNodes(parent)
			}
		}

		countNodes(result)
		assert.Greater(t, nodeCount, 1, "should have multiple nodes in context chain")
	})

	t.Run("inspects context with custom key type", func(t *testing.T) {
		t.Parallel()

		type contextKey string

		key := contextKey("customKey")

		ctx := context.WithValue(t.Context(), key, "customValue") //nolint:staticcheck
		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "valueCtx", result.Struct)

		// Should have fields
		require.NotEmpty(t, result.Fields)
	})

	t.Run("inspects context with struct value", func(t *testing.T) {
		t.Parallel()

		type user struct {
			Name string
			Age  int
		}

		u := user{Name: "Bob", Age: 30}
		ctx := context.WithValue(t.Context(), "user", u) //nolint:staticcheck

		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "valueCtx", result.Struct)

		// Should have fields with the struct value
		require.NotEmpty(t, result.Fields)
	})

	t.Run("inspects context with pointer value", func(t *testing.T) {
		t.Parallel()

		type data struct {
			Value int
		}

		d := &data{Value: 42}
		ctx := context.WithValue(t.Context(), "data", d) //nolint:staticcheck

		result := InspectContext(ctx)

		require.NotNil(t, result)
		assert.Equal(t, "valueCtx", result.Struct)
		require.NotEmpty(t, result.Fields)
	})
}

func TestContextNode(t *testing.T) {
	t.Parallel()

	t.Run("has expected structure", func(t *testing.T) {
		t.Parallel()

		node := &ContextNode{
			Package: "testpkg",
			Struct:  "testStruct",
			Parents: []*ContextNode{
				{
					Package: "parentpkg",
					Struct:  "parentStruct",
				},
			},
			Fields: []*ContextField{
				{
					Name:  "field1",
					Type:  "string",
					Value: "value1",
				},
			},
		}

		assert.Equal(t, "testpkg", node.Package)
		assert.Equal(t, "testStruct", node.Struct)
		require.Len(t, node.Parents, 1)
		assert.Equal(t, "parentpkg", node.Parents[0].Package)
		require.Len(t, node.Fields, 1)
		assert.Equal(t, "field1", node.Fields[0].Name)
	})
}

func TestContextField(t *testing.T) {
	t.Parallel()

	t.Run("has expected structure", func(t *testing.T) {
		t.Parallel()

		field := &ContextField{
			Name:  "testField",
			Type:  "int",
			Value: "42",
		}

		assert.Equal(t, "testField", field.Name)
		assert.Equal(t, "int", field.Type)
		assert.Equal(t, "42", field.Value)
	})
}
