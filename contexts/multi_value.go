package contexts

import (
	"context"
	"fmt"
	"reflect"
	"strings"
)

// WithMultipleValues attaches multiple key-value pairs to a context efficiently.
//
// This function is optimized for cases where you need to attach many values to a context
// at once. Instead of creating a deep chain of contexts (one per value), it stores all
// values in a single context wrapper, keeping the context tree shallow and improving
// performance for Value() lookups.
//
// Type parameter Key must be comparable (can be used as a map key). Common choices are
// string, int, or custom types.
//
// The function panics if parent is nil or if vals is nil. An empty map is allowed and
// will create a valid (though useless) context.
//
// Example:
//
//	ctx := context.Background()
//	vals := map[string]any{
//	    "userId": "12345",
//	    "requestId": "abc-def",
//	    "trace": true,
//	}
//	ctx = contexts.WithMultipleValues(ctx, vals)
//	// Later: retrieve values
//	userId := ctx.Value("userId").(string)
func WithMultipleValues[Key comparable](parent context.Context, vals map[Key]any) context.Context {
	if parent == nil {
		panic("cannot create context from nil parent")
	}

	if vals == nil {
		panic("nil vals passed to WithMultiValue")
	}

	return &multiValueCtx[Key]{parent, vals}
}

// multiValueCtx is a context wrapper that stores multiple key-value pairs efficiently.
//
// It embeds the parent context and adds a map of values. When Value() is called, it first
// checks the local map, then delegates to the parent context if the key is not found.
//
// This implementation keeps the context tree shallow compared to chaining multiple
// context.WithValue calls, which improves lookup performance and reduces memory overhead.
type multiValueCtx[Key comparable] struct {
	context.Context //nolint:containedctx

	vals map[Key]any
}

// stringify converts a value to a human-readable string for debugging and logging.
//
// It handles several cases:
//   - If the value implements fmt.Stringer, use its String() method
//   - If the value is already a string, return it directly
//   - If the value is nil, return "<nil>"
//   - Otherwise, return the type name (e.g., "int", "*User")
//
// This is used by the String() method to create readable debug output.
func stringify(v any) string {
	switch s := v.(type) {
	case fmt.Stringer:
		return s.String()
	case string:
		return s
	case nil:
		return "<nil>"
	}

	return reflect.TypeOf(v).String()
}

// contextName returns a human-readable name for a context.
//
// If the context implements fmt.Stringer, it uses the String() method to get a
// descriptive name (e.g., "context.Background.WithValue(...)"). Otherwise, it
// falls back to the type name (e.g., "*context.valueCtx").
//
// This is used by the String() method to build a hierarchical representation of
// the context chain for debugging.
func contextName(c context.Context) string {
	if s, ok := c.(fmt.Stringer); ok {
		return s.String()
	}

	return reflect.TypeOf(c).String()
}

// String returns a human-readable representation of the context for debugging.
//
// The format shows the parent context followed by the key-value pairs stored in
// this wrapper:
//
//	context.Background.WithMultipleValues(userId=12345, requestId=abc-def)
//
// If the map is empty, it returns:
//
//	context.Background.WithMultipleValues()
//
// Note: The order of key-value pairs is non-deterministic due to map iteration.
func (c *multiValueCtx[T]) String() string {
	if len(c.vals) == 0 {
		return contextName(c.Context) + ".WithMultipleValues()"
	}

	var builder strings.Builder

	builder.WriteString(contextName(c.Context))
	builder.WriteString(".WithMultipleValues(")

	first := true
	for k, v := range c.vals {
		if !first {
			builder.WriteString(", ")
		}

		first = false

		builder.WriteString(stringify(k))
		builder.WriteString("=")
		builder.WriteString(stringify(v))
	}

	builder.WriteString(")")

	return builder.String()
}

// Value retrieves a value from the context by key.
//
// It implements the context.Context interface's Value method. The lookup strategy is:
//
//  1. Check if the key's type is exactly T (the generic Key type parameter)
//  2. If so, look up the key in the local vals map
//  3. If found in the map, return the value
//  4. If not found locally, delegate to the parent context
//
// This delegation pattern ensures that values from parent contexts remain accessible,
// while local values take precedence when keys conflict.
//
// Note: This uses strict type checking - the key must be exactly type T, not just
// convertible to T. This respects the context contract and prevents type confusion.
//
// Example:
//
//	ctx := context.WithValue(context.Background(), "parentKey", "parentValue")
//	vals := map[string]any{"localKey": "localValue"}
//	ctx = contexts.WithMultipleValues(ctx, vals)
//	fmt.Println(ctx.Value("localKey"))   // "localValue" (from map)
//	fmt.Println(ctx.Value("parentKey"))  // "parentValue" (from parent)
func (c *multiValueCtx[T]) Value(key any) any {
	if c.vals != nil {
		// Check if key's type is exactly T (not just convertible to T)
		if reflect.TypeOf(key) == reflect.TypeFor[T]() {
			//nolint:forcetypeassert
			typedKey := key.(T) // Safe because we verified the exact type

			v, found := c.vals[typedKey]
			if found {
				return v
			}
		}
	}

	return c.Context.Value(key)
}
