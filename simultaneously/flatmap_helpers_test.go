package simultaneously

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sizedMap is the subset of Map/OrderedMap behavior the empty-output tests need.
type sizedMap interface {
	Size() int
}

// assertFlatMapIsEmpty runs a FlatMap variant whose transform yields empty maps
// and asserts the result is a non-nil, empty map. Shared by the Map and
// OrderedMap tests, which differ only in the map type they flatten into.
func assertFlatMapIsEmpty[M sizedMap](t *testing.T, flatten func() (M, error)) {
	t.Helper()

	output, err := flatten()
	require.NoError(t, err)
	require.NotNil(t, output)
	assert.Equal(t, 0, output.Size())
}
