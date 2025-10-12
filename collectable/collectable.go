package collectable

import (
	"github.com/amp-labs/amp-common/compare"
	"github.com/amp-labs/amp-common/hashing"
)

// Collectable is an interface that combines the Hashable and
// Comparable interfaces. This is useful for objects that need
// to be stored in a Map or Set, where uniqueness is determined by
// the hashing value, and collisions are resolved by comparing
// the objects.
type Collectable[T any] interface {
	hashing.Hashable
	compare.Comparable[T]
}
