package maps

import "github.com/amp-labs/amp-common/collectable"

// KeyValuePair is a generic key-value pair struct.
type KeyValuePair[K collectable.Collectable[K], V any] struct {
	Key   K
	Value V
}
