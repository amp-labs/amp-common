package collectable

import (
	"errors"
	"fmt"
	"hash"

	"github.com/amp-labs/amp-common/compare"
	"github.com/amp-labs/amp-common/hashing"
)

// ErrUnsupportedType is returned when attempting to hash an unsupported type.
var ErrUnsupportedType = errors.New("unsupported type for hashing")

// Collectable is an interface that combines the Hashable and
// Comparable interfaces. This is useful for objects that need
// to be stored in a Map or Set, where uniqueness is determined by
// the hashing value, and collisions are resolved by comparing
// the objects.
type Collectable[T any] interface {
	hashing.Hashable
	compare.Comparable[T]
}

// comparableWrapper wraps a comparable value and implements Collectable[T].
type comparableWrapper[T comparable] struct {
	value T
}

// UpdateHash implements hashing.Hashable by delegating to the appropriate
// HashableX type based on the actual type of the value.
func (w *comparableWrapper[T]) UpdateHash(h hash.Hash) error { //nolint:varnamelen
	// Use type switch to delegate to the appropriate hashable type
	switch typedValue := any(w.value).(type) {
	case int:
		return hashing.HashableInt(typedValue).UpdateHash(h)
	case int8:
		return hashing.HashableInt8(typedValue).UpdateHash(h)
	case int16:
		return hashing.HashableInt16(typedValue).UpdateHash(h)
	case int32:
		return hashing.HashableInt32(typedValue).UpdateHash(h)
	case int64:
		return hashing.HashableInt64(typedValue).UpdateHash(h)
	case uint:
		return hashing.HashableUint(typedValue).UpdateHash(h)
	case uint8:
		return hashing.HashableUint8(typedValue).UpdateHash(h)
	case uint16:
		return hashing.HashableUint16(typedValue).UpdateHash(h)
	case uint32:
		return hashing.HashableUint32(typedValue).UpdateHash(h)
	case uint64:
		return hashing.HashableUint64(typedValue).UpdateHash(h)
	case float32:
		return hashing.HashableFloat32(typedValue).UpdateHash(h)
	case float64:
		return hashing.HashableFloat64(typedValue).UpdateHash(h)
	case string:
		return hashing.HashableString(typedValue).UpdateHash(h)
	case []byte:
		return hashing.HashableBytes(typedValue).UpdateHash(h)
	case bool:
		return hashing.HashableBool(typedValue).UpdateHash(h)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedType, typedValue)
	}
}

// Equals implements compare.Comparable[T] by using the == operator.
func (w *comparableWrapper[T]) Equals(other T) bool {
	return w.value == other
}

// FromComparable creates a Collectable[T] from any comparable value.
// It supports all numeric types, strings, byte slices, and booleans.
// For unsupported types, the UpdateHash method will return an error.
func FromComparable[T comparable](value T) Collectable[T] {
	return &comparableWrapper[T]{value: value}
}
