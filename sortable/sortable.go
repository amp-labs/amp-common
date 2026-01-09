package sortable

import (
	"github.com/amp-labs/amp-common/compare"
)

// Sortable is a generic interface for types that support both equality comparison
// and ordering. It extends [compare.Comparable] by adding the LessThan method,
// which defines a total ordering on values of type T.
//
// Types implementing Sortable can be used as keys in sorted data structures
// such as red-black trees. The ordering must satisfy the following properties:
//
//   - Antisymmetric: if a.LessThan(b) is true, then b.LessThan(a) must be false
//   - Transitive: if a.LessThan(b) and b.LessThan(c), then a.LessThan(c)
//   - Total: for any a and b, exactly one of a.LessThan(b), a.Equals(b), or b.LessThan(a) is true
//
// The package provides ready-to-use implementations for common types:
// [Int], [Byte], and [String].
//
// Example implementation for a custom type:
//
//	type Person struct {
//	    Age  int
//	    Name string
//	}
//
//	func (p Person) Equals(other Person) bool {
//	    return p.Age == other.Age && p.Name == other.Name
//	}
//
//	func (p Person) LessThan(other Person) bool {
//	    if p.Age != other.Age {
//	        return p.Age < other.Age
//	    }
//	    return p.Name < other.Name
//	}
type Sortable[T any] interface {
	compare.Comparable[T]

	// LessThan returns true if the receiver is strictly less than the other value.
	// This method defines the ordering used by sorted collections.
	LessThan(other T) bool
}
