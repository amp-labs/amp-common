package maps

import "github.com/amp-labs/amp-common/collectable"

// KeyValuePair is a generic key-value pair struct used to represent entries in maps.
// It's particularly used by the OrderedMap.Seq() method to provide both the key and value
// in a single return value, along with an index to indicate insertion order.
//
// The Key must implement the collectable.Collectable interface (hashable and comparable),
// while the Value can be any type.
//
// Example:
//
//	// Returned by OrderedMap iteration
//	for i, entry := range orderedMap.Seq() {
//	    fmt.Printf("Index: %d, Key: %v, Value: %v\n", i, entry.Key, entry.Value)
//	}
type KeyValuePair[K collectable.Collectable[K], V any] struct {
	Key   K
	Value V
}
