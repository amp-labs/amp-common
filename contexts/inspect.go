package contexts

import (
	"context"
	"fmt"
	"reflect"
	"unsafe"
)

// ContextField represents a single field within a context, including its name,
// type, and string representation of its value.
type ContextField struct {
	Name  string `json:"name,omitempty"`
	Type  string `json:"type,omitempty"`
	Value string `json:"value"`
}

// ContextNode represents a node in the context chain, capturing the package and
// struct type information, along with any parent contexts and fields stored in this context.
type ContextNode struct {
	Package string `json:"package,omitempty"`
	Struct  string `json:"struct,omitempty"`

	Fields  []*ContextField `json:"fields,omitempty"`
	Parents []*ContextNode  `json:"parents,omitempty"`
}

// InspectContext uses reflection to introspect the internal structure of a context,
// returning a tree of ContextNodes representing the context chain and its values.
// Returns nil if ctx is nil.
func InspectContext(ctx context.Context) *ContextNode {
	if ctx == nil {
		return nil
	}

	return getContextInternals(ctx)
}

// getContextInternals recursively extracts the internal structure of a context using
// unsafe reflection. It traverses the context chain by following parent context fields
// and captures all non-context fields as ContextField values.
func getContextInternals(ctx any) *ContextNode {
	// Get the reflect value and type
	val := reflect.ValueOf(ctx)
	typ := reflect.TypeOf(ctx)

	// Handle both pointer and non-pointer types
	// Some contexts like emptyCtx are not pointers
	isPointer := val.Kind() == reflect.Ptr
	if isPointer {
		val = val.Elem()
		typ = typ.Elem()
	}

	contextValues := val
	contextKeys := typ

	node := &ContextNode{
		Package: contextKeys.PkgPath(),
		Struct:  contextKeys.Name(),
	}

	// For non-pointer types (like emptyCtx), fields are not addressable
	// emptyCtx has no fields anyway, so we can skip field iteration
	if !isPointer {
		// Non-pointer context types like emptyCtx have no accessible fields
		return node
	}

	if contextKeys.Kind() == reflect.Struct {
		for i := 0; i < contextValues.NumField(); i++ {
			reflectValue := contextValues.Field(i)
			reflectValue = reflect.NewAt(reflectValue.Type(), unsafe.Pointer(reflectValue.UnsafeAddr())).Elem()

			reflectField := contextKeys.Field(i)

			// Handle embedded (anonymous) struct fields by extracting their contents
			if reflectField.Anonymous && reflectField.Type.Kind() == reflect.Struct {
				// Recursively extract the embedded struct's contents
				// The reflectValue is already addressable and has unexported fields accessible via unsafe
				// Pass the address of the embedded struct
				embeddedNode := getContextInternals(reflectValue.Addr().Interface())
				// Merge the embedded struct's parents and fields into the current node
				node.Parents = append(node.Parents, embeddedNode.Parents...)
				node.Fields = append(node.Fields, embeddedNode.Fields...)
			} else if reflectField.Type.AssignableTo(reflect.TypeFor[context.Context]()) {
				parent := getContextInternals(reflectValue.Interface())

				node.Parents = append(node.Parents, parent)
			} else {
				field := &ContextField{
					Name:  fmt.Sprintf("%+v", reflectField.Name),
					Type:  reflectField.Type.String(),
					Value: fmt.Sprintf("%+v", reflectValue.Interface()),
				}

				node.Fields = append(node.Fields, field)
			}
		}
	}

	return node
}
