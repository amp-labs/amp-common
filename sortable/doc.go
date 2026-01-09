// Package sortable provides wrapper types for primitive types that implement
// the Sortable interface, enabling their use as keys in sorted data structures.
//
// # Overview
//
// The sortable package defines the [Sortable] interface and provides ready-to-use
// implementations for common primitive types: [Int], [Byte], and [String].
// These types are designed to work with sorted collections like red-black trees
// (see [github.com/amp-labs/amp-common/set.NewRedBlackTreeSet] and
// [github.com/amp-labs/amp-common/maps.NewRedBlackTreeMap]).
//
// The Sortable interface extends [github.com/amp-labs/amp-common/compare.Comparable]
// by adding a LessThan method, providing both equality comparison and ordering.
//
// # Usage
//
// Use the provided wrapper types when you need sorted collections:
//
//	// Create a sorted set of integers
//	intSet := set.NewRedBlackTreeSet[sortable.Int]()
//	intSet.Add(sortable.Int(42))
//	intSet.Add(sortable.Int(10))
//	intSet.Add(sortable.Int(25))
//
//	// Elements are returned in sorted order: 10, 25, 42
//	for val := range intSet.Seq() {
//	    fmt.Println(int(val))
//	}
//
// # Creating Custom Sortable Types
//
// To create a custom sortable type, implement the Sortable interface:
//
//	type MyType struct {
//	    Priority int
//	    Name     string
//	}
//
//	func (m MyType) Equals(other MyType) bool {
//	    return m.Priority == other.Priority && m.Name == other.Name
//	}
//
//	func (m MyType) LessThan(other MyType) bool {
//	    if m.Priority != other.Priority {
//	        return m.Priority < other.Priority
//	    }
//	    return m.Name < other.Name
//	}
//
// # When to Use Sortable vs Collectable
//
// Use Sortable types when:
//   - You need elements in sorted order
//   - You want O(log n) lookup, insertion, and deletion
//   - The collection supports range queries or ordered iteration
//
// Use [github.com/amp-labs/amp-common/collectable.Collectable] types when:
//   - Order doesn't matter
//   - You want O(1) average-case lookup (hash-based)
//   - The type naturally supports hashing
//
// # Thread Safety
//
// The wrapper types in this package are value types and are inherently thread-safe
// for read operations. However, collections using these types (like red-black trees)
// may not be thread-safe and require external synchronization for concurrent access.
package sortable
