package maps_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cloneable is the subset of Map/OrderedMap behavior that the independent-copy
// tests exercise. Self is the concrete map interface, so Clone stays typed.
type cloneable[Self any] interface {
	Add(key testKey, value int) error
	Size() int
	Clone() Self
}

// assertCloneIsIndependent verifies that a clone does not observe mutations made
// to the original after cloning. Shared by the DefaultMap and DefaultOrderedMap
// tests, which differ only in the map implementation under test.
func assertCloneIsIndependent[Self cloneable[Self]](t *testing.T, original Self) {
	t.Helper()

	require.NoError(t, original.Add(testKey{value: "a"}, 1))
	require.NoError(t, original.Add(testKey{value: "b"}, 2))

	clone := original.Clone()
	assert.Equal(t, 2, clone.Size())

	// Mutating the original must leave the clone untouched.
	require.NoError(t, original.Add(testKey{value: "c"}, 3))

	assert.Equal(t, 3, original.Size())
	assert.Equal(t, 2, clone.Size())
}
