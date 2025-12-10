// Package sortable provides sortable wrapper types for primitive types to implement comparison interfaces.
package sortable

import (
	"github.com/amp-labs/amp-common/compare"
)

type Sortable[T any] interface {
	compare.Comparable[T]

	LessThan(other T) bool
}
